//go:build integration

package server

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	cmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
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

			assert.Equal(t, tt.expectedCode, w.Code)

			if tt.expectedBody != "" {
				body := w.Body.String()
				assert.Equal(t, tt.expectedBody, body)
			}

			if tt.notExpectedBody != "" {
				body := w.Body.String()
				assert.Equal(t, tt.notExpectedBody, body)
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

func TestInternalServer_config_endpoint(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.Config
		expectedCode int
		expectedBody string
	}{
		{
			name: "minimum configuration",
			cfg: &config.Config{
				InternalServer: &config.ServerConfig{
					ListenAddr: "",
					Port:       8080,
					Compress: &config.ServerCompressConfig{
						Enabled: &config.DefaultServerCompressEnabled,
						Level:   config.DefaultServerCompressLevel,
						Types:   config.DefaultServerCompressTypes,
					},
				},
			},
			expectedCode: 200,
			expectedBody: `{"config":{
				"log":null,
				"tracing":null,
				"metrics":null,
				"server":null,
				"internalServer":{
					"timeouts":null,
					"cors":null,
					"cache":null,
					"compress":{
						"enabled":true,
						"types":["text/html","text/css","text/plain","text/javascript","application/javascript","application/x-javascript","application/json","application/atom+xml","application/rss+xml","image/svg+xml"],
						"level":5
					},
					"ssl":null,
					"listenAddr":"",
					"port":8080
				},
				"targets":null,
				"templates":null,
				"authProviders":null,
				"listTargets":null
			}}`,
		},
		{
			name: "1 target configuration",
			cfg: &config.Config{
				Log: &config.LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &config.ServerConfig{
					Port: 8080,
					Compress: &config.ServerCompressConfig{
						Enabled: &config.DefaultServerCompressEnabled,
						Level:   config.DefaultServerCompressLevel,
						Types:   config.DefaultServerCompressTypes,
					},
					Timeouts: &config.ServerTimeoutsConfig{
						ReadHeaderTimeout: config.DefaultServerTimeoutsReadHeaderTimeout,
					},
				},
				InternalServer: &config.ServerConfig{
					Port: 9090,
					Compress: &config.ServerCompressConfig{
						Enabled: &config.DefaultServerCompressEnabled,
						Level:   config.DefaultServerCompressLevel,
						Types:   config.DefaultServerCompressTypes,
					},
					Timeouts: &config.ServerTimeoutsConfig{
						ReadHeaderTimeout: config.DefaultServerTimeoutsReadHeaderTimeout,
					},
				},
				Templates: &config.TemplateConfig{
					Helpers: []string{"templates/_helpers.tpl"},
					FolderList: &config.TemplateConfigItem{
						Path: "templates/folder-list.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "200",
					},
					TargetList: &config.TemplateConfigItem{
						Path: "templates/target-list.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "200",
					},
					NotFoundError: &config.TemplateConfigItem{
						Path: "templates/not-found-error.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "404",
					},
					InternalServerError: &config.TemplateConfigItem{
						Path: "templates/internal-server-error.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "500",
					},
					UnauthorizedError: &config.TemplateConfigItem{
						Path: "templates/unauthorized-error.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "401",
					},
					ForbiddenError: &config.TemplateConfigItem{
						Path: "templates/forbidden-error.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "403",
					},
					BadRequestError: &config.TemplateConfigItem{
						Path: "templates/bad-request-error.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "400",
					},
					Put: &config.TemplateConfigItem{
						Path:    "templates/put.tpl",
						Headers: map[string]string{},
						Status:  "204",
					},
					Delete: &config.TemplateConfigItem{
						Path:    "templates/delete.tpl",
						Headers: map[string]string{},
						Status:  "204",
					},
				},
				Tracing: &config.TracingConfig{Enabled: false},
				Metrics: &config.MetricsConfig{DisableRouterPath: false},
				ListTargets: &config.ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*config.TargetConfig{
					"test": {
						Name: "test",
						Mount: &config.MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &config.BucketConfig{
							Name:                "bucket1",
							Region:              "us-east-1",
							S3ListMaxKeys:       1000,
							S3MaxUploadParts:    10000,
							S3UploadPartSize:    5,
							S3UploadConcurrency: 5,
							Credentials: &config.BucketCredentialConfig{
								AccessKey: &config.CredentialConfig{
									Env:   "FAKE",
									Value: "fake",
								},
								SecretKey: &config.CredentialConfig{
									Path:  "/secret",
									Value: "fake",
								},
							},
						},
						Actions: &config.ActionsConfig{
							GET: &config.GetActionConfig{Enabled: true},
						},
						Templates: &config.TargetTemplateConfig{},
					},
				},
			},
			expectedCode: 200,
			expectedBody: `
{
  "config": {
    "log": { "level": "info", "format": "json", "filePath": "" },
    "tracing": {
      "fixedTags": null,
      "flushInterval": "",
      "udpHost": "",
      "queueSize": 0,
      "enabled": false,
      "logSpan": false
    },
    "metrics": { "disableRouterPath": false },
    "server": {
      "timeouts": {
        "readTimeout": "",
        "readHeaderTimeout": "60s",
        "writeTimeout": "",
        "idleTimeout": ""
      },
      "cors": null,
      "cache": null,
      "compress": {
        "enabled": true,
        "types": [
          "text/html",
          "text/css",
          "text/plain",
          "text/javascript",
          "application/javascript",
          "application/x-javascript",
          "application/json",
          "application/atom+xml",
          "application/rss+xml",
          "image/svg+xml"
        ],
        "level": 5
      },
      "ssl": null,
      "listenAddr": "",
      "port": 8080
    },
    "internalServer": {
      "timeouts": {
        "readTimeout": "",
        "readHeaderTimeout": "60s",
        "writeTimeout": "",
        "idleTimeout": ""
      },
      "cors": null,
      "cache": null,
      "compress": {
        "enabled": true,
        "types": [
          "text/html",
          "text/css",
          "text/plain",
          "text/javascript",
          "application/javascript",
          "application/x-javascript",
          "application/json",
          "application/atom+xml",
          "application/rss+xml",
          "image/svg+xml"
        ],
        "level": 5
      },
      "ssl": null,
      "listenAddr": "",
      "port": 9090
    },
    "targets": {
      "test": {
        "bucket": {
          "credentials": null,
          "requestConfig": null,
          "name": "bucket1",
          "prefix": "",
          "region": "us-east-1",
          "s3Endpoint": "",
          "s3ListMaxKeys": 1000,
          "s3MaxUploadParts": 10000,
          "s3UploadPartSize": 5,
          "s3UploadConcurrency": 5,
          "s3UploadLeavePartsOnError": false,
          "disableSSL": false,
		  "credentials": {
		  	"accessKey": {"env": "FAKE","path":""},
		  	"secretKey": {"path": "/secret", "env":""}
		  }
        },
        "resources": null,
        "mount": { "host": "", "path": ["/test/"] },
        "actions": {
          "GET": { "config": null, "enabled": true },
          "HEAD": null,
          "PUT": null,
          "DELETE": null
        },
        "templates": {
          "folderList": null,
          "notFoundError": null,
          "internalServerError": null,
          "forbiddenError": null,
          "unauthorizedError": null,
          "badRequestError": null,
          "put": null,
          "delete": null,
          "helpers": null
        },
        "keyRewriteList": null
      }
    },
    "templates": {
      "folderList": {
        "path": "templates/folder-list.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "200"
      },
      "targetList": {
        "path": "templates/target-list.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "200"
      },
      "notFoundError": {
        "path": "templates/not-found-error.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "404"
      },
      "internalServerError": {
        "path": "templates/internal-server-error.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "500"
      },
      "unauthorizedError": {
        "path": "templates/unauthorized-error.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "401"
      },
      "forbiddenError": {
        "path": "templates/forbidden-error.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "403"
      },
      "badRequestError": {
        "path": "templates/bad-request-error.tpl",
        "headers": {
          "Content-Type": "{{ template \"main.headers.contentType\" . }}"
        },
        "status": "400"
      },
      "put": { "path": "templates/put.tpl", "headers": {}, "status": "204" },
      "delete": {
        "path": "templates/delete.tpl",
        "headers": {},
        "status": "204"
      },
      "helpers": ["templates/_helpers.tpl"]
    },
    "authProviders": null,
    "listTargets": { "mount": null, "resource": null, "enabled": false }
  }
}
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			// Load configuration in manager
			cfgManagerMock.EXPECT().GetConfig().Return(tt.cfg).AnyTimes()

			svr := &InternalServer{
				logger:     log.NewLogger(),
				cfgManager: cfgManagerMock,
				metricsCl:  metricsCtx,
			}
			got := svr.generateInternalRouter()

			w := httptest.NewRecorder()
			req, err := http.NewRequest(
				"GET",
				"http://localhost/config",
				nil,
			)
			if err != nil {
				t.Error(err)
				return
			}
			got.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)

			body := w.Body.String()
			assert.JSONEq(t, tt.expectedBody, body)
		})
	}
}
