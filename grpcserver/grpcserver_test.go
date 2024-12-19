package grpcserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/vison888/go-vkit/errorsx/neterrors"
	"github.com/vison888/go-vkit/gate"
	"github.com/vison888/go-vkit/grpcclient"
	"github.com/vison888/go-vkit/grpcx"
	"github.com/vison888/go-vkit/logger"
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
	url = "http://localhost:8081" + url
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

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return "", -1, err
	}

	return string(body), res.StatusCode, nil
}

func startGrpcServer() {
	handler1 := func(f HandlerFunc) HandlerFunc {
		return func(ctx context.Context, req *GrpcRequest, resp any) error {
			startTime := time.Now()
			f(ctx, req, resp)
			costTime := time.Since(startTime)

			bb, _ := json.Marshal(req.payload)
			bb2, _ := json.Marshal(resp)

			successStr := fmt.Sprintf("success cost:[%v] url:[%v] req:[%v] resp:[%v]", costTime.Milliseconds(), req.method, string(bb), string(bb2))
			logger.Infof(successStr)
			return nil
		}
	}

	svr := NewServer(
		Name("auth"),
		GrpcWrapHandler(handler1),
	)
	err := svr.RegisterApiEndpoint([]any{&AuthService{}}, []*grpcx.ApiEndpoint{{
		Method:       "AuthService.RefleshUrl",
		Url:          "/rpc/sso/AuthService.RefleshUrl",
		ClientStream: false,
		ServerStream: false,
	}})
	if err != nil {
		logger.Errorf("[main] RegisterApiEndpoint fail %s", err)
		panic(err)
	}

	svr.Run()
}

func TestGrpcServerStart(t *testing.T) {
	go startGrpcServer()

	customHandler := gate.NewGrpcHandler(
		gate.HttpGrpcPort(10000))

	http.HandleFunc("/rpc/", func(w http.ResponseWriter, r *http.Request) {
		customHandler.Handle(w, r)
	})
	go func() {
		err := http.ListenAndServe("0.0.0.0:8081", nil)
		if err != nil {
			logger.Infof("start server fail, err:%s", err)
		} else {
			logger.Infof("start server success")
		}
	}()

	time.Sleep(time.Second)
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

	logger.Infof("server start")

}
