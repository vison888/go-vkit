package grpcclient

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/grpcx"
	"google.golang.org/grpc"
)

var (
	serverName2Addr = map[string]string{}
)

func InvokeByGate(ctx context.Context, addrName string, service, endpoint string, jsonBody []byte) (*json.RawMessage, *neterrors.NetError) {
	ccc, ok := GetClient(addrName)
	if !ok {
		ccc = GetConnClient(addrName)
	}

	reply := &json.RawMessage{}

	err := ccc.Invoke(ctx, service, endpoint, jsonBody, reply)
	if err == nil {
		return reply, nil
	}

	if verr, ok := err.(*neterrors.NetError); ok {
		return nil, verr
	}

	return nil, neterrors.BadRequest(err.Error()).(*neterrors.NetError)
}

func StreamByGate(ctx context.Context, addrName string, service, endpoint string) (grpcx.ClientStream, *neterrors.NetError) {
	//get conn from addrName
	ccc, ok := GetClient(addrName)
	if !ok {
		ccc = GetConnClient(addrName)
	}

	grpcStream, err := ccc.NewStream(ctx,
		&grpc.StreamDesc{
			StreamName:    endpoint,
			ServerStreams: true,
			ClientStreams: true,
		},
		service, endpoint)
	if err == nil {
		return grpcStream, nil
	}

	if verr, ok := err.(*neterrors.NetError); ok {
		return nil, verr
	}

	return nil, neterrors.BadRequest(err.Error()).(*neterrors.NetError)
}

func GetClient(addrName string) (grpcx.Client, bool) {
	addr, ok := serverName2Addr[addrName]
	if !ok {
		return nil, false
	}
	return GetConnClient(addr), true
}

func SetServerName2Addr(m map[string]string) {
	serverName2Addr = m
}

// service Struct.Method /service.Struct/Method
func methodToGRPC(service, method string) string {
	// no method or already grpc method
	if len(method) == 0 || method[0] == '/' {
		return method
	}

	// assume method is Foo.Bar
	mParts := strings.Split(method, ".")
	if len(mParts) != 2 {
		return method
	}

	if len(service) == 0 {
		return fmt.Sprintf("/%s/%s", mParts[0], mParts[1])
	}

	// return /pkg.Foo/Bar
	return fmt.Sprintf("/%s.%s/%s", service, mParts[0], mParts[1])
}
