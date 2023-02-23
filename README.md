# go-vkit
项目限制：  
使用go1.17.7版本及以上  
api统一格式  /前缀/服务名/模块.方法 如:/rpc/speech/AppService.Create  
日志文件只保留七天  

# 主要提供以下功能模块：
1、网关gate  
2、grpcclient  
3、grpcserver  
4、日志  
5、minio文件系统  
6、nats消息队列  
7、mysql数据库  
8、mongodb数据库  
9、redis缓存  
10、错误码规范  

## 1、网关gate
网关作为请求的总入口，主要职责是权限验证、协议适配、请求转发。go的框架底层则提供了两个handler，主要负责http/websock协议的处理。同时也提供一个单点的http服务，该服务通过放射最终回调到业务层。

http网关代理例子：
```
	customHandler := NewHttpHandler(
		HttpGrpcPort(10000),
		HttpAuthHandler(tokenCheck))

	http.HandleFunc("/rpc/", func(w http.ResponseWriter, r *http.Request) {
		customHandler.Handle(w, r)
	})
	err := http.ListenAndServe("0.0.0.0:8080", nil)
	if err != nil {
		logger.Infof("start server fail, err:%s", err)
	} else {
		logger.Infof("start server success")
	}
```

## 2、grpcclient  

原生grpc客户端并不支持连接池，在内部频繁销毁或新建连接将导致请求时间延长、影响服务吞吐量，grpc链路本身支持多路复用，即多个请求可以在一个通道里并行完成，但实际设计不能在一个连接负载所有的流量，这样不满足服务的负载均衡策略，这样设计即使再多的服务器，最总请求都会路由到同个机器，因此，需要限制一个连接能并行的请求数量，在达到上限新开启新的连接来负载。
 
 总结：一个连接池有多个连接，一个连接可以并行多个请求。
```
    cc2 := grpcclient.GetConnClient(
		"127.0.0.1:10000",
		grpcclient.RequestTimeout(time.Second*20),
	)
	Svc.DemoSkillService = nlppb.NewDemoServiceClient("demo", cc2)
	
```

## 3、grpcserver  
每个微服务都将启动一个grpc的服务，为了方便业务开发，对该模块做了封装，主要提供了handler的注册，根据请求URL回调到业务的指定方法。

```
type AuthService struct {
}
type RefleshUrlReq struct {
	Id int64 `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
}
type RefleshUrlResp struct {
	Code int32  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Msg  string `protobuf:"bytes,2,opt,name=msg,proto3" json:"msg,omitempty"`
	Id   int64  `protobuf:"varint,1,opt,name=id,proto3" json:"id,omitempty"`
}

func (the *AuthService) RefleshUrl(ctx context.Context, req *RefleshUrlReq, resp *RefleshUrlResp) error {
	logger.Infof("RefleshUrl req id:%d", req.Id)
	resp.Id = req.Id + 100000
	return nil
}

func startGrpcServer() {
	svr := grpcserver.NewServer()
	err := svr.RegisterApiEndpoint([]interface{}{&AuthService{}}, []*grpcx.ApiEndpoint{{
		Method:       "AuthService.RefleshUrl",
		Url:          "/rpc/sso/AuthService.RefleshUrl",
		ClientStream: false,
		ServerStream: false,
	}})
	if err != nil {
		logger.Errorf("[main] RegisterApiEndpoint fail %s", err)
		panic(err)
	}

	svr.Run("0.0.0.0:10000")
}
```
## 4、nativehandler  
提供一种直接暴露http端口的模块，该模块只支持post协议，内部将post的body通过反射成pb结构，并回调到指定的方法逻辑中。

```
func tokenCheckFunc(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func logFunc(f gate.HandlerFunc) gate.HandlerFunc {
	return func(ctx context.Context, req *gate.HttpRequest, resp *gate.HttpResponse) error {
		startTime := time.Now()
		err := f(ctx, req, resp)
		costTime := time.Since(startTime)
		body, _ := req.Read()
		var logText string
		if err != nil {
			logText = fmt.Sprintf("fail cost:[%v] url:[%v] req:[%v] resp:[%v]", costTime.Milliseconds(), req.Uri(), string(body), err.Error())
		} else {
			logText = fmt.Sprintf("success cost:[%v] url:[%v] req:[%v] resp:[%v]", costTime.Milliseconds(), req.Uri(), string(body), string(resp.Content()))
		}
		logger.Infof(logText)
		return err
	}
}

func Start() {
	//初始化权限数据
	authObj.Start()

	h := gate.NewNativeHandler(
		gate.HttpAuthHandler(tokenCheckFunc),
		gate.HttpWrapHandler(logFunc),
	)
	err := h.RegisterApiEndpoint(handler.GetList(), handler.GetApiEndpoint())
	if err != nil {
		logger.Errorf("[main] RegisterApiEndpoint fail %s", err)
		panic(err)
	}
	http.HandleFunc("/rpc/", func(w http.ResponseWriter, r *http.Request) {
		h.Handle(w, r)
	})

	logger.Infof("[main] Listen port:%d", app.Cfg.Server.HttpPort)
	err = http.ListenAndServe(fmt.Sprintf(":%d", app.Cfg.Server.HttpPort), nil)
	if err != nil {
		logger.Errorf("[main] ListenAndServe fail %s", err)
		panic(err)
	}
}
	
```
## 5、日志  

日志默认7天删除，支持多种级别的日志打印，同时输出到控制台跟文件
```
logger.Infof("日志 %s", "halo")
```
## 6、minio文件系统 
```
	c, err := miniox.NewClient(Cfg.MinIO.DoMain, Cfg.MinIO.EndPoint,
		Cfg.MinIO.AccessKey, Cfg.MinIO.AccessSecret,
		Cfg.MinIO.BucketName)
	if err != nil {
		panic(err)
	}
	Minio = c

```
## 7、nats消息队列  
```
Nats = natsx.NewNatsClient(Cfg.Nats.Url,
		Cfg.Nats.Username, Cfg.Nats.Password)
```
## 8、mysql数据库
```
	c, err := mysqlx.NewClient(Cfg.Mysql.Uri, Cfg.Mysql.MaxConn, Cfg.Mysql.MaxIdel, Cfg.Mysql.MaxLifeTime)
	if err != nil {
		panic(err)
	}
	Mysql = c
```
## 9、mongodb数据库
```
    c, err := mongox.NewClient(Cfg.MongoDB.URI,
		Cfg.MongoDB.DbName,
		time.Duration(Cfg.MongoDB.WithTimeout)*time.Second)

	if err == nil {
		Mgo = c
	} else {
		panic(c)
	}
	
```
## 10、redis缓存
```
	c, err := redisx.NewClient(Cfg.Redis.Address, Cfg.Redis.Password, Cfg.Redis.Db)
	if err != nil {
		panic(err)
	}
	Redis = c
```
## 11、错误码规范
```
var (
	START_ERR_NO = baseNo.NewErrno(4, 1, "开始")
)
```



