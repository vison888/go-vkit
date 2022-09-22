package httphandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/visonlv/go-vkit/codec"
	cerrors "github.com/visonlv/go-vkit/errorsx"
	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/grpcx"
	"github.com/visonlv/go-vkit/logger"
	"google.golang.org/grpc/encoding"
)

type authFunc func(w http.ResponseWriter, r *http.Request) bool
type filterFunc func(w http.ResponseWriter, r *http.Request, b []byte) ([]byte, error)

type handlerInfo struct {
	handler      reflect.Value
	method       reflect.Method
	reqType      reflect.Type
	respType     reflect.Type
	clientStream bool
	serverStream bool
}

type Handler struct {
	sync.RWMutex
	handlers   map[string]*handlerInfo
	authFunc   authFunc
	filterFunc filterFunc
}

func NewHandler() *Handler {
	g := &Handler{
		handlers: make(map[string]*handlerInfo),
	}
	return g
}

func (h *Handler) WithAuthFunc(f authFunc) {
	h.authFunc = f
}

func (h *Handler) WithFilterFunc(f filterFunc) {
	h.filterFunc = f
}

func errorResponse(w http.ResponseWriter, r *http.Request, _err error) {
	var netErr *neterrors.NetError
	if verr, ok := _err.(*neterrors.NetError); ok {
		netErr = verr
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(int(netErr.Status))

	paramJson, err := json.Marshal(*netErr)
	if err != nil {
		logger.Errorf("[httphandler] encode json fail url:%s", r.RequestURI)
		return
	}

	paramStr := string(paramJson)
	w.Header().Set("Content-Length", strconv.Itoa(len(paramJson)))
	logger.Errorf("[httphandler] with error ret:%s url:%s", paramStr, r.RequestURI)
	fmt.Fprintln(w, paramStr)
}

func requestPayload(r *http.Request) (bytes []byte, err error) {
	closeBody := func(body io.ReadCloser) {
		if e := body.Close(); e != nil {
			err = errors.New("[httphandler] body close failed")
			return
		}
	}

	ct := r.Header.Get("Content-Type")
	switch {
	case strings.Contains(ct, "application/json"):
		defer closeBody(r.Body)
		bytes, err = ioutil.ReadAll(r.Body)
		return
	case strings.Contains(ct, "application/x-www-form-urlencoded"):
		r.ParseForm()
		vals := make(map[string]string)
		for k, v := range r.Form {
			vals[k] = strings.Join(v, ",")
		}
		return json.Marshal(vals)
	case strings.Contains(ct, "multipart/form-data"):
		if err := r.ParseMultipartForm(int64(10 << 20)); err != nil {
			return nil, err
		}
		vals := make(map[string]interface{})
		for k, v := range r.MultipartForm.Value {
			vals[k] = strings.Join(v, ",")
		}
		for k := range r.MultipartForm.File {
			f, _, err := r.FormFile(k)
			if err != nil {
				return nil, err
			}
			b, err := ioutil.ReadAll(f)
			if err != nil {
				return nil, err
			}
			vals[k] = b
		}
		return json.Marshal(vals)
	default:
		err = fmt.Errorf("[httphandler] not support contentType:%s", ct)
		return
	}
}

func (h *Handler) RegisterApiEndpoint(list []interface{}, apiEndpointList []*grpcx.ApiEndpoint) (err error) {
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

func (h *Handler) RegisterWithUrl(i interface{}, apiEndpointMap map[string]*grpcx.ApiEndpoint) (err error) {
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
		logger.Infof("[httphandler] Register reqUrl:%s reqMethod:%s", reqUrl, reqMethod)
	}
	return nil
}

func (h *Handler) Register(i interface{}) (err error) {
	return h.RegisterWithUrl(i, nil)
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if re := recover(); re != nil {
			errorStr := fmt.Sprintf("[httphandler] panic recovered:%v ", re)
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
		errorStr := fmt.Sprintf("[httphandler req method:%s only support url:%s", method, r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if h.authFunc != nil {
		if !h.authFunc(w, r) {
			errorStr := fmt.Sprintf("[httphandler] check token fail url:%s", r.RequestURI)
			errorResponse(w, r, neterrors.BadRequest(errorStr))
			return
		}
	}

	reqBytes, err := requestPayload(r)
	if err != nil {
		errorStr := fmt.Sprintf("[httphandler] %s url:%s", err.Error(), r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if h.filterFunc != nil {
		newB, err := h.filterFunc(w, r, reqBytes)
		if err != nil {
			errorStr := fmt.Sprintf("[httphandler] filter data fail url:%s", r.RequestURI)
			errorResponse(w, r, neterrors.BadRequest(errorStr))
			return
		}
		reqBytes = newB
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
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if len(endpoint) == 0 {
		errorStr := fmt.Sprintf("[httphandler] endpoint is empty url:%s", r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	hi, b := h.handlers[endpoint]
	if !b {
		errorStr := fmt.Sprintf("[httphandler] unknown method %s", endpoint)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	argv := reflect.New(hi.reqType.Elem())
	replyv := reflect.New(hi.respType.Elem())

	var cd encoding.Codec
	if cd = codec.DefaultGRPCCodecs[readCt]; cd.Name() != "json" {
		errorStr := fmt.Sprintf("[httphandler] not support content type:%s", r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if err := cd.Unmarshal(reqBytes, argv.Interface()); err != nil {
		errorStr := fmt.Sprintf("[httphandler] Unmarshal error: %s", err.Error())
		errorResponse(w, r, neterrors.BadRequest(errorStr))
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
			errorResponse(w, r, neterrors.BusinessError(-1, errorStr))
			return
		}
	}

	startTime := time.Now()

	in := make([]reflect.Value, 4)
	in[0] = hi.handler
	in[1] = reflect.ValueOf(context.Background())
	in[2] = argv
	in[3] = replyv
	out := hi.method.Func.Call(in)
	if rerr := out[0].Interface(); rerr != nil {
		//处理业务异常
		if verr, ok := rerr.(*cerrors.Errno); ok {
			if verr.Code != 0 {
				errorStr := fmt.Sprintf("[httphandler] call error: %s", verr.Error())
				logger.Errorf(errorStr)
				errorResponse(w, r, neterrors.BusinessError(verr.Code, verr.Msg))
				return
			}
		} else {
			//其他异常统一包装
			errorStr := fmt.Sprintf("[httphandler] call error: %s", rerr.(error).Error())
			logger.Errorf(errorStr)
			errorResponse(w, r, neterrors.BusinessError(-2, errorStr))
			return
		}
	}

	respBytes, err := cd.Marshal(replyv.Interface())
	if err != nil {
		logger.Infof("[httphandler] jsonRaw Marshal fail:%s", err)
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
		logger.Errorf("[httphandler] Write fail url:%v", r.RequestURI)
		return
	}

	successStr := fmt.Sprintf("[httphandler] success cost:[%v] url:[%v] req:[%v] resp:[%v]", costTime.Milliseconds(), r.RequestURI, string(reqBytes), string(respBytes))
	logger.Infof(successStr)

}
