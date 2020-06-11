package middlewares

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/utils"
	"golang.org/x/net/context"
)

// contextKey is a value for use with context.WithValue. It's used as
// a pointer so it fits in an interface{} without allocation. This technique
// for defining context keys was copied from Go 1.7's new use of context in net/http.
type contextKey struct {
	name string
}

var bucketRequestContextKey = &contextKey{name: "bucket-request-context"}

// nolint:whitespace
func BucketRequestContext(
	tgt *config.TargetConfig, tplConfig *config.TemplateConfig,
	path string, metricsCli metrics.Client,
) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			// Get logger
			logEntry := GetLogEntry(req)
			// Get request URI
			requestURI := req.URL.RequestURI()
			errorhandlers := &bucket.ErrorHandlers{
				HandleForbiddenWithTemplate:           utils.HandleForbiddenWithTemplate,
				HandleNotFoundWithTemplate:            utils.HandleNotFoundWithTemplate,
				HandleInternalServerErrorWithTemplate: utils.HandleInternalServerErrorWithTemplate,
				HandleBadRequestWithTemplate:          utils.HandleBadRequestWithTemplate,
				HandleUnauthorizedWithTemplate:        utils.HandleUnauthorizedWithTemplate,
			}
			// Generate new bucket client
			brctx, err := bucket.NewClient(tgt, tplConfig, logEntry, path, rw, metricsCli, errorhandlers)
			if err != nil {
				logEntry.Error(err)
				utils.HandleInternalServerError(logEntry, rw, tplConfig, requestURI, err)
				// Stop
				return
			}
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
