package mediasoup

import (
	"fmt"
	"path"
	"runtime"

	"github.com/sirupsen/logrus"
)

var logger = logrus.New()

func init() {
	logger.AddHook(ContextHook{})
}

func Logger() *logrus.Logger {
	return logger
}

func AppLogger() logrus.FieldLogger {
	return logger.WithField("app", "mediasoup")
}

func TypeLogger(value string) logrus.FieldLogger {
	return AppLogger().WithField("type", value)
}

type ContextHook struct{}

func (hook ContextHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

func (hook ContextHook) Fire(entry *logrus.Entry) error {
	if pc, file, line, ok := runtime.Caller(10); ok {
		funcName := runtime.FuncForPC(pc).Name()

		entry.Data["source"] = fmt.Sprintf("%s:%v:%s", path.Base(file), line, path.Base(funcName))
	}

	return nil
}
