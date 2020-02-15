// +build integration

package server

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/johannesboyne/gofakes3"
	"github.com/johannesboyne/gofakes3/backend/s3mem"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/metrics"
	"github.com/sirupsen/logrus"
)

// Generate metrics instance
var metricsCtx = metrics.NewClient()

func TestInternalRouter(t *testing.T) {
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
			got := GenerateInternalRouter(&logrus.Logger{}, metricsCtx)
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
				t.Errorf("Integration test on GenerateRouter() status code = %v, expected status code %v", w.Code, tt.expectedCode)
				return
			}

			if tt.expectedBody != "" {
				body := w.Body.String()
				if tt.expectedBody != body {
					t.Errorf("Integration test on GenerateRouter() body = \"%v\", expected body \"%v\"", body, tt.expectedBody)
					return
				}
			}

			if tt.notExpectedBody != "" {
				body := w.Body.String()
				if tt.notExpectedBody == body {
					t.Errorf("Integration test on GenerateRouter() body = \"%v\", not expected body \"%v\"", body, tt.notExpectedBody)
					return
				}
			}
		})
	}
}

func TestPublicRouter(t *testing.T) {
	trueValue := true
	accessKey := "YOUR-ACCESSKEYID"
	secretAccessKey := "YOUR-SECRETACCESSKEY"
	region := "eu-central-1"
	bucket := "test-bucket"
	s3server, err := setupFakeS3(
		accessKey,
		secretAccessKey,
		region,
		bucket,
	)
	defer s3server.Close()
	if err != nil {
		t.Error(err)
		return
	}

	type args struct {
		cfg *config.Config
	}
	tests := []struct {
		name               string
		args               args
		inputMethod        string
		inputURL           string
		inputBasicUser     string
		inputBasicPassword string
		inputBody          string
		inputFileName      string
		inputFileKey       string
		expectedCode       int
		expectedBody       string
		notExpectedBody    string
		wantErr            bool
	}{
		{
			name: "GET a not found path",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/not-found/",
			expectedCode: 404,
			expectedBody: "404 page not found\n",
		},
		{
			name: "GET a folder without index document enabled",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/",
			expectedCode: 200,
		},
		{
			name: "GET a file with success",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 200,
			expectedBody: "Hello folder1!",
		},
		{
			name: "GET a file with a not found error",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt-not-existing",
			expectedCode: 404,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Not Found folder1/test.txt-not-existing</h1>
  </body>
</html>
`,
		},
		{
			name: "GET a file with a not found error because of not valid host",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
								Host: "test.local",
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 404,
			expectedBody: "Not Found\n",
		},
		{
			name: "GET a file with success on specific host",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
								Host: "test.local",
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://test.local/mount/folder1/test.txt",
			expectedCode: 200,
			expectedBody: "Hello folder1!",
		},
		{
			name: "GET a file with forbidden error in case of no resource found",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": &config.BasicAuthConfig{
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								&config.Resource{
									Path:     "/mount/folder2/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											&config.BasicAuthUserConfig{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 403,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Forbidden</h1>
  </body>
</html>
`,
		},
		{
			name: "GET a file with forbidden error in case of no resource found because no valid http methods",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": &config.BasicAuthConfig{
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								&config.Resource{
									Path:     "/mount/folder2/*",
									Methods:  []string{"PUT"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											&config.BasicAuthUserConfig{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 403,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Forbidden</h1>
  </body>
</html>
`,
		},
		{
			name: "GET a file with unauthorized error in case of no basic auth",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": &config.BasicAuthConfig{
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								&config.Resource{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											&config.BasicAuthUserConfig{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 401,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
  </body>
</html>
`,
		},
		{
			name: "GET a file with unauthorized error in case of not found basic auth user",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": &config.BasicAuthConfig{
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								&config.Resource{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											&config.BasicAuthUserConfig{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:        "GET",
			inputURL:           "http://localhost/mount/folder1/test.txt",
			inputBasicUser:     "user2",
			inputBasicPassword: "pass2",
			expectedCode:       401,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
  </body>
</html>
`,
		},
		{
			name: "GET a file with unauthorized error in case of wrong basic auth password",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": &config.BasicAuthConfig{
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								&config.Resource{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											&config.BasicAuthUserConfig{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:        "GET",
			inputURL:           "http://localhost/mount/folder1/test.txt",
			inputBasicUser:     "user1",
			inputBasicPassword: "pass2",
			expectedCode:       401,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Unauthorized</h1>
  </body>
</html>
`,
		},
		{
			name: "GET a file with success in case of valid basic auth",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": &config.BasicAuthConfig{
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								&config.Resource{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											&config.BasicAuthUserConfig{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:        "GET",
			inputURL:           "http://localhost/mount/folder1/test.txt",
			inputBasicUser:     "user1",
			inputBasicPassword: "pass1",
			expectedCode:       200,
			expectedBody:       "Hello folder1!",
		},
		{
			name: "GET a file with unauthorized error in case of no oidc cookie or bearer token",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						OIDC: map[string]*config.OIDCAuthConfig{
							"provider1": &config.OIDCAuthConfig{
								ClientID:     "fake-client-id",
								CookieName:   "oidc",
								RedirectURL:  "http://fake-s3-proxy/",
								CallbackPath: "/auth/provider1/callback",
								IssuerURL:    "https://fake-idp/",
								LoginPath:    "/auth/provider1/",
							},
						},
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								&config.Resource{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									OIDC: &config.ResourceOIDC{
										AuthorizationAccesses: []*config.OIDCAuthorizationAccess{},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod: "GET",
			inputURL:    "http://localhost/mount/folder1/test.txt",
			wantErr:     true,
		},
		{
			name: "GET a file with success in case of whitelist",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					AuthProviders: &config.AuthProviderConfig{
						Basic: map[string]*config.BasicAuthConfig{
							"provider1": &config.BasicAuthConfig{
								Realm: "realm1",
							},
						},
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Resources: []*config.Resource{
								&config.Resource{
									Path:      "/mount/folder1/test.txt",
									Methods:   []string{"GET"},
									WhiteList: &trueValue,
								},
								&config.Resource{
									Path:     "/mount/folder1/*",
									Methods:  []string{"GET"},
									Provider: "provider1",
									Basic: &config.ResourceBasic{
										Credentials: []*config.BasicAuthUserConfig{
											&config.BasicAuthUserConfig{
												User: "user1",
												Password: &config.CredentialConfig{
													Value: "pass1",
												},
											},
										},
									},
								},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/test.txt",
			expectedCode: 200,
			expectedBody: "Hello folder1!",
		},
		{
			name: "GET target list",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{
						Enabled: true,
						Mount: &config.MountConfig{
							Path: []string{"/"},
						},
					},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/",
			expectedCode: 200,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Target buckets list</h1>
    <ul>
        <li>target1:
          <ul>
            <li><a href="/mount/">/mount/</a></li>
          </ul>
        </li>
    </ul>
  </body>
</html>
`,
		},
		{
			name: "GET index document with index document enabled with success",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							IndexDocument: "index.html",
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "GET",
			inputURL:     "http://localhost/mount/folder1/",
			expectedCode: 200,
			expectedBody: "<!DOCTYPE html><html><body><h1>Hello folder1!</h1></body></html>",
		},
		{
			name: "GET a path with index document enabled with success",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							IndexDocument: "index.html-fake",
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:     "GET",
			inputURL:        "http://localhost/mount/folder1/",
			expectedCode:    200,
			notExpectedBody: "<!DOCTYPE html><html><body><h1>Hello folder1!</h1></body></html>",
		},
		{
			name: "DELETE a path with a 405 error (method not allowed) because DELETE not enabled",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "DELETE",
			inputURL:     "http://localhost/mount/folder1/text.txt",
			expectedCode: 405,
		},
		{
			name: "DELETE a path with success",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET:    &config.GetActionConfig{Enabled: true},
								DELETE: &config.DeleteActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			inputMethod:  "DELETE",
			inputURL:     "http://localhost/mount/folder1/text.txt",
			expectedCode: 204,
		},
		{
			name: "PUT in a path with success without allow override and don't need it",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
								PUT: &config.PutActionConfig{
									Enabled: true,
									Config: &config.PutActionConfigConfig{
										StorageClass: "Standard",
										Metadata: map[string]string{
											"meta1": "meta1",
										},
									},
								},
							},
						},
					},
				},
			},
			inputMethod:   "PUT",
			inputURL:      "http://localhost/mount/folder1/",
			inputFileName: "test2.txt",
			inputFileKey:  "file",
			inputBody:     "Hello test2!",
			expectedCode:  204,
		},
		{
			name: "PUT in a path without allow override should failed",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
								PUT: &config.PutActionConfig{
									Enabled: true,
									Config: &config.PutActionConfigConfig{
										StorageClass: "Standard",
										Metadata: map[string]string{
											"meta1": "meta1",
										},
									},
								},
							},
						},
					},
				},
			},
			inputMethod:   "PUT",
			inputURL:      "http://localhost/mount/folder1/",
			inputFileName: "test.txt",
			inputFileKey:  "file",
			inputBody:     "Hello test1!",
			expectedCode:  403,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Forbidden</h1>
  </body>
</html>
`,
		},
		{
			name: "PUT in a path with allow override should be ok",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
								PUT: &config.PutActionConfig{
									Enabled: true,
									Config: &config.PutActionConfigConfig{
										StorageClass: "Standard",
										Metadata: map[string]string{
											"meta1": "meta1",
										},
										AllowOverride: true,
									},
								},
							},
						},
					},
				},
			},
			inputMethod:   "PUT",
			inputURL:      "http://localhost/mount/folder1/",
			inputFileName: "test.txt",
			inputFileKey:  "file",
			inputBody:     "Hello test1!",
			expectedCode:  204,
		},
		{
			name: "PUT in a path should fail because no input",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
								PUT: &config.PutActionConfig{
									Enabled: true,
								},
							},
						},
					},
				},
			},
			inputMethod:  "PUT",
			inputURL:     "http://localhost/mount/folder1/",
			expectedCode: 500,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>missing form body</p>
  </body>
</html>
`,
		},
		{
			name: "PUT in a path should fail because wrong key in form",
			args: args{
				cfg: &config.Config{
					ListTargets: &config.ListTargetsConfig{},
					Templates: &config.TemplateConfig{
						FolderList:          "../../templates/folder-list.tpl",
						TargetList:          "../../templates/target-list.tpl",
						NotFound:            "../../templates/not-found.tpl",
						Forbidden:           "../../templates/forbidden.tpl",
						BadRequest:          "../../templates/bad-request.tpl",
						InternalServerError: "../../templates/internal-server-error.tpl",
						Unauthorized:        "../../templates/unauthorized.tpl",
					},
					Targets: []*config.TargetConfig{
						&config.TargetConfig{
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
							Mount: &config.MountConfig{
								Path: []string{"/mount/"},
							},
							Actions: &config.ActionsConfig{
								GET: &config.GetActionConfig{Enabled: true},
								PUT: &config.PutActionConfig{
									Enabled: true,
								},
							},
						},
					},
				},
			},
			inputMethod:   "PUT",
			inputURL:      "http://localhost/mount/folder1/",
			inputFileName: "test.txt",
			inputFileKey:  "wrongkey",
			inputBody:     "Hello test1!",
			expectedCode:  500,
			expectedBody: `<!DOCTYPE html>
<html>
  <body>
    <h1>Internal Server Error</h1>
    <p>http: no such file</p>
  </body>
</html>
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GenerateRouter(&logrus.Logger{}, tt.args.cfg, metricsCtx)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateRouter() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			// If want error at this moment => stop
			if tt.wantErr {
				return
			}
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
			// multipart form
			if tt.inputBody != "" {
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				part, err := writer.CreateFormFile(tt.inputFileKey, filepath.Base(tt.inputFileName))
				if err != nil {
					t.Error(err)
					return
				}
				_, err = io.Copy(part, strings.NewReader(tt.inputBody))
				err = writer.Close()
				if err != nil {
					t.Error(err)
					return
				}
				req, err = http.NewRequest(
					tt.inputMethod,
					tt.inputURL,
					body,
				)
				if err != nil {
					t.Error(err)
					return
				}
				req.Header.Set("Content-Type", writer.FormDataContentType())
			}
			// Add basic auth
			if tt.inputBasicUser != "" {
				req.SetBasicAuth(tt.inputBasicUser, tt.inputBasicPassword)
			}
			got.ServeHTTP(w, req)
			if tt.expectedCode != w.Code {
				t.Errorf("Integration test on GenerateRouter() status code = %v, expected status code %v", w.Code, tt.expectedCode)
				return
			}

			if tt.expectedBody != "" {
				body := w.Body.String()
				if tt.expectedBody != body {
					t.Errorf("Integration test on GenerateRouter() body = \"%v\", expected body \"%v\"", body, tt.expectedBody)
					return
				}
			}

			if tt.notExpectedBody != "" {
				body := w.Body.String()
				if tt.notExpectedBody == body {
					t.Errorf("Integration test on GenerateRouter() body = \"%v\", not expected body \"%v\"", body, tt.notExpectedBody)
					return
				}
			}
		})
	}
}

