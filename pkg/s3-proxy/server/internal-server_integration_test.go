//go:build integration

package server

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	cmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/stretchr/testify/assert"
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

			// Load configuration in manager
			cfgManagerMock.EXPECT().GetConfig().Return(&config.Config{
				InternalServer: &config.ServerConfig{
					ListenAddr: "",
					Port:       8080,
					Compress: &config.ServerCompressConfig{
						Enabled: &config.DefaultServerCompressEnabled,
						Level:   config.DefaultServerCompressLevel,
						Types:   config.DefaultServerCompressTypes,
					},
				},
			})

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

func TestInternal_Server_Listen(t *testing.T) {
	// Create go mock controller
	ctrl := gomock.NewController(t)
	cfgManagerMock := cmocks.NewMockManager(ctrl)

	// Load configuration in manager
	cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(&config.Config{
		InternalServer: &config.ServerConfig{
			ListenAddr: "",
			Port:       8080,
			Compress: &config.ServerCompressConfig{
				Enabled: &config.DefaultServerCompressEnabled,
				Level:   config.DefaultServerCompressLevel,
				Types:   config.DefaultServerCompressTypes,
			},
		},
	})

	svr := NewInternalServer(log.NewLogger(), cfgManagerMock, metricsCtx)
	// Generate server
	svr.GenerateServer()

	var wg sync.WaitGroup
	// Add a wait
	wg.Add(1)
	// Listen and synchronize wait
	go func() error {
		wg.Done()
		return svr.Listen()
	}()
	// Wait server up and running
	wg.Wait()
	// Sleep 1 second in order to wait again start server
	time.Sleep(time.Second)

	// Do a request
	resp, err := http.Get("http://localhost:8080/health")
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 200, resp.StatusCode)
	// Defer close server
	err = svr.server.Close()
	assert.NoError(t, err)
}
