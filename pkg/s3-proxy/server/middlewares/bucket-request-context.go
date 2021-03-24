package middlewares

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/utils"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
	"golang.org/x/net/context"
)

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type contextKey struct {
	name string
}

var bucketRequestContextKey = &contextKey{name: "bucket-request-context"}

func BucketRequestContext(
	tgt *config.TargetConfig,
	tplConfig *config.TemplateConfig,
	path string,
	metricsCli metrics.Client,
	s3clientManager s3client.Manager,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Get logger
			logEntry := log.GetLoggerFromContext(req.Context())
			// Get request URI
			errorhandlers := &bucket.ErrorHandlers{
				HandleForbiddenWithTemplate:           utils.HandleForbiddenWithTemplate,
				HandleNotFoundWithTemplate:            utils.HandleNotFoundWithTemplate,
				HandleInternalServerErrorWithTemplate: utils.HandleInternalServerErrorWithTemplate,
				HandleBadRequestWithTemplate:          utils.HandleBadRequestWithTemplate,
				HandleUnauthorizedWithTemplate:        utils.HandleUnauthorizedWithTemplate,
			}
			// Get request trace
			trace := tracing.GetTraceFromContext(req.Context())
			// Generate new bucket client
			brctx := bucket.NewClient(tgt, tplConfig, logEntry, path, rw, req, metricsCli, errorhandlers, trace, s3clientManager)
			// Add bucket structure to request context by creating a new context
			ctx := context.WithValue(req.Context(), bucketRequestContextKey, brctx)
			// Create new request with new context
			req = req.WithContext(ctx)
			// Next
			next.ServeHTTP(rw, req)
		})
	}
}

func GetBucketRequestContext(req *http.Request) bucket.Client {
	res, _ := req.Context().Value(bucketRequestContextKey).(bucket.Client)

	return res
}
