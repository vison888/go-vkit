package httpclient

import (
	"testing"
	"time"

	"github.com/visonlv/go-vkit/logger"
)

func TestSever(t *testing.T) {
	resp, err := NewRequest().Get("https://baidu.com")
	logger.Infof("resp:%v err:%v", resp, err)
	time.Sleep(2000)
}
