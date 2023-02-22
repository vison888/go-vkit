package gate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/grpcclient"
	"github.com/visonlv/go-vkit/grpcx"
	"github.com/visonlv/go-vkit/logger"
	meta "github.com/visonlv/go-vkit/metadata"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

const (
	pingPeriod     = 60 * time.Second
	maxMessageSize = 1024 * 1024 * 4
)

var (
	upgrader = websocket.Upgrader{
		//设置读缓冲区
		ReadBufferSize: 1024,
		//设置写缓冲区
		WriteBufferSize: 1024,
		//允许跨域访问
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
		//协商消息压缩
		EnableCompression: true,
	}
)

type WsHandler struct {
	auth     authFunc
	grpcPort int
}

type StreamContext struct {
	ctx       context.Context
	ctxCancel context.CancelFunc
	conn      *websocket.Conn
	stream    grpcx.ClientStream
	wsReadCh  chan []byte
	wsWriteCh chan *json.RawMessage
	closeLock *sync.Mutex
	isClose   bool
}

func NewWsHandler(grpcPort int, auth authFunc) *WsHandler {
	return &WsHandler{
		grpcPort: grpcPort,
		auth:     auth,
	}
}

func (h *WsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if re := recover(); re != nil {
			errorStr := fmt.Sprintf("[gate] WsHandler panic recovered:%v ", re)
			logger.Errorf(errorStr)
			logger.Error(string(debug.Stack()))
			ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		}
	}()

	if cerr := h.auth(w, r); cerr != nil {
		logger.Errorf("[gate] WsHandler authCheck Err url:%s err:%s", r.RequestURI, cerr.Error())
		ErrorResponse(w, r, cerr)
		return
	}

	conn, err := upgrader.Upgrade(w, r, r.Header)
	if err != nil {
		logger.Errorf("[gate] WsHandler Upgrade Err url:%s err:%s", r.RequestURI, err)
		ErrorResponse(w, r, neterrors.Forbidden(err.Error()))
		return
	}
	defer conn.Close()

	readCt := "application/json"

	if strings.ToLower(readCt) != "application/json" {
		logger.Errorf("[gate] WsHandler url:%s content-type not application/json", r.RequestURI)
		ErrorResponse(w, r, neterrors.Forbidden("content-type not application/json"))
	}

	// 将header转context
	requestId := strings.ReplaceAll(uuid.New().String(), "-", "")
	md := meta.Metadata{}
	md["x-content-type"] = readCt
	md["request_id"] = requestId
	for k, v := range r.Header {
		if k == "Connection" {
			continue
		}
		md[strings.ToLower(k)] = strings.Join(v, ",")
	}
	ctx := meta.NewContext(context.Background(), md)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	sc := &StreamContext{
		ctx:       ctx,
		ctxCancel: cancel,
		conn:      conn,
		wsReadCh:  make(chan []byte, 10),
		wsWriteCh: make(chan *json.RawMessage, 10),
		closeLock: new(sync.Mutex),
		isClose:   false,
	}

	if err = h.connectGrpcServer(sc, w, r); err != nil {
		return
	}
	h.readsAndWrites(sc)
}

func (h *WsHandler) Close(sc *StreamContext) {
	if sc.isClose {
		return
	}

	sc.closeLock.Lock()
	defer sc.closeLock.Unlock()
	if sc.isClose {
		return
	}
	sc.isClose = true

	sc.ctxCancel()
	sc.conn.Close()
	sc.stream.Close()
}

func (h *WsHandler) connectGrpcServer(sc *StreamContext, w http.ResponseWriter, r *http.Request) error {
	var service, endpoint string
	path := strings.Split(r.RequestURI, "/")
	if len(path) > 3 {
		service = path[2]
		endpoint = path[3]
	}

	if len(service) == 0 {
		return errors.New("service is empty")
	}

	if len(endpoint) == 0 {
		return errors.New("endpoint is empty")
	}

	// 连接grpc服务
	target := fmt.Sprintf("%s:%d", service, h.grpcPort)
	stream, netErr := grpcclient.StreamByGate(sc.ctx, target, service, endpoint)
	if netErr != nil {
		return fmt.Errorf("StreamByGate fail:%s", netErr.Error())
	}
	sc.stream = stream
	return nil
}

