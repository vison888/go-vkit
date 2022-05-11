package grpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/visonlv/go-vkit/codec"
	"github.com/visonlv/go-vkit/errors"
	"github.com/visonlv/go-vkit/errors/neterrors"
	"github.com/visonlv/go-vkit/logger"
	meta "github.com/visonlv/go-vkit/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
)

const (
	DefaultContentType    = "application/grpc"
	DefaultMaxRecvMsgSize = 1024 * 1024 * 16
	DefaultMaxSendMsgSize = 1024 * 1024 * 16
)

type handlerInfo struct {
	handler  reflect.Value
	method   reflect.Method
	reqType  reflect.Type
	respType reflect.Type
}

type grpcServer struct {
	srv *grpc.Server

	sync.RWMutex
	handlers map[string]*handlerInfo
}

func init() {
	encoding.RegisterCodec(codec.WrapCodec{codec.JsonCodec{}})
	encoding.RegisterCodec(codec.WrapCodec{codec.ProtoCodec{}})
}

func NewServer(opts ...grpc.ServerOption) *grpcServer {
	g := &grpcServer{
		handlers: make(map[string]*handlerInfo),
	}
	maxRecvMsgSize := DefaultMaxRecvMsgSize
	maxSendMsgSize := DefaultMaxSendMsgSize

	gopts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(maxRecvMsgSize),
		grpc.MaxSendMsgSize(maxSendMsgSize),
		grpc.UnknownServiceHandler(g.handler),
	}

	gopts = append(gopts, opts...)
	g.srv = grpc.NewServer(gopts...)
	reflection.Register(g.srv)
	return g
}

func serviceMethod(m string) (string, string, error) {
	if len(m) == 0 {
		return "", "", fmt.Errorf("[grpcserver] malformed method name: %q", m)
	}

	if m[0] == '/' {
		parts := strings.Split(m, "/")
		if len(parts) != 3 || len(parts[1]) == 0 || len(parts[2]) == 0 {
			return "", "", fmt.Errorf("[grpcserver] malformed method name: %q", m)
		}
		service := strings.Split(parts[1], ".")
		return service[len(service)-1], m, nil
	}

	parts := strings.Split(m, ".")

	if len(parts) != 2 {
		return "", "", fmt.Errorf("[grpcserver] malformed method name: %q", m)
	}

	return parts[0], m, nil
}

func (g *grpcServer) handler(srv interface{}, stream grpc.ServerStream) (err error) {
	defer func() {
		if r := recover(); r != nil {
			errorStr := fmt.Sprintf("[grpcserver] panic recovered:%v ", r)
			logger.Errorf(errorStr)
			logger.Error(string(debug.Stack()))
			err = neterrors.BadRequest(errorStr)
		}
	}()

	fullMethod, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		errorStr := "[grpcserver] method does not exist in context"
		logger.Errorf(errorStr)
		return neterrors.NotFound(errorStr)
	}

	// service, endpoint, err := serviceMethod(fullMethod)
	// if err != nil {
	// 	errorStr := fmt.Sprintf("[grpcserver] serviceMethod error: %v", err)
	// 	logger.Errorf(errorStr)
	// 	return neterrors.NotFound(errorStr)
	// }

	// if len(service) == 0 {
	// 	errorStr := fmt.Sprintf("[grpcserver] service not found name: %q", fullMethod)
	// 	logger.Errorf(errorStr)
	// 	return neterrors.NotFound(errorStr)
	// }

	// if len(endpoint) == 0 {
	// 	errorStr := fmt.Sprintf("[grpcserver] endpoint not found name: %q", fullMethod)
	// 	logger.Errorf(errorStr)
	// 	return neterrors.NotFound(errorStr)
	// }

	methodName := fullMethod

	gmd, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		gmd = metadata.MD{}
	}

	md := meta.Metadata{}
	for k, v := range gmd {
		md[k] = strings.Join(v, ", ")
	}

	to := md["timeout"]
	requestSq := md["request_sq"]
	requestId := md["request_id"]

	xct := DefaultContentType

	if ctype, ok := md["x-content-type"]; ok {
		xct = ctype
	} else {
		if ctype, ok := md["content-type"]; ok {
			xct = ctype
		}
	}
	ct := xct
	if ctype, ok := md["content-type"]; ok {
		ct = ctype
	}
	md["x-content-type"] = xct
	md["content-type"] = ct
	delete(md, "timeout")

	// create new context
	ctx := meta.NewContext(stream.Context(), md)

	// get peer from context
	if p, ok := peer.FromContext(stream.Context()); ok {
		md["Remote"] = p.Addr.String()
		ctx = peer.NewContext(ctx, p)
	}

	// set the timeout if we have it
	if len(to) > 0 {
		if n, err := strconv.ParseUint(to, 10, 64); err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(n))
			defer cancel()
		}
	}

	h, b := g.handlers[methodName]
	if !b {
		errorStr := fmt.Sprintf("[grpcserver] unknown method %s", methodName)
		logger.Errorf(errorStr)
		return neterrors.NotFound(errorStr)
	}

	return g.processRequest(stream, h, ct, xct, requestId, requestSq, methodName, ctx)
}

