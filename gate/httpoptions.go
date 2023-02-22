package gate

import (
	"context"
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/logger"
)

type HandlerFunc func(ctx context.Context, req *HttpRequest, resp *HttpResponse) error
type HandlerWrapper func(HandlerFunc) HandlerFunc

var (
	DefaultGrpcPort   = 10000
	DefaultErrHandler = func(w http.ResponseWriter, r *http.Request, err interface{}) {
		errorStr := fmt.Sprintf("[gate] HttpHandler panic recovered:%v ", err)
		logger.Errorf(errorStr)
		logger.Error(string(debug.Stack()))
		ErrorResponse(w, r, neterrors.BadRequest(errorStr))
	}
)

type HttpOptions struct {
	GrpcPort     int
	ErrHandler   func(w http.ResponseWriter, r *http.Request, err interface{})
	AuthHandler  func(w http.ResponseWriter, r *http.Request) error
	HdlrWrappers []HandlerWrapper
}

type HttpOption func(o *HttpOptions)

func newHttpOptions(opts ...HttpOption) HttpOptions {
	opt := HttpOptions{
		GrpcPort:     DefaultGrpcPort,
		ErrHandler:   DefaultErrHandler,
		HdlrWrappers: make([]HandlerWrapper, 0),
	}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

func HttpWrapHandler(w HandlerWrapper) HttpOption {
	return func(o *HttpOptions) {
		o.HdlrWrappers = append(o.HdlrWrappers, w)
	}
}

func HttpErrHandler(h func(w http.ResponseWriter, r *http.Request, err interface{})) HttpOption {
	return func(o *HttpOptions) {
		o.ErrHandler = h
	}
}

func HttpGrpcPort(port int) HttpOption {
	return func(o *HttpOptions) {
		o.GrpcPort = port
	}
}

func HttpAuthHandler(h func(w http.ResponseWriter, r *http.Request) error) HttpOption {
	return func(o *HttpOptions) {
		o.AuthHandler = h
	}
}
