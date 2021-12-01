package authentication

import (
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gobwas/glob"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/bucket"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
	"github.com/thoas/go-funk"
)

var errAuthenticationMiddlewareNotSupported = errors.New("authentication not supported")

type service struct {
	allVerifiers map[string]*oidc.IDTokenVerifier
	cfg          *config.Config
	metricsCl    metrics.Client
	// This has been saved only for response handler.
	// Not used inside the service functions because it is better to use fixed configuration to avoid conflict in case of reload and incoming request.
	cfgManager config.Manager
}

// Middleware will redirect authentication to basic auth or OIDC depending on request path and resources declared.
func (s *service) Middleware(resources []*config.Resource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get logger
			logEntry := log.GetLoggerFromContext(r.Context())
			// Get response handler
			resHan := responsehandler.GetResponseHandlerFromContext(r.Context())

			// Check if resources are empty
			if len(resources) == 0 {
				// In this case, continue without authentication
				logEntry.Info("no resource declared => skip authentication")
				next.ServeHTTP(w, r)

				return
			}

			// Get request data
			requestURI := r.URL.RequestURI()
			httpMethod := r.Method

			// Get bucket request context
			brctx := bucket.GetBucketRequestContextFromContext(r.Context())
			// Find resource
			res, err := findResource(resources, requestURI, httpMethod)
			if err != nil {
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					responsehandler.GeneralInternalServerError(r, w, s.cfgManager, err)
				} else {
					resHan.InternalServerError(brctx.LoadFileContent, err)
				}

				return
			}

			// Check if resource isn't found
			if res == nil {
				// In this case, resource isn't found because not path not declared
				// So access is forbidden
				err2 := fmt.Errorf("no resource found for path %s and method %s => Forbidden access", requestURI, httpMethod)
				// Add stack trace
				err2 = errors.WithStack(err2)
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					responsehandler.GeneralForbiddenError(r, w, s.cfgManager, err2)
				} else {
					resHan.ForbiddenError(brctx.LoadFileContent, err2)
				}

				return
			}

			// Resource found case

			// Add resource to request context in order to keep it ready for authorization
			ctx := models.SetRequestResourceInContext(r.Context(), res)
			// Create new request with new context
			r = r.WithContext(ctx)

			// Check if OIDC is enabled
			if res.OIDC != nil {
				logEntry.Debug("authentication with oidc detected")
				s.oidcAuthorizationMiddleware(res)(next).ServeHTTP(w, r)

				return
			}

			// Check if Basic auth is enabled
			if res.Basic != nil {
				logEntry.Debug("authentication with basic auth detected")
				s.basicAuthMiddleware(res)(next).ServeHTTP(w, r)

				return
			}

			// Last case must be whitelist
			if *res.WhiteList {
				logEntry.Debug("authentication skipped because resource is whitelisted")
				next.ServeHTTP(w, r)

				return
			}

			// Error, this case shouldn't arrive
			err = errAuthenticationMiddlewareNotSupported
			// Check if bucket request context doesn't exist to use local default files
			if brctx == nil {
				responsehandler.GeneralInternalServerError(r, w, s.cfgManager, err)
			} else {
				resHan.InternalServerError(brctx.LoadFileContent, err)
			}
		})
	}
}

func findResource(resL []*config.Resource, requestURI string, httpMethod string) (*config.Resource, error) {
	// Loop over the list
	for _, res := range resL {
		// Check if http method is declared in resource
		if !funk.Contains(res.Methods, httpMethod) {
			// Http method not declared in resource
			// Stop here and continue to next resource
			continue
		}
		// Compile a glob pattern for uri matching
		g, err := glob.Compile(res.Path)
		// Check if error exists
		if err != nil {
			return nil, errors.WithStack(err)
		}
		// Check if request uri match glob pattern declared in resource
		if g.Match(requestURI) {
			return res, nil
		}
	}

	// Not found case
	return nil, nil // nolint: nilnil // No need for a sentinel error in this case
}
