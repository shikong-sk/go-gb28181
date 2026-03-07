package log

import (
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Logger 全局日志实例
var Logger zerolog.Logger

// SetLogger 设置全局日志实例
func SetLogger(l *zerolog.Logger) {
	Logger = *l
}

// GetLogger 获取全局日志实例
func GetLogger() *zerolog.Logger {
	return &Logger
}

// Log 获取全局日志实例 (向后兼容)
func Log() *zerolog.Logger {
	return &Logger
}

// InitLogger 初始化日志
func InitLogger(debug bool) zerolog.Logger {
	output := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stdout
		w.TimeFormat = time.RFC3339
	})

	logger := zerolog.New(output).With().Timestamp().Logger()

	if debug {
		logger = logger.Level(zerolog.DebugLevel)
	} else {
		logger = logger.Level(zerolog.InfoLevel)
	}

	Logger = logger
	return logger
}

// SetOutput 设置日志输出目标
func SetOutput(w io.Writer) {
	Logger = Logger.Output(w)
}

// Debug 记录调试日志
func Debug() *zerolog.Event {
	return Logger.Debug()
}

// Info 记录信息日志
func Info() *zerolog.Event {
	return Logger.Info()
}

// Warn 记录警告日志
func Warn() *zerolog.Event {
	return Logger.Warn()
}

// Error 记录错误日志
func Error() *zerolog.Event {
	return Logger.Error()
}

// Fatal 记录致命日志并退出
func Fatal() *zerolog.Event {
	return Logger.Fatal()
}
