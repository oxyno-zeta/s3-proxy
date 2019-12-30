package middlewares

import (
	"errors"
	"net/http"

	"github.com/gobwas/glob"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/server/utils"
)

var errAuthMiddlewareNotSupported = errors.New("not supported")

func AuthMiddleware(cfg *config.Config, resources []*config.Resource) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logEntry := GetLogEntry(r)
			requestURI := r.URL.RequestURI()
			logEntry.Info(requestURI)
			// Find resource
			res, err := findResource(resources, requestURI)
			if err != nil {
				logEntry.Error(err)
				utils.HandleInternalServerError(w, err, requestURI, &logEntry, cfg.Templates)
			}

			// Check if resource isn't found
			if res == nil {
				// No resource matching
				next.ServeHTTP(w, r)
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
			utils.HandleInternalServerError(w, err, requestURI, &logEntry, cfg.Templates)
		})
	}
}

func findResource(resL []*config.Resource, requestURI string) (*config.Resource, error) {
	for i := 0; i < len(resL); i++ {
		res := resL[i]
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
