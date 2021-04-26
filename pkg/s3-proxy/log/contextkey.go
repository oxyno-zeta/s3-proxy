package log

import "context"

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type contextKey struct {
	name string
}

var loggerContextKey = &contextKey{name: "LOGGER_CONTEXT_KEY"}

func GetLoggerFromContext(ctx context.Context) Logger {
	res, _ := ctx.Value(loggerContextKey).(Logger)

	return res
}

func SetLoggerInContext(ctx context.Context, logger Logger) context.Context {
	return context.WithValue(ctx, loggerContextKey, logger)
}
