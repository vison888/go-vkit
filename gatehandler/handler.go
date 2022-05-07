package gatehandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/visonlv/go-vkit/errors/neterrors"
	"github.com/visonlv/go-vkit/grpcclient"
	"github.com/visonlv/go-vkit/logger"
	meta "github.com/visonlv/go-vkit/metadata"
	"github.com/google/uuid"
)

type authFunc func(w http.ResponseWriter, r *http.Request) bool

type Handler struct {
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
		logger.Errorf("[gatehandler] encode json fail url:%s", r.RequestURI)
		return
	}

	paramStr := string(paramJson)
	w.Header().Set("Content-Length", strconv.Itoa(len(paramJson)))
	logger.Errorf("[gatehandler] with error ret:%s url:%s", paramStr, r.RequestURI)
	fmt.Fprintln(w, paramStr)
}

func requestPayload(r *http.Request) (bytes []byte, err error) {
	closeBody := func(body io.ReadCloser) {
		if e := body.Close(); e != nil {
			err = errors.New("[gatehandler] body close failed")
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
		err = fmt.Errorf("[gatehandler] not support contentType:%s", ct)
		return
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request, authCheck authFunc) {
	defer func() {
		if re := recover(); re != nil {
			errorStr := fmt.Sprintf("[gatehandler] panic recovered:%v ", re)
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
		errorStr := fmt.Sprintf("[gatehandler req method:%s only support url:%s", method, r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if !authCheck(w, r) {
		errorStr := fmt.Sprintf("[gatehandler] check token fail url:%s", r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	reqBytes, err := requestPayload(r)
	if err != nil {
		errorStr := fmt.Sprintf("[gatehandler] %s url:%s", err.Error(), r.RequestURI)
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
		errorStr := fmt.Sprintf("[gatehandler] service is empty url:%s", r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	if len(endpoint) == 0 {
		errorStr := fmt.Sprintf("[gatehandler] endpoint is empty url:%s", r.RequestURI)
		errorResponse(w, r, neterrors.BadRequest(errorStr))
		return
	}

	startTime := time.Now()

	requestId := strings.ReplaceAll(uuid.New().String(), "-", "")
	requestSq := 0
	md := meta.Metadata{}
	md["x-content-type"] = readCt
	md["request_id"] = requestId
	ctx := meta.NewContext(context.Background(), md)

	jsonRaw, netErr := grpcclient.InvokeByGate(ctx, service, endpoint, reqBytes)
	if netErr != nil {
		logger.Infof("[gatehandler] InvokeWithJson response netErr:%s", netErr)
		errorResponse(w, r, netErr)
		return
	}

	respBytes, err := jsonRaw.MarshalJSON()
	if err != nil {
		logger.Infof("[gatehandler] jsonRaw MarshalJSON fail:%s", err)
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
		logger.Errorf("[gatehandler] requestId:%s requestSq:%d response fail url:%v", requestId, requestSq, r.RequestURI)
		return
	}

	if logger.CanServerLog(readCt) {
		successStr := fmt.Sprintf("[gatehandler] success requestId:%s requestSq:%d cost:[%v] url:[%v] req:[%v] resp:[%v]", requestId, requestSq, costTime.Milliseconds(), r.RequestURI, string(reqBytes), string(respBytes))
		logger.Infof(successStr)
	}

}
