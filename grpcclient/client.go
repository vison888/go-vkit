package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/visonlv/go-vkit/codec"
	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/grpcx"
	"github.com/visonlv/go-vkit/logger"
	"github.com/visonlv/go-vkit/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	gmetadata "google.golang.org/grpc/metadata"
)

var (
	addr2conn sync.Map
	mutex     sync.Mutex

	DefaultPoolMaxStreams = 20
	DefaultPoolMaxIdle    = 50
	DefaultPoolSize       = 100
	DefaultPoolTTL        = time.Minute

	DefaultMaxRecvMsgSize = 1024 * 1024 * 16
	DefaultMaxSendMsgSize = 1024 * 1024 * 16
	DefaultDialTimeout    = time.Second * 5
	DefaultRequestTimeout = time.Second * 20
)

type customClient struct {
	pool *pool
	addr string
	opts Options
}

func init() {
	encoding.RegisterCodec(codec.WrapCodec{codec.JsonCodec{}})
	encoding.RegisterCodec(codec.WrapCodec{codec.ProtoCodec{}})
}

func NewClient(addr string, opts ...Option) grpcx.Client {
	pool := newPool(DefaultPoolSize, DefaultPoolTTL, DefaultPoolMaxIdle, DefaultPoolMaxStreams)

	ccc := &customClient{
		pool: pool,
		addr: addr,
		opts: newOptions(opts...),
	}

	return ccc
}

func DelConnClient(addr string) {
	addr2conn.Delete(addr)
}

func GetConnClient(addr string, opts ...Option) grpcx.Client {
	iccc, ok := addr2conn.Load(addr)
	if ok {
		return iccc.(*customClient)
	}

	mutex.Lock()
	defer mutex.Unlock()
	//double check
	iccc, ok = addr2conn.Load(addr)
	if ok {
		return iccc.(*customClient)
	}
	ccc := NewClient(addr, opts...)

	addr2conn.Store(addr, ccc)
	return ccc
}

// rc.pool = newPool(options.PoolSize, options.PoolTTL, rc.poolMaxIdle(), rc.poolMaxStreams())
func (ccc *customClient) Invoke(ctx context.Context, service, endpoint string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	method := methodToGRPC(service, endpoint)
	header := make(map[string]string)

	if md, ok := metadata.FromContext(ctx); ok {
		for k, v := range md {
			header[strings.ToLower(k)] = v
		}
	} else {
		header = make(map[string]string)
	}

	xContentType := "application/grpc"
	if v, ok := header["x-content-type"]; ok {
		xContentType = v
	}

	requestTimeout := ccc.opts.RequestTimeout
	d, ok := ctx.Deadline()
	if !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, requestTimeout)
		defer cancel()
	} else {
		requestTimeout = time.Until(d)
	}

	header["timeout"] = fmt.Sprintf("%d", requestTimeout)
	header["x-content-type"] = xContentType
	md := gmetadata.New(header)
	ctx = gmetadata.NewOutgoingContext(ctx, md)

	cf, ok := codec.DefaultGRPCCodecs[xContentType]
	if !ok {
		return neterrors.BadRequest("[grpcclient] codec not found")
	}

	grpcDialOptions := []grpc.DialOption{
		grpc.WithTimeout(ccc.opts.DialTimeout),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(ccc.opts.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(ccc.opts.MaxSendMsgSize),
		),
	}

	cc, err := ccc.pool.getConn(ccc.addr, grpcDialOptions...)
	if err != nil {
		return neterrors.BadRequest("[grpcclient] Error sending request: %v", err)
	}

	var grr error
	defer func() {
		//有error 连接将自动关闭
		ccc.pool.release(ccc.addr, cc, grr)
		//服务不可用则直接删除client
		if grr != nil {
			if verr, ok := grr.(*neterrors.NetError); ok {
				//服务不可用 删除客户端
				if verr.Status == http.StatusServiceUnavailable {
					logger.Info("[grpcclient] remove client ccc.addr:%s", ccc.addr)
					DelConnClient(ccc.addr)
				}
			}
		}
	}()

	ch := make(chan error, 1)

	go func() {
		grpcCallOptions := []grpc.CallOption{
			grpc.ForceCodec(cf),
			grpc.CallContentSubtype(cf.Name())}
		grpcCallOptions = append(grpcCallOptions, opts...)
		err := cc.ClientConn.Invoke(ctx, method, args, reply, grpcCallOptions...)
		if err == nil {
			ch <- nil
			return
		}
		errorStr := err.Error()
		index := strings.Index(errorStr, "{\"")
		if index != -1 {
			errorRune := []rune(errorStr)
			errorJson := errorRune[index:]
			ch <- neterrors.Parse(string(errorJson))
			return
		}
		index = strings.Index(errorStr, "code = Unavailable")
		if index != -1 {
			ch <- neterrors.ServiceUnavailable(err.Error())
		} else {
			ch <- neterrors.BadRequest("[grpcclient] req fail %v", err.Error())
		}
	}()

	select {
	case err := <-ch:
		grr = err
	case <-ctx.Done():
		grr = neterrors.Timeout("[grpcclient] req fail %v", ctx.Err())
	}

	var jsonArgv []byte
	if raw, ok := args.([]byte); ok {
		jsonArgv = raw
	} else {
		jsonArgv, _ = json.Marshal(args)
	}

	jsonReplyv, _ := json.Marshal(reply)
	successStr := fmt.Sprintf("[grpcclient] request success methodName:%s argv:%s replyv:%s", method, jsonArgv, jsonReplyv)
	logger.Info(successStr)

	return grr
}

