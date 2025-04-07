package models

import (
	"context"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type contextKey struct {
	name string
}

var (
	userContextKey     = &contextKey{name: "USER_CONTEXT_KEY"}
	resourceContextKey = &contextKey{name: "RESOURCE_CONTEXT_KEY"}
)

func SetAuthenticatedUserInContext(ctx context.Context, user GenericUser) context.Context {
	// Add value to context
	return context.WithValue(ctx, userContextKey, user)
}

// GetAuthenticatedUserFromContext will get authenticated user in context.
func GetAuthenticatedUserFromContext(ctx context.Context) GenericUser {
	res, _ := ctx.Value(userContextKey).(GenericUser)

	return res
}

func SetRequestResourceInContext(ctx context.Context, res *config.Resource) context.Context {
	return context.WithValue(ctx, resourceContextKey, res)
}

// GetRequestResourceFromContext will get request resource in context.
func GetRequestResourceFromContext(ctx context.Context) *config.Resource {
	res, _ := ctx.Value(resourceContextKey).(*config.Resource)

	return res
}
