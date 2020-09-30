package mediasoup

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// Level defines log levels.
type Level = zerolog.Level

const (
	// DebugLevel defines debug log level.
	DebugLevel = zerolog.DebugLevel
	// WarnLevel defines warn log level.
	WarnLevel = zerolog.WarnLevel
	// ErrorLevel defines error log level.
	ErrorLevel = zerolog.ErrorLevel
	// NoLevel defines an absent log level.
	NoLevel = zerolog.NoLevel
)

var DefaultLevel = DebugLevel

type Logger interface {
	Debug(format string, v ...interface{})
	Warn(format string, v ...interface{})
	Error(format string, v ...interface{})
}

var NewLogger = func(prefix string) Logger {
	return NewDefaultLogger(prefix)
}

func NewNopLogger(prefix string) Logger {
	return &DefaultLogger{
		logger: zerolog.Nop(),
	}
}

type DefaultLogger struct {
	logger zerolog.Logger
}

func NewDefaultLogger(prefix string) *DefaultLogger {
	writer := zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
	context := zerolog.New(writer).With().Timestamp()

	if len(prefix) > 0 {
		context = context.Str(zerolog.CallerFieldName, prefix)
	}

	return &DefaultLogger{
		logger: context.Logger().Level(DefaultLevel),
	}
}

func (l DefaultLogger) GetLogger() zerolog.Logger {
	return l.logger
}

func (l *DefaultLogger) SetLogger(logger zerolog.Logger) {
	l.logger = logger
}

func (l DefaultLogger) Debug(format string, v ...interface{}) {
	l.logger.Debug().Msgf(format, v...)
}

func (l DefaultLogger) Warn(format string, v ...interface{}) {
	l.logger.Warn().Msgf(format, v...)
}

func (l DefaultLogger) Error(format string, v ...interface{}) {
	l.logger.Error().Msgf(format, v...)
}
