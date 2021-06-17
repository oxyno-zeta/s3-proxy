package authorization

import (
	"errors"
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/authentication"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/middlewares"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/utils"
)

var errAuthorizationMiddlewareNotSupported = errors.New("authorization not supported")

func Middleware(cfg *config.Config, metricsCl metrics.Client) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logger := middlewares.GetLogEntry(r)

			// Get request resource from request
			resource := authentication.GetRequestResource(r)
			// Check if resource exists
			if resource == nil {
				// Resource doesn't exists
				// In this case, authentication is skipped, need to skip authorization too
				logger.Debug("no resource found in authorization, means that authentication was skipped => skip authorization too")
				next.ServeHTTP(w, r)

				return
			}

			// Check if resource is whitelisted
			if resource.WhiteList != nil && *resource.WhiteList {
				// Resource is whitelisted
				logger.Debug("authorization skipped because resource is whitelisted")
				next.ServeHTTP(w, r)

				return
			}

			// Get user from context
			user := authentication.GetAuthenticatedUser(r)

			// Check if resource is basic authentication
			if resource.Basic != nil {
				// Case user in basic auth user
				buser := user.(*models.BasicAuthUser)
				// Resource is basic authenticated
				logger.Debug("authorization for basic authentication => nothing needed")
				logger.Infof("Basic auth user %s authorized", buser.GetIdentifier())
				metricsCl.IncAuthorized("basic-auth")
				next.ServeHTTP(w, r)

				return
			}

			// Get request data
			requestURI := r.URL.RequestURI()
			// httpMethod := r.Method

			// Get bucket request context
			brctx := middlewares.GetBucketRequestContext(r)

			// Check if resource is OIDC
			if resource.OIDC != nil {
				// Cast user in oidc user
				ouser := user.(*models.OIDCUser)

				// Authorization part

				authorizationProvider := ""
				authorized := false
				// Check if case of opa server
				if resource.OIDC.AuthorizationOPAServer != nil {
					authorizationProvider = "oidc-opa"
					var err error
					authorized, err = isOPAServerAuthorized(r, ouser, resource)
					if err != nil {
						logger.Error(err)
						// Check if bucket request context doesn't exist to use local default files
						if brctx == nil {
							utils.HandleInternalServerError(logger, w, cfg.Templates, requestURI, err)
						} else {
							brctx.HandleInternalServerError(r.Context(), err, requestURI)
						}

						return
					}
				} else {
					authorizationProvider = "oidc-basic"
					authorized = isOIDCAuthorizedBasic(ouser.Groups, ouser.Email, resource.OIDC.AuthorizationAccesses)
				}

				// Check if not authorized
				if !authorized {
					logger.Errorf("Forbidden user %s", ouser.GetIdentifier())
					// Check if bucket request context doesn't exist to use local default files
					if brctx == nil {
						utils.HandleForbidden(logger, w, cfg.Templates, requestURI)
					} else {
						brctx.HandleForbidden(r.Context(), requestURI)
					}

					return
				}

				// User is authorized

				logger.Infof("OIDC user %s authorized", ouser.GetIdentifier())
				metricsCl.IncAuthorized(authorizationProvider)
				next.ServeHTTP(w, r)

				return
			}

			// Error, this case shouldn't arrive
			err := errAuthorizationMiddlewareNotSupported
			logger.Error(err)
			// Check if bucket request context doesn't exist to use local default files
			if brctx == nil {
				utils.HandleInternalServerError(logger, w, cfg.Templates, requestURI, err)
			} else {
				brctx.HandleInternalServerError(r.Context(), err, requestURI)
			}
		})
	}
}