func (g *grpcServer) processRequest(stream grpc.ServerStream, h *handlerInfo, ct string, xct string, requestId string, requestSq string, methodName string, ctx context.Context) error {
	argv := reflect.New(h.reqType.Elem())
	replyv := reflect.New(h.respType.Elem())

	if cd := codec.DefaultGRPCCodecs[xct]; cd.Name() != "json" {
		if err := stream.RecvMsg(argv.Interface()); err != nil {
			errorStr := fmt.Sprintf("[grpcserver] RecvMsg error: %s", err.Error())
			logger.Errorf(errorStr)
			return neterrors.BadRequest(errorStr)
		}
	} else {
		var raw json.RawMessage
		if err := stream.RecvMsg(&raw); err != nil {
			errorStr := fmt.Sprintf("[grpcserver] RecvMsg error: %s", err.Error())
			logger.Errorf(errorStr)
			return neterrors.BadRequest(errorStr)
		}

		if err := cd.Unmarshal(raw, argv.Interface()); err != nil {
			errorStr := fmt.Sprintf("[grpcserver] Unmarshal error: %s", err.Error())
			logger.Errorf(errorStr)
			return neterrors.BadRequest(errorStr)
		}
	}

	// validate
	validateFunc, b := h.reqType.MethodByName("Validate")
	if b {
		out := validateFunc.Func.Call([]reflect.Value{reflect.ValueOf(argv.Interface())})
		errValue := out[0]
		if errValue.Interface() != nil {
			err := errValue.Interface().(error)
			errorStr := fmt.Sprintf("[grpcserver] requestId:%s requestSq:%s param error: %s", requestId, requestSq, err.Error())
			logger.Errorf(errorStr)
			return neterrors.BusinessError(-1, errorStr)
		}
	}

	in := make([]reflect.Value, 4)
	in[0] = h.handler
	in[1] = reflect.ValueOf(ctx)
	in[2] = argv
	in[3] = replyv
	out := h.method.Func.Call(in)
	if rerr := out[0].Interface(); rerr != nil {
		//处理业务异常
		if verr, ok := rerr.(*errors.Errno); ok {
			if verr.Code != 0 {
				errorStr := fmt.Sprintf("[grpcserver] requestId:%s requestSq:%s call error: %s", requestId, requestSq, verr.Error())
				logger.Errorf(errorStr)
				return neterrors.BusinessError(verr.GetFullCode(), verr.Msg)
			}
		} else {
			//其他异常统一包装
			errorStr := fmt.Sprintf("[grpcserver] requestId:%s requestSq:%s call error: %s", requestId, requestSq, rerr.(error).Error())
			logger.Errorf(errorStr)
			return neterrors.BusinessError(-2, errorStr)
		}

	}

	if err := stream.SendMsg(replyv.Interface()); err != nil {
		errorStr := fmt.Sprintf("[grpcserver] requestId:%s requestSq:%s send error: %s", requestId, requestSq, err.Error())
		logger.Errorf(errorStr)
		return neterrors.BusinessError(-2, errorStr)
	}

	if logger.CanServerLog(xct) {
		jsonArgv, _ := json.Marshal(argv.Interface())
		jsonReplyv, _ := json.Marshal(replyv.Interface())
		successStr := fmt.Sprintf("[grpcserver] handler success requestId:%s requestSq:%s methodName:%s argv:%s replyv:%s", requestId, requestSq, methodName, jsonArgv, jsonReplyv)
		logger.Info(successStr)
	}

	return nil

}

func (g *grpcServer) RegisterList(list []interface{}, urls map[string]string) (err error) {
	for _, v := range list {
		err := g.RegisterWithUrl(v, urls)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *grpcServer) RegisterWithUrl(i interface{}, urls map[string]string) (err error) {
	o := reflect.ValueOf(i)
	hType := o.Type()
	hName := hType.Elem().Name()
	mCount := hType.NumMethod()
	//反射方法
	for i := 0; i < mCount; i++ {
		m := hType.Method(i)
		pType1 := m.Type.In(2)
		pType2 := m.Type.In(3)
		handler := &handlerInfo{
			handler:  o,
			method:   m,
			reqType:  pType1,
			respType: pType2,
		}
		methodName := hName + "." + m.Name
		if urls != nil {
			path, b := urls[methodName]
			if b {
				methodName = path
			}
		}
		g.handlers[methodName] = handler
		logger.Infof("[grpcServer] Register methodName:[%v]", methodName)
	}
	return nil
}

func (g *grpcServer) Register(i interface{}) (err error) {
	return g.RegisterWithUrl(i, nil)
}

func (g *grpcServer) Run(listenPort string) {
	logger.Info("[grpcServer] Listen start port:[%s]", listenPort)

	lis, err := net.Listen("tcp", listenPort)
	if err != nil {
		logger.Error("[grpcServer] Listen failed e: %v", err.Error())
		return
	}

	if err := g.srv.Serve(lis); err != nil {
		logger.Error("failed to serve: %v", err)
	}

}
