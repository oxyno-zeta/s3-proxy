//go:build integration

package server

import (
	"context"
	"crypto/tls"
	"io"
	"net/http"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	cmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/webhook"
	"github.com/stretchr/testify/assert"
)

func TestTLSServer(t *testing.T) {
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"
	tracingConfig := &config.TracingConfig{}

	makeConfig := func(port int, sslConfig *config.ServerSSLConfig) *config.Config {
		return &config.Config{
			ListTargets: &config.ListTargetsConfig{},
			Server: &config.ServerConfig{
				ListenAddr: "127.0.0.1",
				Port:       port,
				SSL:        sslConfig,
			},
			Tracing: tracingConfig,
			Targets: map[string]*config.TargetConfig{
				"target1": {
					Name: "target1",
					Bucket: &config.BucketConfig{
						Name:   bucket,
						Region: region,
						Credentials: &config.BucketCredentialConfig{
							AccessKey: &config.CredentialConfig{Value: accessKey},
							SecretKey: &config.CredentialConfig{Value: secretAccessKey},
						},
						DisableSSL: true,
					},
					Mount: &config.MountConfig{
						Path: []string{"/mount/"},
					},
					Actions: &config.ActionsConfig{
						GET: &config.GetActionConfig{Enabled: true},
					},
				},
			},
			Templates: testsDefaultGeneralTemplateConfig,
		}
	}

	tests := []struct {
		name         string
		config       *config.Config
		inputMethod  string
		inputURL     string
		expectedCode int
		expectedBody string
		wantErr      bool
	}{
		{
			name: "Self-signed certificates only",
			config: makeConfig(8081, &config.ServerSSLConfig{
				Enabled:             true,
				SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
			}),
			inputMethod:  "GET",
			inputURL:     "https://localhost:8081/mount/folder1/test.txt",
			expectedCode: 200,
			expectedBody: "Hello folder1!",
		},
		{
			name: "Test supplied certificate",
			config: makeConfig(8082, &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []config.ServerSSLCertificate{
					{
						Certificate: &testCertificate,
						PrivateKey:  &testPrivateKey,
					},
				},
			}),
			inputMethod:  "GET",
			inputURL:     "https://localhost:8082/mount/folder1/test.txt",
			expectedCode: 200,
			expectedBody: "Hello folder1!",
		},
		{
			name: "Test both supplied and generated certificates",
			config: makeConfig(8083, &config.ServerSSLConfig{
				Enabled: true,
				Certificates: []config.ServerSSLCertificate{
					{
						Certificate: &testCertificate,
						PrivateKey:  &testPrivateKey,
					},
				},
				SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
			}),
			inputMethod:  "GET",
			inputURL:     "https://localhost:8083/mount/folder1/test.txt",
			expectedCode: 200,
			expectedBody: "Hello folder1!",
		},
	}

	for _, currentTest := range tests {
		// Capture the current test for parallel processing. Otherwise currentTest will be modified during our test run.
		tt := currentTest

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Create an S3 mock server specifically for this test.
			s3server, err := setupFakeS3(
				accessKey,
				secretAccessKey,
				region,
				bucket,
			)
			if err != nil {
				t.Fatal(err)
				return
			}
			defer s3server.Close()

			// Update each target to use this mock server.
			for _, targetCfg := range tt.config.Targets {
				targetCfg.Bucket.S3Endpoint = s3server.URL
			}

			// Create go mock controller
			ctrl := gomock.NewController(t)
			cfgManagerMock := cmocks.NewMockManager(ctrl)

			// Load configuration in manager
			cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(tt.config)
			cfgManagerMock.EXPECT().AddOnChangeHook(gomock.Any()).AnyTimes()

			logger := log.NewLogger()

			// Create tracing service
			tsvc, err := tracing.New(cfgManagerMock, logger)
			assert.NoError(t, err)

			// Create webhook manager
			webhookManager := webhook.NewManager(cfgManagerMock, metricsCtx)

			// Create S3 Manager
			s3Manager := s3client.NewManager(cfgManagerMock, metricsCtx)
			err = s3Manager.Load()
			assert.NoError(t, err)

			svr := &Server{
				logger:          logger,
				cfgManager:      cfgManagerMock,
				metricsCl:       metricsCtx,
				tracingSvc:      tsvc,
				s3clientManager: s3Manager,
				webhookManager:  webhookManager,
			}

			err = svr.GenerateServer()
			assert.NoError(t, err)
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}

			go svr.Listen()

			httpClient := &http.Client{Transport: tr}
			response, err := httpClient.Get(tt.inputURL)

			// Test status code
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got %v", err)
				} else {
					if response.StatusCode != tt.expectedCode {
						t.Errorf("Expected status code %d but got %d", tt.expectedCode, response.StatusCode)
					}

					body, err := io.ReadAll(response.Body)
					assert.NoError(t, err)
					assert.Equal(t, tt.expectedBody, string(body))
				}
			}

			svr.server.Shutdown(context.Background())
		})
	}
}
