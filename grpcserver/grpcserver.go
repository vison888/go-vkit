package grpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"reflect"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"github.com/vison888/go-vkit/codec"
	"github.com/vison888/go-vkit/errorsx"
	"github.com/vison888/go-vkit/errorsx/neterrors"
	"github.com/vison888/go-vkit/grpcx"
	"github.com/vison888/go-vkit/logger"
	meta "github.com/vison888/go-vkit/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/encoding"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
)

const (
	DefaultContentType = "application/grpc"
)

type handlerInfo struct {
	handler      reflect.Value
	method       reflect.Method
	reqType      reflect.Type
	respType     reflect.Type
	clientStream bool
	serverStream bool
}

type GrpcServer struct {
	srv      *grpc.Server
	handlers map[string]*handlerInfo

	opts GrpcOptions
}

func init() {
	encoding.RegisterCodec(codec.WrapCodec{codec.JsonCodec{}})
	encoding.RegisterCodec(codec.WrapCodec{codec.ProtoCodec{}})
}

func NewServer(opts ...GrpcOption) *GrpcServer {
	g := &GrpcServer{
		handlers: make(map[string]*handlerInfo),
		opts:     newGrpcOptions(opts...),
	}

	gopts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(g.opts.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(g.opts.MaxSendMsgSize),
		grpc.UnknownServiceHandler(g.handler),
	}

	gopts = append(gopts, g.opts.Gopts...)
	g.srv = grpc.NewServer(gopts...)
	reflection.Register(g.srv)
	return g
}

func (g *GrpcServer) handler(srv any, stream grpc.ServerStream) (err error) {
	defer func() {
		if r := recover(); r != nil {
			errorStr := fmt.Sprintf("[Grpcserver] panic recovered:%v ", r)
			logger.Errorf(errorStr)
			logger.Error(string(debug.Stack()))
			err = neterrors.BadRequest(errorStr)
		}
	}()

	fullMethod, ok := grpc.MethodFromServerStream(stream)
	if !ok {
		errorStr := "[Grpcserver] method does not exist in context"
		logger.Errorf(errorStr)
		return neterrors.NotFound(errorStr)
	}

	serviceName, methodName, err := ServiceMethod(fullMethod)
	if err != nil {
		errorStr := "[Grpcserver] ServiceMethod err:" + err.Error()
		logger.Errorf(errorStr)
		return neterrors.NotFound(errorStr)
	}
	methodName = fmt.Sprintf("%s.%s", serviceName, methodName)

	gmd, ok := metadata.FromIncomingContext(stream.Context())
	if !ok {
		gmd = metadata.MD{}
	}

	md := meta.Metadata{}
	for k, v := range gmd {
		md[k] = strings.Join(v, ", ")
	}

	to := md["timeout"]
	xct := DefaultContentType

	if ctype, ok := md["x-content-type"]; ok {
		xct = ctype
	} else {
		if ctype, ok := md["content-type"]; ok {
			xct = ctype
		}
	}
	ct := xct
	if ctype, ok := md["content-type"]; ok {
		ct = ctype
	}
	md["content-type"] = ct
	delete(md, "timeout")
	delete(md, "x-content-type")

	// create new context
	ctx := meta.NewContext(stream.Context(), md)

	// get peer from context
	if p, ok := peer.FromContext(stream.Context()); ok {
		md["Remote"] = p.Addr.String()
		ctx = peer.NewContext(ctx, p)
	}

	// set the timeout if we have it
	if len(to) > 0 {
		if n, err := strconv.ParseUint(to, 10, 64); err == nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(n))
			defer cancel()
		}
	}

	h, b := g.handlers[methodName]
	if !b {
		h, b = g.handlers[fullMethod]
	}
	if !b {
		errorStr := fmt.Sprintf("[Grpcserver] unknown method %s", methodName)
		logger.Errorf(errorStr)
		return neterrors.NotFound(errorStr)
	}

	if h.clientStream || h.serverStream {
		return g.processStream(stream, h, ct, xct, methodName, ctx)
	}

	return g.processRequest(stream, h, ct, xct, methodName, ctx)
}

