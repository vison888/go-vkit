package grpcclient

import "time"

type Options struct {
	MaxRecvMsgSize int
	MaxSendMsgSize int
	DialTimeout    time.Duration
	RequestTimeout time.Duration
}

type Option func(o *Options)

func newOptions(opts ...Option) Options {
	opt := Options{
		MaxRecvMsgSize: DefaultMaxRecvMsgSize,
		MaxSendMsgSize: DefaultMaxSendMsgSize,
		DialTimeout:    DefaultDialTimeout,
		RequestTimeout: DefaultRequestTimeout,
	}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

func MaxRecvMsgSize(maxRecvMsgSize int) Option {
	return func(o *Options) {
		o.MaxRecvMsgSize = maxRecvMsgSize
	}
}

func MaxSendMsgSize(maxSendMsgSize int) Option {
	return func(o *Options) {
		o.MaxSendMsgSize = maxSendMsgSize
	}
}

func DialTimeout(dialTimeout time.Duration) Option {
	return func(o *Options) {
		o.DialTimeout = dialTimeout
	}
}

func RequestTimeout(requestTimeout time.Duration) Option {
	return func(o *Options) {
		o.RequestTimeout = requestTimeout
	}
}
