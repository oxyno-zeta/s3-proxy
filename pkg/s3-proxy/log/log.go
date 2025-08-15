package log

import (
	"github.com/sirupsen/logrus"
)

type Logger interface {
	Configure(level, format, filePath string) error

	WithField(key string, value any) Logger
	WithFields(fields map[string]any) Logger
	WithError(err error) Logger

	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Printf(format string, args ...any)
	Warnf(format string, args ...any)
	Warningf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Panicf(format string, args ...any)

	Debug(args ...any)
	Info(args ...any)
	Print(args ...any)
	Warn(args ...any)
	Warning(args ...any)
	Error(args ...any)
	Fatal(args ...any)
	Panic(args ...any)

	Debugln(args ...any)
	Infoln(args ...any)
	Println(args ...any)
	Warnln(args ...any)
	Warningln(args ...any)
	Errorln(args ...any)
	Fatalln(args ...any)
	Panicln(args ...any)

	GetTracingLogger() TracingLogger
	GetCorsLogger() CorsLogger
}

type TracingLogger interface {
	Error(msg string)
	Infof(msg string, args ...any)
	Debugf(msg string, args ...any)
}

type CorsLogger interface {
	Printf(msg string, args ...any)
}

func NewLogger() Logger {
	return &loggerIns{
		FieldLogger: logrus.New(),
	}
}
