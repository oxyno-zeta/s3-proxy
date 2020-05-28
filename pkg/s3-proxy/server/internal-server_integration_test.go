// +build integration

package server

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/golang/mock/gomock"
	cmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
)

func TestInternalServer_generateInternalRouter(t *testing.T) {
	tests := []struct {
		name            string
		inputMethod     string
		inputURL        string
		expectedCode    int
		expectedBody    string
		notExpectedBody string
	}{
		{
			name:         "Should be ok to call /health",
			inputMethod:  "GET",
			inputURL:     "http://localhost/health",
			expectedCode: 200,
			expectedBody: "{\"status\":\"UP\"}\n",
		},
		{
			name:         "Should be ok to call /metrics",
			inputMethod:  "GET",
			inputURL:     "http://localhost/metrics",
			expectedCode: 200,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			svr := &InternalServer{
				logger:     log.NewLogger(),
				cfgManager: cfgManagerMock,
				metricsCl:  metricsCtx,
			}
			got := svr.generateInternalRouter()

			w := httptest.NewRecorder()
			req, err := http.NewRequest(
				tt.inputMethod,
				tt.inputURL,
				nil,
			)
			if err != nil {
				t.Error(err)
				return
			}
			got.ServeHTTP(w, req)
			if tt.expectedCode != w.Code {
				t.Errorf("Integration test on generateInternalRouter() status code = %v, expected status code %v", w.Code, tt.expectedCode)
				return
			}

			if tt.expectedBody != "" {
				body := w.Body.String()
				if tt.expectedBody != body {
					t.Errorf("Integration test on generateInternalRouter() body = \"%v\", expected body \"%v\"", body, tt.expectedBody)
					return
				}
			}

			if tt.notExpectedBody != "" {
				body := w.Body.String()
				if tt.notExpectedBody == body {
					t.Errorf("Integration test on generateInternalRouter() body = \"%v\", not expected body \"%v\"", body, tt.notExpectedBody)
					return
				}
			}
		})
	}
}
