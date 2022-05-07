package errors

import (
	"encoding/json"
	"fmt"
)

var (
	OK         error = &Errno{Project: 0, Code: 0, Msg: "OK"}
	PARAME_ERR error = &Errno{Project: 0, Code: -1, Msg: "参数错误"}
	SYSTEM_ERR error = &Errno{Project: 0, Code: -2, Msg: "系统异常"}
)

type Errno struct {
	Project int32  `json:"project"`
	Code    int32  `json:"code"`
	Msg     string `json:"msg"`
}

func NewErrno(project int32, code int32, msg string) *Errno {
	if project < 1000 {
		panic("project invalid, should be >= 1000")
	}
	if code < 1 || code > 999 {
		panic("code invalid, should be 1~999")
	}
	err := &Errno{Project: project, Code: code, Msg: msg}
	err.Project = project
	return err
}

func (e *Errno) GetFullCode() int32 {
	return e.Project*1000 + int32(e.Code)
}

func (e *Errno) Fail(format string, v ...interface{}) *Errno {
	ret := &Errno{Project: e.Project, Code: e.Code, Msg: ""}
	ret.Msg = fmt.Sprintf(format, v...)
	return ret
}

func (e *Errno) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}
