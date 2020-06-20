// +build integration

package config

import (
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/stretchr/testify/assert"
)

func Test_managercontext_Load(t *testing.T) {
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
			name: "Empty",
			configs: map[string]string{
				"config.yaml": "",
			},
			wantErr: true,
		},
		{
			name: "Test all default values with minimal config",
			configs: map[string]string{
				"config.yaml": `
targets:
- name: test
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
				},
				InternalServer: &ServerConfig{
					Port: 9090,
				},
				Templates: &TemplateConfig{
					FolderList:          "templates/folder-list.tpl",
					TargetList:          "templates/target-list.tpl",
					NotFound:            "templates/not-found.tpl",
					InternalServerError: "templates/internal-server-error.tpl",
					Unauthorized:        "templates/unauthorized.tpl",
					Forbidden:           "templates/forbidden.tpl",
					BadRequest:          "templates/bad-request.tpl",
				},
				Tracing: &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: []*TargetConfig{
					{
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:   "bucket1",
							Region: "us-east-1",
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
			name: "Test secrets from environment variable",
			configs: map[string]string{
				"config.yaml": `
targets:
- name: test
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
					Port: 8080,
				},
				InternalServer: &ServerConfig{
					Port: 9090,
				},
				Templates: &TemplateConfig{
					FolderList:          "templates/folder-list.tpl",
					TargetList:          "templates/target-list.tpl",
					NotFound:            "templates/not-found.tpl",
					InternalServerError: "templates/internal-server-error.tpl",
					Unauthorized:        "templates/unauthorized.tpl",
					Forbidden:           "templates/forbidden.tpl",
					BadRequest:          "templates/bad-request.tpl",
				},
				Tracing: &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: []*TargetConfig{
					{
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:   "bucket1",
							Region: "us-east-1",
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
- name: test
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
- name: test
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + os.TempDir() + `/secret1
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
- name: test
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + os.TempDir() + `/secret1
      secretKey:
        value: VALUE2`,
			},
			secretFiles: map[string]string{
				os.TempDir() + "/secret1": "VALUE1",
			},
			wantErr: false,
			expectedResult: &Config{
				Log: &LogConfig{
					Level:  "info",
					Format: "json",
				},
				Server: &ServerConfig{
					Port: 8080,
				},
				InternalServer: &ServerConfig{
					Port: 9090,
				},
				Templates: &TemplateConfig{
					FolderList:          "templates/folder-list.tpl",
					TargetList:          "templates/target-list.tpl",
					NotFound:            "templates/not-found.tpl",
					InternalServerError: "templates/internal-server-error.tpl",
					Unauthorized:        "templates/unauthorized.tpl",
					Forbidden:           "templates/forbidden.tpl",
					BadRequest:          "templates/bad-request.tpl",
				},
				Tracing: &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: []*TargetConfig{
					{
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:   "bucket1",
							Region: "us-east-1",
							Credentials: &BucketCredentialConfig{
								AccessKey: &CredentialConfig{
									Path:  os.TempDir() + "/secret1",
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
- name: test
  mount:
    path: /test/
  templates:
    notFound:
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
- name: test
  mount:
    path: /test/
  templates:
    notFound:
      inBucket: false
      path: "/fake"
    internalServerError:
      inBucket: true
      path: "/fake2"
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
					Port: 8080,
				},
				InternalServer: &ServerConfig{
					Port: 9090,
				},
				Templates: &TemplateConfig{
					FolderList:          "templates/folder-list.tpl",
					TargetList:          "templates/target-list.tpl",
					NotFound:            "templates/not-found.tpl",
					InternalServerError: "templates/internal-server-error.tpl",
					Unauthorized:        "templates/unauthorized.tpl",
					Forbidden:           "templates/forbidden.tpl",
					BadRequest:          "templates/bad-request.tpl",
				},
				Tracing: &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: []*TargetConfig{
					{
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:   "bucket1",
							Region: "us-east-1",
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
							NotFound: &TargetTemplateConfigItem{
								InBucket: false,
								Path:     "/fake",
							},
							InternalServerError: &TargetTemplateConfigItem{
								InBucket: true,
								Path:     "/fake2",
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
- name: test
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
					Port: 8080,
				},
				InternalServer: &ServerConfig{
					Port: 9090,
				},
				Templates: &TemplateConfig{
					FolderList:          "templates/folder-list.tpl",
					TargetList:          "templates/target-list.tpl",
					NotFound:            "templates/not-found.tpl",
					InternalServerError: "templates/internal-server-error.tpl",
					Unauthorized:        "templates/unauthorized.tpl",
					Forbidden:           "templates/forbidden.tpl",
					BadRequest:          "templates/bad-request.tpl",
				},
				Tracing: &TracingConfig{Enabled: false},
				ListTargets: &ListTargetsConfig{
					Enabled: false,
				},
				Targets: []*TargetConfig{
					{
						Name: "test",
						Mount: &MountConfig{
							Path: []string{"/test/"},
						},
						Bucket: &BucketConfig{
							Name:   "bucket1",
							Region: "us-east-1",
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

			if !reflect.DeepEqual(res, tt.expectedResult) {
				t.Errorf("Load() source = %+v, want %+v", res, tt.expectedResult)
			}
		})
	}
}

func Test_Load_reload_config(t *testing.T) {
	// Channel for wait watch
	waitCh := make(chan bool)

	dir, err := ioutil.TempDir("", "s3-proxy-config-reload")
	assert.NoError(t, err)

	configs := map[string]string{
		"log.yaml": `
log:
  level: error
`,
		"targets.yaml": `
targets:
- name: test
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + os.TempDir() + `/secret1
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
		os.TempDir() + "/secret1": "VALUE1",
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
			Port: 8080,
		},
		InternalServer: &ServerConfig{
			Port: 9090,
		},
		Templates: &TemplateConfig{
			FolderList:          "templates/folder-list.tpl",
			TargetList:          "templates/target-list.tpl",
			NotFound:            "templates/not-found.tpl",
			InternalServerError: "templates/internal-server-error.tpl",
			Unauthorized:        "templates/unauthorized.tpl",
			Forbidden:           "templates/forbidden.tpl",
			BadRequest:          "templates/bad-request.tpl",
		},
		Tracing: &TracingConfig{Enabled: false},
		ListTargets: &ListTargetsConfig{
			Enabled: false,
		},
		Targets: []*TargetConfig{
			{
				Name: "test",
				Mount: &MountConfig{
					Path: []string{"/test/"},
				},
				Bucket: &BucketConfig{
					Name:   "bucket1",
					Region: "us-east-1",
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
				Port: 8080,
			},
			InternalServer: &ServerConfig{
				Port: 9090,
			},
			Templates: &TemplateConfig{
				FolderList:          "templates/folder-list.tpl",
				TargetList:          "templates/target-list.tpl",
				NotFound:            "templates/not-found.tpl",
				InternalServerError: "templates/internal-server-error.tpl",
				Unauthorized:        "templates/unauthorized.tpl",
				Forbidden:           "templates/forbidden.tpl",
				BadRequest:          "templates/bad-request.tpl",
			},
			Tracing: &TracingConfig{Enabled: false},
			ListTargets: &ListTargetsConfig{
				Enabled: false,
			},
			Targets: []*TargetConfig{
				{
					Name: "test",
					Mount: &MountConfig{
						Path: []string{"/test/"},
					},
					Bucket: &BucketConfig{
						Name:   "bucket1",
						Region: "us-east-1",
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
		return
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "shouldn't call this")
	}
}

func Test_Load_reload_secret(t *testing.T) {
	// Channel for wait watch
	waitCh := make(chan bool)

	dir, err := ioutil.TempDir("", "s3-proxy-config-reload-secret")
	assert.NoError(t, err)

	configs := map[string]string{
		"log.yaml": `
log:
  level: error
  format: text
`,
		"targets.yaml": `
targets:
- name: test
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + os.TempDir() + `/secret1
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
		os.TempDir() + "/secret1": "VALUE1",
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
			Port: 8080,
		},
		InternalServer: &ServerConfig{
			Port: 9090,
		},
		Templates: &TemplateConfig{
			FolderList:          "templates/folder-list.tpl",
			TargetList:          "templates/target-list.tpl",
			NotFound:            "templates/not-found.tpl",
			InternalServerError: "templates/internal-server-error.tpl",
			Unauthorized:        "templates/unauthorized.tpl",
			Forbidden:           "templates/forbidden.tpl",
			BadRequest:          "templates/bad-request.tpl",
		},
		Tracing: &TracingConfig{Enabled: false},
		ListTargets: &ListTargetsConfig{
			Enabled: false,
		},
		Targets: []*TargetConfig{
			{
				Name: "test",
				Mount: &MountConfig{
					Path: []string{"/test/"},
				},
				Bucket: &BucketConfig{
					Name:   "bucket1",
					Region: "us-east-1",
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

	secretFiles = map[string]string{
		os.TempDir() + "/secret1": "SECRET1",
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
				Port: 8080,
			},
			InternalServer: &ServerConfig{
				Port: 9090,
			},
			Templates: &TemplateConfig{
				FolderList:          "templates/folder-list.tpl",
				TargetList:          "templates/target-list.tpl",
				NotFound:            "templates/not-found.tpl",
				InternalServerError: "templates/internal-server-error.tpl",
				Unauthorized:        "templates/unauthorized.tpl",
				Forbidden:           "templates/forbidden.tpl",
				BadRequest:          "templates/bad-request.tpl",
			},
			Tracing: &TracingConfig{Enabled: false},
			ListTargets: &ListTargetsConfig{
				Enabled: false,
			},
			Targets: []*TargetConfig{
				{
					Name: "test",
					Mount: &MountConfig{
						Path: []string{"/test/"},
					},
					Bucket: &BucketConfig{
						Name:   "bucket1",
						Region: "us-east-1",
						Credentials: &BucketCredentialConfig{
							AccessKey: &CredentialConfig{
								Value: "SECRET1",
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
		return
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "shouldn't call this")
	}
}

func Test_Load_reload_config_with_wrong_config(t *testing.T) {
	// Channel for wait watch
	waitCh := make(chan bool)

	dir, err := ioutil.TempDir("", "s3-proxy-config-reload-wrong-config")
	assert.NoError(t, err)

	configs := map[string]string{
		"log.yaml": `
log:
  level: error
  format: text
`,
		"targets.yaml": `
targets:
- name: test
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + os.TempDir() + `/secret1
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
		os.TempDir() + "/secret1": "VALUE1",
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
			Port: 8080,
		},
		InternalServer: &ServerConfig{
			Port: 9090,
		},
		Templates: &TemplateConfig{
			FolderList:          "templates/folder-list.tpl",
			TargetList:          "templates/target-list.tpl",
			NotFound:            "templates/not-found.tpl",
			InternalServerError: "templates/internal-server-error.tpl",
			Unauthorized:        "templates/unauthorized.tpl",
			Forbidden:           "templates/forbidden.tpl",
			BadRequest:          "templates/bad-request.tpl",
		},
		Tracing: &TracingConfig{Enabled: false},
		ListTargets: &ListTargetsConfig{
			Enabled: false,
		},
		Targets: []*TargetConfig{
			{
				Name: "test",
				Mount: &MountConfig{
					Path: []string{"/test/"},
				},
				Bucket: &BucketConfig{
					Name:   "bucket1",
					Region: "us-east-1",
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

	configs = map[string]string{
		"log.yaml": `
configuration with error
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
				Port: 8080,
			},
			InternalServer: &ServerConfig{
				Port: 9090,
			},
			Templates: &TemplateConfig{
				FolderList:          "templates/folder-list.tpl",
				TargetList:          "templates/target-list.tpl",
				NotFound:            "templates/not-found.tpl",
				InternalServerError: "templates/internal-server-error.tpl",
				Unauthorized:        "templates/unauthorized.tpl",
				Forbidden:           "templates/forbidden.tpl",
				BadRequest:          "templates/bad-request.tpl",
			},
			Tracing: &TracingConfig{Enabled: false},
			ListTargets: &ListTargetsConfig{
				Enabled: false,
			},
			Targets: []*TargetConfig{
				{
					Name: "test",
					Mount: &MountConfig{
						Path: []string{"/test/"},
					},
					Bucket: &BucketConfig{
						Name:   "bucket1",
						Region: "us-east-1",
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
	}
}

func Test_Load_reload_config_map_structure(t *testing.T) {
	// Channel for wait watch
	waitCh := make(chan bool)

	dir, err := ioutil.TempDir("", "s3-proxy-config-reload-map-structure")
	assert.NoError(t, err)

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
- name: test
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + os.TempDir() + `/secret1
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
		os.TempDir() + "/secret1": "VALUE1",
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
			Port: 8080,
		},
		InternalServer: &ServerConfig{
			Port: 9090,
		},
		Templates: &TemplateConfig{
			FolderList:          "templates/folder-list.tpl",
			TargetList:          "templates/target-list.tpl",
			NotFound:            "templates/not-found.tpl",
			InternalServerError: "templates/internal-server-error.tpl",
			Unauthorized:        "templates/unauthorized.tpl",
			Forbidden:           "templates/forbidden.tpl",
			BadRequest:          "templates/bad-request.tpl",
		},
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
		Targets: []*TargetConfig{
			{
				Name: "test",
				Mount: &MountConfig{
					Path: []string{"/test/"},
				},
				Bucket: &BucketConfig{
					Name:   "bucket1",
					Region: "us-east-1",
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
				Port: 8080,
			},
			InternalServer: &ServerConfig{
				Port: 9090,
			},
			Templates: &TemplateConfig{
				FolderList:          "templates/folder-list.tpl",
				TargetList:          "templates/target-list.tpl",
				NotFound:            "templates/not-found.tpl",
				InternalServerError: "templates/internal-server-error.tpl",
				Unauthorized:        "templates/unauthorized.tpl",
				Forbidden:           "templates/forbidden.tpl",
				BadRequest:          "templates/bad-request.tpl",
			},
			Tracing: &TracingConfig{Enabled: false},
			ListTargets: &ListTargetsConfig{
				Enabled: false,
			},
			AuthProviders: &AuthProviderConfig{
				Basic: map[string]*BasicAuthConfig{
					"provider1": {Realm: "prov1"},
				},
			},
			Targets: []*TargetConfig{
				{
					Name: "test",
					Mount: &MountConfig{
						Path: []string{"/test/"},
					},
					Bucket: &BucketConfig{
						Name:   "bucket1",
						Region: "us-east-1",
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
		return
	case <-time.After(5 * time.Second):
		assert.FailNow(t, "shouldn't call this")
	}
}

func Test_Load_reload_config_ignore_hidden_file_and_directory(t *testing.T) {
	// Channel for wait watch
	waitCh := make(chan bool)

	dir, err := ioutil.TempDir("", "s3-proxy-config-reload-ignore")
	assert.NoError(t, err)
	err = os.MkdirAll(path.Join(dir, "dir1"), os.ModePerm)
	assert.NoError(t, err)

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
- name: test
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
    credentials:
      accessKey:
        path: ` + os.TempDir() + `/secret1
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
		os.TempDir() + "/secret1": "VALUE1",
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
			Port: 8080,
		},
		InternalServer: &ServerConfig{
			Port: 9090,
		},
		Tracing: &TracingConfig{Enabled: false},
		Templates: &TemplateConfig{
			FolderList:          "templates/folder-list.tpl",
			TargetList:          "templates/target-list.tpl",
			NotFound:            "templates/not-found.tpl",
			InternalServerError: "templates/internal-server-error.tpl",
			Unauthorized:        "templates/unauthorized.tpl",
			Forbidden:           "templates/forbidden.tpl",
			BadRequest:          "templates/bad-request.tpl",
		},
		AuthProviders: &AuthProviderConfig{
			Basic: map[string]*BasicAuthConfig{
				"provider1": {Realm: "prov1"},
				"provider2": {Realm: "prov2"},
			},
		},
		ListTargets: &ListTargetsConfig{
			Enabled: false,
		},
		Targets: []*TargetConfig{
			{
				Name: "test",
				Mount: &MountConfig{
					Path: []string{"/test/"},
				},
				Bucket: &BucketConfig{
					Name:   "bucket1",
					Region: "us-east-1",
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
}
