package log

import (
	"os"
	"path/filepath"

	logrus "github.com/sirupsen/logrus"
)

type loggerIns struct {
	logrus.FieldLogger
}

func (ll *loggerIns) GetTracingLogger() TracingLogger {
	return &tracingLogger{
		logger: ll,
	}
}

func (ll *loggerIns) Configure(level string, format string, filePath string) error {
	// Parse log level
	lvl, err := logrus.ParseLevel(level)
	if err != nil {
		return err
	}

	// Get logrus logger
	lll := ll.FieldLogger.(*logrus.Logger)

	// Set log level
	lll.SetLevel(lvl)

	// Set format
	if format == "json" {
		lll.SetFormatter(&logrus.JSONFormatter{})
	} else {
		lll.SetFormatter(&logrus.TextFormatter{})
	}

	if filePath != "" {
		// Create directory if necessary
		err2 := os.MkdirAll(filepath.Dir(filePath), os.ModePerm)
		if err2 != nil {
			return err2
		}

		// Open file
		f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return err
		}

		// Set output file
		lll.SetOutput(f)
	}

	return nil
}

func (ll *loggerIns) WithField(key string, value interface{}) Logger {
	// Create new field logger
	fieldL := ll.FieldLogger.WithField(key, value)

	return &loggerIns{
		FieldLogger: fieldL,
	}
}

func (ll *loggerIns) WithFields(fields map[string]interface{}) Logger {
	// Transform fields
	var ff logrus.Fields = fields
	// Create new field logger
	fieldL := ll.FieldLogger.WithFields(ff)

	return &loggerIns{
		FieldLogger: fieldL,
	}
}

func (ll *loggerIns) WithError(err error) Logger {
	// Create new field logger
	fieldL := ll.FieldLogger.WithError(err)

	return &loggerIns{
		FieldLogger: fieldL.Logger,
	}
}
