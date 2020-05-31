package authentication

import (
	"fmt"
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/middlewares"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/utils"
	"github.com/thoas/go-funk"
	"golang.org/x/net/context"
)

var bucketRequestContextKey = &contextKey{name: "bucket-request-context"}

// nolint:whitespace
func basicAuthMiddleware(basicConfig *config.BasicAuthConfig,
	basicAuthUserConfigList []*config.BasicAuthUserConfig, templateConfig *config.TemplateConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logEntry := middlewares.GetLogEntry(r)
			path := r.URL.RequestURI()
			// Get bucket request context from request
			brctx := middlewares.GetBucketRequestContext(r)

			// Get basic auth information
			username, password, ok := r.BasicAuth()
			if !ok {
				logEntry.Error("No basic auth detected in request")
				w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, basicConfig.Realm))
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					utils.HandleUnauthorized(logEntry, w, templateConfig, path)
				} else {
					brctx.HandleUnauthorized(path)
				}
				return
			}

			// Find user credentials
			cred := funk.Find(basicAuthUserConfigList, func(cred *config.BasicAuthUserConfig) bool {
				return cred.User == username
			})

			if cred == nil {
				logEntry.Errorf("Username %s not found in authorized users", username)
				w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, basicConfig.Realm))
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					utils.HandleUnauthorized(logEntry, w, templateConfig, path)
				} else {
					brctx.HandleUnauthorized(path)
				}
				return
			}

			// Check password
			if cred.(*config.BasicAuthUserConfig).Password.Value == "" || cred.(*config.BasicAuthUserConfig).Password.Value != password {
				logEntry.Errorf("Username %s not authorized", username)
				w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, basicConfig.Realm))
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					utils.HandleUnauthorized(logEntry, w, templateConfig, path)
				} else {
					brctx.HandleUnauthorized(path)
				}
				return
			}

			// Create Basic auth user
			buser := &models.BasicAuthUser{Username: username}

			// Add user to request context by creating a new context
			ctx := context.WithValue(r.Context(), userContextKey, buser)
			// Create new request with new context
			r = r.WithContext(ctx)

			logEntry.Info("Basic auth user %s authenticated", username)

			next.ServeHTTP(w, r)
		})
	}
}
