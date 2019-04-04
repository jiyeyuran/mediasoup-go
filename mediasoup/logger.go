package mediasoup

import "github.com/sirupsen/logrus"

var logger = logrus.New()

func Logger() *logrus.Logger {
	return logger
}

func AppLogger() logrus.FieldLogger {
	return logger.WithField("app", "mediasoup")
}
