package grpcclient

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/visonlv/go-vkit/errors/neterrors"
)

var (
	serverName2Addr = map[string]string{}
)

func InvokeByGate(ctx context.Context, addrName string, endpoint string, jsonBody []byte) (*json.RawMessage, *neterrors.NetError) {
	//get conn from addrName
	ccc, ok := GetClient(addrName)
	if !ok {
		ccc = GetConnClient(addrName + ":10000")
	}

	reply := &json.RawMessage{}

	methodName := strings.ReplaceAll(endpoint, ".", "/")
	err := ccc.Invoke(ctx, "/"+methodName, jsonBody, reply)
	switch err {
	case nil:
		return reply, nil
	}

	if verr, ok := err.(*neterrors.NetError); ok {
		return nil, verr
	}

	return nil, neterrors.BadRequest(err.Error()).(*neterrors.NetError)
}

func GetClient(addrName string) (*CustomClientConn, bool) {
	addr, ok := serverName2Addr[addrName]
	if !ok {
		return nil, false
	}
	return GetConnClient(addr), true
}

func SetServerName2Addr(m map[string]string) {
	serverName2Addr = m
}
