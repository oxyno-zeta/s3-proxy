//go:build unit

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetObjectRequestPath(t *testing.T) {
	for _, mount := range []string{"", "/files"} {
		for _, target := range []string{
			"/private/file.txt", "/%70rivate/file.txt", "/private%2Ffile.txt", "/%2570rivate/file.txt",
			"/private/file%3F.txt?next=/public", "/private/file%25.txt", "/private/file%2B.txt",
		} {
			t.Run(mount+target, func(t *testing.T) {
				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, mount+target, nil)
				router := chi.NewRouter()
				called := false

				router.Route(mount+"/", func(r chi.Router) {
					r.Get("/*", func(_ http.ResponseWriter, r *http.Request) {
						called = true
						got, err := getObjectRequestPath(r)
						assert.NoError(t, err)
						assert.Equal(t, req.URL.Path[len(mount)+1:], got)
					})
				})
				router.ServeHTTP(httptest.NewRecorder(), req)
				assert.True(t, called)
			})
		}
	}
}
