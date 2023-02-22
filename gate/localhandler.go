package gate

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/visonlv/go-vkit/codec"
	cerrors "github.com/visonlv/go-vkit/errorsx"
	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/grpcx"
	"github.com/visonlv/go-vkit/logger"
	"google.golang.org/grpc/encoding"
	gmetadata "google.golang.org/grpc/metadata"
)

type NativeHandler struct {
	handlers map[string]*handlerInfo
	opts     HttpOptions
}

func NewNativeHandler(opts ...HttpOption) *NativeHandler {
	return &NativeHandler{
		handlers: make(map[string]*handlerInfo),
		opts:     newHttpOptions(opts...),
	}
}

func (h *NativeHandler) Init(opts ...HttpOption) {
	for _, o := range opts {
		o(&h.opts)
	}
}

type handlerInfo struct {
	handler      reflect.Value
	method       reflect.Method
	reqType      reflect.Type
	respType     reflect.Type
	clientStream bool
	serverStream bool
}

func (h *NativeHandler) RegisterApiEndpoint(list []interface{}, apiEndpointList []*grpcx.ApiEndpoint) (err error) {
	apiEndpointMap := make(map[string]*grpcx.ApiEndpoint, 0)
	for _, v := range apiEndpointList {
		apiEndpointMap[v.Method] = v
	}
	for _, v := range list {
		err := h.RegisterWithUrl(v, apiEndpointMap)
		if err != nil {
			return err
		}
	}
	return nil
}

func (h *NativeHandler) RegisterWithUrl(i interface{}, apiEndpointMap map[string]*grpcx.ApiEndpoint) (err error) {
	o := reflect.ValueOf(i)
	hType := o.Type()
	hName := hType.Elem().Name()
	mCount := hType.NumMethod()
	//反射方法
	for i := 0; i < mCount; i++ {
		m := hType.Method(i)
		methodName := hName + "." + m.Name
		reqUrl := methodName
		reqMethod := ""
		clientStream := false
		serverStream := false
		if apiEndpointMap != nil {
			desc, b := apiEndpointMap[methodName]
			if !b {
				continue
			}
			reqUrl = desc.Url
			clientStream = desc.ClientStream
			serverStream = desc.ServerStream
			reqMethod = desc.Method
		}
		var reqType reflect.Type
		var respType reflect.Type
		if m.Type.NumIn() == 3 {
			respType = m.Type.In(2)
		} else if m.Type.NumIn() == 4 {
			reqType = m.Type.In(2)
			respType = m.Type.In(3)
		} else {
			panic("in param numbre error methodName:=" + methodName)
		}

		handler := &handlerInfo{
			handler:      o,
			method:       m,
			reqType:      reqType,
			respType:     respType,
			clientStream: clientStream,
			serverStream: serverStream,
		}

		h.handlers[reqUrl] = handler
		if reqMethod != "" {
			h.handlers[reqMethod] = handler
		}
	}
	return nil
}

func (h *NativeHandler) Register(i interface{}) (err error) {
	return h.RegisterWithUrl(i, nil)
}

func (h *NativeHandler) Handle(w http.ResponseWriter, r *http.Request) {
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
		errorStr := fmt.Sprintf("[httphandler req method:%s only support url:%s", method, r.RequestURI)
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

	reqBytes, err := requestPayload(r)
	if err != nil {
		errorStr := fmt.Sprintf("[httphandler] %s url:%s", err.Error(), r.RequestURI)
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
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
		errorStr := fmt.Sprintf("[httphandler] service is empty url:%s", r.RequestURI)
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if len(endpoint) == 0 {
		errorStr := fmt.Sprintf("[httphandler] endpoint is empty url:%s", r.RequestURI)
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	hi, b := h.handlers[endpoint]
	if !b {
		errorStr := fmt.Sprintf("[httphandler] unknown method %s", endpoint)
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	argv := reflect.New(hi.reqType.Elem())
	replyv := reflect.New(hi.respType.Elem())

	var cd encoding.Codec
	if cd = codec.DefaultGRPCCodecs[readCt]; cd.Name() != "json" {
		errorStr := fmt.Sprintf("[httphandler] not support content type:%s", r.RequestURI)
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if err := cd.Unmarshal(reqBytes, argv.Interface()); err != nil {
		errorStr := fmt.Sprintf("[httphandler] Unmarshal error: %s", err.Error())
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	// validate
	validateFunc, b := hi.reqType.MethodByName("Validate")
	if b {
		out := validateFunc.Func.Call([]reflect.Value{reflect.ValueOf(argv.Interface())})
		errValue := out[0]
		if errValue.Interface() != nil {
			err := errValue.Interface().(error)
			errorStr := fmt.Sprintf("[httphandler] param error: %s", err.Error())
			ErrorResponse(w, r, neterrors.BusinessError(-1, errorStr))
			return
		}
	}

	startTime := time.Now()
	hdr := map[string]string{}
	for k := range r.Header {
		hdr[k] = r.Header.Get(k)
	}
	md := gmetadata.New(hdr)
	ctx := gmetadata.NewIncomingContext(context.Background(), md)

	in := make([]reflect.Value, 4)
	in[0] = hi.handler
	in[1] = reflect.ValueOf(ctx)
	in[2] = argv
	in[3] = replyv
	out := hi.method.Func.Call(in)
	if rerr := out[0].Interface(); rerr != nil {
		//处理业务异常
		if verr, ok := rerr.(*cerrors.Errno); ok {
			if verr.Code != 0 {
				errorStr := fmt.Sprintf("[httphandler] call error: %s", verr.Error())
				logger.Errorf(errorStr)
				ErrorResponse(w, r, neterrors.BusinessError(verr.Code, verr.Msg))
				return
			}
		} else {
			//其他异常统一包装
			errorStr := fmt.Sprintf("[httphandler] call error: %s", rerr.(error).Error())
			logger.Errorf(errorStr)
			ErrorResponse(w, r, neterrors.BusinessError(-2, errorStr))
			return
		}
	}

	respBytes, err := cd.Marshal(replyv.Interface())
	if err != nil {
		logger.Infof("[httphandler] jsonRaw Marshal fail:%s", err)
		ErrorResponse(w, r, neterrors.BadRequest(err.Error()))
		return
	}

	costTime := time.Since(startTime)

	//success
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(200)
	w.Header().Set("Content-Length", strconv.Itoa(len(respBytes)))
	_, err = w.Write(respBytes)
	if err != nil {
		logger.Errorf("[httphandler] Write fail url:%v", r.RequestURI)
		return
	}

	successStr := fmt.Sprintf("[httphandler] success cost:%v url:%v req:%v resp:%v", costTime.Milliseconds(), r.RequestURI, string(reqBytes), string(respBytes))
	logger.Infof(successStr)
}
