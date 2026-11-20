//go:build integration

package server

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	cmocks "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config/mocks"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/webhook"
)

func TestEncodedPathAuthentication(t *testing.T) {
	trueValue := true

	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"

	s3api, s3server, err := setupFakeS3(accessKey, secretAccessKey, region, bucket)
	require.NoError(t, err)

	defer s3server.Close()

	cfg := &config.Config{
		Server: &config.ServerConfig{
			Compress: &config.ServerCompressConfig{
				Enabled: &config.DefaultServerCompressEnabled,
				Level:   config.DefaultServerCompressLevel,
				Types:   config.DefaultServerCompressTypes,
			},
		},
		ListTargets: &config.ListTargetsConfig{},
		Tracing:     &config.TracingConfig{},
		Templates:   testsDefaultGeneralTemplateConfig,
		AuthProviders: &config.AuthProviderConfig{
			Basic: map[string]*config.BasicAuthConfig{
				"provider1": {Realm: "realm1"},
			},
		},
		Targets: map[string]*config.TargetConfig{
			"target1": {
				Name: "target1",
				Bucket: &config.BucketConfig{
					Name:       bucket,
					Region:     region,
					S3Endpoint: s3server.URL,
					Credentials: &config.BucketCredentialConfig{
						AccessKey: &config.CredentialConfig{Value: accessKey},
						SecretKey: &config.CredentialConfig{Value: secretAccessKey},
					},
					DisableSSL: true,
				},
				Mount: &config.MountConfig{Path: []string{"/"}},
				Resources: []*config.Resource{
					{
						Path:     "/private/**",
						Methods:  []string{"GET", "HEAD", "PUT", "DELETE"},
						Provider: "provider1",
						Basic: &config.ResourceBasic{Credentials: []*config.BasicAuthUserConfig{
							{User: "user1", Password: &config.CredentialConfig{Value: "pass1"}},
						}},
					},
					{Path: "/**", Methods: []string{"GET", "HEAD", "PUT", "DELETE"}, WhiteList: &trueValue},
				},
				Actions: &config.ActionsConfig{
					GET:    &config.GetActionConfig{Enabled: true},
					HEAD:   &config.HeadActionConfig{Enabled: true},
					PUT:    &config.PutActionConfig{Enabled: true},
					DELETE: &config.DeleteActionConfig{Enabled: true},
				},
			},
		},
	}

	ctrl := gomock.NewController(t)
	cfgManagerMock := cmocks.NewMockManager(ctrl)
	cfgManagerMock.EXPECT().GetConfig().AnyTimes().Return(cfg)

	logger := log.NewLogger()
	tsvc, err := tracing.New(cfgManagerMock, logger)
	require.NoError(t, err)

	s3Manager := s3client.NewManager(cfgManagerMock, metricsCtx)
	err = s3Manager.Load()
	require.NoError(t, err)

	webhookManager := webhook.NewManager(cfgManagerMock, metricsCtx)

	svr := &Server{
		logger:          logger,
		cfgManager:      cfgManagerMock,
		metricsCl:       metricsCtx,
		tracingSvc:      tsvc,
		s3clientManager: s3Manager,
		webhookManager:  webhookManager,
	}

	var router http.Handler

	for _, key := range []string{"private/payroll/q3.csv", "public/file.txt", "%70rivate/payroll/q3.csv"} {
		_, err = s3api.PutObjectWithContext(t.Context(), &s3.PutObjectInput{
			Bucket: new(bucket), Key: new(key), Body: strings.NewReader(key),
		})
		require.NoError(t, err)
	}

	for _, mount := range []string{"/", "/files"} {
		cfg.Targets["target1"].Mount.Path = []string{mount}
		cfg.Targets["target1"].Resources[0].Path = strings.TrimSuffix(mount, "/") + "/private/**"
		router, err = svr.generateRouter()
		require.NoError(t, err)

		for _, method := range []string{http.MethodGet, http.MethodHead, http.MethodPut, http.MethodDelete} {
			for _, path := range []string{
				"/private/payroll/q3.csv", "/%70rivate/payroll/q3.csv", "/privat%65/payroll/q3.csv",
				"/pr%69vate/p%61yroll/q3.csv", "/private%2Fpayroll/q3.csv", "/private%2fpayroll/q3.csv",
			} {
				t.Run(mount+method+path, func(t *testing.T) {
					for _, password := range []string{"", "wrong"} {
						req := httptest.NewRequestWithContext(t.Context(), method, strings.TrimSuffix(mount, "/")+path, nil)
						if password != "" {
							req.SetBasicAuth("user1", password)
						}

						w := httptest.NewRecorder()
						router.ServeHTTP(w, req)
						assert.Equal(t, http.StatusUnauthorized, w.Code)
					}
				})
			}
		}

		for _, tc := range []struct{ path, password, body string }{
			{"/%70rivate/payroll/q3.csv", "pass1", "private/payroll/q3.csv"},
			{"/private%2Fpayroll/q3.csv", "pass1", "private/payroll/q3.csv"},
			{"/public/file.txt?next=/private/payroll/q3.csv", "", "public/file.txt"},
			{"/%2570rivate/payroll/q3.csv", "", "%70rivate/payroll/q3.csv"},
		} {
			t.Run(mount+tc.path, func(t *testing.T) {
				req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, strings.TrimSuffix(mount, "/")+tc.path, nil)
				if tc.password != "" {
					req.SetBasicAuth("user1", tc.password)
				}

				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)
				assert.Equal(t, http.StatusOK, w.Code)
				assert.Equal(t, tc.body, w.Body.String())
			})
		}
	}
}
