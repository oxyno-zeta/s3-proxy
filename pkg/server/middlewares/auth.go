package middlewares

import (
	"errors"
	"net/http"

	"github.com/gobwas/glob"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/server/utils"
	"github.com/thoas/go-funk"
)

var errAuthMiddlewareNotSupported = errors.New("not supported")

// AuthMiddleware will redirect authentication to basic auth or OIDC depending on request path and resources declared
func AuthMiddleware(cfg *config.Config, resources []*config.Resource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logEntry := GetLogEntry(r)
			requestURI := r.URL.RequestURI()
			httpMethod := r.Method
			logEntry.Info(requestURI)
			// Find resource
			res, err := findResource(resources, requestURI, httpMethod)
			if err != nil {
				logEntry.Error(err)
				utils.HandleInternalServerError(w, err, requestURI, logEntry, cfg.Templates)
			}

			// Check if resource isn't found
			if res == nil {
				// Check if resources are empty
				if len(resources) == 0 {
					// In this case, continue without authentication
					next.ServeHTTP(w, r)
					return
				}
				// In this case, resource isn't found because not path not declared
				// So access is forbidden
				logEntry.Errorf("no resource found for path %s => Forbidden access", requestURI)
				utils.HandleForbidden(w, requestURI, logEntry, cfg.Templates)
				return
			}

			// Resource found case

			// Check if OIDC is enabled
			if res.OIDC != nil {
				oidcAuthorizationMiddleware(cfg.AuthProviders.OIDC[res.Provider], cfg.Templates, res.OIDC.AuthorizationAccesses)(next).ServeHTTP(w, r)
				return
			}

			// Check if Basic auth is enabled
			if res.Basic != nil {
				basicAuthMiddleware(cfg.AuthProviders.Basic[res.Provider], res.Basic.Credentials, cfg.Templates)(next).ServeHTTP(w, r)
				return
			}

			// Last case must be whitelist
			if *res.WhiteList {
				next.ServeHTTP(w, r)
				return
			}

			// Error, this case shouldn't arrive
			err = errAuthMiddlewareNotSupported
			logEntry.Error(err)
			utils.HandleInternalServerError(w, err, requestURI, logEntry, cfg.Templates)
		})
	}
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
