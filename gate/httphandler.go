package gate

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/grpcclient"
	"github.com/visonlv/go-vkit/logger"
	meta "github.com/visonlv/go-vkit/metadata"
)

type HttpHandler struct {
	auth     authFunc
	grpcPort int
}

func NewHttpHandler(grpcPort int, auth authFunc) *HttpHandler {
	return &HttpHandler{
		grpcPort: grpcPort,
		auth:     auth,
	}
}

func (h *HttpHandler) Handle(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if re := recover(); re != nil {
			errorStr := fmt.Sprintf("[gate] HttpHandler panic recovered:%v ", re)
			logger.Errorf(errorStr)
			logger.Error(string(debug.Stack()))
			errorResponse(w, r, neterrors.BadRequest(errorStr))
		}
	}()

	method := strings.ToUpper(r.Method)
	if method == "OPTIONS" {
		return
	}

	if method != "POST" {
		errorStr := fmt.Sprintf("[gate] req method:%s not support url:%s", method, r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if cerr := h.auth(w, r); cerr != nil {
		errorResponse(w, r, cerr)
		return
	}

	reqBytes, err := requestPayload(r)
	if err != nil {
		errorStr := fmt.Sprintf("[gate] %s url:%s", err.Error(), r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	readCt := r.Header.Get("Content-Type")
	index := strings.Index(readCt, ";")
	if index != -1 {
		readCt = readCt[:index]
	}

	var service, endpoint string
	path := strings.Split(r.RequestURI, "/")
	if len(path) > 3 {
		service = path[2]
		endpoint = path[3]
	}

	if len(service) == 0 {
		errorStr := fmt.Sprintf("[gate] service is empty url:%s", r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if len(endpoint) == 0 {
		errorStr := fmt.Sprintf("[gate] endpoint is empty url:%s", r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	startTime := time.Now()

	requestId := strings.ReplaceAll(uuid.New().String(), "-", "")
	requestSq := 0
	md := meta.Metadata{}
	md["x-content-type"] = readCt
	md["request_id"] = requestId
	//插入header
	for k, v := range r.Header {
		if k == "Connection" {
			continue
		}
		md[strings.ToLower(k)] = strings.Join(v, ",")
	}
	ctx := meta.NewContext(context.Background(), md)

	target := fmt.Sprintf("%s:%d", service, h.grpcPort)
	jsonRaw, netErr := grpcclient.InvokeByGate(ctx, target, service, endpoint, reqBytes)
	if netErr != nil {
		logger.Infof("[gate] InvokeWithJson response netErr:%s", netErr)
		errorResponse(w, r, netErr)
		return
	}

	respBytes, err := jsonRaw.MarshalJSON()
	if err != nil {
		logger.Infof("[gate] jsonRaw MarshalJSON fail:%s", err)
		errorResponse(w, r, neterrors.BadRequest(err.Error()))
		return
	}

	costTime := time.Since(startTime)

	//success
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	w.Header().Set("Content-Length", strconv.Itoa(len(respBytes)))
	_, err = w.Write(respBytes)
	if err != nil {
		logger.Errorf("[gate] requestId:%s requestSq:%d response fail url:%v", requestId, requestSq, r.RequestURI)
		return
	}

	successStr := fmt.Sprintf("[gate] success requestId:%s requestSq:%d cost:[%v] url:[%v] req:[%v] resp:[%v]", requestId, requestSq, costTime.Milliseconds(), r.RequestURI, string(reqBytes), string(respBytes))
	logger.Infof(successStr)
}