func (ccc *customClient) NewStream(ctx context.Context, desc *grpc.StreamDesc, service, endpoint string, opts ...grpc.CallOption) (grpcx.ClientStream, error) {
	method := methodToGRPC(service, endpoint)
	header := make(map[string]string)

	if md, ok := metadata.FromContext(ctx); ok {
		for k, v := range md {
			header[strings.ToLower(k)] = v
		}
	} else {
		header = make(map[string]string)
	}

	xContentType := "application/grpc"
	if v, ok := header["x-content-type"]; ok {
		xContentType = v
	}

	// set the content type for the request
	header["x-content-type"] = xContentType
	md := gmetadata.New(header)
	ctx = gmetadata.NewOutgoingContext(ctx, md)

	cf, ok := codec.DefaultGRPCCodecs[xContentType]
	if !ok {
		return nil, neterrors.BadRequest("[grpcclient] codec not found")
	}

	grpcDialOptions := []grpc.DialOption{
		grpc.WithTimeout(ccc.opts.DialTimeout),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(ccc.opts.MaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(ccc.opts.MaxSendMsgSize),
		),
	}

	cc, err := ccc.pool.getConn(ccc.addr, grpcDialOptions...)
	if err != nil {
		return nil, neterrors.BadRequest("[grpcclient] Error sending request: %v", err)
	}

	grpcCallOptions := []grpc.CallOption{
		grpc.ForceCodec(cf),
		grpc.CallContentSubtype(cf.Name()),
	}

	var cancel context.CancelFunc
	ctx, cancel = context.WithCancel(ctx)

	st, err := cc.NewStream(ctx, desc, method, grpcCallOptions...)
	if err != nil {
		cancel()
		ccc.pool.release(ccc.addr, cc, err)
		return nil, neterrors.BadRequest(fmt.Sprintf("Error creating stream: %v", err))
	}

	stream := &grpcStream{
		ClientStream: st,
		context:      ctx,
		conn:         cc,
		close: func(err error) {
			if err != nil {
				cancel()
			}

			logger.Infof("close err:%v", err)
			ccc.pool.release(ccc.addr, cc, err)
		},
	}

	return stream, nil
}
