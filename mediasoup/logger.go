package mediasoup

import "github.com/sirupsen/logrus"

var logger logrus.FieldLogger = logrus.WithField("app", "mediasoup")

func Logger() logrus.FieldLogger {
	return logger
}

func SetLogger(l logrus.FieldLogger) {
	logger = l
}
