package authorization

import (
	"fmt"
	"net/http"

	"emperror.dev/errors"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
)

var errAuthorizationMiddlewareNotSupported = errors.New("authorization not supported")

func Middleware(
	// This has been saved only for response handler.
	// Not used inside the service functions because it is better to use fixed configuration to avoid conflict in case of reload and incoming request.
	cfgManager config.Manager,
	metricsCl metrics.Client,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get logger from request
			logger := log.GetLoggerFromContext(r.Context())

			// Get request resource from request
			resource := models.GetRequestResourceFromContext(r.Context())
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
			user := models.GetAuthenticatedUserFromContext(r.Context())

			// Check if resource is basic authentication
			if resource.Basic != nil {
				// Case user in basic auth user
				buser, _ := user.(*models.BasicAuthUser)
				// Resource is basic authenticated
				logger.Debug("authorization for basic authentication => nothing needed")
				logger.Infof("Basic auth user %s authorized", buser.GetIdentifier())
				metricsCl.IncAuthorized("basic-auth")
				next.ServeHTTP(w, r)

				return
			}

			// Get bucket request context
			brctx := bucket.GetBucketRequestContextFromContext(r.Context())
			// Get response handler
			resHan := responsehandler.GetResponseHandlerFromContext(r.Context())

			// Check if resource is OIDC or Header
			if resource.OIDC != nil || resource.Header != nil {
				// Initialize variables
				var authorizationProvider string
				// Initialize variables
				var headerOIDCResource *config.ResourceHeaderOIDC

				// Check if resource is OIDC
				if resource.OIDC != nil {
					// OIDC case
					headerOIDCResource = resource.OIDC

					if headerOIDCResource.AuthorizationOPAServer != nil {
						authorizationProvider = "oidc-opa"
					} else {
						authorizationProvider = "oidc-basic"
					}
				} else {
					// Header case
					headerOIDCResource = resource.Header

					if headerOIDCResource.AuthorizationOPAServer != nil {
						authorizationProvider = "header-opa"
					} else {
						authorizationProvider = "header-basic"
					}
				}

				// Authorization part

				var authorized bool
				// Check if case of opa server
				if headerOIDCResource.AuthorizationOPAServer != nil {
					var err error
					authorized, err = isOPAServerAuthorized(r, user, headerOIDCResource)
					// Check error
					if err != nil {
						// Check if bucket request context doesn't exist to use local default files
						if brctx == nil {
							responsehandler.GeneralInternalServerError(r, w, cfgManager, err)
						} else {
							resHan.InternalServerError(
								brctx.LoadFileContent,
								err,
							)
						}

						return
					}
				} else {
					authorized = isHeaderOIDCAuthorizedBasic(
						user.GetGroups(),
						user.GetEmail(),
						headerOIDCResource.AuthorizationAccesses,
					)
				}

				// Check if not authorized
				if !authorized {
					// Create error
					err := fmt.Errorf("forbidden user %s", user.GetIdentifier())
					// Add stack trace
					err = errors.WithStack(err)
					// Check if bucket request context doesn't exist to use local default files
					if brctx == nil {
						responsehandler.GeneralForbiddenError(r, w, cfgManager, err)
					} else {
						resHan.ForbiddenError(brctx.LoadFileContent, err)
					}

					return
				}

				// User is authorized

				logger.Infof("%s user %s authorized", user.GetType(), user.GetIdentifier())
				metricsCl.IncAuthorized(authorizationProvider)
				next.ServeHTTP(w, r)

				return
			}

			// Error, this case shouldn't arrive
			err := errAuthorizationMiddlewareNotSupported
			// Check if bucket request context doesn't exist to use local default files
			if brctx == nil {
				responsehandler.GeneralInternalServerError(r, w, cfgManager, err)
			} else {
				resHan.InternalServerError(brctx.LoadFileContent, err)
			}
		})
	}
}
