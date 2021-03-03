package middlewares

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

// CacheManagement is a middleware to manage cache header output.
func CacheManagement(cfg *config.CacheConfig) func(http.Handler) http.Handler {
	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
			// Check if expires header is set
			if cfg.Expires != "" {
				rw.Header().Set("Expires", cfg.Expires)
			}
			// Check if cache control is set
			if cfg.CacheControl != "" {
				rw.Header().Set("Cache-Control", cfg.CacheControl)
			}
			// Check if pragma is set
			if cfg.Pragma != "" {
				rw.Header().Set("Pragma", cfg.Pragma)
			}
			// Check if x-accel-expires
			if cfg.XAccelExpires != "" {
				rw.Header().Set("X-Accel-Expires", cfg.XAccelExpires)
			}

			// Next
			h.ServeHTTP(rw, r)
		})
	}
}
