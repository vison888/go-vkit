package logger

import (
	"bytes"
	"fmt"
	"os/exec"
	"testing"
)

//阻塞式的执行外部shell命令的函数,等待执行完毕并返回标准输出
func exec_shell(s string) (string, error) {
	//函数返回一个*Cmd，用于使用给出的参数执行name指定的程序
	cmd := exec.Command("/bin/bash", "-c", s)
	//读取io.Writer类型的cmd.Stdout，再通过bytes.Buffer(缓冲byte类型的缓冲器)将byte类型转化为string类型(out.String():这是bytes类型提供的接口)
	var out bytes.Buffer
	cmd.Stdout = &out
	//Run执行c包含的命令，并阻塞直到完成。  这里stdout被取出，cmd.Wait()无法正确获取stdin,stdout,stderr，则阻塞在那了
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
	return out.String(), err
}
func TestLogger(t *testing.T) {
	exec_shell("export NODE_NAME=node1 && export POD_NAME=pod1 && export APP_NAME=app1 && export logDir=./logs/")
	Info("0")
	Info("1")
	Info("2")
	Info("3")
	Info("4")
}
