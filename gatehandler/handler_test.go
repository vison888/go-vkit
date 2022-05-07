package gatehandler

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSever(t *testing.T) {
	tokenCheck := func(w http.ResponseWriter, r *http.Request) bool {
		return strings.Contains(r.Header.Get("AuthToken"), "Testing")
	}

	http.HandleFunc("/rpc/", func(w http.ResponseWriter, r *http.Request) {
		Handle(w, r, tokenCheck)
	})
	go http.ListenAndServe(":9999", nil)
	time.Sleep(1000)
}
