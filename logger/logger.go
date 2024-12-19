package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
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

const (
	SplitTypeDate = iota
	SplitTypeHour
)

type LogCfg struct {
	// 日志保存时间
	KeepSecond int64
	// 日志目录
	LogDir string
	// 日志切割方式
	SplitType int
}

var (
	initOnce sync.Once

	logCfg         LoggerOptions
	wMutex         sync.Mutex
	stdWrite       io.Writer
	logFile        *os.File
	outputBuf      []byte
	lineCount      int32
	createFileDate int32
	createFileHour int32
	deleteFileDate int32
)

type LoggerOptions struct {
	// 日志保存时间
	KeepSecond int64
	// 日志目录
	LogDir string
	// 日志切割方式
	SplitType int
	// 回调
	CallBack func(string)
}

type LoggerOption func(o *LoggerOptions)

func newLoggerOptions(opts ...LoggerOption) LoggerOptions {
	opt := LoggerOptions{
		KeepSecond: 3600 * 24 * 7,
		LogDir:     "./logs/",
		SplitType:  SplitTypeDate,
		CallBack:   nil,
	}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

func WithKeepSecond(keepSecond int64) LoggerOption {
	return func(o *LoggerOptions) {
		o.KeepSecond = keepSecond
	}
}

func WithLogDir(logDir string) LoggerOption {
	return func(o *LoggerOptions) {
		o.LogDir = logDir
	}
}

func WithSplitType(splitType int) LoggerOption {
	return func(o *LoggerOptions) {
		o.SplitType = splitType
	}
}

func WithCallBack(callBack func(string)) LoggerOption {
	return func(o *LoggerOptions) {
		o.CallBack = callBack
	}
}

func tryInit() {
	Init()
}

func Init(opts ...LoggerOption) {
	initOnce.Do(func() {
		logCfg = newLoggerOptions(opts...)
		stdWrite = os.Stdout
		tryNewFile(true)
		go mainloop()
	})

}

// logs/appname/podname.time.log
func tryNewFile(force bool) {
	// 日期不一样或者行数达到上限
	if force || lineCount > FileMaxLine ||
		(logCfg.SplitType == SplitTypeDate && createFileDate != int32(time.Now().YearDay())) ||
		(logCfg.SplitType == SplitTypeHour && createFileHour != int32(time.Now().Hour())) {
		cur := time.Now()
		timeStr := cur.Format("20060102150405")
		fileDir := logCfg.LogDir
		filePath := fmt.Sprintf("%s/%s.log", fileDir, timeStr)
		//try create dir
		_, err := os.Stat(fileDir)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(fileDir, os.ModePerm)
				if err != nil {
					panic(fmt.Sprintf("create forder fail fileDir=:%s err:%s", fileDir, err))
				}
			}
		}
		// new file
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			panic(fmt.Sprintf("open log file failed, err:%s", err))
		}
		lineCount = 0
		createFileDate = int32(cur.YearDay())
		createFileHour = int32(cur.Hour())
		if logFile != nil {
			logFile.Close()
		}
		logFile = file
	}
}

func mainloop() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			now := time.Now()
			tryDelteFile(&now)
		}
	}
}

func tryDelteFile(date *time.Time) {
	// 每日凌晨删除
	if deleteFileDate != int32(date.YearDay()) {
		deleteFileDate = int32(date.YearDay())
		fileInfoList, err := os.ReadDir(logCfg.LogDir)
		if err != nil {
			return
		}

		for i := range fileInfoList {
			fileName := fileInfoList[i].Name()
			filePath := logCfg.LogDir + fileName
			if needDelete(date, filePath) {
				os.Remove(filePath)
				fmt.Printf("delete file%s \n", filePath)
			}
		}
	}
}

func needDelete(date *time.Time, filePath string) bool {
	f, err := os.Open(filePath)
	if err != nil {
		fmt.Printf("needDeletel open file:[%s] fail err:%v \n", filePath, err)
		return false
	}

	fi, err := f.Stat()
	f.Close()
	if err != nil {
		fmt.Printf("needDelete Stat file:[%s] fail err:%v \n", filePath, err)
		return false
	}

	if date.Unix()-fi.ModTime().Unix() > int64(logCfg.KeepSecond) {
		return true
	}

	return false
}

func formatAndWrite(l Level, format string, v ...any) {
	tryInit()
	now := time.Now()
	wMutex.Lock()
	defer wMutex.Unlock()

	outputBuf = outputBuf[:0]
	formatHeader(&outputBuf, l, now)
	s := fmt.Sprintf(format, v...)
	outputBuf = append(outputBuf, s...)
	outputBuf = append(outputBuf, '\n')
	stdWrite.Write(outputBuf)
	logFile.Write(outputBuf)
	lineCount++
	if logCfg.CallBack != nil {
		logCfg.CallBack(string(outputBuf))
	}
	tryNewFile(false)
}

// [level][time][NODE_NAME][POD_NAME][APP_NAME] msg
func formatHeader(buf *[]byte, l Level, t time.Time) {
	*buf = append(*buf, l.String()...)
	timeStr := t.Format("[2006-01-02 15:04:05.000000]")
	*buf = append(*buf, timeStr...)
}

func Infof(format string, v ...any) {
	formatAndWrite(InfoLevel, format, v...)
}

func Warnf(format string, v ...any) {
	formatAndWrite(WarnLevel, format, v...)
}

func Errorf(format string, v ...any) {
	formatAndWrite(ErrorLevel, format, v...)
}

func Debugf(format string, v ...any) {
	formatAndWrite(DebugLevel, format, v...)
}

func Info(v ...any) {
	Infof(fmt.Sprint(v...))
}

func Warn(v ...any) {
	Warnf(fmt.Sprint(v...))
}

func Error(v ...any) {
	Errorf(fmt.Sprint(v...))
}

func Debug(v ...any) {
	Debugf(fmt.Sprint(v...))
}

func JsonInfo(format string, v any) {
	bb, e := json.Marshal(v)
	if e != nil {
		Errorf("e:%s", e)
		return
	}
	Debugf(format, string(bb))
}
