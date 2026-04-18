package fmt

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/cors1900/kitchen/file"

	"github.com/fatih/color"
	"github.com/pkg/errors"
)

var Sprintf = fmt.Sprintf
var Printf = fmt.Printf
var Sscanf = fmt.Sscanf
var Scanln = fmt.Scanln
var Scanf = fmt.Scanf

var appName = "app: "

const (
	levelFatal   = 5
	levelError   = 4
	levelWarning = 3
	levelInfo    = 2
	levelDebug   = 1
)

var isDebug bool

func SetDebug(v bool) {
	isDebug = v
}

func SetAppName(name string) {
	appName = name
}

func getLabel(level int) string {
	switch level {
	case levelFatal:
		return "FATAL"
	case levelError:
		return "ERROR"
	case levelWarning:
		return "WARNING"
	case levelDebug:
		return "DEBUG"
	default:
		return ""
	}
}

func Clear() {
	fmt.Printf("\033[2K\r")
}
func callerInfo(skip int) (funcName, fileName string, line int) {
	pc, file, line, ok := runtime.Caller(skip)
	if !ok {
		return "unknown", "unknown", 0
	}
	funcName = runtime.FuncForPC(pc).Name()
	fileName = filepath.Base(file)
	return
}

func msgf(c *color.Color, level int, msg string) {
	if c == nil {
		c = color.New()
	}

	label := getLabel(level)
	var prefix string
	prefix = prefix + appName
	if len(label) > 0 {
		prefix = prefix + label + " "
	}

	if isDebug {
		_, fileName, line := callerInfo(3)
		fileName = filepath.Base(fileName)
		const maxLen = 13
		if len(fileName) > maxLen {
			fileName = "~" + fileName[len(fileName)-maxLen+1:]
		}
		prefix = Sprintf("[%-*s:%3d] ", maxLen, fileName, line) + prefix
	}

	if len(msg) > 0 && msg[0] == '\r' {
		prefix = "\033[2K\r" + prefix
		msg = msg[1:]
	}

	if !strings.Contains(msg, "\n") {
		c.Printf("%s%s", prefix, msg)
		return
	}

	c.Printf("%s", prefix)

	for {
		i := strings.Index(msg, "\n")
		if i < 0 {
			c.Printf("%s", msg)
			break
		}
		c.Printf(msg[:i+1])
		msg = msg[i+1:]
	}
}

func Debug(format string, args ...any) {
	if !isDebug {
		return
	}
	msgf(color.New(), levelDebug, fmt.Sprintf(format, args...)+"\n")
}

func Infof(format string, args ...any) {
	msgf(color.New(), levelInfo, Sprintf(format, args...))
}

// 自动添加换行符
func Info(format string, args ...any) {
	msgf(color.New(), levelInfo, Sprintf(format, args...)+"\n")
}

func Errorf(format string, args ...any) {
	msgf(color.New(color.FgRed), levelError, fmt.Sprintf(format, args...))
}

func Error(format string, args ...any) {
	msgf(color.New(color.FgRed), levelError, fmt.Sprintf(format, args...)+"\n")
}

func Warningf(format string, args ...any) {
	msgf(color.New(color.FgYellow), levelWarning, fmt.Sprintf(format, args...))
}

func Fatalf(format string, args ...any) {
	msgf(color.New(color.FgRed), levelFatal, fmt.Sprintf(format, args...))
	os.Exit(1)
}

func Size(s int64) string {
	if s > 1024*1024 {
		return fmt.Sprintf("%.1f MB", float64(s)/1024/1024)
	}
	if s > 1024 {
		return fmt.Sprintf("%.1f KB", float64(s)/1024)
	}
	return fmt.Sprintf("%d Byte", s)
}

func Duration(d time.Duration) string {
	isNegative := d < 0
	if isNegative {
		d = -d
	}
	ms := d.Milliseconds()
	hours := ms / 1000 / 60 / 60
	minutes := ms % (1000 * 60 * 60) / 1000 / 60
	seconds := float64(ms-minutes*60*1000) / 1000

	var parts []string
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%d h", hours))
	}
	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%d min", minutes))
	}
	// 秒部分始终显示，保留两位小数
	parts = append(parts, fmt.Sprintf("%.2f s", seconds))

	result := strings.Join(parts, " ")
	if isNegative {
		result = "-" + result
	}
	return result
}
func Tree(dir string, prefix string, handler func(prefix, item string, isDir bool)) error {
	inner_pointers := []string{"├──", "│  "}
	final_pointers := []string{"└──", "   "}

	isDir := file.IsDir(dir)
	if prefix == "" {
		handler(prefix, dir, isDir)
	}

	if !isDir {
		return nil
	}

	items, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return errors.WithStack(err)
	}

	for index, item := range items {
		var pointers []string
		if index == len(items)-1 {
			pointers = final_pointers
		} else {
			pointers = inner_pointers
		}

		// line := fmt.Sprintf("%s%s", prefix, pointers[0])
		// Info("%s%s%s", prefix, pointers[0], filepath.Base(item))
		isDir := file.IsDir(item)
		handler(prefix+pointers[0], item, isDir)
		if isDir {
			if err := Tree(item, prefix+pointers[1], handler); err != nil {
				return err
			}
			continue
		}
	}
	return nil
}
