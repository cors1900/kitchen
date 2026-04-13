package log

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const (
	TimeFormat = "2006-01-02 15:04:05"
	Red        = "\033[31m"
	Yellow     = "\033[33m"
	Green      = "\033[32m"
	ColorEnd   = "\033[0m"
)

func getCallerInfo(skip int) (funcName, fileName string, line int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", "unknown", 0
	}
	funcName = runtime.FuncForPC(pc).Name()
	fileName = filepath.Base(file)
	return
}

func Info(format string, a ...any) {
	logPrint("I", Green, format, a...)
}
func Infof(format string, a ...any) {
	logPrint("I", Green, format, a...)
}

func Debug(format string, a ...any) {
	logPrint("D", "", format, a...)
}

func Debugf(format string, a ...any) {
	logPrint("D", "", format, a...)
}

func Warn(msg string, a ...any) {
	logPrint("W", Red, msg, a...)
}

func Warnf(msg string, a ...any) {
	logPrint("W", Red, msg, a...)
}

func Error(format string, a ...any) {
	logPrint("E", Red, format, a...)
}

func Errorf(format string, a ...any) {
	logPrint("E", Red, format, a...)
}

func Fatalf(format string, a ...any) {
	logPrint("E", Red, format, a...)
	os.Exit(1)
}

func logPrint(level, color, msg string, args ...any) {
	_, file, line := getCallerInfo(3)
	// file = strings.TrimRight(file, filepath.Ext(file))
	const maxLen = 12
	if len(file) > maxLen {
		file = "..." + file[len(file)-maxLen:]
	}
	fmt.Printf("%s[%s %s %d] %s%s\n", color, level, file, line, fmt.Sprintf(msg, args...), ColorEnd)
}
