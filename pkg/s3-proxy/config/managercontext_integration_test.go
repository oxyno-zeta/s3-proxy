//go:build integration

package config

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/stretchr/testify/assert"
)

var defaultTemplateCfg = &TemplateConfig{
	Helpers: []string{"templates/_helpers.tpl"},
	FolderList: &TemplateConfigItem{
		Path: "templates/folder-list.tpl",
		Headers: map[string]string{
			"Content-Type": "{{ template \"main.headers.contentType\" . }}",
		},
		Status: "200",
	},
	TargetList: &TemplateConfigItem{
		Path: "templates/target-list.tpl",
		Headers: map[string]string{
			"Content-Type": "{{ template \"main.headers.contentType\" . }}",
		},
		Status: "200",
	},
	NotFoundError: &TemplateConfigItem{
		Path: "templates/not-found-error.tpl",
		Headers: map[string]string{
			"Content-Type": "{{ template \"main.headers.contentType\" . }}",
		},
		Status: "404",
	},
	InternalServerError: &TemplateConfigItem{
		Path: "templates/internal-server-error.tpl",
		Headers: map[string]string{
			"Content-Type": "{{ template \"main.headers.contentType\" . }}",
		},
		Status: "500",
	},
	UnauthorizedError: &TemplateConfigItem{
		Path: "templates/unauthorized-error.tpl",
		Headers: map[string]string{
			"Content-Type": "{{ template \"main.headers.contentType\" . }}",
		},
		Status: "401",
	},
	ForbiddenError: &TemplateConfigItem{
		Path: "templates/forbidden-error.tpl",
		Headers: map[string]string{
			"Content-Type": "{{ template \"main.headers.contentType\" . }}",
		},
		Status: "403",
	},
	BadRequestError: &TemplateConfigItem{
		Path: "templates/bad-request-error.tpl",
		Headers: map[string]string{
			"Content-Type": "{{ template \"main.headers.contentType\" . }}",
		},
		Status: "400",
	},
	Put: &TemplateConfigItem{
		Path:    "templates/put.tpl",
		Headers: map[string]string{},
		Status:  "204",
	},
	Delete: &TemplateConfigItem{
		Path:    "templates/delete.tpl",
		Headers: map[string]string{},
		Status:  "204",
	},
}

