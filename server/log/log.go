package log

import (
	"fmt"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
)

var rootLogger = logrus.New()

type Log struct {
	logger *logrus.Entry
}

// Fields wraps logrus.Fields, which is a map[string]interface{}
type Fields logrus.Fields

func NewLogger(fields Fields) *Log {
	return &Log{
		logger: rootLogger.WithFields(logrus.Fields(fields)),
	}
}

func SetLogLevel(level logrus.Level) {
	rootLogger.Level = level
}

func SetLogFormatter(formatter logrus.Formatter) {
	rootLogger.Formatter = formatter
}

// Print logs a message at level Print on the standard logger.
func (l *Log) Print(args ...interface{}) {
	entry := l.logger.WithFields(logrus.Fields{})
	entry.Data["file"] = fileInfo(2)
	entry.Print(args...)
}

// Print logs a message at level Print on the standard logger.
func (l *Log) Printf(format string, args ...interface{}) {
	entry := l.logger.WithFields(logrus.Fields{})
	entry.Data["file"] = fileInfo(2)
	entry.Printf(format, args...)
}

// Print logs a message at level Print on the standard logger.
func (l *Log) Println(args ...interface{}) {
	entry := l.logger.WithFields(logrus.Fields{})
	entry.Data["file"] = fileInfo(2)
	entry.Println(args...)
}

// Debug logs a message at level Debug on the standard logger.
func (l *Log) Debug(args ...interface{}) {
	if rootLogger.Level >= logrus.DebugLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Debug(args...)
	}
}

// Debug logs a message at level Debug on the standard logger.
func (l *Log) Debugf(format string, args ...interface{}) {
	if rootLogger.Level >= logrus.DebugLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Debugf(format, args...)
	}
}

// Info logs a message at level Info on the standard logger.
func (l *Log) Info(args ...interface{}) {
	if rootLogger.Level >= logrus.InfoLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Info(args...)
	}
}

// Info logs a message at level Info on the standard logger.
func (l *Log) Infof(format string, args ...interface{}) {
	if rootLogger.Level >= logrus.InfoLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Infof(format, args...)
	}
}

// Warn logs a message at level Warn on the standard logger.
func (l *Log) Warn(args ...interface{}) {
	if rootLogger.Level >= logrus.WarnLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Warn(args...)
	}
}

// Warnf logs a message at level Warn on the standard logger.
func (l *Log) Warnf(format string, args ...interface{}) {
	if rootLogger.Level >= logrus.WarnLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Warnf(format, args...)
	}
}

// Error logs a message at level Error on the standard logger.
func (l *Log) Error(args ...interface{}) {
	if rootLogger.Level >= logrus.ErrorLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Error(args...)
	}
}

// Error logs a message at level Error on the standard logger.
func (l *Log) Errorf(format string, args ...interface{}) {
	if rootLogger.Level >= logrus.ErrorLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Errorf(format, args...)
	}
}

// Fatal logs a message at level Fatal on the standard logger.
func (l *Log) Fatal(args ...interface{}) {
	if rootLogger.Level >= logrus.FatalLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Fatal(args...)
	}
}

// Fatal logs a message at level Fatal on the standard logger.
func (l *Log) Fatalf(format string, args ...interface{}) {
	if rootLogger.Level >= logrus.FatalLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Fatalf(format, args...)
	}
}

// Panic logs a message at level Panic on the standard logger.
func (l *Log) Panic(args ...interface{}) {
	if rootLogger.Level >= logrus.PanicLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Panic(args...)
	}
}

// Panic logs a message at level Panic on the standard logger.
func (l *Log) Panicf(format string, args ...interface{}) {
	if rootLogger.Level >= logrus.PanicLevel {
		entry := l.logger.WithFields(logrus.Fields{})
		entry.Data["file"] = fileInfo(2)
		entry.Panicf(format, args...)
	}
}

func fileInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip)
	if !ok {
		file = "<???>"
		line = 1
	} else {
		slash := strings.LastIndex(file, "/")
		if slash >= 0 {
			file = file[slash+1:]
		}
	}
	return fmt.Sprintf("%s:%d", file, line)
}
