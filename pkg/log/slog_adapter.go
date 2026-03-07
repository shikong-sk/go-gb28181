package log

import (
	"context"
	"log/slog"

	"github.com/rs/zerolog"
)

// SlogHandler 将 zerolog 适配为 slog.Handler
// 用于 sipgo 等使用 slog 的库
type SlogHandler struct {
	logger *zerolog.Logger
}

// NewSlogHandler 创建 slog 适配器
func NewSlogHandler(logger *zerolog.Logger) *SlogHandler {
	return &SlogHandler{logger: logger}
}

// Enabled 实现 slog.Handler 接口
func (h *SlogHandler) Enabled(_ context.Context, level slog.Level) bool {
	switch level {
	case slog.LevelDebug:
		return h.logger.GetLevel() <= zerolog.DebugLevel
	case slog.LevelInfo:
		return h.logger.GetLevel() <= zerolog.InfoLevel
	case slog.LevelWarn:
		return h.logger.GetLevel() <= zerolog.WarnLevel
	case slog.LevelError:
		return h.logger.GetLevel() <= zerolog.ErrorLevel
	default:
		return false
	}
}

// Handle 实现 slog.Handler 接口
func (h *SlogHandler) Handle(_ context.Context, r slog.Record) error {
	var event *zerolog.Event
	switch r.Level {
	case slog.LevelDebug:
		event = h.logger.Debug()
	case slog.LevelInfo:
		event = h.logger.Info()
	case slog.LevelWarn:
		event = h.logger.Warn()
	case slog.LevelError:
		event = h.logger.Error()
	default:
		event = h.logger.Info()
	}

	event.Str("caller", r.Source().Function).Msg(r.Message)
	return nil
}

// WithAttrs 实现 slog.Handler 接口
func (h *SlogHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	// 简化实现，忽略属性
	return h
}

// WithGroup 实现 slog.Handler 接口
func (h *SlogHandler) WithGroup(name string) slog.Handler {
	// 简化实现，忽略分组
	return h
}

// SlogLogger 返回适配后的 slog.Logger
func SlogLogger() *slog.Logger {
	return slog.New(NewSlogHandler(&Logger))
}

// NewSlogLogger 创建 slog.Logger 用于指定 zerolog.Logger
func NewSlogLogger(logger *zerolog.Logger) *slog.Logger {
	return slog.New(NewSlogHandler(logger))
}
