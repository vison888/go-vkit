package main

import (
	_ "github.com/vison888/go-vkit/codec"
	_ "github.com/vison888/go-vkit/errorsx"
	_ "github.com/vison888/go-vkit/errorsx/neterrors"
	_ "github.com/vison888/go-vkit/gate"
	_ "github.com/vison888/go-vkit/grpcclient"
	_ "github.com/vison888/go-vkit/grpcserver"
	_ "github.com/vison888/go-vkit/logger"
	_ "github.com/vison888/go-vkit/metadata"
	_ "github.com/vison888/go-vkit/miniox"
	_ "github.com/vison888/go-vkit/mongox"
	_ "github.com/vison888/go-vkit/mysqlx"
	_ "github.com/vison888/go-vkit/natsx"
	_ "github.com/vison888/go-vkit/redisx"
	_ "github.com/vison888/go-vkit/utilsx"
)

func main() {
	//mongo
	//nats
	//mysql
	//minio
}
