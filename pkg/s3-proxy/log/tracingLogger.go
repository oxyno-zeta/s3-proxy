package log

type tracingLogger struct {
	logger Logger
}

func (tl *tracingLogger) Error(msg string) {
	tl.logger.Errorf(msg)
}

func (tl *tracingLogger) Infof(msg string, args ...interface{}) {
	tl.logger.Infof(msg, args...)
}

// Debugf logs a message at debug priority.
func (tl *tracingLogger) Debugf(msg string, args ...interface{}) {
	tl.logger.Debugf(msg, args...)
}
