package responsehandler

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

// HTTPMiddleware will add a new response handler on each request.
func HTTPMiddleware(cfgManager config.Manager, targetKey string) func(next http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Get request context
			ctx := r.Context()

			// Create response handler object
			rh := NewHandler(r, rw, cfgManager, targetKey)

			// Inject in context
			ctx = SetResponseHandlerInContext(ctx, rh)
			// Inject in request
			r = r.WithContext(ctx)

			// Next
			h.ServeHTTP(rw, r)
		})
	}
}
