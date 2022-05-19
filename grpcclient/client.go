package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/visonlv/go-vkit/codec"
	"github.com/visonlv/go-vkit/errors/neterrors"
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
	DefaultMaxRecvMsgSize = 1024 * 1024 * 16
	DefaultMaxSendMsgSize = 1024 * 1024 * 16
	DefaultDialTimeout    = time.Second * 5
	DefaultRequestTimeout = time.Second * 10
)

type CustomClientConn struct {
	grpc.ClientConnInterface
	pool *pool
	addr string
}

func init() {
	encoding.RegisterCodec(codec.WrapCodec{codec.JsonCodec{}})
	encoding.RegisterCodec(codec.WrapCodec{codec.ProtoCodec{}})
}

// rc.pool = newPool(options.PoolSize, options.PoolTTL, rc.poolMaxIdle(), rc.poolMaxStreams())
func (ccc *CustomClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
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

	if _, ok := header["request_id"]; !ok {
		header["request_id"] = strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	if sq, ok := header["request_sq"]; ok {
		sqInt64, _ := strconv.ParseInt(sq, 10, 64)
		header["request_sq"] = strconv.FormatInt(sqInt64+1, 10)
	} else {
		header["request_sq"] = "0"
	}
	requestId := header["request_id"]
	requestSq := header["request_sq"]

	// set timeout in nanoseconds
	header["timeout"] = fmt.Sprintf("%d", DefaultRequestTimeout)
	// set the content type for the request
	header["x-content-type"] = xContentType
	md := gmetadata.New(header)
	ctx = gmetadata.NewOutgoingContext(ctx, md)

	logger.Infof("xContentType=%s", xContentType)
	cf, ok := codec.DefaultGRPCCodecs[xContentType]
	if !ok {
		return neterrors.BadRequest("[grpcclient] codec not found")
	}

	grpcDialOptions := []grpc.DialOption{
		grpc.WithTimeout(DefaultDialTimeout),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(DefaultMaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(DefaultMaxSendMsgSize),
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

	if logger.CanServerLog(xContentType) {
		var jsonArgv []byte
		if raw, ok := args.([]byte); ok {
			jsonArgv = raw
		} else {
			jsonArgv, _ = json.Marshal(args)
		}

		jsonReplyv, _ := json.Marshal(reply)
		successStr := fmt.Sprintf("[grpcclient] request success requestId:%s requestSq:%s methodName:%s argv:%s replyv:%s", requestId, requestSq, method, jsonArgv, jsonReplyv)
		logger.Info(successStr)
	}

	return grr
}

func (ccc *CustomClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
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

	if _, ok := header["request_id"]; !ok {
		header["request_id"] = strings.ReplaceAll(uuid.New().String(), "-", "")
	}
	if sq, ok := header["request_sq"]; ok {
		sqInt64, _ := strconv.ParseInt(sq, 10, 64)
		header["request_sq"] = strconv.FormatInt(sqInt64+1, 10)
	} else {
		header["request_sq"] = "0"
	}
	// set timeout in nanoseconds
	header["timeout"] = fmt.Sprintf("%d", DefaultRequestTimeout)
	// set the content type for the request
	header["x-content-type"] = xContentType
	md := gmetadata.New(header)
	ctx = gmetadata.NewOutgoingContext(ctx, md)

	logger.Infof("xContentType=%s", xContentType)
	cf, ok := codec.DefaultGRPCCodecs[xContentType]
	if !ok {
		return nil, neterrors.BadRequest("[grpcclient] codec not found")
	}

	grpcDialOptions := []grpc.DialOption{
		grpc.WithTimeout(DefaultDialTimeout),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(DefaultMaxRecvMsgSize),
			grpc.MaxCallSendMsgSize(DefaultMaxSendMsgSize),
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

	// stream := &grpcStream{
	// 	ClientStream: st,
	// 	context:      ctx,
	// 	conn:         cc,
	// 	close: func(err error) {
	// 		// cancel the context if an error occured
	// 		if err != nil {
	// 			cancel()
	// 		}
	// 		// defer execution of release
	// 		ccc.pool.release(ccc.addr, cc, err)
	// 	},
	// }

	return st, nil
}

// func (g *grpcClient) stream(ctx context.Context, addr string, req client.Request, rsp interface{}, opts client.CallOptions) error {
// 	var header map[string]string

// 	if md, ok := metadata.FromContext(ctx); ok {
// 		header = make(map[string]string, len(md))
// 		for k, v := range md {
// 			header[k] = v
// 		}
// 	} else {
// 		header = make(map[string]string)
// 	}

// 	// set timeout in nanoseconds
// 	if opts.StreamTimeout > time.Duration(0) {
// 		header["timeout"] = fmt.Sprintf("%d", opts.StreamTimeout)
// 	}
// 	// set the content type for the request
// 	header["x-content-type"] = req.ContentType()

// 	md := gmetadata.New(header)
// 	ctx = gmetadata.NewOutgoingContext(ctx, md)

// 	cf, err := g.newGRPCCodec(req.ContentType())
// 	if err != nil {
// 		return errors.InternalServerError("go.micro.client", err.Error())
// 	}

// 	maxRecvMsgSize := g.maxRecvMsgSizeValue()
// 	maxSendMsgSize := g.maxSendMsgSizeValue()

// 	grpcDialOptions := []grpc.DialOption{
// 		grpc.WithTimeout(opts.DialTimeout),
// 		g.secure(addr),
// 		grpc.WithDefaultCallOptions(
// 			grpc.MaxCallRecvMsgSize(maxRecvMsgSize),
// 			grpc.MaxCallSendMsgSize(maxSendMsgSize),
// 		),
// 	}

// 	if opts := g.getGrpcDialOptions(); opts != nil {
// 		grpcDialOptions = append(grpcDialOptions, opts...)
// 	}

// 	cc, err := g.pool.getConn(addr, grpcDialOptions...)
// 	if err != nil {
// 		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error sending request: %v", err))
// 	}

// 	desc := &grpc.StreamDesc{
// 		StreamName:    req.Service() + req.Endpoint(),
// 		ClientStreams: true,
// 		ServerStreams: true,
// 	}

// 	grpcCallOptions := []grpc.CallOption{
// 		grpc.ForceCodec(cf),
// 		grpc.CallContentSubtype(cf.Name()),
// 	}
// 	if opts := g.getGrpcCallOptions(); opts != nil {
// 		grpcCallOptions = append(grpcCallOptions, opts...)
// 	}

// 	var cancel context.CancelFunc
// 	ctx, cancel = context.WithCancel(ctx)

// 	st, err := cc.NewStream(ctx, desc, methodToGRPC(req.Service(), req.Endpoint()), grpcCallOptions...)
// 	if err != nil {
// 		// we need to cleanup as we dialled and created a context
// 		// cancel the context
// 		cancel()
// 		// release the connection
// 		g.pool.release(addr, cc, err)
// 		// now return the error
// 		return errors.InternalServerError("go.micro.client", fmt.Sprintf("Error creating stream: %v", err))
// 	}

// 	codec := &grpcCodec{
// 		s: st,
// 		c: wc,
// 	}

// 	// set request codec
// 	if r, ok := req.(*grpcRequest); ok {
// 		r.codec = codec
// 	}

// 	// setup the stream response
// 	stream := &grpcStream{
// 		ClientStream: st,
// 		context:      ctx,
// 		request:      req,
// 		response: &response{
// 			conn:   cc,
// 			stream: st,
// 			codec:  cf,
// 			gcodec: codec,
// 		},
// 		conn: cc,
// 		close: func(err error) {
// 			// cancel the context if an error occured
// 			if err != nil {
// 				cancel()
// 			}

// 			// defer execution of release
// 			g.pool.release(addr, cc, err)
// 		},
// 	}

// 	// set the stream as the response
// 	val := reflect.ValueOf(rsp).Elem()
// 	val.Set(reflect.ValueOf(stream).Elem())
// 	return nil
// }

func DelConnClient(addr string) {
	addr2conn.Delete(addr)
}

func GetConnClient(addr string) *CustomClientConn {
	iccc, ok := addr2conn.Load(addr)
	if ok {
		return iccc.(*CustomClientConn)
	}

	mutex.Lock()
	defer mutex.Unlock()
	//double check
	iccc, ok = addr2conn.Load(addr)
	if ok {
		return iccc.(*CustomClientConn)
	}
	pool := newPool(100, time.Minute, 50, 20)

	ccc := &CustomClientConn{
		pool: pool,
		addr: addr,
	}
	addr2conn.Store(addr, ccc)
	return ccc
}
