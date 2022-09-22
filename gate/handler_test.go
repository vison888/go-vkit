package gate

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/visonlv/go-vkit/errorsx/neterrors"
	"github.com/visonlv/go-vkit/logger"
)

func TestSever(t *testing.T) {
	tokenCheck := func(w http.ResponseWriter, r *http.Request) error {
		token := r.Header.Get("AuthToken")
		if token == "" {
			return neterrors.Unauthorized("header not conatain token")
		}
		if strings.Contains(token, "Testing") {
			return nil
		}
		return neterrors.Forbidden("url not permission")
	}

	http.HandleFunc("/rpc/", func(w http.ResponseWriter, r *http.Request) {
		NewHttpHandler(10000, tokenCheck).Handle(w, r)
	})
	go http.ListenAndServe(":9999", nil)
	logger.Info("server start")
	time.Sleep(1000)
}
