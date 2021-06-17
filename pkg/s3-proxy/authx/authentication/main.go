package authentication

import (
	"errors"
	"net/http"

	oidc "github.com/coreos/go-oidc/v3/oidc"
	"github.com/gobwas/glob"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/middlewares"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/server/utils"
	"github.com/thoas/go-funk"
	"golang.org/x/net/context"
)

var userContextKey = &contextKey{name: "USER_CONTEXT_KEY"}
var resourceContextKey = &contextKey{name: "RESOURCE_CONTEXT_KEY"}
var errAuthenticationMiddlewareNotSupported = errors.New("authentication not supported")

type service struct {
	allVerifiers map[string]*oidc.IDTokenVerifier
	cfg          *config.Config
	metricsCl    metrics.Client
}

// Middleware will redirect authentication to basic auth or OIDC depending on request path and resources declared.
func (s *service) Middleware(resources []*config.Resource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logEntry := middlewares.GetLogEntry(r)

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
			brctx := middlewares.GetBucketRequestContext(r)
			// Find resource
			res, err := findResource(resources, requestURI, httpMethod)
			if err != nil {
				logEntry.Error(err)
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					utils.HandleInternalServerError(logEntry, w, s.cfg.Templates, requestURI, err)
				} else {
					brctx.HandleInternalServerError(r.Context(), err, requestURI)
				}

				return
			}

			// Check if resource isn't found
			if res == nil {
				// In this case, resource isn't found because not path not declared
				// So access is forbidden
				logEntry.Errorf("no resource found for path %s and method %s => Forbidden access", requestURI, httpMethod)
				// Check if bucket request context doesn't exist to use local default files
				if brctx == nil {
					utils.HandleForbidden(logEntry, w, s.cfg.Templates, requestURI)
				} else {
					brctx.HandleForbidden(r.Context(), requestURI)
				}

				return
			}

			// Resource found case

			// Add resource to request context in order to keep it ready for authorization
			ctx := context.WithValue(r.Context(), resourceContextKey, res)
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
			logEntry.Error(err)
			// Check if bucket request context doesn't exist to use local default files
			if brctx == nil {
				utils.HandleInternalServerError(logEntry, w, s.cfg.Templates, requestURI, err)
			} else {
				brctx.HandleInternalServerError(r.Context(), err, requestURI)
			}
		})
	}
}

// GetAuthenticatedUser will get authenticated user in context.
func GetAuthenticatedUser(req *http.Request) models.GenericUser {
	res, _ := req.Context().Value(userContextKey).(models.GenericUser)

	return res
}

// GetRequestResource will get request resource in context.
func GetRequestResource(req *http.Request) *config.Resource {
	res, _ := req.Context().Value(resourceContextKey).(*config.Resource)

	return res
}

func findResource(resL []*config.Resource, requestURI string, httpMethod string) (*config.Resource, error) {
	for i := 0; i < len(resL); i++ {
		res := resL[i]
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
			return nil, err
		}
		// Check if request uri match glob pattern declared in resource
		if g.Match(requestURI) {
			return res, nil
		}
	}
	// Not found case
	return nil, nil
}
