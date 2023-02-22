package neterrors

import (
	er "errors"
	"testing"
)

func TestFromError(t *testing.T) {
	err := NotFound("err msg %s", "example")
	merr := FromError(err)
	if merr.Code != -1 {
		t.Fatalf("invalid conversation %v != %v", err, merr)
	}
	err = er.New(err.Error())
	merr = FromError(err)
	if merr.Status != 404 {
		t.Fatalf("invalid conversation %v != %v", err, merr)
	}

}
