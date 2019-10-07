package server

import (
	"fmt"
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/thoas/go-funk"
)

func basicAuthMiddleware(basicConfig *config.BasicAuthConfig, templateConfig *config.TemplateConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			logEntry := GetLogEntry(r)
			path := r.URL.RequestURI()
			username, password, ok := r.BasicAuth()
			if !ok {
				w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, basicConfig.Realm))
				handleUnauthorized(w, path, &logEntry, templateConfig)
				return
			}

			// Find user credentials
			cred := funk.Find(basicConfig.Credentials, func(cred *config.BasicAuthUserConfig) bool {
				return cred.User == username
			})

			if cred == nil {
				w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, basicConfig.Realm))
				handleUnauthorized(w, path, &logEntry, templateConfig)
				return
			}

			// Check password
			if cred.(*config.BasicAuthUserConfig).Password.Value == "" || cred.(*config.BasicUserConfig).Password.Value != password {
				w.Header().Add("WWW-Authenticate", fmt.Sprintf(`Basic realm="%s"`, basicConfig.Realm))
				handleUnauthorized(w, path, &logEntry, templateConfig)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
