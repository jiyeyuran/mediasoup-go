package mediasoup

import "github.com/sirupsen/logrus"

const APP_NAME = "mediasoup"

var defaultLogger logrus.FieldLogger = logrus.WithField("app", APP_NAME)

func GetLogger() logrus.FieldLogger {
	return defaultLogger
}

func SetLogger(logger logrus.FieldLogger) {
	defaultLogger = logger
}
