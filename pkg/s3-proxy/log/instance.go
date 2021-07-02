package log

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	logrus "github.com/sirupsen/logrus"
)

type loggerIns struct {
	logrus.FieldLogger
}

// This is dirty pkg/errors.
type stackTracer interface {
	StackTrace() errors.StackTrace
}

func (ll *loggerIns) GetTracingLogger() TracingLogger {
	return &tracingLogger{
		logger: ll,
	}
}

func (ll *loggerIns) GetCorsLogger() CorsLogger {
	return &corsLogger{
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
			return errors.WithStack(err2)
		}

		// Open file
		f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			return errors.WithStack(err)
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

	addStackTrace := func(pError stackTracer) {
		// Get stack trace from error
		st := pError.StackTrace()
		// Stringify stack trace
		valued := fmt.Sprintf("%+v", st)
		// Remove all tabs
		valued = strings.ReplaceAll(valued, "\t", "")
		// Split on new line
		stack := strings.Split(valued, "\n")
		// Remove first empty string
		stack = stack[1:]
		// Add stack trace to field logger
		fieldL = fieldL.WithField("stack", strings.Join(stack, ","))
	}

	// Check if error is matching stack trace interface
	// nolint: errorlint // Ignore this because the aim is to catch stack trace error at first level
	if err2, ok := err.(stackTracer); ok {
		addStackTrace(err2)
	}

	// Check if error cause is matching stack trace interface
	// nolint: errorlint // Ignore this because the aim is to catch stack trace error at first level
	if err2, ok := errors.Cause(err).(stackTracer); ok {
		addStackTrace(err2)
	}

	return &loggerIns{
		FieldLogger: fieldL,
	}
}

func (ll *loggerIns) addPotentialWithError(elem interface{}) {
	// Try to cast element to error
	err, ok := elem.(error)
	// Check if can be casted to error
	if ok {
		// Call with error
		res := ll.WithError(err)

		// Change internal field logger
		ll.FieldLogger = res.(*loggerIns).FieldLogger
	}
}

func (ll *loggerIns) Error(args ...interface{}) {
	// Add potential "WithError"
	ll.addPotentialWithError(args[0])

	// Call logger error method
	ll.FieldLogger.Error(args...)
}

func (ll *loggerIns) Fatal(args ...interface{}) {
	// Add potential "WithError"
	ll.addPotentialWithError(args[0])

	// Call logger fatal method
	ll.FieldLogger.Fatal(args...)
}

func (ll *loggerIns) Errorf(format string, args ...interface{}) {
	// Create error
	err := fmt.Errorf(format, args...)

	// Log error
	ll.Error(err)
}

func (ll *loggerIns) Fatalf(format string, args ...interface{}) {
	// Create error
	err := fmt.Errorf(format, args...)

	// Log fatal
	ll.Fatal(err)
}

func (ll *loggerIns) Errorln(args ...interface{}) {
	// Add potential "WithError"
	ll.addPotentialWithError(args[0])

	// Log error
	ll.FieldLogger.Errorln(args...)
}

func (ll *loggerIns) Fatalln(args ...interface{}) {
	// Add potential "WithError"
	ll.addPotentialWithError(args[0])

	// Log fatal
	ll.FieldLogger.Fatalln(args...)
}
