package grpcserver

import (
	"context"

	"google.golang.org/grpc"
)

type HandlerFunc func(ctx context.Context, req *GrpcRequest, rsp any) error
type HandlerWrapper func(HandlerFunc) HandlerFunc

var (
	DefaultGrpcAddr       = "0.0.0.0:10000"
	DefaultMaxRecvMsgSize = 1024 * 1024 * 16
	DefaultMaxSendMsgSize = 1024 * 1024 * 16
)

type GrpcOptions struct {
	Name           string
	GrpcAddr       string
	MaxRecvMsgSize int
	MaxSendMsgSize int
	HdlrWrappers   []HandlerWrapper
	Gopts          []grpc.ServerOption
}

type GrpcOption func(o *GrpcOptions)

func newGrpcOptions(opts ...GrpcOption) GrpcOptions {
	opt := GrpcOptions{
		GrpcAddr:       DefaultGrpcAddr,
		MaxRecvMsgSize: DefaultMaxRecvMsgSize,
		MaxSendMsgSize: DefaultMaxSendMsgSize,
		HdlrWrappers:   make([]HandlerWrapper, 0),
		Gopts:          make([]grpc.ServerOption, 0),
		Name:           "",
	}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

func GrpcWrapHandler(w HandlerWrapper) GrpcOption {
	return func(o *GrpcOptions) {
		o.HdlrWrappers = append(o.HdlrWrappers, w)
	}
}

func GrpcAddr(addr string) GrpcOption {
	return func(o *GrpcOptions) {
		o.GrpcAddr = addr
	}
}

func Name(name string) GrpcOption {
	return func(o *GrpcOptions) {
		o.Name = name
	}
}

func MaxSendMsgSize(maxSendMsgSize int) GrpcOption {
	return func(o *GrpcOptions) {
		o.MaxSendMsgSize = maxSendMsgSize
	}
}

func MaxRecvMsgSize(maxRecvMsgSize int) GrpcOption {
	return func(o *GrpcOptions) {
		o.MaxRecvMsgSize = maxRecvMsgSize
	}
}

func Gopts(opts ...grpc.ServerOption) GrpcOption {
	return func(o *GrpcOptions) {
		for _, opt := range opts {
			o.Gopts = append(o.Gopts, opt)
		}
	}
}