func Test_managercontext_Load(t *testing.T) {
	svrCompressCfg := &ServerCompressConfig{
		Enabled: &DefaultServerCompressEnabled,
		Level:   DefaultServerCompressLevel,
		Types:   DefaultServerCompressTypes,
	}
	svrTimeoutsCfg := &ServerTimeoutsConfig{
		ReadHeaderTimeout: DefaultServerTimeoutsReadHeaderTimeout,
	}
	falseValue := false

	secret1Filename := path.Join(os.TempDir(), "secret1")
	secretWithNewLineFilename := path.Join(os.TempDir(), "secret-with-new-line")

	tests := []struct {
		name           string
		configs        map[string]string
		envVariables   map[string]string
		secretFiles    map[string]string
		expectedResult *Config
		wantErr        bool
	}{
		{
			name: "Configuration not found",
			configs: map[string]string{
				"config": "",
			},
			wantErr: true,
		},
		{
			name: "Not a yaml",
			configs: map[string]string{
				"config.yaml": "notayaml",
			},
			wantErr: true,
		},
		{
			name: "Empty configuration file",
			configs: map[string]string{
				"config.yaml": "",
			},
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: nil,
			},
		},
		{
			name:    "no configuration file",
			configs: map[string]string{},
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: nil,
			},
		},
		{
			name: "Test all default values with minimal config",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
`,
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{},
					},
				},
			},
		},
		{
			name: "Should merge target across multiple files",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
`,
				"config2.yaml": `
targets:
 test:
  bucket:
    name: bucket1
`,
				"config3.yaml": `
targets:
 test:
  bucket:
    region: us-east-1
`,
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{},
					},
				},
			},
		},
		{
			name: "Should merge multiple targets across multiple files",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
 test2:
  mount:
    path: /test2/
`,
				"config2.yaml": `
targets:
 test:
  bucket:
    name: bucket1
 test2:
  bucket:
    name: bucket2
    region: us-east-1
`,
				"config3.yaml": `
targets:
 test:
  bucket:
    region: us-east-1
`,
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{},
					},
					"test2": {
						Name: "test2",
						Mount: &MountConfig{
							Path: []string{"/test2/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket2",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{},
					},
				},
			},
		},
		{
			name: "Test disable server compress",
			configs: map[string]string{
				"config.yaml": `
server:
  compress:
    enabled: false
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
`,
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port: 8080,
					Compress: &ServerCompressConfig{
						Enabled: &falseValue,
						Level:   DefaultServerCompressLevel,
						Types:   DefaultServerCompressTypes,
					},
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{},
					},
				},
			},
		},
		{
			name: "override template should have default headers templates when not override",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
  templates:
    folderList:
      path: fake1.tpl
      headers:
        override: value
    notFoundError:
      path: fake2.tpl
    internalServerError:
      path: fake2.tpl
    forbiddenError:
      path: fake2.tpl
    unauthorizedError:
      path: fake2.tpl
    badRequestError:
      path: fake2.tpl
    put:
      path: fake2.tpl
    delete:
      path: fake2.tpl
  bucket:
    name: bucket1
    region: us-east-1
`,
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{
							FolderList: &TargetTemplateConfigItem{
								Path: "fake1.tpl",
								Headers: map[string]string{
									"override": "value",
								},
							},
							NotFoundError: &TargetTemplateConfigItem{
								Path: "fake2.tpl",
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
							},
							InternalServerError: &TargetTemplateConfigItem{
								Path: "fake2.tpl",
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
							},
							UnauthorizedError: &TargetTemplateConfigItem{
								Path: "fake2.tpl",
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
							},
							ForbiddenError: &TargetTemplateConfigItem{
								Path: "fake2.tpl",
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
							},
							BadRequestError: &TargetTemplateConfigItem{
								Path: "fake2.tpl",
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
							},
							Put: &TargetTemplateConfigItem{
								Path:    "fake2.tpl",
								Headers: map[string]string{},
							},
							Delete: &TargetTemplateConfigItem{
								Path:    "fake2.tpl",
								Headers: map[string]string{},
							},
						},
					},
				},
			},
		},
		{
			name: "override template status code should work in general and target override",
			configs: map[string]string{
				"config.yaml": `
templates:
  folderList:
    status: "400"
targets:
 test:
  mount:
    path: /test/
  templates:
    folderList:
      path: fake1.tpl
      headers:
        override: value
      status: "499"
    notFoundError:
      path: fake2.tpl
    internalServerError:
      path: fake2.tpl
    forbiddenError:
      path: fake2.tpl
    unauthorizedError:
      path: fake2.tpl
    badRequestError:
      path: fake2.tpl
  bucket:
    name: bucket1
    region: us-east-1
`,
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: &TemplateConfig{
					Helpers: []string{"templates/_helpers.tpl"},
					FolderList: &TemplateConfigItem{
						Path: "templates/folder-list.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "400",
					},
					TargetList: &TemplateConfigItem{
						Path: "templates/target-list.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "200",
					},
					NotFoundError: &TemplateConfigItem{
						Path: "templates/not-found-error.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "404",
					},
					InternalServerError: &TemplateConfigItem{
						Path: "templates/internal-server-error.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "500",
					},
					UnauthorizedError: &TemplateConfigItem{
						Path: "templates/unauthorized-error.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "401",
					},
					ForbiddenError: &TemplateConfigItem{
						Path: "templates/forbidden-error.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "403",
					},
					BadRequestError: &TemplateConfigItem{
						Path: "templates/bad-request-error.tpl",
						Headers: map[string]string{
							"Content-Type": "{{ template \"main.headers.contentType\" . }}",
						},
						Status: "400",
					},
					Put: &TemplateConfigItem{
						Path:    "templates/put.tpl",
						Headers: map[string]string{},
						Status:  "204",
					},
					Delete: &TemplateConfigItem{
						Path:    "templates/delete.tpl",
						Headers: map[string]string{},
						Status:  "204",
					},
				},
				Tracing: &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{
							FolderList: &TargetTemplateConfigItem{
								Path: "fake1.tpl",
								Headers: map[string]string{
									"override": "value",
								},
								Status: "499",
							},
							NotFoundError: &TargetTemplateConfigItem{
								Path: "fake2.tpl",
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
							},
							InternalServerError: &TargetTemplateConfigItem{
								Path: "fake2.tpl",
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
							},
							UnauthorizedError: &TargetTemplateConfigItem{
								Path: "fake2.tpl",
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
							},
							ForbiddenError: &TargetTemplateConfigItem{
								Path: "fake2.tpl",
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
							},
							BadRequestError: &TargetTemplateConfigItem{
								Path: "fake2.tpl",
								Headers: map[string]string{
									"Content-Type": "{{ template \"main.headers.contentType\" . }}",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Test server compress configurations error (level)",
			configs: map[string]string{
				"config.yaml": `
server:
  compress:
    enabled: true
	level: 0
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
`,
			},
			wantErr: true,
		},
		{
			name: "Test server compress configurations error (types)",
			configs: map[string]string{
				"config.yaml": `
server:
  compress:
    enabled: true
	types: []
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
`,
			},
			wantErr: true,
		},
		{
			name: "Test secrets from environment variable",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        env: ENV1
      secretKey:
        env: ENV2`,
			},
			envVariables: map[string]string{
				"ENV1": "VALUE1",
				"ENV2": "VALUE2",
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
							Credentials: &BucketCredentialConfig{
								AccessKey: &CredentialConfig{
									Env:   "ENV1",
									Value: "VALUE1",
								},
								SecretKey: &CredentialConfig{
									Env:   "ENV2",
									Value: "VALUE2",
								},
							},
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{},
					},
				},
			},
		},
		{
			name: "Test secrets from environment variable with empty environment variable",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        env: ENV1
      secretKey:
        env: ENV2`,
			},
			envVariables: map[string]string{
				"ENV1": "VALUE1",
			},
			wantErr: true,
		},
		{
			name: "Test secrets from a not found file",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + secret1Filename + `
      secretKey:
        value: VALUE2`,
			},
			wantErr: true,
		},
		{
			name: "Test secrets from a file and direct value",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + secret1Filename + `
      secretKey:
        value: VALUE2`,
			},
			secretFiles: map[string]string{
				secret1Filename: "VALUE1",
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
							Credentials: &BucketCredentialConfig{
								AccessKey: &CredentialConfig{
									Path:  secret1Filename,
									Value: "VALUE1",
								},
								SecretKey: &CredentialConfig{
									Value: "VALUE2",
								},
							},
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{},
					},
				},
			},
		},
		{
			name: "Test secrets from a file and direct value with new lines cleaned",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + secretWithNewLineFilename + `
      secretKey:
        value: VALUE2`,
			},
			secretFiles: map[string]string{
				secretWithNewLineFilename: `
VALUE1
`,
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
							Credentials: &BucketCredentialConfig{
								AccessKey: &CredentialConfig{
									Path:  secretWithNewLineFilename,
									Value: "VALUE1",
								},
								SecretKey: &CredentialConfig{
									Value: "VALUE2",
								},
							},
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{},
					},
				},
			},
		},
		{
			name: "should fail when target templates configuration are invalid",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
  templates:
    notFoundError:
      inBucket: false
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        value: ENV1
      secretKey:
        value: ENV2`,
			},
			wantErr: true,
		},
		{
			name: "should load complete configuration with target custom templates",
			configs: map[string]string{
				"config.yaml": `
targets:
 test:
  mount:
    path: /test/
  templates:
    notFoundError:
      inBucket: false
      path: "/fake"
      headers:
        h1: "{{ .Fake }}"
    internalServerError:
      inBucket: true
      path: "/fake2"
      headers:
        h2: "{{ .Fake }}"
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        value: VALUE1
      secretKey:
        value: VALUE2`,
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
							Credentials: &BucketCredentialConfig{
								AccessKey: &CredentialConfig{
									Value: "VALUE1",
								},
								SecretKey: &CredentialConfig{
									Value: "VALUE2",
								},
							},
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{
							NotFoundError: &TargetTemplateConfigItem{
								InBucket: false,
								Path:     "/fake",
								Headers:  map[string]string{"h1": "{{ .Fake }}"},
							},
							InternalServerError: &TargetTemplateConfigItem{
								InBucket: true,
								Path:     "/fake2",
								Headers:  map[string]string{"h2": "{{ .Fake }}"},
							},
						},
					},
				},
			},
		},
		{
			name: "Test with multiple configuration files",
			configs: map[string]string{
				"log.yaml": `
log:
  level: error
`,
				"targets.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        value: value1
      secretKey:
        value: value2`,
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "error",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
							Credentials: &BucketCredentialConfig{
								AccessKey: &CredentialConfig{
									Value: "value1",
								},
								SecretKey: &CredentialConfig{
									Value: "value2",
								},
							},
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{},
					},
				},
			},
		},
		{
			name: "Test key rewrite list",
			configs: map[string]string{
				"cfg.yaml": `
log:
  level: error
targets:
 test:
  mount:
    path: /test/
  keyRewriteList:
    - source: ^/(?P<one>\w+)/file.html$
      target: /$one/fake/$one/file.html
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        value: value1
      secretKey:
        value: value2`,
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "error",
					Format: "json",
				},
				Server: &ServerConfig{
					Port:     8080,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				InternalServer: &ServerConfig{
					Port:     9090,
					Compress: svrCompressCfg,
					Timeouts: svrTimeoutsCfg,
				},
				Templates: defaultTemplateCfg,
				Tracing:   &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: map[string]*TargetConfig{
					"test": {
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						KeyRewriteList: []*TargetKeyRewriteConfig{{
							Source:      `^/(?P<one>\w+)/file.html$`,
							SourceRegex: regexp.MustCompile(`^/(?P<one>\w+)/file.html$`),
							Target:      "/$one/fake/$one/file.html",
							TargetType:  RegexTargetKeyRewriteTargetType,
						}},
						Bucket: &BucketConfig{
							Name:          "bucket1",
							Region:        "us-east-1",
							S3ListMaxKeys: 1000,
							Credentials: &BucketCredentialConfig{
								AccessKey: &CredentialConfig{
									Value: "value1",
								},
								SecretKey: &CredentialConfig{
									Value: "value2",
								},
							},
						},
						Actions: &ActionsConfig{
							GET: &GetActionConfig{Enabled: true},
						},
						Templates: &TargetTemplateConfig{},
					},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "s3-proxy-config")
			if err != nil {
				t.Error(err)
				return
			}

			defer os.RemoveAll(dir) // clean up
			for k, v := range tt.configs {
				tmpfn := filepath.Join(dir, k)
				err = ioutil.WriteFile(tmpfn, []byte(v), 0666)
				if err != nil {
					t.Error(err)
					return
				}
			}

			// Set environment variables
			for k, v := range tt.envVariables {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			// Create secret files
			for k, v := range tt.secretFiles {
				dirToCr := filepath.Dir(k)
				err = os.MkdirAll(dirToCr, 0666)
				if err != nil {
					t.Error(err)
					return
				}
				err = ioutil.WriteFile(k, []byte(v), 0666)
				if err != nil {
					t.Error(err)
					return
				}
				defer os.Remove(k)
			}

			// Change var for main configuration file
			mainConfigFolderPath = dir

			ctx := &managercontext{
				logger: log.NewLogger(),
			}

			// Load config
			err = ctx.Load()

			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Get configuration
			res := ctx.GetConfig()

			assert.Equal(t, tt.expectedResult, res)
		})
	}
}

