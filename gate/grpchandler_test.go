package gate

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/grpcclient"
	"github.com/visonlv/go-vkit/grpcserver"
	"github.com/visonlv/go-vkit/grpcx"
	"github.com/visonlv/go-vkit/logger"
)

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

func postData(url string, reqBody, token string) (string, int, error) {
	url = "http://localhost:8080" + url
	payload := strings.NewReader(reqBody)
	client := &http.Client{Timeout: time.Second * 3}
	req, err := http.NewRequest(http.MethodPost, url, payload)
	if err != nil {
		fmt.Println(err)
		return "", -1, err
	}
	req.Header.Add("Content-Type", "application/json")
	if token != "" {
		req.Header.Add("AuthToken", token)
	}
	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return "", -1, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return "", -1, err
	}

	return string(body), res.StatusCode, nil
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

func TestAuth(t *testing.T) {
	tokenCheck := func(w http.ResponseWriter, r *http.Request) error {
		token := r.Header.Get("AuthToken")
		if token == "" {
			return neterrors.Unauthorized("header not conatain token ")
		}
		if strings.Contains(token, "Testing") {
			return nil
		}
		return neterrors.Forbidden("url not permission")
	}

	go startGrpcServer()
	customHandler := NewGrpcHandler(
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

	type args struct {
		token  string
		status int
		url    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"不带token", args{"", 401, "/rpc/sso/AuthService.RefleshUrl"}, true},
		{"带错误token", args{"hhhh", 403, "/rpc/sso/AuthService.RefleshUrl"}, true},
		{"带正确token", args{"Testing", 200, "/rpc/sso/AuthService.RefleshUrl"}, true},
	}

	grpcclient.SetServerName2Addr(map[string]string{"sso:10000": "localhost:10000"})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, status, err := postData(tt.args.url, "{\"id\":111}", tt.args.token)
			if (err != nil) && tt.wantErr {
				t.Fatalf("check token fail %s body:%s", err, resp)
				return
			}
			netErr := &neterrors.NetError{}
			json.Unmarshal([]byte(resp), netErr)
			if netErr.Code != 0 && status != (tt.args.status) && tt.wantErr {
				t.Fatalf("check token fail %s body:%s status err", err, resp)
				return
			}
		})
	}
}

func TestWrapHandler(t *testing.T) {
	go startGrpcServer()

	handler1 := func(f HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req *HttpRequest, resp *HttpResponse) error {
			logger.Infof("=========步骤1 handler1 start")
			f(ctx, req, resp)
			logger.Infof("=========步骤4 handler1 end data:%s", string(resp.content))
			return nil
		}
	}

	handler2 := func(f HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req *HttpRequest, resp *HttpResponse) error {
			logger.Infof("=========步骤2 handler2 start")
			f(ctx, req, resp)
			logger.Infof("=========步骤3 handler2 end data:%s", string(resp.content))
			return nil
		}
	}

	customHandler := NewGrpcHandler(
		HttpWrapHandler(handler1),
		HttpWrapHandler(handler2),
		HttpGrpcPort(10000))

	http.HandleFunc("/rpc/", func(w http.ResponseWriter, r *http.Request) {
		customHandler.Handle(w, r)
	})
	go func() {
		err := http.ListenAndServe("0.0.0.0:8080", nil)
		if err != nil {
			logger.Infof("start server fail, err:%s", err)
		} else {
			logger.Infof("start server success")
		}
	}()

	type args struct {
		token  string
		status int
		url    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"多个拦截器", args{"Testing", 200, "/rpc/sso/AuthService.RefleshUrl"}, true},
	}

	grpcclient.SetServerName2Addr(map[string]string{"sso:10000": "localhost:10000"})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, status, err := postData(tt.args.url, "{\"id\":111}", tt.args.token)
			if (err != nil) && tt.wantErr {
				t.Fatalf("check token fail %s body:%s", err, resp)
				return
			}
			netErr := &neterrors.NetError{}
			json.Unmarshal([]byte(resp), netErr)
			if netErr.Code != 0 && status != (tt.args.status) && tt.wantErr {
				t.Fatalf("check token fail %s body:%s status err", err, resp)
				return
			}
		})
	}
}

