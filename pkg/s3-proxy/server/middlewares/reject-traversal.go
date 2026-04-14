package middlewares

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"emperror.dev/errors"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	responsehandler "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler"
)

// RejectTraversal returns a middleware that rejects requests whose URL path
// contains dot-segment traversals before routing and auth run. It fully decodes
// the escaped path (including %2F and %2E) and compares the non-empty segment
// count before and after path.Clean. A change in count means a . or .. segment
// was present — either literal or encoded — and the request is rejected with
// 400 Bad Request.
//
// Consecutive slashes (//) do not change the non-empty segment count and are
// allowed through (the router normalises them).
func RejectTraversal(cfgManager config.Manager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			hasDot, err := isPathTraversalAttempt(r.URL.EscapedPath())
			if err != nil {
				responsehandler.GeneralBadRequestError(r, w, cfgManager, errors.WithStack(err))

				return
			}

			if hasDot {
				responsehandler.GeneralBadRequestError(
					r,
					w,
					cfgManager,
					errors.New("path traversal detected"),
				)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func isPathTraversalAttempt(escapedPath string) (bool, error) {
	decoded, err := url.PathUnescape(escapedPath)
	if err != nil {
		return false, err
	}

	return countSegments(decoded) != countSegments(path.Clean(decoded)), nil
}

func countSegments(rawPath string) int {
	count := 0

	for segment := range strings.SplitSeq(rawPath, "/") {
		if segment != "" {
			count++
		}
	}

	return count
}
