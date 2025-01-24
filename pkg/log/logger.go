package log

import "github.com/rs/zerolog"

var logger *zerolog.Logger

func SetLogger(l *zerolog.Logger) {
	logger = l
}

func Log() *zerolog.Logger {
	return logger
}

var GetLogger = Log
