package main

import (
	_ "github.com/visonlv/go-vkit/codec"
	_ "github.com/visonlv/go-vkit/errorsx"
	_ "github.com/visonlv/go-vkit/errorsx/neterrors"
	_ "github.com/visonlv/go-vkit/gate"
	_ "github.com/visonlv/go-vkit/grpcclient"
	_ "github.com/visonlv/go-vkit/grpcserver"
	_ "github.com/visonlv/go-vkit/httphandler"
	_ "github.com/visonlv/go-vkit/logger"
	_ "github.com/visonlv/go-vkit/metadata"
	_ "github.com/visonlv/go-vkit/miniox"
	_ "github.com/visonlv/go-vkit/mongox"
	_ "github.com/visonlv/go-vkit/mysqlx"
	_ "github.com/visonlv/go-vkit/natsx"
	_ "github.com/visonlv/go-vkit/redisx"
	_ "github.com/visonlv/go-vkit/utilsx"
)

func main() {
	//mongo
	//nats
	//mysql
	//minio
}
