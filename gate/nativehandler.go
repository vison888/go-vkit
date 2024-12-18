package gate

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/vison888/go-vkit/codec"
	"github.com/vison888/go-vkit/errorsx"
	"github.com/vison888/go-vkit/errorsx/neterrors"
	"github.com/vison888/go-vkit/grpcx"
	"github.com/vison888/go-vkit/logger"
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

func (h *NativeHandler) RegisterApiEndpoint(list []any, apiEndpointList []*grpcx.ApiEndpoint) (err error) {
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

func (h *NativeHandler) RegisterWithUrl(i any, apiEndpointMap map[string]*grpcx.ApiEndpoint) (err error) {
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

func (h *NativeHandler) Register(i any) (err error) {
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
		errorStr := fmt.Sprintf("method:%s not support, url:%s", method, r.RequestURI)
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
		errorStr := fmt.Sprintf("service is empty url:%s", r.RequestURI)
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if len(endpoint) == 0 {
		errorStr := fmt.Sprintf("endpoint is empty url:%s", r.RequestURI)
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

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

	hdr := map[string]string{}
	for k := range r.Header {
		hdr[k] = r.Header.Get(k)
	}
	md := gmetadata.New(hdr)
	fullCtx := gmetadata.NewIncomingContext(context.Background(), md)
	// 主逻辑
	fn := func(ctx context.Context, req *HttpRequest, resp *HttpResponse) error {
		reqBytes, files, err := request.Read()
		if err != nil {
			errorStr := fmt.Sprintf("获取body失败 %s url:%s", err.Error(), r.RequestURI)
			return neterrors.BadRequest(errorStr)
		}

		hi, b := h.handlers[endpoint]
		if !b {
			hi, b = h.handlers[request.uri]
		}
		if !b {
			errorStr := fmt.Sprintf("unknown method %s", endpoint)
			return neterrors.BadRequest(errorStr)
		}

		argv := reflect.New(hi.reqType.Elem())
		replyv := reflect.New(hi.respType.Elem())

		var cd encoding.Codec
		if cd = codec.DefaultGRPCCodecs[readCt]; cd.Name() != "json" {
			errorStr := fmt.Sprintf("not support content type:%s", r.RequestURI)
			return neterrors.BadRequest(errorStr)
		}

		if err := cd.Unmarshal(reqBytes, argv.Interface()); err != nil {
			errorStr := fmt.Sprintf("Unmarshal error: %s", err.Error())
			return neterrors.BadRequest(errorStr)
		}

		if files != nil {
			filesRet := make(map[string][]byte, 0)
			for k, v := range files {
				filesRet[k] = v.Content
			}

			filenamesRet := make(map[string]string, 0)
			for k, v := range files {
				filenamesRet[k] = v.Filename
			}
			// 检查回调
			field1 := argv.Elem().FieldByName("Files")
			if field1.CanSet() {
				field1.Set(reflect.ValueOf(filesRet))
			}
			field2 := argv.Elem().FieldByName("Filenames")
			if field2.CanSet() {
				field2.Set(reflect.ValueOf(filenamesRet))
			}
		}

		// validate
		validateFunc, b := hi.reqType.MethodByName("Validate")
		if b {
			out := validateFunc.Func.Call([]reflect.Value{reflect.ValueOf(argv.Interface())})
			errValue := out[0]
			if errValue.Interface() != nil {
				err := errValue.Interface().(error)
				errorStr := fmt.Sprintf("param error: %s", err.Error())
				return neterrors.BusinessError(-1, errorStr)
			}
		}

		in := make([]reflect.Value, 4)
		in[0] = hi.handler
		in[1] = reflect.ValueOf(ctx)
		in[2] = argv
		in[3] = replyv
		out := hi.method.Func.Call(in)
		if rerr := out[0].Interface(); rerr != nil {
			//处理业务异常
			if verr, ok := rerr.(*errorsx.Errno); ok {
				if verr.Code != 0 {
					errorStr := fmt.Sprintf("call error: %s", verr.Error())
					logger.Errorf(errorStr)
					return neterrors.BusinessError(verr.Code, verr.Msg)
				}
			} else {
				//其他异常统一包装
				errorStr := fmt.Sprintf("call error: %s", rerr.(error).Error())
				logger.Errorf(errorStr)
				return neterrors.BusinessError(-2, errorStr)
			}
		}

		respBytes, err := cd.Marshal(replyv.Interface())
		if err != nil {
			logger.Infof("jsonRaw Marshal fail:%s", err)
			return neterrors.BadRequest(err.Error())
		}

		resp.content = respBytes
		return nil
	}
	// 拦截器
	for i := len(h.opts.HdlrWrappers); i > 0; i-- {
		fn = h.opts.HdlrWrappers[i-1](fn)
	}

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
	w.Header().Set("Content-Length", strconv.Itoa(len(response.content)))
	_, err := w.Write(response.content)
	if err != nil {
		logger.Errorf("[nativehandler] response fail url:%v respBytes:%s", r.RequestURI, string(response.content))
	}
}
