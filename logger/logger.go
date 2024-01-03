package logger

import "github.com/sirupsen/logrus"

func SetLogger(logLevel logrus.Level) {
	logrus.SetFormatter(&logrus.TextFormatter{
		DisableTimestamp: true,
	})
	logrus.SetLevel(logLevel)
}
