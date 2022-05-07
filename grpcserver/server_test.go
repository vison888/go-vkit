package grpcserver

import (
	"testing"
	"time"
)

func TestSever(t *testing.T) {
	svr := NewServer()
	go svr.Run("0.0.0.0:10000")
	time.Sleep(1000)
}
