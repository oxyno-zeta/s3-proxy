// +build unit

package config

import (
	"testing"
)

func Test_validatePath(t *testing.T) {
	type args struct {
		beginErrorMessage string
		path              string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "Path must start with a /",
			args: args{
				beginErrorMessage: "begin",
				path:              "test",
			},
			wantErr: true,
		},
		{
			name: "Path must end with a /",
			args: args{
				beginErrorMessage: "begin",
				path:              "/test",
			},
			wantErr: true,
		},
		{
			name: "Path must be ok",
			args: args{
				beginErrorMessage: "begin",
				path:              "/test/",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := validatePath(tt.args.beginErrorMessage, tt.args.path); (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func Test_validateResource(t *testing.T) {
	falseValue := false
	// trueValue := true
	type args struct {
		beginErrorMessage string
		res               *Resource
		authProviders     *AuthProviderConfig
		mountPathList     []string
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		errorString string
	}{
		{
			name: "Resource don't have a valid http method",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods: []string{"POST"},
				},
				authProviders: &AuthProviderConfig{},
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error must have a HTTP method in GET, PUT or DELETE",
		},
		{
			name: "Resource don't have a valid http method (2)",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods: []string{"GET", "POST"},
				},
				authProviders: &AuthProviderConfig{},
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error must have a HTTP method in GET, PUT or DELETE",
		},
		{
			name: "Resource don't have any whitelist or authentication settings",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods: []string{"GET"},
				},
				authProviders: &AuthProviderConfig{},
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error must have whitelist, basic configuration or oidc configuration",
		},
		{
			name: "Resource don't have any whitelist and no provider is set",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods:   []string{"GET"},
					WhiteList: &falseValue,
				},
				authProviders: &AuthProviderConfig{},
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error must have a provider",
		},
		{
			name: "Resource don't have any whitelist and no authentication settings are set",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods:   []string{"GET"},
					WhiteList: &falseValue,
					Provider:  "test",
				},
				authProviders: &AuthProviderConfig{},
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error must have authentication configuration declared (oidc or basic)",
		},
		{
			name: "Resource declare a provider but authorization providers are nil",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods:   []string{"GET"},
					WhiteList: &falseValue,
					Provider:  "test",
					Basic:     &ResourceBasic{},
				},
				authProviders: nil,
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error has declared a provider but authentication providers aren't declared",
		},
		{
			name: "Resource use a not declared provider (Basic auth case) and no provider declared",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods:   []string{"GET"},
					WhiteList: &falseValue,
					Provider:  "test",
					Basic:     &ResourceBasic{},
				},
				authProviders: &AuthProviderConfig{},
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error must have a valid provider declared in authentication providers",
		},
		{
			name: "Resource use a not declared provider (OIDC auth case) and no provider declared",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods:   []string{"GET"},
					WhiteList: &falseValue,
					Provider:  "test",
					OIDC:      &ResourceOIDC{},
				},
				authProviders: &AuthProviderConfig{},
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error must have a valid provider declared in authentication providers",
		},
		{
			name: "Resource use a not declared provider (Basic auth case)",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods:   []string{"GET"},
					WhiteList: &falseValue,
					Provider:  "test",
					Basic:     &ResourceBasic{},
				},
				authProviders: &AuthProviderConfig{
					OIDC: map[string]*OIDCAuthConfig{
						"test": {},
					},
				},
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error must use a valid authentication configuration with selected authentication provider: basic auth not allowed",
		},
		{
			name: "Resource use a not declared provider (OIDC auth case)",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods:   []string{"GET"},
					WhiteList: &falseValue,
					Provider:  "test",
					OIDC:      &ResourceOIDC{},
				},
				authProviders: &AuthProviderConfig{
					Basic: map[string]*BasicAuthConfig{
						"test": {},
					},
				},
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error must use a valid authentication configuration with selected authentication provider: oidc not allowed",
		},
		{
			name: "Resource with invalid oidc authorization methods",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods:   []string{"GET"},
					WhiteList: &falseValue,
					Provider:  "test",
					OIDC: &ResourceOIDC{
						AuthorizationAccesses: []*OIDCAuthorizationAccess{
							{Email: "fake@fake.com"},
						},
						AuthorizationOPAServer: &OPAServerAuthorization{
							URL: "http://fake.com",
						},
					},
				},
				authProviders: &AuthProviderConfig{
					OIDC: map[string]*OIDCAuthConfig{
						"test": {},
					},
				},
				mountPathList: []string{"/"},
			},
			wantErr:     true,
			errorString: "begin error cannot contain oidc authorization accesses and OPA server together at the same time",
		},
		{
			name: "Resource path must begin by mount path",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods:   []string{"GET"},
					WhiteList: &falseValue,
					Provider:  "test",
					Basic:     &ResourceBasic{},
					Path:      "/test/",
				},
				authProviders: &AuthProviderConfig{
					Basic: map[string]*BasicAuthConfig{
						"test": {},
					},
				},
				mountPathList: []string{"/v1/"},
			},
			wantErr:     true,
			errorString: "begin error must start with path declared in mount path section",
		},
		{
			name: "Resource is valid",
			args: args{
				beginErrorMessage: "begin error",
				res: &Resource{
					Methods:   []string{"GET"},
					WhiteList: &falseValue,
					Provider:  "test",
					Basic:     &ResourceBasic{},
					Path:      "/v1/test/",
				},
				authProviders: &AuthProviderConfig{
					Basic: map[string]*BasicAuthConfig{
						"test": {},
					},
				},
				mountPathList: []string{"/v1/"},
			},
			wantErr:     false,
			errorString: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResource(tt.args.beginErrorMessage, tt.args.res, tt.args.authProviders, tt.args.mountPathList)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateResource() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.errorString {
				t.Errorf("validateResource() error = %v, wantErr %v", err.Error(), tt.errorString)
			}
		})
	}
}

