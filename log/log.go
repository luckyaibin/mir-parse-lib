package log

import (
	"fmt"
	"runtime"
	"strings"
	"time"
)

const (
	TRACE byte = iota
	DEBUG
	INFO
	WARN
	ERROR
	FATAL
)

type levelColor struct {
	levelStr string
	colorStr string
}

var levelColorMap = [...]levelColor{
	TRACE: {"TRACE", "\033[1;34m"},
	DEBUG: {"DEBUG", "\033[1;36m"},
	INFO:  {"INFO ", "\033[1;37m"},
	WARN:  {"WARN ", "\033[1;33m"},
	ERROR: {"ERROR", "\033[1;31m"},
	FATAL: {"FATAL", "\033[1;41m"},
}

var globalConsoleLogger *consoleLogger
var globalLevel byte

func init() {
	var level string
	level = "trace"
	globalConsoleLogger = &consoleLogger{}
	switch level {
	case "trace":
		globalLevel = TRACE
	case "debug":
		globalLevel = DEBUG
	case "info":
		globalLevel = INFO
	case "warn":
		globalLevel = WARN
	case "error":
		globalLevel = ERROR
	case "fatal":
		globalLevel = FATAL
	default:
		panic("invalid log Level: " + level)
	}
}
func Tracef(format string, args ...interface{}) {
	if globalLevel > TRACE {
		return
	}
	globalConsoleLogger.Log(TRACE,
		fmt.Sprintf(format, args...), "", 0)
}

func Debugf(format string, args ...interface{}) {
	if globalLevel > DEBUG {
		return
	}
	globalConsoleLogger.Log(DEBUG,
		fmt.Sprintf(format, args...), "", 0)
}

func Infof(format string, args ...interface{}) {
	if globalLevel > INFO {
		return
	}
	globalConsoleLogger.Log(INFO,
		fmt.Sprintf(format, args...), "", 0)
}

func Warnf(format string, args ...interface{}) {
	if globalLevel > WARN {
		return
	}
	_, fn, ln, _ := runtime.Caller(1)
	globalConsoleLogger.Log(WARN,
		fmt.Sprintf(format, args...), fn, ln)
}

func Errorf(format string, args ...interface{}) {
	_, fn, ln, _ := runtime.Caller(1)
	globalConsoleLogger.Log(ERROR,
		fmt.Sprintf(format, args...), fn, ln)
}

func Fatalf(format string, args ...interface{}) {
	_, fn, ln, _ := runtime.Caller(1)
	globalConsoleLogger.Log(FATAL,
		fmt.Sprintf(format, args...), fn, ln)
	panic(fmt.Sprintf(format, args...))
}

func Println(args ...interface{}) {
	if globalLevel > TRACE {
		return
	}
	format := strings.Repeat("%v", len(args))
	globalConsoleLogger.Log(TRACE, fmt.Sprintf(format, args...), "", 0)
}

type consoleLogger struct {
}

func (c *consoleLogger) Log(lvl byte, content string, file string, line int) {
	level := lvl
	sb := strings.Builder{}
	now := time.Now()
	sb.WriteString(now.Format("2006-01-02 15:04:05.000"))
	sb.WriteString(levelColorMap[level].colorStr)
	sb.WriteString(" " + levelColorMap[level].levelStr)

	if len(file) != 0 {
		sb.WriteString(fmt.Sprintf(" %s(%d)", file, line))
	}
	sb.WriteString(" " + content)
	sb.WriteString("\033[0m") //颜色结束。8进制表示的27
	fmt.Println(sb.String())
}
