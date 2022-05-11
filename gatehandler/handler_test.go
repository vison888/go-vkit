package gatehandler

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/visonlv/go-vkit/errors/neterrors"
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
		NewHandler().Handle(w, r, tokenCheck)
	})
	go http.ListenAndServe(":9999", nil)
	time.Sleep(1000)
}