func (g *GrpcServer) processStream(stream grpc.ServerStream, h *handlerInfo, ct string, xct string, methodName string, ctx context.Context) error {
	replyv := reflect.New(h.respType.Elem())
	setStreamFunc, b := h.respType.MethodByName("SetStream")
	if b {
		setStreamFunc.Func.Call([]reflect.Value{reflect.ValueOf(replyv.Interface()), reflect.ValueOf(stream)})
	}

	r := &GrpcRequest{
		service:     g.opts.Name,
		contentType: ct,
		method:      methodName,
		stream:      false,
	}

	var argv reflect.Value
	if h.reqType != nil {
		argv = reflect.New(h.reqType.Elem())
		r.payload = argv.Interface()
	}

	fn := func(ctx context.Context, req *GrpcRequest, rsp any) (err error) {
		var in []reflect.Value
		var out []reflect.Value
		if h.reqType != nil {
			//read first data
			if err := stream.RecvMsg(argv.Interface()); err != nil {
				return err
			}
			in = make([]reflect.Value, 4)
			in[0] = h.handler
			in[1] = reflect.ValueOf(ctx)
			in[2] = argv
			in[3] = replyv
			out = h.method.Func.Call(in)
		} else {
			in = make([]reflect.Value, 3)
			in[0] = h.handler
			in[1] = reflect.ValueOf(ctx)
			in[2] = replyv
			out = h.method.Func.Call(in)
		}
		if rerr := out[0].Interface(); rerr != nil {
			if verr, ok := rerr.(error); ok {
				return verr
			} else {
				return fmt.Errorf("stream error %v", rerr)
			}
		}

		return nil
	}

	for i := len(g.opts.HdlrWrappers); i > 0; i-- {
		fn = g.opts.HdlrWrappers[i-1](fn)
	}

	if appErr := fn(ctx, r, replyv.Interface()); appErr != nil {
		switch verr := appErr.(type) {
		case *neterrors.NetError:
			return appErr
		default:
			return verr
		}
	}

	return nil
}

func (g *GrpcServer) processRequest(stream grpc.ServerStream, h *handlerInfo, ct string, xct string, methodName string, ctx context.Context) error {
	argv := reflect.New(h.reqType.Elem())
	replyv := reflect.New(h.respType.Elem())

	if cd := codec.DefaultGRPCCodecs[xct]; cd.Name() != "json" {
		if err := stream.RecvMsg(argv.Interface()); err != nil {
			errorStr := fmt.Sprintf("[Grpcserver] RecvMsg error: %s", err.Error())
			logger.Errorf(errorStr)
			return neterrors.BadRequest(errorStr)
		}
	} else {
		var raw json.RawMessage
		if err := stream.RecvMsg(&raw); err != nil {
			errorStr := fmt.Sprintf("[Grpcserver] RecvMsg error: %s", err.Error())
			logger.Errorf(errorStr)
			return neterrors.BadRequest(errorStr)
		}

		if err := cd.Unmarshal(raw, argv.Interface()); err != nil {
			errorStr := fmt.Sprintf("[Grpcserver] Unmarshal error: %s", err.Error())
			logger.Errorf(errorStr)
			return neterrors.BadRequest(errorStr)
		}
	}

	r := &GrpcRequest{
		service:     g.opts.Name,
		contentType: ct,
		method:      methodName,
		stream:      false,
		payload:     argv.Interface(),
	}

	fn := func(ctx context.Context, req *GrpcRequest, rsp any) (err error) {
		// validate
		validateFunc, b := h.reqType.MethodByName("Validate")
		if b {
			out := validateFunc.Func.Call([]reflect.Value{reflect.ValueOf(argv.Interface())})
			errValue := out[0]
			if errValue.Interface() != nil {
				err := errValue.Interface().(error)
				errorStr := fmt.Sprintf("[Grpcserver]  param error: %s", err.Error())
				logger.Errorf(errorStr)
				return neterrors.BusinessError(-1, errorStr)
			}
		}

		in := make([]reflect.Value, 4)
		in[0] = h.handler
		in[1] = reflect.ValueOf(ctx)
		in[2] = argv
		in[3] = replyv
		out := h.method.Func.Call(in)
		if rerr := out[0].Interface(); rerr != nil {
			//处理业务异常
			if verr, ok := rerr.(*errorsx.Errno); ok {
				if verr.Code != 0 {
					errorStr := fmt.Sprintf("[Grpcserver] call error: %s", verr.Error())
					logger.Errorf(errorStr)
					return neterrors.BusinessError(verr.Code, verr.Msg)
				}
			} else {
				//其他异常统一包装
				errorStr := fmt.Sprintf("[Grpcserver] call error: %s", rerr.(error).Error())
				logger.Errorf(errorStr)
				return neterrors.BusinessError(-2, errorStr)
			}
		}
		return nil
	}

	for i := len(g.opts.HdlrWrappers); i > 0; i-- {
		fn = g.opts.HdlrWrappers[i-1](fn)
	}

	// execute the handler
	if appErr := fn(ctx, r, replyv.Interface()); appErr != nil {
		switch verr := appErr.(type) {
		case *neterrors.NetError:
			// 检查回调
			field1 := argv.Elem().FieldByName("Code")
			field2 := argv.Elem().FieldByName("Msg")
			if field1.CanSet() && field2.CanSet() {
				field1.Set(reflect.ValueOf(verr.Code))
				field2.Set(reflect.ValueOf(verr.Msg))
			} else {
				return appErr
			}
		default:
			return verr
		}
	}

	if err := stream.SendMsg(replyv.Interface()); err != nil {
		errorStr := fmt.Sprintf("[Grpcserver] send error: %s", err.Error())
		logger.Errorf(errorStr)
		return neterrors.BusinessError(-2, errorStr)
	}

	return nil
}

