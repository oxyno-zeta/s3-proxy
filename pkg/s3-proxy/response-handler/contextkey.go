package responsehandler

import "context"

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type contextKey struct {
	name string
}

var responseHandlerCtxKey = &contextKey{name: "ResponseHandlerCtxKey"}

// GetResponseHandlerFromContext will return the response handler object from context.
func GetResponseHandlerFromContext(ctx context.Context) ResponseHandler {
	//nolint: forcetypeassert // Ignore this
	return ctx.Value(responseHandlerCtxKey).(ResponseHandler)
}

// SetResponseHandlerInContext will set a response handler object in a context.
func SetResponseHandlerInContext(ctx context.Context, resH ResponseHandler) context.Context {
	return context.WithValue(ctx, responseHandlerCtxKey, resH)
}
