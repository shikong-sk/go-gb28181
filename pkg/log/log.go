package log

import (
	"io"
	"os"
	"path/filepath"
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

// SetupFileLogging 配置日志文件输出。
// 这是必要的公共入口说明：后台运行时主要依赖日志文件排查，
// 因此这里统一把结构化日志同时写到控制台和指定文件，避免丢失现场。
func SetupFileLogging(logFile string) (*os.File, error) {
	if logFile == "" {
		return nil, nil
	}

	dir := filepath.Dir(logFile)
	if dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}

	consoleWriter := zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		w.Out = os.Stdout
		w.TimeFormat = time.RFC3339
	})
	multiWriter := io.MultiWriter(consoleWriter, file)
	Logger = Logger.Output(multiWriter)
	return file, nil
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

// SIPTracer 实现 sip.SIPTracer 接口，用于输出可视化格式的 SIP 日志
type SIPTracer struct{}

// SIPTraceRead 记录 SIP 消息读取
func (t *SIPTracer) SIPTraceRead(transport, laddr, raddr string, sipmsg []byte) {
	Debug().Msgf("%s read from %s <- %s:\n%s", transport, laddr, raddr, string(sipmsg))
}

// SIPTraceWrite 记录 SIP 消息写入
func (t *SIPTracer) SIPTraceWrite(transport, laddr, raddr string, sipmsg []byte) {
	Debug().Msgf("%s write to %s -> %s:\n%s", transport, laddr, raddr, string(sipmsg))
}

// NewSIPTracer 创建 SIP 日志跟踪器
func NewSIPTracer() *SIPTracer {
	return &SIPTracer{}
}
