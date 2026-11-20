//go:build unit

package authorization

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/opentracing/opentracing-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

func TestOPARequestPath(t *testing.T) {
	for _, target := range []string{
		"/private/payroll/q3.csv", "/%70rivate/payroll/q3.csv", "/private%2Fpayroll/q3.csv",
		"/pr%69vate/p%61yroll/q3.csv?next=/public/file.txt",
	} {
		t.Run(target, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				var input inputOPA
				if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
					t.Error(err)
					w.WriteHeader(http.StatusBadRequest)

					return
				}

				assert.Equal(t, "/private/payroll/q3.csv", input.Input.Request.Path)
				assert.Equal(t, []string{"private", "payroll", "q3.csv"}, input.Input.Request.ParsedPath)

				_, err := w.Write([]byte(`{"result":false}`))
				assert.NoError(t, err)
			}))
			defer server.Close()

			span := opentracing.StartSpan("test")
			defer span.Finish()

			req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
			req = req.WithContext(opentracing.ContextWithSpan(req.Context(), span))
			authorized, err := isOPAServerAuthorized(req, nil, &config.ResourceHeaderOIDC{
				AuthorizationOPAServer: &config.OPAServerAuthorization{URL: server.URL},
			})
			require.NoError(t, err)
			assert.False(t, authorized)
		})
	}
}
