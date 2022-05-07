package auth

import (
	"testing"

	"github.com/visonlv/go-vkit/logger"
)

func TestMatch(t *testing.T) {
	white := []string{"/rpc/speech/*/sdsd", "/haldsfsdf"}
	a := NewAuth(white)
	r := &AuthRole{code: "ADMIN", urls: []string{"/rpc/speech/**", "/haldsfsdf"}}
	a.SetRole(r)

	bbb := a.IsPemission([]string{"ADMIN"}, "/rpc/speech/sdsd/sdsd")
	logger.Infof("===========%v", bbb)
}
