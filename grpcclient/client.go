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

	"github.com/visonlv/go-vkit/codec"
	"github.com/visonlv/go-vkit/errors/neterrors"
	"github.com/visonlv/go-vkit/logger"
	"github.com/visonlv/go-vkit/metadata"
	"github.com/google/uuid"
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
	return nil, nil
}

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
