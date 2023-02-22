package gate

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/grpcclient"
	"github.com/visonlv/go-vkit/logger"
	meta "github.com/visonlv/go-vkit/metadata"
)

type GrpcHandler struct {
	opts HttpOptions
}

func NewGrpcHandler(opts ...HttpOption) *GrpcHandler {
	return &GrpcHandler{
		opts: newHttpOptions(opts...),
	}
}

func (h *GrpcHandler) Init(opts ...HttpOption) {
	for _, o := range opts {
		o(&h.opts)
	}
}

func (h *GrpcHandler) Handle(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if re := recover(); re != nil {
			if h.opts.ErrHandler != nil {
				h.opts.ErrHandler(w, r, re)
			}
		}
	}()

	method := strings.ToUpper(r.Method)
	if method == "OPTIONS" {
		return
	}

	if method != "POST" {
		errorStr := fmt.Sprintf("[gate] req method:%s not support url:%s", method, r.RequestURI)
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	// 鉴权
	if h.opts.AuthHandler != nil {
		if cerr := h.opts.AuthHandler(w, r); cerr != nil {
			ErrorResponse(w, r, cerr)
			return
		}
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
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if len(endpoint) == 0 {
		errorStr := fmt.Sprintf("[gate] endpoint is empty url:%s", r.RequestURI)
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	md := meta.Metadata{}
	md["x-content-type"] = readCt

	request := &HttpRequest{
		uri:         r.RequestURI,
		r:           r,
		service:     service,
		method:      method,
		contentType: readCt,
		body:        nil,
		hasRead:     false,
	}

	response := &HttpResponse{
		w:        w,
		header:   nil,
		hasWrite: false,
		content:  nil,
	}

	reqBytes, err := request.Read()
	if err != nil {
		errorStr := fmt.Sprintf("[gate] %s url:%s", err.Error(), r.RequestURI)
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	fullCtx := requestToContext(context.Background(), md, r)
	// 主逻辑
	fn := func(ctx context.Context, req *HttpRequest, resp *HttpResponse) error {
		target := fmt.Sprintf("%s:%d", service, h.opts.GrpcPort)
		jsonRaw, netErr := grpcclient.InvokeByGate(ctx, target, service, endpoint, reqBytes)
		if netErr != nil {
			logger.Infof("[gate] InvokeWithJson response netErr:%s", netErr)
			return netErr
		}

		respBytes, err := jsonRaw.MarshalJSON()
		if err != nil {
			logger.Infof("[gate] jsonRaw MarshalJSON fail:%s", err)
			return neterrors.BadRequest(err.Error())
		}

		resp.content = respBytes
		return nil
	}
	// 拦截器
	for i := len(h.opts.HdlrWrappers); i > 0; i-- {
		fn = h.opts.HdlrWrappers[i-1](fn)
	}

	resp := make([]byte, 0)
	if appErr := fn(fullCtx, request, response); appErr != nil {
		switch verr := appErr.(type) {
		case *neterrors.NetError:
			ErrorResponse(w, r, verr)
		default:
			ErrorResponse(w, r, neterrors.BadRequest(verr.Error()))
		}
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	w.Header().Set("Content-Length", strconv.Itoa(len(resp)))
	_, err = w.Write(resp)
	if err != nil {
		logger.Errorf("[gate] response fail url:%v respBytes:%s", r.RequestURI, string(resp))
	}
}
