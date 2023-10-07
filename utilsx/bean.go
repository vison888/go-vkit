package utilsx

import (
	"bytes"
	"encoding/gob"
)

// golang 完全是按值传递，所以正常的赋值都是值拷贝，当然如果类型里面嵌套的有指针，也是指针值的拷贝，此时就会出现两个类型变量的内部有一部分是共享的。
func DeepCopy(src, dst any) error {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(src); err != nil {
		return err
	}
	return gob.NewDecoder(bytes.NewBuffer(buf.Bytes())).Decode(dst)
}