func Test_Load_reload_config(t *testing.T) {
	svrCompressCfg := &ServerCompressConfig{
		Enabled: &DefaultServerCompressEnabled,
		Level:   DefaultServerCompressLevel,
		Types:   DefaultServerCompressTypes,
	}
	svrTimeoutsCfg := &ServerTimeoutsConfig{
		ReadHeaderTimeout: DefaultServerTimeoutsReadHeaderTimeout,
	}

	// Channel for wait watch
	waitCh := make(chan bool)

	dir, err := ioutil.TempDir("", "s3-proxy-config-reload")
	assert.NoError(t, err)

	secret1Filename := path.Join(os.TempDir(), "secret1")

	configs := map[string]string{
		"log.yaml": `
log:
  level: error
`,
		"targets.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + secret1Filename + `
      secretKey:
        value: value2`,
	}

	defer os.RemoveAll(dir) // clean up
	for k, v := range configs {
		tmpfn := filepath.Join(dir, k)
		err = ioutil.WriteFile(tmpfn, []byte(v), 0666)
		assert.NoError(t, err)
	}

	secretFiles := map[string]string{
		secret1Filename: "VALUE1",
	}
	// Create secret files
	for k, v := range secretFiles {
		dirToCr := filepath.Dir(k)
		err = os.MkdirAll(dirToCr, 0666)
		assert.NoError(t, err)
		err = ioutil.WriteFile(k, []byte(v), 0666)
		assert.NoError(t, err)
		defer os.Remove(k)
	}

	// Change var for main configuration file
	mainConfigFolderPath = dir

	ctx := &managercontext{
		logger: log.NewLogger(),
	}

	ctx.AddOnChangeHook(func() {
		waitCh <- true
	})

	// Load config
	err = ctx.Load()
	assert.NoError(t, err)
	// Get configuration
	res := ctx.GetConfig()

	assert.Equal(t, &Config{
		Log: &LogConfig{
			Level:  "error",
			Format: "json",
		},
		Server: &ServerConfig{
			Port:     8080,
			Compress: svrCompressCfg,
			Timeouts: svrTimeoutsCfg,
		},
		InternalServer: &ServerConfig{
			Port:     9090,
			Compress: svrCompressCfg,
			Timeouts: svrTimeoutsCfg,
		},
		Templates: defaultTemplateCfg,
		Tracing:   &TracingConfig{Enabled: false},
		ListTargets: &ListTargetsConfig{
			Enabled: false,
		},
		Targets: map[string]*TargetConfig{
			"test": {
				Name: "test",
				Mount: &MountConfig{
					Path: []string{"/test/"},
				},
				Bucket: &BucketConfig{
					Name:          "bucket1",
					Region:        "us-east-1",
					S3ListMaxKeys: 1000,
					Credentials: &BucketCredentialConfig{
						AccessKey: &CredentialConfig{
							Value: "VALUE1",
							Path:  secret1Filename,
						},
						SecretKey: &CredentialConfig{
							Value: "value2",
						},
					},
				},
				Actions: &ActionsConfig{
					GET: &GetActionConfig{Enabled: true},
				},
				Templates: &TargetTemplateConfig{},
			},
		},
	}, res)

	configs = map[string]string{
		"log.yaml": `
log:
  level: debug
  format: text
`,
	}

	defer os.RemoveAll(dir) // clean up
	for k, v := range configs {
		tmpfn := filepath.Join(dir, k)
		err = ioutil.WriteFile(tmpfn, []byte(v), 0666)
		assert.NoError(t, err)
	}

	select {
	case <-waitCh:
		// Get configuration
		res = ctx.GetConfig()

		assert.Equal(t, &Config{
			Log: &LogConfig{
				Level:  "debug",
				Format: "text",
			},
			Server: &ServerConfig{
				Port:     8080,
				Compress: svrCompressCfg,
				Timeouts: svrTimeoutsCfg,
			},
			InternalServer: &ServerConfig{
				Port:     9090,
				Compress: svrCompressCfg,
				Timeouts: svrTimeoutsCfg,
			},
			Templates: defaultTemplateCfg,
			Tracing:   &TracingConfig{Enabled: false},
			ListTargets: &ListTargetsConfig{
				Enabled: false,
			},
			Targets: map[string]*TargetConfig{
				"test": {
					Name: "test",
					Mount: &MountConfig{
						Path: []string{"/test/"},
					},
					Bucket: &BucketConfig{
						Name:          "bucket1",
						Region:        "us-east-1",
						S3ListMaxKeys: 1000,
						Credentials: &BucketCredentialConfig{
							AccessKey: &CredentialConfig{
								Value: "VALUE1",
								Path:  secret1Filename,
							},
							SecretKey: &CredentialConfig{
								Value: "value2",
							},
						},
					},
					Actions: &ActionsConfig{
						GET: &GetActionConfig{Enabled: true},
					},
					Templates: &TargetTemplateConfig{},
				},
			},
		}, res)
		return
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "shouldn't call this")
	}
}

func Test_Load_reload_secret(t *testing.T) {
	svrCompressCfg := &ServerCompressConfig{
		Enabled: &DefaultServerCompressEnabled,
		Level:   DefaultServerCompressLevel,
		Types:   DefaultServerCompressTypes,
	}
	svrTimeoutsCfg := &ServerTimeoutsConfig{
		ReadHeaderTimeout: DefaultServerTimeoutsReadHeaderTimeout,
	}

	// Channel for wait watch
	waitCh := make(chan bool)

	dir, err := ioutil.TempDir("", "s3-proxy-config-reload-secret")
	assert.NoError(t, err)

	secret1Filename := path.Join(os.TempDir(), "secret1")

	configs := map[string]string{
		"log.yaml": `
log:
  level: error
  format: text
`,
		"targets.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + secret1Filename + `
      secretKey:
        value: value2`,
	}

	defer os.RemoveAll(dir) // clean up
	for k, v := range configs {
		tmpfn := filepath.Join(dir, k)
		err = ioutil.WriteFile(tmpfn, []byte(v), 0666)
		assert.NoError(t, err)
	}

	secretFiles := map[string]string{
		secret1Filename: "VALUE1",
	}
	// Create secret files
	for k, v := range secretFiles {
		dirToCr := filepath.Dir(k)
		err = os.MkdirAll(dirToCr, 0666)
		assert.NoError(t, err)
		err = ioutil.WriteFile(k, []byte(v), 0666)
		assert.NoError(t, err)
		defer os.Remove(k)
	}

	// Change var for main configuration file
	mainConfigFolderPath = dir

	ctx := &managercontext{
		logger: log.NewLogger(),
	}

	ctx.AddOnChangeHook(func() {
		waitCh <- true
	})

	// Load config
	err = ctx.Load()
	assert.NoError(t, err)
	// Get configuration
	res := ctx.GetConfig()

	assert.Equal(t, &Config{
		Log: &LogConfig{
			Level:  "error",
			Format: "text",
		},
		Server: &ServerConfig{
			Port:     8080,
			Compress: svrCompressCfg,
			Timeouts: svrTimeoutsCfg,
		},
		InternalServer: &ServerConfig{
			Port:     9090,
			Compress: svrCompressCfg,
			Timeouts: svrTimeoutsCfg,
		},
		Templates: defaultTemplateCfg,
		Tracing:   &TracingConfig{Enabled: false},
		ListTargets: &ListTargetsConfig{
			Enabled: false,
		},
		Targets: map[string]*TargetConfig{
			"test": {
				Name: "test",
				Mount: &MountConfig{
					Path: []string{"/test/"},
				},
				Bucket: &BucketConfig{
					Name:          "bucket1",
					Region:        "us-east-1",
					S3ListMaxKeys: 1000,
					Credentials: &BucketCredentialConfig{
						AccessKey: &CredentialConfig{
							Value: "VALUE1",
							Path:  secret1Filename,
						},
						SecretKey: &CredentialConfig{
							Value: "value2",
						},
					},
				},
				Actions: &ActionsConfig{
					GET: &GetActionConfig{Enabled: true},
				},
				Templates: &TargetTemplateConfig{},
			},
		},
	}, res)

	secretFiles = make(map[string]string)
	secretFiles[secret1Filename] = "SECRET1"
	// Create secret files
	for k, v := range secretFiles {
		dirToCr := filepath.Dir(k)
		err = os.MkdirAll(dirToCr, 0666)
		assert.NoError(t, err)
		err = ioutil.WriteFile(k, []byte(v), 0666)
		assert.NoError(t, err)
		defer os.Remove(k)
	}

	select {
	case <-waitCh:
		// Get configuration
		res = ctx.GetConfig()

		assert.Equal(t, &Config{
			Log: &LogConfig{
				Level:  "error",
				Format: "text",
			},
			Server: &ServerConfig{
				Port:     8080,
				Compress: svrCompressCfg,
				Timeouts: svrTimeoutsCfg,
			},
			InternalServer: &ServerConfig{
				Port:     9090,
				Compress: svrCompressCfg,
				Timeouts: svrTimeoutsCfg,
			},
			Templates: defaultTemplateCfg,
			Tracing:   &TracingConfig{Enabled: false},
			ListTargets: &ListTargetsConfig{
				Enabled: false,
			},
			Targets: map[string]*TargetConfig{
				"test": {
					Name: "test",
					Mount: &MountConfig{
						Path: []string{"/test/"},
					},
					Bucket: &BucketConfig{
						Name:          "bucket1",
						Region:        "us-east-1",
						S3ListMaxKeys: 1000,
						Credentials: &BucketCredentialConfig{
							AccessKey: &CredentialConfig{
								Value: "SECRET1",
								Path:  secret1Filename,
							},
							SecretKey: &CredentialConfig{
								Value: "value2",
							},
						},
					},
					Actions: &ActionsConfig{
						GET: &GetActionConfig{Enabled: true},
					},
					Templates: &TargetTemplateConfig{},
				},
			},
		}, res)
		return
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "shouldn't call this")
	}
}

func Test_Load_reload_config_with_wrong_config(t *testing.T) {
	svrCompressCfg := &ServerCompressConfig{
		Enabled: &DefaultServerCompressEnabled,
		Level:   DefaultServerCompressLevel,
		Types:   DefaultServerCompressTypes,
	}
	svrTimeoutsCfg := &ServerTimeoutsConfig{
		ReadHeaderTimeout: DefaultServerTimeoutsReadHeaderTimeout,
	}

	// Channel for wait watch
	waitCh := make(chan bool)

	dir, err := ioutil.TempDir("", "s3-proxy-config-reload-wrong-config")
	assert.NoError(t, err)

	secret1Filename := path.Join(os.TempDir(), "secret1")

	configs := map[string]string{
		"log.yaml": `
log:
  level: error
  format: text
`,
		"targets.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + secret1Filename + `
      secretKey:
        value: value2`,
	}

	defer os.RemoveAll(dir) // clean up
	for k, v := range configs {
		tmpfn := filepath.Join(dir, k)
		err = ioutil.WriteFile(tmpfn, []byte(v), 0666)
		assert.NoError(t, err)
	}

	secretFiles := map[string]string{
		secret1Filename: "VALUE1",
	}
	// Create secret files
	for k, v := range secretFiles {
		dirToCr := filepath.Dir(k)
		err = os.MkdirAll(dirToCr, 0666)
		assert.NoError(t, err)
		err = ioutil.WriteFile(k, []byte(v), 0666)
		assert.NoError(t, err)
		defer os.Remove(k)
	}

	// Change var for main configuration file
	mainConfigFolderPath = dir

	ctx := &managercontext{
		logger: log.NewLogger(),
	}

	ctx.AddOnChangeHook(func() {
		waitCh <- true
	})

	// Load config
	err = ctx.Load()
	assert.NoError(t, err)
	// Get configuration
	res := ctx.GetConfig()

	assert.Equal(t, &Config{
		Log: &LogConfig{
			Level:  "error",
			Format: "text",
		},
		Server: &ServerConfig{
			Port:     8080,
			Compress: svrCompressCfg,
			Timeouts: svrTimeoutsCfg,
		},
		InternalServer: &ServerConfig{
			Port:     9090,
			Compress: svrCompressCfg,
			Timeouts: svrTimeoutsCfg,
		},
		Templates: defaultTemplateCfg,
		Tracing:   &TracingConfig{Enabled: false},
		ListTargets: &ListTargetsConfig{
			Enabled: false,
		},
		Targets: map[string]*TargetConfig{
			"test": {
				Name: "test",
				Mount: &MountConfig{
					Path: []string{"/test/"},
				},
				Bucket: &BucketConfig{
					Name:          "bucket1",
					Region:        "us-east-1",
					S3ListMaxKeys: 1000,
					Credentials: &BucketCredentialConfig{
						AccessKey: &CredentialConfig{
							Value: "VALUE1",
							Path:  path.Join(os.TempDir(), "/secret1"),
						},
						SecretKey: &CredentialConfig{
							Value: "value2",
						},
					},
				},
				Actions: &ActionsConfig{
					GET: &GetActionConfig{Enabled: true},
				},
				Templates: &TargetTemplateConfig{},
			},
		},
	}, res)

	configs = make(map[string]string)
	configs["log.yaml"] = `
configuration with error
`

	defer os.RemoveAll(dir) // clean up
	for k, v := range configs {
		tmpfn := filepath.Join(dir, k)
		err = ioutil.WriteFile(tmpfn, []byte(v), 0666)
		assert.NoError(t, err)
	}

	select {
	case <-waitCh:
		assert.FailNow(t, "shouldn't call this")
		return
	case <-time.After(5 * time.Second):
		// Get configuration
		res = ctx.GetConfig()

		assert.Equal(t, &Config{
			Log: &LogConfig{
				Level:  "error",
				Format: "text",
			},
			Server: &ServerConfig{
				Port:     8080,
				Compress: svrCompressCfg,
				Timeouts: svrTimeoutsCfg,
			},
			InternalServer: &ServerConfig{
				Port:     9090,
				Compress: svrCompressCfg,
				Timeouts: svrTimeoutsCfg,
			},
			Templates: defaultTemplateCfg,
			Tracing:   &TracingConfig{Enabled: false},
			ListTargets: &ListTargetsConfig{
				Enabled: false,
			},
			Targets: map[string]*TargetConfig{
				"test": {
					Name: "test",
					Mount: &MountConfig{
						Path: []string{"/test/"},
					},
					Bucket: &BucketConfig{
						Name:          "bucket1",
						Region:        "us-east-1",
						S3ListMaxKeys: 1000,
						Credentials: &BucketCredentialConfig{
							AccessKey: &CredentialConfig{
								Value: "VALUE1",
								Path:  secret1Filename,
							},
							SecretKey: &CredentialConfig{
								Value: "value2",
							},
						},
					},
					Actions: &ActionsConfig{
						GET: &GetActionConfig{Enabled: true},
					},
					Templates: &TargetTemplateConfig{},
				},
			},
		}, res)
	}
}

func Test_Load_reload_config_map_structure(t *testing.T) {
	svrCompressCfg := &ServerCompressConfig{
		Enabled: &DefaultServerCompressEnabled,
		Level:   DefaultServerCompressLevel,
		Types:   DefaultServerCompressTypes,
	}
	svrTimeoutsCfg := &ServerTimeoutsConfig{
		ReadHeaderTimeout: DefaultServerTimeoutsReadHeaderTimeout,
	}

	// Channel for wait watch
	waitCh := make(chan bool)

	dir, err := ioutil.TempDir("", "s3-proxy-config-reload-map-structure")
	assert.NoError(t, err)

	secret1Filename := path.Join(os.TempDir(), "secret1")

	configs := map[string]string{
		"log.yaml": `
log:
  level: error
`,
		"providers.yaml": `
authProviders:
  basic:
    provider1:
      realm: prov1
    provider2:
      realm: prov2
`,
		"targets.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + secret1Filename + `
      secretKey:
        value: value2`,
	}

	defer os.RemoveAll(dir) // clean up
	for k, v := range configs {
		tmpfn := filepath.Join(dir, k)
		err = ioutil.WriteFile(tmpfn, []byte(v), 0666)
		assert.NoError(t, err)
	}

	secretFiles := map[string]string{
		secret1Filename: "VALUE1",
	}
	// Create secret files
	for k, v := range secretFiles {
		dirToCr := filepath.Dir(k)
		err = os.MkdirAll(dirToCr, 0666)
		assert.NoError(t, err)
		err = ioutil.WriteFile(k, []byte(v), 0666)
		assert.NoError(t, err)
		defer os.Remove(k)
	}

	// Change var for main configuration file
	mainConfigFolderPath = dir

	ctx := &managercontext{
		logger: log.NewLogger(),
	}

	ctx.AddOnChangeHook(func() {
		waitCh <- true
	})

	// Load config
	err = ctx.Load()
	assert.NoError(t, err)
	// Get configuration
	res := ctx.GetConfig()

	assert.Equal(t, &Config{
		Log: &LogConfig{
			Level:  "error",
			Format: "json",
		},
		Server: &ServerConfig{
			Port:     8080,
			Compress: svrCompressCfg,
			Timeouts: svrTimeoutsCfg,
		},
		InternalServer: &ServerConfig{
			Port:     9090,
			Compress: svrCompressCfg,
			Timeouts: svrTimeoutsCfg,
		},
		Templates: defaultTemplateCfg,
		AuthProviders: &AuthProviderConfig{
			Basic: map[string]*BasicAuthConfig{
				"provider1": {Realm: "prov1"},
				"provider2": {Realm: "prov2"},
			},
		},
		Tracing: &TracingConfig{Enabled: false},
		ListTargets: &ListTargetsConfig{
			Enabled: false,
		},
		Targets: map[string]*TargetConfig{
			"test": {
				Name: "test",
				Mount: &MountConfig{
					Path: []string{"/test/"},
				},
				Bucket: &BucketConfig{
					Name:          "bucket1",
					Region:        "us-east-1",
					S3ListMaxKeys: 1000,
					Credentials: &BucketCredentialConfig{
						AccessKey: &CredentialConfig{
							Value: "VALUE1",
							Path:  secret1Filename,
						},
						SecretKey: &CredentialConfig{
							Value: "value2",
						},
					},
				},
				Actions: &ActionsConfig{
					GET: &GetActionConfig{Enabled: true},
				},
				Templates: &TargetTemplateConfig{},
			},
		},
	}, res)

	configs = map[string]string{
		"providers.yaml": `
authProviders:
  basic:
    provider1:
      realm: prov1
`,
	}

	defer os.RemoveAll(dir) // clean up
	for k, v := range configs {
		tmpfn := filepath.Join(dir, k)
		err = ioutil.WriteFile(tmpfn, []byte(v), 0666)
		assert.NoError(t, err)
	}

	select {
	case <-waitCh:
		// Get configuration
		res = ctx.GetConfig()

		assert.Equal(t, &Config{
			Log: &LogConfig{
				Level:  "error",
				Format: "json",
			},
			Server: &ServerConfig{
				Port:     8080,
				Compress: svrCompressCfg,
				Timeouts: svrTimeoutsCfg,
			},
			InternalServer: &ServerConfig{
				Port:     9090,
				Compress: svrCompressCfg,
				Timeouts: svrTimeoutsCfg,
			},
			Templates: defaultTemplateCfg,
			Tracing:   &TracingConfig{Enabled: false},
			ListTargets: &ListTargetsConfig{
				Enabled: false,
			},
			AuthProviders: &AuthProviderConfig{
				Basic: map[string]*BasicAuthConfig{
					"provider1": {Realm: "prov1"},
				},
			},
			Targets: map[string]*TargetConfig{
				"test": {
					Name: "test",
					Mount: &MountConfig{
						Path: []string{"/test/"},
					},
					Bucket: &BucketConfig{
						Name:          "bucket1",
						Region:        "us-east-1",
						S3ListMaxKeys: 1000,
						Credentials: &BucketCredentialConfig{
							AccessKey: &CredentialConfig{
								Value: "VALUE1",
								Path:  secret1Filename,
							},
							SecretKey: &CredentialConfig{
								Value: "value2",
							},
						},
					},
					Actions: &ActionsConfig{
						GET: &GetActionConfig{Enabled: true},
					},
					Templates: &TargetTemplateConfig{},
				},
			},
		}, res)
		return
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "shouldn't call this")
	}
}

func Test_Load_reload_config_ignore_hidden_file_and_directory(t *testing.T) {
	svrCompressCfg := &ServerCompressConfig{
		Enabled: &DefaultServerCompressEnabled,
		Level:   DefaultServerCompressLevel,
		Types:   DefaultServerCompressTypes,
	}
	svrTimeoutsCfg := &ServerTimeoutsConfig{
		ReadHeaderTimeout: DefaultServerTimeoutsReadHeaderTimeout,
	}

	// Channel for wait watch
	waitCh := make(chan bool)

	dir, err := ioutil.TempDir("", "s3-proxy-config-reload-ignore")
	assert.NoError(t, err)
	err = os.MkdirAll(path.Join(dir, "dir1"), os.ModePerm)
	assert.NoError(t, err)

	secret1Filename := path.Join(os.TempDir(), "secret1")

	configs := map[string]string{
		"..log.yaml": `
log:
  level: error
`,
		".log2.yaml": `
log:
  format: fake
`,
		"dir1/log2.yaml": `
server:
  port: 8181
`,
		"providers.yaml": `
authProviders:
  basic:
    provider1:
      realm: prov1
    provider2:
      realm: prov2
`,
		"targets.yaml": `
targets:
 test:
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + secret1Filename + `
      secretKey:
        value: value2`,
	}

	defer os.RemoveAll(dir) // clean up
	for k, v := range configs {
		tmpfn := filepath.Join(dir, k)
		err = ioutil.WriteFile(tmpfn, []byte(v), 0666)
		assert.NoError(t, err)
	}

	secretFiles := map[string]string{
		secret1Filename: "VALUE1",
	}
	// Create secret files
	for k, v := range secretFiles {
		dirToCr := filepath.Dir(k)
		err = os.MkdirAll(dirToCr, 0666)
		assert.NoError(t, err)
		err = ioutil.WriteFile(k, []byte(v), 0666)
		assert.NoError(t, err)
		defer os.Remove(k)
	}

	// Change var for main configuration file
	mainConfigFolderPath = dir

	ctx := &managercontext{
		logger: log.NewLogger(),
	}

	ctx.AddOnChangeHook(func() {
		waitCh <- true
	})

	// Load config
	err = ctx.Load()
	assert.NoError(t, err)
	// Get configuration
	res := ctx.GetConfig()

	assert.Equal(t, &Config{
		Log: &LogConfig{
			Level:  "info",
			Format: "json",
		},
		Server: &ServerConfig{
			Port:     8080,
			Compress: svrCompressCfg,
			Timeouts: svrTimeoutsCfg,
		},
		InternalServer: &ServerConfig{
			Port:     9090,
			Compress: svrCompressCfg,
			Timeouts: svrTimeoutsCfg,
		},
		Tracing:   &TracingConfig{Enabled: false},
		Templates: defaultTemplateCfg,
		AuthProviders: &AuthProviderConfig{
			Basic: map[string]*BasicAuthConfig{
				"provider1": {Realm: "prov1"},
				"provider2": {Realm: "prov2"},
			},
		},
		ListTargets: &ListTargetsConfig{
			Enabled: false,
		},
		Targets: map[string]*TargetConfig{
			"test": {
				Name: "test",
				Mount: &MountConfig{
					Path: []string{"/test/"},
				},
				Bucket: &BucketConfig{
					Name:          "bucket1",
					Region:        "us-east-1",
					S3ListMaxKeys: 1000,
					Credentials: &BucketCredentialConfig{
						AccessKey: &CredentialConfig{
							Value: "VALUE1",
							Path:  secret1Filename,
						},
						SecretKey: &CredentialConfig{
							Value: "value2",
						},
					},
				},
				Actions: &ActionsConfig{
					GET: &GetActionConfig{Enabled: true},
				},
				Templates: &TargetTemplateConfig{},
			},
		},
	}, res)
}
