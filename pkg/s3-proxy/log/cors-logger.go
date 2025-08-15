package log

type corsLogger struct {
	logger Logger
}

func (c *corsLogger) Printf(format string, args ...any) {
	c.logger.Debugf(format, args...)
}