func (h *WsHandler) readsAndWrites(sc *StreamContext) {
	stopWsCtx, wsCancel := context.WithCancel(context.Background())
	stopGrpcCtx, grpcCancel := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}
	wg.Add(4)

	go h.wsRead(stopWsCtx, wsCancel, &wg, sc)
	go h.wsWrite(stopWsCtx, wsCancel, &wg, sc)
	go h.grpcRead(stopGrpcCtx, grpcCancel, &wg, sc)
	go h.grpcWrite(stopGrpcCtx, grpcCancel, &wg, sc)
	wg.Wait()
}

func (h *WsHandler) wsRead(stopCtx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, sc *StreamContext) {
	defer func() {
		cancel()
		wg.Done()
		h.Close(sc)
		logger.Info("defer wsRead")
	}()
	sc.conn.SetReadLimit(maxMessageSize)

	for {
		select {
		case <-stopCtx.Done():
			return
		default:
		}

		mt, msg, err := sc.conn.ReadMessage()
		if err != nil {
			logger.Errorf("[gate] WsHandler wsRead err: %+v", err)
			return
		}

		if mt == websocket.BinaryMessage {
			logger.Errorf("[gate] WsHandler wsRead not support websocket.BinaryMessage mt: %d", mt)
			return
		}
		sc.wsReadCh <- msg
	}
}

func (h *WsHandler) wsWrite(stopCtx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, sc *StreamContext) {
	defer func() {
		cancel()
		wg.Done()
		h.Close(sc)
		logger.Info("defer wsWrite")
	}()
	ticker := time.NewTicker(pingPeriod)
	defer ticker.Stop()
	for {
		select {
		case <-stopCtx.Done():
			return
		case <-sc.ctx.Done():
			return
		case <-sc.stream.Context().Done():
			return
		case <-ticker.C:
			if err := sc.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				logger.Errorf("[gate] WsHandler wsWrite conn.WriteMessage err: %+v", err)
				return
			}
		case msg := <-sc.wsWriteCh:
			respBytes, err := msg.MarshalJSON()
			if err != nil {
				logger.Errorf("[gate] WsHandler wsWrite msg.MarshalJSON err: %+v", err)
				return
			}

			if err := sc.conn.WriteMessage(websocket.TextMessage, respBytes); err != nil {
				logger.Errorf("[gate] WsHandler wsWrite conn.WriteMessage err: %+v", err)
				return
			}
		}
	}
}

func (h *WsHandler) grpcWrite(stopCtx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, sc *StreamContext) {
	defer func() {
		cancel()
		wg.Done()
		h.Close(sc)
		logger.Info("defer grpcWrite")
	}()

	for {
		select {
		case <-stopCtx.Done():
			return
		case msg := <-sc.wsReadCh:
			if err := sc.stream.Send(msg); err != nil {
				logger.Errorf("[gate] WsHandler grpcWrite stream.Send err: %+v", err)
				return
			}
		}
	}
}

func (h *WsHandler) grpcRead(stopCtx context.Context, cancel context.CancelFunc, wg *sync.WaitGroup, sc *StreamContext) {
	defer func() {
		cancel()
		wg.Done()
		h.Close(sc)
		logger.Info("defer grpcRead")
	}()

	for {
		select {
		case <-stopCtx.Done():
			return
		default:
		}

		data := &json.RawMessage{}
		err := sc.stream.Recv(data)
		if err != nil {
			logger.Errorf("[gate] WsHandler grpcRead stream.Recv err: %+v", err)
			return
		}
		sc.wsWriteCh <- data
	}

}
