package server

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

// getObjectRequestPath decodes the routed object path exactly once, matching
// URL.Path used by authentication and authorization. Chi routes on RawPath when
// present, otherwise on the already decoded Path; only the former needs decoding.
func getObjectRequestPath(req *http.Request) (string, error) {
	requestPath := chi.URLParam(req, "*")
	if req.URL.RawPath != "" {
		// Chi routes on RawPath when present, so the wildcard parameter is
		// still encoded (e.g. "%70rivate/file.txt"). Decode it to match
		// URL.Path ("/private/file.txt"), which authentication checks.
		// Otherwise Chi uses the already decoded URL.Path; unescaping that
		// parameter again could turn a literal "%70" in an object name into "p".
		return url.PathUnescape(requestPath)
	}

	return requestPath, nil
}
