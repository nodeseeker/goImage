package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// LogLevel 日志级别
type LogLevel int

const (
	DebugLevel LogLevel = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

var (
	// 当前日志级别，默认为 InfoLevel
	currentLevel = InfoLevel

	// 日志前缀
	levelPrefix = map[LogLevel]string{
		DebugLevel: "[DEBUG]",
		InfoLevel:  "[INFO] ",
		WarnLevel:  "[WARN] ",
		ErrorLevel: "[ERROR]",
		FatalLevel: "[FATAL]",
	}
)

// SetLevel 设置当前日志级别
func SetLevel(level LogLevel) {
	currentLevel = level
}

// GetLevel 获取当前日志级别
func GetLevel() LogLevel {
	return currentLevel
}

// Debug 记录调试级别的日志
func Debug(format string, v ...interface{}) {
	if currentLevel <= DebugLevel {
		logWithCaller(DebugLevel, format, v...)
	}
}

// Info 记录普通信息级别的日志
func Info(format string, v ...interface{}) {
	if currentLevel <= InfoLevel {
		logWithCaller(InfoLevel, format, v...)
	}
}

// Warn 记录警告级别的日志
func Warn(format string, v ...interface{}) {
	if currentLevel <= WarnLevel {
		logWithCaller(WarnLevel, format, v...)
	}
}

// Error 记录错误级别的日志
func Error(format string, v ...interface{}) {
	if currentLevel <= ErrorLevel {
		logWithCaller(ErrorLevel, format, v...)
	}
}

// Fatal 记录严重错误级别的日志并退出程序
func Fatal(format string, v ...interface{}) {
	if currentLevel <= FatalLevel {
		logWithCaller(FatalLevel, format, v...)
	}
	os.Exit(1)
}

// logWithCaller 记录带调用者信息的日志
func logWithCaller(level LogLevel, format string, v ...interface{}) {
	// 获取调用者信息
	_, file, line, ok := runtime.Caller(2)
	callerInfo := "???"
	if ok {
		callerInfo = fmt.Sprintf("%s:%d", filepath.Base(file), line)
	}

	// 时间戳
	timestamp := time.Now().Format("2006-01-02 15:04:05.000")

	// 构造并记录日志
	prefix := fmt.Sprintf("%s %s %s", timestamp, levelPrefix[level], callerInfo)
	message := fmt.Sprintf(format, v...)
	log.Printf("%s - %s", prefix, message)
}

// InitLogger 初始化日志系统
func InitLogger(level LogLevel) {
	// 设置日志级别
	SetLevel(level)

	// 设置标准库日志格式
	log.SetFlags(0) // 清除默认前缀，我们会添加自己的前缀
}
