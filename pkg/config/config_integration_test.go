// +build integration

package config

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadAndValidateConfig(t *testing.T) {
	tests := []struct {
		name              string
		configFileContent string
		configFileName    string
		envVariables      map[string]string
		secretFiles       map[string]string
		expectedResult    *Config
		wantErr           bool
	}{
		{
			name:              "Configuration not found",
			configFileName:    "config",
			configFileContent: "",
			wantErr:           true,
		},
		{
			name:              "Not a yaml",
			configFileName:    "config.yaml",
			configFileContent: "notayaml",
			wantErr:           true,
		},
		{
			name:              "Empty",
			configFileName:    "config.yaml",
			configFileContent: "",
			wantErr:           true,
		},
		{
			name:           "Test all default values with minimal config",
			configFileName: "config.yaml",
			configFileContent: `
targets:
- name: test
  mount:
    path: /test/
  bucket:
    name: bucket1
    region: us-east-1
`,
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
			name:           "Test secrets from environment variable",
			configFileName: "config.yaml",
			configFileContent: `
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
			name:           "Test secrets from environment variable with empty environment variable",
			configFileName: "config.yaml",
			configFileContent: `
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
			envVariables: map[string]string{
				"ENV1": "VALUE1",
			},
			wantErr: true,
		},
		{
			name:           "Test secrets from a not found file",
			configFileName: "config.yaml",
			configFileContent: `
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
			wantErr: true,
		},
		{
			name:           "Test secrets from a file and direct value",
			configFileName: "config.yaml",
			configFileContent: `
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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir, err := ioutil.TempDir("", "s3-proxy-config")
			if err != nil {
				t.Error(err)
				return
			}

			defer os.RemoveAll(dir) // clean up
			tmpfn := filepath.Join(dir, tt.configFileName)
			err = ioutil.WriteFile(tmpfn, []byte(tt.configFileContent), 0666)
			if err != nil {
				t.Error(err)
				return
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
			MainConfigFolderPath = dir
			// Load config
			res, err := Load()

			if (err != nil) != tt.wantErr {
				t.Errorf("Load() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(res, tt.expectedResult) {
				t.Errorf("Load() source = %+v, want %+v", res, tt.expectedResult)
			}
		})
	}
}
