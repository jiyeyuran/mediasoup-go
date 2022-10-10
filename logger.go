package mediasoup

import (
	"os"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	"github.com/go-logr/zerologr"
	"github.com/gobwas/glob"
	"github.com/rs/zerolog"
)

var (
	// defaultLoggerImpl is a zerolog instance with console writer
	defaultLoggerImpl = zerolog.New(zerolog.NewConsoleWriter(func(w *zerolog.ConsoleWriter) {
		color, _ := strconv.ParseBool(os.Getenv("DEBUG_COLORS"))
		w.NoColor = !color
		w.TimeFormat = "2006-01-02 15:04:05.999"
	})).With().Timestamp().Caller().Logger()

	defaultLoggerLevel = zerolog.InfoLevel

	// NewLogger defines function to create logger instance.
	NewLogger = func(scope string) logr.Logger {
		shouldDebug := false
		if debug := os.Getenv("DEBUG"); len(debug) > 0 {
			for _, part := range strings.Split(debug, ",") {
				part := strings.TrimSpace(part)
				if len(part) == 0 {
					continue
				}
				shouldMatch := true
				if part[0] == '-' {
					shouldMatch = false
					part = part[1:]
				}
				if g, err := glob.Compile(part); err == nil && g.Match(scope) {
					shouldDebug = shouldMatch
				}
			}
		}

		level := defaultLoggerLevel

		if shouldDebug {
			level = zerolog.DebugLevel
		}

		logger := defaultLoggerImpl.Level(level)

		return zerologr.New(&logger).WithName(scope)
	}
)

func init() {
	zerolog.TimeFieldFormat = "2006-01-02T15:04:05.999Z07:00"
	zerologr.VerbosityFieldName = ""
}
