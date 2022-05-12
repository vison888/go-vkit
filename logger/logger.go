package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	DebugLevel  = Level(1)
	InfoLevel   = Level(2)
	WarnLevel   = Level(3)
	ErrorLevel  = Level(4)
	FileMaxLine = 65536
)

// Level 日志输出级别。
type Level int32

func (level Level) String() string {
	switch level {
	case DebugLevel:
		return "[debug]"
	case InfoLevel:
		return "[info]"
	case WarnLevel:
		return "[warn]"
	case ErrorLevel:
		return "[error]"
	default:
		return "[none]"
	}
}

var (
	nodeName string
	appName  string
	podName  string
	podInfo  string
	logDir   string

	mutex     sync.Mutex
	stdWrite  io.Writer
	logFile   *os.File
	outputBuf []byte
	lineCount int32
)

func getSystemEnvAndCmdArg() map[string]string {
	m := make(map[string]string)
	for _, env := range os.Environ() {
		ss := strings.SplitN(env, "=", 2)
		k := ss[0]
		if len(k) > 0 && len(ss) > 1 {
			v := ss[1]
			m[k] = v
		}
	}

	for i := 0; i < len(os.Args); i++ {
		s := os.Args[i]
		if strings.HasPrefix(s, "--") {
			ss := strings.SplitN(strings.TrimPrefix(s, "--"), "=", 2)
			k, v := ss[0], ""
			if len(ss) > 1 {
				v = ss[1]
			}
			m[k] = v
			continue
		}
	}
	return m
}

func init() {
	m := getSystemEnvAndCmdArg()
	key := "NODE_NAME"
	nodeName = ""
	if val, ok := m[key]; ok {
		nodeName = val
		podInfo = podInfo + fmt.Sprintf("[%s]", nodeName)
	}

	key = "POD_NAME"
	podName = ""
	if val, ok := m[key]; ok {
		podName = val
		podInfo = podInfo + fmt.Sprintf("[%s]", podName)
	}

	key = "APP_NAME"
	appName = ""
	if val, ok := m[key]; ok {
		appName = val
		podInfo = podInfo + fmt.Sprintf("[%s] ", appName)
	}

	stdWrite = os.Stdout
	key = "LOG_DIR"
	logDir = "./logs/"
	if val, ok := m[key]; ok {
		logDir = val
	}
	tryNewFile(true)
}

//logs/appname/podname.time.log
func tryNewFile(force bool) {
	if lineCount > FileMaxLine || force {
		// builf file path
		timeStr := time.Now().Format("2006-01-02-15:04:05")
		fileDir := fmt.Sprintf("%s%s", logDir, appName)
		filePath := fmt.Sprintf("%s/%s.%s.log", fileDir, podName, timeStr)
		//try create dir
		_, err := os.Stat(fileDir)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(fileDir, os.ModePerm)
				if err != nil {
					fmt.Println("create forder fail fileDir=:" + fileDir)
					return
				}
			}
		}
		// new file
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			fmt.Println("open log file failed, err:", err)
			return
		}
		lineCount = 0
		if logFile != nil {
			logFile.Close()
		}
		logFile = file
	}
}

func formatAndWrite(l Level, format string, v ...interface{}) {
	now := time.Now()
	mutex.Lock()
	defer mutex.Unlock()

	outputBuf = outputBuf[:0]
	formatHeader(&outputBuf, l, now)
	s := fmt.Sprintf(format, v...)
	outputBuf = append(outputBuf, s...)
	outputBuf = append(outputBuf, '\n')
	stdWrite.Write(outputBuf)
	logFile.Write(outputBuf)
	lineCount++
	tryNewFile(false)
}

//[level][time][NODE_NAME][POD_NAME][APP_NAME] msg
func formatHeader(buf *[]byte, l Level, t time.Time) {
	*buf = append(*buf, l.String()...)
	timeStr := t.Format("[2006-01-02 15:04:05.000000]")
	*buf = append(*buf, timeStr...)
	*buf = append(*buf, podInfo...)
}

func Infof(format string, v ...interface{}) {
	formatAndWrite(InfoLevel, format, v...)
}

func Warnf(format string, v ...interface{}) {
	formatAndWrite(WarnLevel, format, v...)
}

func Errorf(format string, v ...interface{}) {
	formatAndWrite(ErrorLevel, format, v...)
}

func Debugf(format string, v ...interface{}) {
	formatAndWrite(DebugLevel, format, v...)
}

func Info(format string, v ...interface{}) {
	Infof(format, v...)
}

func Warn(format string, v ...interface{}) {
	Warnf(format, v...)
}

func Error(format string, v ...interface{}) {
	Errorf(format, v...)
}

func Debug(format string, v ...interface{}) {
	Debugf(format, v...)
}

func JsonInfo(format string, v interface{}) {
	bb, e := json.Marshal(v)
	if e != nil {
		Errorf("e:%s", e)
		return
	}

	Debugf(format, string(bb))
}

func CanServerLog(xct string) bool {
	return !strings.Contains(xct, "multipart/form-data")
}