func TestLog(t *testing.T) {
	go startGrpcServer()

	handler1 := func(f HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req *HttpRequest, resp *HttpResponse) error {
			startTime := time.Now()
			f(ctx, req, resp)
			costTime := time.Since(startTime)

			body, _ := req.Read()
			successStr := fmt.Sprintf("success cost:[%v] url:[%v] req:[%v] resp:[%v]", costTime.Milliseconds(), req.Uri(), string(body), string(resp.Content()))
			logger.Infof(successStr)
			return nil
		}
	}

	customHandler := NewGrpcHandler(
		HttpWrapHandler(handler1),
		HttpGrpcPort(10000))

	http.HandleFunc("/rpc/", func(w http.ResponseWriter, r *http.Request) {
		customHandler.Handle(w, r)
	})
	go func() {
		err := http.ListenAndServe("0.0.0.0:8080", nil)
		if err != nil {
			logger.Infof("start server fail, err:%s", err)
		} else {
			logger.Infof("start server success")
		}
	}()

	type args struct {
		token  string
		status int
		url    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"拦截日志", args{"Testing", 200, "/rpc/sso/AuthService.RefleshUrl"}, true},
	}

	grpcclient.SetServerName2Addr(map[string]string{"sso:10000": "localhost:10000"})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, status, err := postData(tt.args.url, "{\"id\":111}", tt.args.token)
			if (err != nil) && tt.wantErr {
				t.Fatalf("check token fail %s body:%s", err, resp)
				return
			}
			netErr := &neterrors.NetError{}
			json.Unmarshal([]byte(resp), netErr)
			if netErr.Code != 0 && status != (tt.args.status) && tt.wantErr {
				t.Fatalf("check token fail %s body:%s status err", err, resp)
				return
			}
		})
	}
}

func TestTraceLog(t *testing.T) {
	go startGrpcServer()

	handler1 := func(f HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req *HttpRequest, resp *HttpResponse) error {
			startTime := time.Now()
			f(ctx, req, resp)
			costTime := time.Since(startTime)

			body, _ := req.Read()
			successStr := fmt.Sprintf("success cost:[%v] url:[%v] req:[%v] resp:[%v]", costTime.Milliseconds(), req.Uri(), string(body), string(resp.Content()))
			logger.Infof(successStr)
			return nil
		}
	}

	customHandler := NewGrpcHandler(
		HttpWrapHandler(handler1),
		HttpGrpcPort(10000))

	http.HandleFunc("/rpc/", func(w http.ResponseWriter, r *http.Request) {
		customHandler.Handle(w, r)
	})
	go func() {
		err := http.ListenAndServe("0.0.0.0:8080", nil)
		if err != nil {
			logger.Infof("start server fail, err:%s", err)
		} else {
			logger.Infof("start server success")
		}
	}()

	type args struct {
		token  string
		status int
		url    string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{"拦截日志", args{"Testing", 200, "/rpc/sso/AuthService.RefleshUrl"}, true},
	}

	grpcclient.SetServerName2Addr(map[string]string{"sso:10000": "localhost:10000"})
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, status, err := postData(tt.args.url, "{\"id\":111}", tt.args.token)
			if (err != nil) && tt.wantErr {
				t.Fatalf("check token fail %s body:%s", err, resp)
				return
			}
			netErr := &neterrors.NetError{}
			json.Unmarshal([]byte(resp), netErr)
			if netErr.Code != 0 && status != (tt.args.status) && tt.wantErr {
				t.Fatalf("check token fail %s body:%s status err", err, resp)
				return
			}
		})
	}
}
