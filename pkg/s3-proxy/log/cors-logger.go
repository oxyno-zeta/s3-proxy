package log

type corsLogger struct {
	logger Logger
}

func (c *corsLogger) Printf(format string, args ...interface{}) {
	c.logger.Debugf(format, args...)
}
