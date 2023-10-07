package neterrors

import (
	"encoding/json"
	"fmt"

	"github.com/visonlv/go-vkit/logger"
)

type NetError struct {
	Code   int32  `json:"code"`
	Status int32  `json:"status"`
	Msg    string `json:"msg"`
}

func (e *NetError) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}

func New(msg, detail string, code int32, status int32) error {
	return &NetError{
		Msg:    msg,
		Code:   code,
		Status: status,
	}
}

func FromError(err error) *NetError {
	if err == nil {
		return nil
	}
	if verr, ok := err.(*NetError); ok && verr != nil {
		return verr
	}

	return Parse(err.Error())
}

func Parse(err string) *NetError {
	e := new(NetError)
	errr := json.Unmarshal([]byte(err), e)
	if errr != nil {
		logger.Infof("parse fail %s", e)
		e.Msg = e.Error()
	}
	return e
}

func BusinessError(code int32, format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   code,
		Status: 200,
	}
}

func BadRequest(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 400,
	}
}

func Unauthorized(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 401,
	}
}

func Forbidden(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 403,
	}
}

func NotFound(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 404,
	}
}

func MethodNotAllowed(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 405,
	}
}

func Timeout(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 408,
	}
}

func Conflict(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 409,
	}
}

func InternalServerError(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 500,
	}
}

func NotImplemented(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 501,
	}
}

func BadGateway(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 502,
	}
}

func ServiceUnavailable(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 503,
	}
}

func GatewayTimeout(format string, a ...any) error {
	return &NetError{
		Msg:    fmt.Sprintf(format, a...),
		Code:   -1,
		Status: 504,
	}
}