func Test_validateBusinessConfig(t *testing.T) {
	type args struct {
		out *Config
	}
	tests := []struct {
		name        string
		args        args
		wantErr     bool
		errorString string
	}{
		{
			name: "Path is invalid in target",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET: &GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			wantErr:     true,
			errorString: "path 0 in target 0 must ends with /",
		},
		{
			name: "Resource is invalid in target",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1"},
							},
							Resources: []*Resource{
								{
									Path:     "/*",
									Provider: "doesn't exists",
								},
							},
							Actions: &ActionsConfig{
								GET: &GetActionConfig{Enabled: true},
							},
						},
					},
				},
			},
			wantErr:     true,
			errorString: "resource 0 from target 0 must have whitelist, basic configuration or oidc configuration",
		},
		{
			name: "No actions are present in target",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1/"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET:    nil,
								PUT:    nil,
								DELETE: nil,
							},
						},
					},
				},
			},
			wantErr:     true,
			errorString: "at least one action must be declared in target 0",
		},
		{
			name: "No actions are enabled in target",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1/"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET:    &GetActionConfig{Enabled: false},
								PUT:    &PutActionConfig{Enabled: false},
								DELETE: &DeleteActionConfig{Enabled: false},
							},
						},
					},
				},
			},
			wantErr:     true,
			errorString: "at least one action must be enabled in target 0",
		},
		{
			name: "Configuration is valid without list targets",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1/"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET:    &GetActionConfig{Enabled: true},
								PUT:    &PutActionConfig{Enabled: false},
								DELETE: &DeleteActionConfig{Enabled: false},
							},
						},
					},
				},
			},
			wantErr:     false,
			errorString: "",
		},
		{
			name: "Configuration is valid with list targets disabled",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1/"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET:    &GetActionConfig{Enabled: true},
								PUT:    &PutActionConfig{Enabled: false},
								DELETE: &DeleteActionConfig{Enabled: false},
							},
						},
					},
					ListTargets: &ListTargetsConfig{Enabled: false},
				},
			},
			wantErr:     false,
			errorString: "",
		},
		{
			name: "List targets resource is invalid",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1/"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET:    &GetActionConfig{Enabled: true},
								PUT:    &PutActionConfig{Enabled: false},
								DELETE: &DeleteActionConfig{Enabled: false},
							},
						},
					},
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Mount: &MountConfig{
							Path: []string{"/"},
						},
						Resource: &Resource{
							Path:     "/*",
							Provider: "doesn't exists",
						},
					},
				},
			},
			wantErr:     true,
			errorString: "resource from list targets must have whitelist, basic configuration or oidc configuration",
		},
		{
			name: "List targets path is invalid",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1/"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET:    &GetActionConfig{Enabled: true},
								PUT:    &PutActionConfig{Enabled: false},
								DELETE: &DeleteActionConfig{Enabled: false},
							},
						},
					},
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Mount: &MountConfig{
							Path: []string{"/list"},
						},
						Resource: nil,
					},
				},
			},
			wantErr:     true,
			errorString: "path 0 in list targets must ends with /",
		},
		{
			name: "OIDC provider with wrong callback path",
			args: args{
				out: &Config{
					AuthProviders: &AuthProviderConfig{
						OIDC: map[string]*OIDCAuthConfig{
							"provider1": {
								CallbackPath: "/",
							},
						},
					},
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1/"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET:    &GetActionConfig{Enabled: true},
								PUT:    &PutActionConfig{Enabled: false},
								DELETE: &DeleteActionConfig{Enabled: false},
							},
						},
					},
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Mount: &MountConfig{
							Path: []string{"/"},
						},
						Resource: nil,
					},
				},
			},
			wantErr:     true,
			errorString: "provider provider1 can't have a callback path equal to / (to avoid redirect loop)",
		},
		{
			name: "OIDC provider with wrong login path",
			args: args{
				out: &Config{
					AuthProviders: &AuthProviderConfig{
						OIDC: map[string]*OIDCAuthConfig{
							"provider1": {
								LoginPath: "/",
							},
						},
					},
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1/"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET:    &GetActionConfig{Enabled: true},
								PUT:    &PutActionConfig{Enabled: false},
								DELETE: &DeleteActionConfig{Enabled: false},
							},
						},
					},
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Mount: &MountConfig{
							Path: []string{"/"},
						},
						Resource: nil,
					},
				},
			},
			wantErr:     true,
			errorString: "provider provider1 can't have a login path equal to / (to avoid redirect loop)",
		},
		{
			name: "OIDC provider with same login and callback path",
			args: args{
				out: &Config{
					AuthProviders: &AuthProviderConfig{
						OIDC: map[string]*OIDCAuthConfig{
							"provider1": {
								LoginPath:    "/fake",
								CallbackPath: "/fake",
							},
						},
					},
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1/"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET:    &GetActionConfig{Enabled: true},
								PUT:    &PutActionConfig{Enabled: false},
								DELETE: &DeleteActionConfig{Enabled: false},
							},
						},
					},
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Mount: &MountConfig{
							Path: []string{"/"},
						},
						Resource: nil,
					},
				},
			},
			wantErr:     true,
			errorString: "provider provider1 can't have same login and callback path (to avoid redirect loop)",
		},
		{
			name: "Configuration with list target and target is valid",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Name: "test1",
							Bucket: &BucketConfig{
								Name:   "bucket1",
								Region: "region1",
							},
							Mount: &MountConfig{
								Path: []string{"/mount1/"},
							},
							Resources: nil,
							Actions: &ActionsConfig{
								GET:    &GetActionConfig{Enabled: true},
								PUT:    &PutActionConfig{Enabled: false},
								DELETE: &DeleteActionConfig{Enabled: false},
							},
						},
					},
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Mount: &MountConfig{
							Path: []string{"/"},
						},
						Resource: nil,
					},
				},
			},
			wantErr:     false,
			errorString: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateBusinessConfig(tt.args.out)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateBusinessConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.errorString {
				t.Errorf("validateBusinessConfig() error = %v, wantErr %v", err.Error(), tt.errorString)
			}
		})
	}
}
