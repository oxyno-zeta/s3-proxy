package middlewares

import (
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/utils"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
)

func ImproveTracing() func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Get request id
			reqID := middleware.GetReqID(req.Context())
			// Get trace trace from request
			trace := tracing.GetTraceFromRequest(req)
			// Add request id to trace
			trace.SetTag("http.request_id", reqID)

			// Add request host
			trace.SetTag("http.request_host", utils.RequestHost(req))

			// Add request path
			trace.SetTag("http.request_path", req.URL.Path)

			// Next
			next.ServeHTTP(rw, req)
		})
	}
}
