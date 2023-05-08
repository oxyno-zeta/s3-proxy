//go:build integration

package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
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
	accessKey := "MINIO_ACCESS_KEY"
	secretAccessKey := "MINIO_SECRET_KEY"
	s3Endpoint := "http://localhost:9000"
	region := "eu-central-1"
	bucket := "test-bucket"
	tracingConfig := &config.TracingConfig{}

	// S3 mock server for all subtests.
	_, err := setupFakeS3(accessKey, secretAccessKey, region, bucket)
	if err != nil {
		t.Fatal(err)
		return
	}

	nextPort := 20000

	// Write the test certificate/private key to a file. We use a directory that is
	certDir, err := os.MkdirTemp("", "test#?tls_*")
	if err != nil {
		t.Fatal("Unable to create temporary directory:", err)
		return
	}
	t.Cleanup(func() { os.RemoveAll(certDir) })

	certFilename := filepath.Join(certDir, "cert.pem")
	certFile, err := os.Create(certFilename)
	if err != nil {
		t.Fatal("Unable to create certificate file:", err)
		return
	}
	certFile.WriteString(testCertificate)
	certFile.Close()

	privKeyFilename := filepath.Join(certDir, "key.pem")
	privKeyFile, err := os.Create(privKeyFilename)
	if err != nil {
		t.Fatal("Unable to create private key file:", err)
		return
	}
	privKeyFile.WriteString(testPrivateKey)
	privKeyFile.Close()

	// makeConfig creates a new config for a server, assigning it the next port in sequence.
	makeConfig := func(sslConfig *config.ServerSSLConfig) *config.Config {
		port := nextPort
		nextPort++

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

	type testExpect int
	const (
		testExpectSuccess testExpect = iota
		testExpectErr
		testExpectSetupErr
	)

	tests := []struct {
		name     string
		config   *config.Config
		expect   testExpect
		errorStr string
	}{
		{
			name: "Self-signed certificates only",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled:             true,
				SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
			}),
		},
		{
			name: "Test supplied certificate",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate: &testCertificate,
						PrivateKey:  &testPrivateKey,
					},
				},
			}),
		},
		{
			name: "Test both supplied and generated certificates",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						Certificate: &testCertificate,
						PrivateKey:  &testPrivateKey,
					},
				},
				SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
			}),
		},
		{
			name: "Certificate/PrivateKey stored in S3",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("s3://test-bucket/ssl/certificate.pem"),
						CertificateURLConfig: &config.SSLURLConfig{
							HTTPTimeout:   "1s",
							AWSRegion:     region,
							AWSEndpoint:   "TEST",
							AWSDisableSSL: true,
							AWSCredentials: &config.BucketCredentialConfig{
								AccessKey: &config.CredentialConfig{
									Value: accessKey,
								},
								SecretKey: &config.CredentialConfig{
									Value: secretAccessKey,
								},
							},
						},
						PrivateKeyURL: aws.String("s3://test-bucket/ssl/privateKey.pem"),
						PrivateKeyURLConfig: &config.SSLURLConfig{
							HTTPTimeout:   "1s",
							AWSRegion:     region,
							AWSEndpoint:   "TEST",
							AWSDisableSSL: true,
							AWSCredentials: &config.BucketCredentialConfig{
								AccessKey: &config.CredentialConfig{
									Value: accessKey,
								},
								SecretKey: &config.CredentialConfig{
									Value: secretAccessKey,
								},
							},
						},
					},
				},
			}),
		},
		{
			name: "Certificate/PrivateKey stored in S3 (ARN)",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("arn:aws:s3:::test-bucket/ssl/certificate.pem"),
						CertificateURLConfig: &config.SSLURLConfig{
							HTTPTimeout:   "1s",
							AWSRegion:     region,
							AWSEndpoint:   "TEST",
							AWSDisableSSL: true,
							AWSCredentials: &config.BucketCredentialConfig{
								AccessKey: &config.CredentialConfig{
									Value: accessKey,
								},
								SecretKey: &config.CredentialConfig{
									Value: secretAccessKey,
								},
							},
						},
						PrivateKeyURL: aws.String("arn:aws:s3:::test-bucket/ssl/privateKey.pem"),
						PrivateKeyURLConfig: &config.SSLURLConfig{
							HTTPTimeout:   "1s",
							AWSRegion:     region,
							AWSEndpoint:   "TEST",
							AWSDisableSSL: true,
							AWSCredentials: &config.BucketCredentialConfig{
								AccessKey: &config.CredentialConfig{
									Value: accessKey,
								},
								SecretKey: &config.CredentialConfig{
									Value: secretAccessKey,
								},
							},
						},
					},
				},
			}),
		},
		{
			name: "Certificate/PrivateKey stored in HTTP",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("HTTP_TEST"),
						CertificateURLConfig: &config.SSLURLConfig{
							HTTPTimeout: "10s",
						},
						PrivateKeyURL: aws.String("HTTP_TEST"),
					},
				},
			}),
		},
		{
			name: "Certificate stored in HTTP invalid host",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("http://169.254.0.30:80/certificate.pem"),
						CertificateURLConfig: &config.SSLURLConfig{
							HTTPTimeout: "1ms",
						},
						PrivateKeyURL: aws.String("HTTP_TEST"),
					},
				},
			}),
			expect: testExpectSetupErr,
		},
		{
			name: "Certificate stored in HTTP not found",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("HTTP_NOT_FOUND"),
						PrivateKeyURL:  aws.String("HTTP_TEST"),
					},
				},
			}),
			expect: testExpectSetupErr,
		},
		{
			name: "Certificate stored in file",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: &certFilename,
						PrivateKeyURL:  &privKeyFilename,
					},
				},
			}),
		},
		{
			name: "Certificate stored in file (URL)",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String(filePathAsURL(certFilename)),
						PrivateKeyURL:  aws.String(filePathAsURL(privKeyFilename)),
					},
				},
			}),
		},
		{
			name: "Invalid HTTP URL with fragment",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("http://example.com/certificate.pem#fragment"),
						PrivateKeyURL:  aws.String("http://example.com/privateKey.pem"),
					},
				},
			}),
			expect:   testExpectSetupErr,
			errorStr: "failed to create TLS configuration for server: unable to load certificate: failed to get certificate from URL: http://example.com/certificate.pem#fragment: http URL cannot contain fragment",
		},
		{
			name: "Invalid file URL with fragment",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("file:///tmp/certificate.pem#fragment"),
						PrivateKeyURL:  aws.String("http://example.com/privateKey.pem"),
					},
				},
			}),
			expect:   testExpectSetupErr,
			errorStr: "failed to create TLS configuration for server: unable to load certificate: failed to get certificate from URL: file:///tmp/certificate.pem#fragment: file URL cannot contain fragment",
		},
		{
			name: "Invalid file URL with query",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("file:///tmp/certificate.pem?query"),
						PrivateKeyURL:  aws.String("http://example.com/privateKey.pem"),
					},
				},
			}),
			expect:   testExpectSetupErr,
			errorStr: "failed to create TLS configuration for server: unable to load certificate: failed to get certificate from URL: file:///tmp/certificate.pem?query: file URL cannot contain query",
		},
		{
			name: "Invalid S3 custom endpoint",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("s3://bucket/key"),
						PrivateKeyURL:  aws.String("http://example.com/privateKey.pem"),
						CertificateURLConfig: &config.SSLURLConfig{
							AWSEndpoint: ":r&qwer+asdf",
							AWSRegion:   "us-east-7",
						},
					},
				},
			}),
			expect:   testExpectSetupErr,
			errorStr: "failed to create TLS configuration for server: unable to load certificate: failed to get certificate from URL: s3://bucket/key: invalid S3 endpoint URL: :r&qwer+asdf:",
		},
		{
			name: "Invalid S3 URL (wrong key)",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("s3://test-bucket/ssl/not_found.pem"),
						CertificateURLConfig: &config.SSLURLConfig{
							HTTPTimeout:   "1s",
							AWSRegion:     region,
							AWSEndpoint:   "TEST",
							AWSDisableSSL: true,
							AWSCredentials: &config.BucketCredentialConfig{
								AccessKey: &config.CredentialConfig{
									Value: accessKey,
								},
								SecretKey: &config.CredentialConfig{
									Value: secretAccessKey,
								},
							},
						},
						PrivateKeyURL: aws.String("http://example.com/privateKey.pem"),
					},
				},
			}),
			expect:   testExpectSetupErr,
			errorStr: "failed to create TLS configuration for server: unable to load certificate: failed to get certificate from URL: s3://test-bucket/ssl/not_found.pem: NoSuchKey",
		},
		{
			name: "Invalid S3 URL (no key)",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("s3://test-bucket"),
						CertificateURLConfig: &config.SSLURLConfig{
							HTTPTimeout:   "1s",
							AWSRegion:     region,
							AWSEndpoint:   "TEST",
							AWSDisableSSL: true,
							AWSCredentials: &config.BucketCredentialConfig{
								AccessKey: &config.CredentialConfig{
									Value: accessKey,
								},
								SecretKey: &config.CredentialConfig{
									Value: secretAccessKey,
								},
							},
						},
						PrivateKeyURL: aws.String("http://example.com/privateKey.pem"),
					},
				},
			}),
			expect:   testExpectSetupErr,
			errorStr: "failed to create TLS configuration for server: unable to load certificate: failed to get certificate from URL: s3://test-bucket: missing S3 key",
		},
		{
			name: "Invalid S3 ARN with account",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("arn:aws:s3::123456789012:bucket/certificate.pem"),
						PrivateKeyURL:  aws.String("http://example.com/privateKey.pem"),
					},
				},
			}),
			expect:   testExpectSetupErr,
			errorStr: "failed to create TLS configuration for server: unable to load certificate: failed to get certificate from URL: arn:aws:s3::123456789012:bucket/certificate.pem: invalid S3 ARN: account ID cannot be set",
		},
		{
			name: "Invalid S3 ARN with region",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("arn:aws:s3:us-east-7::bucket/certificate.pem"),
						PrivateKeyURL:  aws.String("http://example.com/privateKey.pem"),
					},
				},
			}),
			expect:   testExpectSetupErr,
			errorStr: "failed to create TLS configuration for server: unable to load certificate: failed to get certificate from URL: arn:aws:s3:us-east-7::bucket/certificate.pem: invalid S3 ARN: region cannot be set",
		},
		{
			name: "Invalid S3 ARN without key",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("arn:aws:s3:::bucket"),
						PrivateKeyURL:  aws.String("http://example.com/privateKey.pem"),
					},
				},
			}),
			expect:   testExpectSetupErr,
			errorStr: "failed to create TLS configuration for server: unable to load certificate: failed to get certificate from URL: arn:aws:s3:::bucket: missing S3 key",
		},
		{
			name: "Invalid AWS service in ARN",
			config: makeConfig(&config.ServerSSLConfig{
				Enabled: true,
				Certificates: []*config.ServerSSLCertificate{
					{
						CertificateURL: aws.String("arn:aws:iam::123456789012:role/myrole"),
						PrivateKeyURL:  aws.String("http://example.com/privateKey.pem"),
					},
				},
			}),
			expect:   testExpectSetupErr,
			errorStr: "failed to create TLS configuration for server: unable to load certificate: failed to get certificate from URL: arn:aws:iam::123456789012:role/myrole: unsupported AWS service in ARN: iam",
		},
	}

	for _, currentTest := range tests {
		// Capture the current test for parallel processing. Otherwise currentTest will be modified during our test run.
		tt := currentTest

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// An HTTP test server that returns the certificate and private key (created if needed).
			var httpServer *httptest.Server

			// Update each target to use this mock server.
			for _, targetCfg := range tt.config.Targets {
				targetCfg.Bucket.S3Endpoint = s3Endpoint
			}

			if tt.config.Server.SSL != nil {
				for _, cert := range tt.config.Server.SSL.Certificates {
					if cert.CertificateURL != nil && (*cert.CertificateURL == "HTTP_TEST" || *cert.CertificateURL == "HTTP_NOT_FOUND") {
						if httpServer == nil {
							httpServer = httptest.NewServer(http.HandlerFunc(serveTestCertificate))
							defer httpServer.Close()
						}

						if *cert.CertificateURL == "HTTP_TEST" {
							cert.CertificateURL = aws.String(fmt.Sprintf("%s/certificate.pem", httpServer.URL))
						} else {
							cert.CertificateURL = aws.String(fmt.Sprintf("%s/not_found.pem", httpServer.URL))
						}
					}

					if cert.PrivateKeyURL != nil && (*cert.PrivateKeyURL == "HTTP_TEST" || *cert.PrivateKeyURL == "HTTP_NOT_FOUND") {
						if httpServer == nil {
							httpServer = httptest.NewServer(http.HandlerFunc(serveTestCertificate))
							defer httpServer.Close()
						}

						if *cert.PrivateKeyURL == "HTTP_TEST" {
							cert.PrivateKeyURL = aws.String(fmt.Sprintf("%s/privateKey.pem", httpServer.URL))
						} else {
							cert.PrivateKeyURL = aws.String(fmt.Sprintf("%s/not_found.pem", httpServer.URL))
						}
					}

					if cert.CertificateURLConfig != nil && cert.CertificateURLConfig.AWSEndpoint == "TEST" {
						fmt.Printf("Replacing Certificate endpoint with %#v\n", s3Endpoint)
						cert.CertificateURLConfig.AWSEndpoint = s3Endpoint
					}

					if cert.PrivateKeyURLConfig != nil && cert.PrivateKeyURLConfig.AWSEndpoint == "TEST" {
						cert.PrivateKeyURLConfig.AWSEndpoint = s3Endpoint
					}
				}
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
			if !assert.NoError(t, err) {
				return
			}

			// Create webhook manager
			webhookManager := webhook.NewManager(cfgManagerMock, metricsCtx)

			// Create S3 Manager
			s3Manager := s3client.NewManager(cfgManagerMock, metricsCtx)
			err = s3Manager.Load()
			if !assert.NoError(t, err) {
				return
			}

			svr := &Server{
				logger:          logger,
				cfgManager:      cfgManagerMock,
				metricsCl:       metricsCtx,
				tracingSvc:      tsvc,
				s3clientManager: s3Manager,
				webhookManager:  webhookManager,
			}

			err = svr.GenerateServer()
			if tt.expect == testExpectSetupErr {
				if assert.Error(t, err) {
					if !strings.HasPrefix(err.Error(), tt.errorStr) {
						t.Errorf("Got error %v, wanted %v", err.Error(), tt.errorStr)
					}
				}

				return
			} else if !assert.NoError(t, err) {
				return
			}

			// We're using untrusted test certificates, so we skip verification here.
			tr := &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}

			go svr.Listen()

			httpClient := &http.Client{Transport: tr}

			// FIXME: This is a race condition with the server actually calling Listen(). Fixing this requires
			// modifications to the server code to incorporate a condition variable (or similar).
			time.Sleep(500 * time.Millisecond)

			// The request URL is fixed for all of the above integration tests.
			requestURL := fmt.Sprintf("https://127.0.0.1:%d/mount/folder1/test.txt", tt.config.Server.Port)
			response, err := httpClient.Get(requestURL)

			if tt.expect != testExpectSuccess {
				if assert.Error(t, err) {
					if !strings.HasPrefix(err.Error(), tt.errorStr) {
						t.Errorf("Got error %v, wanted %v", err.Error(), tt.errorStr)
					}
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got %v", err)
				} else {
					// For all non-error tests, we expect HTTP 200 (OK)
					if response.StatusCode != http.StatusOK {
						t.Errorf("Expected status code %d but got %d", http.StatusOK, response.StatusCode)
					}

					// And the body should be "Hello folder1" (see setupMockS3).
					body, err := io.ReadAll(response.Body)
					assert.NoError(t, err)
					assert.Equal(t, "Hello folder1!", string(body))
				}
			}

			svr.server.Shutdown(context.Background())
		})
	}
}

func serveTestCertificate(rw http.ResponseWriter, req *http.Request) {
	defer req.Body.Close()

	header := rw.Header()
	header.Add("Content-Type", "text/plain; charset=utf-8")

	if req.Method != http.MethodGet {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("Invalid HTTP method"))
		return
	}

	var body string

	switch req.URL.Path {
	case "/certificate.pem":
		body = testCertificate
	case "/privateKey.pem":
		body = testPrivateKey
	default:
		rw.WriteHeader(http.StatusNotFound)
		rw.Write([]byte("Path not found"))
	}

	rw.WriteHeader(http.StatusOK)
	rw.Write([]byte(body))
}

func filePathAsURL(filename string) string {
	parts := strings.Split(filename, string(os.PathSeparator))
	escapedParts := make([]string, 0, len(parts))

	for _, part := range parts {
		escapedParts = append(escapedParts, url.PathEscape(part))
	}

	return fmt.Sprintf("file://%s", strings.Join(escapedParts, "/"))
}