func setupFakeS3(accessKey, secretAccessKey, region, bucket string) (*httptest.Server, error) {
	backend := s3mem.New()
	faker := gofakes3.New(backend)
	ts := httptest.NewServer(faker.Server())

	// configure S3 client
	s3Config := &aws.Config{
		Credentials:      credentials.NewStaticCredentials(accessKey, secretAccessKey, ""),
		Endpoint:         aws.String(ts.URL),
		Region:           aws.String(region),
		DisableSSL:       aws.Bool(true),
		S3ForcePathStyle: aws.Bool(true),
	}
	newSession := session.New(s3Config)

	s3Client := s3.New(newSession)
	cparams := &s3.CreateBucketInput{
		Bucket: aws.String(bucket),
	}

	// Create a new bucket using the CreateBucket call.
	_, err := s3Client.CreateBucket(cparams)
	if err != nil {
		return nil, err
	}

	files := map[string]string{
		"folder1/test.txt":   "Hello folder1!",
		"folder1/index.html": "<!DOCTYPE html><html><body><h1>Hello folder1!</h1></body></html>",
		"folder2/index.html": "<!DOCTYPE html><html><body><h1>Hello folder2!</h1></body></html>",
	}

	// Upload files
	for k, v := range files {
		_, err = s3Client.PutObject(&s3.PutObjectInput{
			Body:   strings.NewReader(v),
			Bucket: aws.String(bucket),
			Key:    aws.String(k),
		})
		if err != nil {
			return nil, err
		}
	}

	return ts, nil
}
