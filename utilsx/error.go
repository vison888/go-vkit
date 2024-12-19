package utilsx

import (
	"reflect"

	"github.com/vison888/go-vkit/errorsx/neterrors"
)

func FetchErrWithCode(resp any, code int, err error) {
	respv := reflect.ValueOf(resp)
	codeField := respv.Elem().FieldByName("Code")
	msgField := respv.Elem().FieldByName("Msg")
	if verr, ok := err.(*neterrors.NetError); ok {
		if codeField.CanSet() {
			codeField.Set(reflect.ValueOf(verr.Code))
		}
		if msgField.CanSet() {
			msgField.Set(reflect.ValueOf(verr.Msg))
		}
	} else {
		if codeField.CanSet() {
			codeField.Set(reflect.ValueOf(code))
		}
		if msgField.CanSet() {
			msgField.Set(reflect.ValueOf(err.Error()))
		}
	}
}

func FetchErr(resp any, err error) {
	respv := reflect.ValueOf(resp)
	codeField := respv.Elem().FieldByName("Code")
	msgField := respv.Elem().FieldByName("Msg")
	if verr, ok := err.(*neterrors.NetError); ok {
		if codeField.CanSet() {
			codeField.Set(reflect.ValueOf(verr.Code))
		}
		if msgField.CanSet() {
			msgField.Set(reflect.ValueOf(verr.Msg))
		}
	} else {
		if codeField.CanSet() {
			codeField.Set(reflect.ValueOf(-1))
		}
		if msgField.CanSet() {
			msgField.Set(reflect.ValueOf(err.Error()))
		}
	}
}

func ErrMsg(err error) string {
	if verr, ok := err.(*neterrors.NetError); ok {
		return verr.Msg
	}
	return err.Error()
}
