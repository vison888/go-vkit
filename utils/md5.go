package utils

import (
	"crypto/md5"
	"fmt"
)

func Md5Encode(str string) (strMd5 string) {
	strByte := []byte(str)
	strMd5Byte := md5.Sum(strByte)
	strMd5 = fmt.Sprintf("%x", strMd5Byte)
	return strMd5
}
