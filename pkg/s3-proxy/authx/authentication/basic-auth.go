package authentication

import (
	"fmt"
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
	"github.com/pkg/errors"
	"github.com/thoas/go-funk"
)

func (s *service) basicAuthMiddleware(res *config.Resource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get data
			basicConfig := s.cfg.AuthProviders.Basic[res.Provider]
			basicAuthUserConfigList := res.Basic.Credentials
			// Get logger from request
			logEntry := log.GetLoggerFromContext(r.Context())
			// Get bucket request context from request
			brctx := bucket.GetBucketRequestContextFromContext(r.Context())
			// Get response handler
			resHan := responsehandler.GetResponseHandlerFromContext(r.Context())

			// Get basic auth information
			username, password, ok := r.BasicAuth()
			if !ok {
				// Create error
				err := errors.New("no basic auth detected in request")
				// Add header for basic auth realm
				w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, basicConfig.Realm))
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					responsehandler.GeneralUnauthorizedError(r, w, s.cfgManager, err)
				} else {
					resHan.UnauthorizedError(brctx.LoadFileContent, err)
				}

				return
			}

			// Create Basic auth user
			buser := &models.BasicAuthUser{Username: username}

			// Add user to request context by creating a new context
			ctx := models.SetAuthenticatedUserInContext(r.Context(), buser)
			// Create new request with new context
			r = r.WithContext(ctx)

			// Update response handler to have the latest context values
			resHan.UpdateRequestAndResponse(r, w)

			// Find user credentials
			cred := funk.Find(basicAuthUserConfigList, func(cred *config.BasicAuthUserConfig) bool {
				return cred.User == username
			})

			// Check if credential exists
			if cred == nil {
				// Create error
				err := fmt.Errorf("username %s not found in authorized users", username)
				// Add stack trace
				err = errors.WithStack(err)
				// Add header for basic auth realm
				w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, basicConfig.Realm))
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					responsehandler.GeneralUnauthorizedError(r, w, s.cfgManager, err)
				} else {
					resHan.UnauthorizedError(brctx.LoadFileContent, err)
				}

				return
			}

			// Check password
			if cred.(*config.BasicAuthUserConfig).Password.Value == "" || cred.(*config.BasicAuthUserConfig).Password.Value != password {
				// Create error
				err := fmt.Errorf("username %s not authorized", username)
				// Add stack trace
				err = errors.WithStack(err)
				// Add header for basic auth realm
				w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, basicConfig.Realm))
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					responsehandler.GeneralUnauthorizedError(r, w, s.cfgManager, err)
				} else {
					resHan.UnauthorizedError(brctx.LoadFileContent, err)
				}

				return
			}

			logEntry.Info("Basic auth user %s authenticated", buser.GetIdentifier())
			s.metricsCl.IncAuthenticated("basic-auth", res.Provider)

			next.ServeHTTP(w, r)
		})
	}
}