func (g *GrpcServer) RegisterApiEndpoint(list []any, apiEndpointList []*grpcx.ApiEndpoint) (err error) {
	apiEndpointMap := make(map[string]*grpcx.ApiEndpoint, 0)
	for _, v := range apiEndpointList {
		apiEndpointMap[v.Method] = v
	}
	for _, v := range list {
		err := g.RegisterWithUrl(v, apiEndpointMap)
		if err != nil {
			return err
		}
	}
	return nil
}

func (g *GrpcServer) RegisterWithUrl(i any, apiEndpointMap map[string]*grpcx.ApiEndpoint) (err error) {
	o := reflect.ValueOf(i)
	hType := o.Type()
	hName := hType.Elem().Name()
	mCount := hType.NumMethod()
	//反射方法
	for i := 0; i < mCount; i++ {
		m := hType.Method(i)
		methodName := hName + "." + m.Name
		reqUrl := methodName
		reqMethod := ""
		clientStream := false
		serverStream := false
		if apiEndpointMap != nil {
			desc, b := apiEndpointMap[methodName]
			if !b {
				continue
			}
			reqUrl = desc.Url
			clientStream = desc.ClientStream
			serverStream = desc.ServerStream
			reqMethod = desc.Method
		}
		var reqType reflect.Type
		var respType reflect.Type
		if m.Type.NumIn() == 3 {
			respType = m.Type.In(2)
		} else if m.Type.NumIn() == 4 {
			reqType = m.Type.In(2)
			respType = m.Type.In(3)
		} else {
			panic("in param numbre error methodName:=" + methodName)
		}

		handler := &handlerInfo{
			handler:      o,
			method:       m,
			reqType:      reqType,
			respType:     respType,
			clientStream: clientStream,
			serverStream: serverStream,
		}

		g.handlers[reqUrl] = handler
		if reqMethod != "" {
			g.handlers[reqMethod] = handler
		}
		logger.Infof("[GrpcServer] Register reqUrl:%s reqMethod:%s", reqUrl, reqMethod)
	}
	return nil
}

func (g *GrpcServer) Register(i any) (err error) {
	return g.RegisterWithUrl(i, nil)
}

func (g *GrpcServer) Run() {
	logger.Infof("[GrpcServer] Listen start port:[%s]", g.opts.GrpcAddr)

	lis, err := net.Listen("tcp", g.opts.GrpcAddr)
	if err != nil {
		logger.Errorf("[GrpcServer] Listen failed e: %v", err.Error())
		return
	}

	if err := g.srv.Serve(lis); err != nil {
		logger.Errorf("failed to serve: %v", err)
	}
}
