package errorsx

import (
	"encoding/json"
	"fmt"
)

var (
	OK         = NewErrno(0, 0, "OK")
	FAIL       = NewErrno(0, -1, "未知错误")
	PARAM_ERR  = NewErrno(0, -2, "参数错误")
	SYSTEM_ERR = NewErrno(0, -3, "系统异常")
)

type Errno struct {
	Project int32  `json:"project"`
	Code    int32  `json:"code"`
	Msg     string `json:"msg"`
}

func NewErrno(project int32, code int32, msg string) *Errno {
	if project > 1000 {
		panic("project invalid, should be <= 1000")
	}
	if code > 999 {
		panic("code invalid, should be 1~999")
	}
	err := &Errno{Project: project, Code: project*1000 + code, Msg: msg}
	return err
}

func (e *Errno) Fail(format string, v ...interface{}) *Errno {
	ret := &Errno{Project: e.Project, Code: e.Code, Msg: fmt.Sprintf(format, v...)}
	return ret
}

func (e *Errno) Error() string {
	b, _ := json.Marshal(e)
	return string(b)
}
