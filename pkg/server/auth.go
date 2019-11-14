package server

import (
	"errors"
	"net/http"

	"github.com/gobwas/glob"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
)

var errAuthMiddlewareNotSupported = errors.New("not supported")

func authMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logEntry := GetLogEntry(r)
			requestURI := r.URL.RequestURI()
			// Find resource
			res, err := findResource(cfg.Resources, requestURI)
			if err != nil {
				logEntry.Error(err)
				handleInternalServerError(w, err, requestURI, &logEntry, cfg.Templates)
			}

			// Check if resource isn't found or resource exists without any auth or whitelist
			if res == nil || (res != nil && res.OIDC == nil && res.Basic == nil && *res.WhiteList == false) {
				// No resource matching
				// Check if there is a default auth case

				// Check OIDC default auth case
				if cfg.Auth != nil && cfg.Auth.OIDC != nil {
					// OIDC default case detected
					oidcAuthorizationMiddleware(cfg.Auth.OIDC, cfg.Templates, cfg.Auth.OIDC.AuthorizationAccesses)(next).ServeHTTP(w, r)
					return
				}

				// Check basic auth default case
				if cfg.Auth != nil && cfg.Auth.Basic != nil {
					// Basic auth case detected
					basicAuthMiddleware(cfg.Auth.Basic, cfg.Templates)(next).ServeHTTP(w, r)
					return
				}

				// No default case detected => next
				next.ServeHTTP(w, r)
				return
			}

			// Resource found case

			// Check if OIDC is enabled
			if res.OIDC != nil {
				oidcAuthorizationMiddleware(cfg.Auth.OIDC, cfg.Templates, res.OIDC.AuthorizationAccesses)(next).ServeHTTP(w, r)
				return
			}

			// Check if Basic auth is enabled
			if res.Basic != nil {
				basicAuthMiddleware(res.Basic, cfg.Templates)(next).ServeHTTP(w, r)
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
			handleInternalServerError(w, err, requestURI, &logEntry, cfg.Templates)
		})
	}
}

func findResource(resL []*config.Resource, requestURI string) (*config.Resource, error) {
	for i := 0; i < len(resL); i++ {
		res := resL[i]
		g, err := glob.Compile(res.Path)
		if err != nil {
			return nil, err
		}
		if g.Match(requestURI) {
			return res, nil
		}
	}
	return nil, nil
}
