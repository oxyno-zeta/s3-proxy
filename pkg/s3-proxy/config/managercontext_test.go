// +build unit

package config

import (
	"reflect"
	"regexp"
	"testing"
)

func Test_loadBusinessDefaultValues(t *testing.T) {
	type args struct {
		out *Config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		result  *Config
	}{
		{
			name: "Load default values with nothing in it",
			args: args{
				out: &Config{},
			},
			wantErr: false,
			result: &Config{
				ListTargets: &ListTargetsConfig{Enabled: false},
			},
		},
		{
			name: "Load default values skipped for list targets config",
			args: args{
				out: &Config{
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Mount: &MountConfig{
							Path: []string{"/"},
						},
					},
				},
			},
			wantErr: false,
			result: &Config{
				ListTargets: &ListTargetsConfig{
					Enabled: true,
					Mount: &MountConfig{
						Path: []string{"/"},
					},
				},
			},
		},
		{
			name: "Load default values for oidc auth providers",
			args: args{
				out: &Config{
					AuthProviders: &AuthProviderConfig{
						OIDC: map[string]*OIDCAuthConfig{
							"provider1": {},
						},
					},
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Mount: &MountConfig{
							Path: []string{"/"},
						},
					},
				},
			},
			wantErr: false,
			result: &Config{
				AuthProviders: &AuthProviderConfig{
					OIDC: map[string]*OIDCAuthConfig{
						"provider1": {
							Scopes:       DefaultOIDCScopes,
							GroupClaim:   DefaultOIDCGroupClaim,
							CookieName:   DefaultOIDCCookieName,
							LoginPath:    "/auth/provider1",
							CallbackPath: "/auth/provider1/callback",
						},
					},
				},
				ListTargets: &ListTargetsConfig{
					Enabled: true,
					Mount: &MountConfig{
						Path: []string{"/"},
					},
				},
			},
		},
		{
			name: "Load default values for oidc auth providers (2)",
			args: args{
				out: &Config{
					AuthProviders: &AuthProviderConfig{
						OIDC: map[string]*OIDCAuthConfig{
							"provider1": {
								Scopes:       []string{"test"},
								GroupClaim:   "test",
								CookieName:   "test",
								LoginPath:    "/test",
								CallbackPath: "/test/callback",
							},
						},
					},
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Mount: &MountConfig{
							Path: []string{"/"},
						},
					},
				},
			},
			wantErr: false,
			result: &Config{
				AuthProviders: &AuthProviderConfig{
					OIDC: map[string]*OIDCAuthConfig{
						"provider1": {
							Scopes:       []string{"test"},
							GroupClaim:   "test",
							CookieName:   "test",
							LoginPath:    "/test",
							CallbackPath: "/test/callback",
						},
					},
				},
				ListTargets: &ListTargetsConfig{
					Enabled: true,
					Mount: &MountConfig{
						Path: []string{"/"},
					},
				},
			},
		},
		{
			name: "Load default values for list targets (methods)",
			args: args{
				out: &Config{
					ListTargets: &ListTargetsConfig{
						Enabled:  true,
						Resource: &Resource{},
					},
				},
			},
			wantErr: false,
			result: &Config{
				ListTargets: &ListTargetsConfig{
					Enabled: true,
					Resource: &Resource{
						Methods: []string{"GET"},
					},
				},
			},
		},
		{
			name: "Load default values for list targets (OIDC Group Regexp)",
			args: args{
				out: &Config{
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Resource: &Resource{
							OIDC: &ResourceOIDC{
								AuthorizationAccesses: []*OIDCAuthorizationAccess{
									{
										Group:  ".*",
										Regexp: true,
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			result: &Config{
				ListTargets: &ListTargetsConfig{
					Enabled: true,
					Resource: &Resource{
						Methods: []string{"GET"},
						OIDC: &ResourceOIDC{
							AuthorizationAccesses: []*OIDCAuthorizationAccess{
								{
									Group:       ".*",
									Regexp:      true,
									GroupRegexp: regexp.MustCompile(".*"),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Load default values for list targets (OIDC Email Regexp)",
			args: args{
				out: &Config{
					ListTargets: &ListTargetsConfig{
						Enabled: true,
						Resource: &Resource{
							OIDC: &ResourceOIDC{
								AuthorizationAccesses: []*OIDCAuthorizationAccess{
									{
										Email:  ".*",
										Regexp: true,
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			result: &Config{
				ListTargets: &ListTargetsConfig{
					Enabled: true,
					Resource: &Resource{
						Methods: []string{"GET"},
						OIDC: &ResourceOIDC{
							AuthorizationAccesses: []*OIDCAuthorizationAccess{
								{
									Email:       ".*",
									Regexp:      true,
									EmailRegexp: regexp.MustCompile(".*"),
								},
							},
						},
					},
				},
			},
		},
		{
			name: "Load default values for targets (Actions)",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{},
					},
				},
			},
			wantErr: false,
			result: &Config{
				Targets: []*TargetConfig{
					{
						Actions:   &ActionsConfig{GET: &GetActionConfig{Enabled: true}},
						Templates: &TargetTemplateConfig{},
					},
				},
				ListTargets: &ListTargetsConfig{Enabled: false},
			},
		},
		{
			name: "Load default values for targets (Bucket region)",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Actions:   &ActionsConfig{GET: &GetActionConfig{Enabled: false}},
							Bucket:    &BucketConfig{},
							Templates: &TargetTemplateConfig{},
						},
					},
				},
			},
			wantErr: false,
			result: &Config{
				Targets: []*TargetConfig{
					{
						Actions:   &ActionsConfig{GET: &GetActionConfig{Enabled: false}},
						Bucket:    &BucketConfig{Region: DefaultBucketRegion},
						Templates: &TargetTemplateConfig{},
					},
				},
				ListTargets: &ListTargetsConfig{Enabled: false},
			},
		},
		{
			name: "Load default values for targets (resource)",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Actions: &ActionsConfig{GET: &GetActionConfig{Enabled: false}},
							Bucket:  &BucketConfig{Region: "test"},
							Resources: []*Resource{
								{
									OIDC: &ResourceOIDC{
										AuthorizationAccesses: []*OIDCAuthorizationAccess{
											{
												Email:  ".*",
												Regexp: true,
											},
										},
									},
								},
							},
							Templates: &TargetTemplateConfig{},
						},
					},
				},
			},
			wantErr: false,
			result: &Config{
				Targets: []*TargetConfig{
					{
						Actions: &ActionsConfig{GET: &GetActionConfig{Enabled: false}},
						Bucket:  &BucketConfig{Region: "test"},
						Resources: []*Resource{
							{
								Methods: []string{"GET"},
								OIDC: &ResourceOIDC{
									AuthorizationAccesses: []*OIDCAuthorizationAccess{
										{
											Email:       ".*",
											Regexp:      true,
											EmailRegexp: regexp.MustCompile(".*"),
										},
									},
								},
							},
						},
						Templates: &TargetTemplateConfig{},
					},
				},
				ListTargets: &ListTargetsConfig{Enabled: false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := loadBusinessDefaultValues(tt.args.out)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadDefaultValues() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && !reflect.DeepEqual(tt.args.out, tt.result) {
				t.Errorf("loadDefaultValues() source = %+v, want %+v", tt.args.out, tt.result)
			}
		})
	}
}

func Test_loadAllCredentials(t *testing.T) {
	type args struct {
		out *Config
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
		result  []*CredentialConfig
		cfg     *Config
	}{
		{
			name: "Skip all load credential",
			args: args{
				out: &Config{},
			},
			wantErr: false,
			cfg:     &Config{},
			result:  []*CredentialConfig{},
		},
		{
			name: "Skip target load credential",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Bucket: &BucketConfig{},
						},
					},
				},
			},
			wantErr: false,
			cfg: &Config{
				Targets: []*TargetConfig{
					{
						Bucket: &BucketConfig{},
					},
				},
			},
			result: []*CredentialConfig{},
		},
		{
			name: "Skip target resource basic auth load credential",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Resources: []*Resource{
								{},
							},
							Bucket: &BucketConfig{},
						},
					},
				},
			},
			wantErr: false,
			cfg: &Config{
				Targets: []*TargetConfig{
					{
						Resources: []*Resource{
							{},
						},
						Bucket: &BucketConfig{},
					},
				},
			},
			result: []*CredentialConfig{},
		},
		{
			name: "Load target resource basic auth credentials",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Resources: []*Resource{
								{
									Basic: &ResourceBasic{
										Credentials: []*BasicAuthUserConfig{
											{
												Password: &CredentialConfig{
													Value: "value1",
												},
											},
										},
									},
								},
							},
							Bucket: &BucketConfig{},
						},
					},
				},
			},
			wantErr: false,
			cfg: &Config{
				Targets: []*TargetConfig{
					{
						Resources: []*Resource{
							{
								Basic: &ResourceBasic{
									Credentials: []*BasicAuthUserConfig{
										{
											Password: &CredentialConfig{
												Value: "value1",
											},
										},
									},
								},
							},
						},
						Bucket: &BucketConfig{},
					},
				},
			},
			result: []*CredentialConfig{
				{
					Value: "value1",
				},
			},
		},
		{
			name: "Load target bucket credentials",
			args: args{
				out: &Config{
					Targets: []*TargetConfig{
						{
							Bucket: &BucketConfig{
								Credentials: &BucketCredentialConfig{
									AccessKey: &CredentialConfig{
										Value: "value1",
									},
									SecretKey: &CredentialConfig{
										Value: "value2",
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			cfg: &Config{
				Targets: []*TargetConfig{
					{
						Bucket: &BucketConfig{
							Credentials: &BucketCredentialConfig{
								AccessKey: &CredentialConfig{
									Value: "value1",
								},
								SecretKey: &CredentialConfig{
									Value: "value2",
								},
							},
						},
					},
				},
			},
			result: []*CredentialConfig{
				{
					Value: "value1",
				},
				{
					Value: "value2",
				},
			},
		},
		{
			name: "Load list targets resource basic auth credentials",
			args: args{
				out: &Config{
					ListTargets: &ListTargetsConfig{
						Resource: &Resource{
							Basic: &ResourceBasic{
								Credentials: []*BasicAuthUserConfig{
									{
										Password: &CredentialConfig{
											Value: "value1",
										},
									},
								},
							},
						},
					},
				},
			},
			wantErr: false,
			cfg: &Config{
				ListTargets: &ListTargetsConfig{
					Resource: &Resource{
						Basic: &ResourceBasic{
							Credentials: []*BasicAuthUserConfig{
								{
									Password: &CredentialConfig{
										Value: "value1",
									},
								},
							},
						},
					},
				},
			},
			result: []*CredentialConfig{
				{
					Value: "value1",
				},
			},
		},
		{
			name: "Load auth providers oidc credentials",
			args: args{
				out: &Config{
					AuthProviders: &AuthProviderConfig{
						OIDC: map[string]*OIDCAuthConfig{
							"test": {
								ClientSecret: &CredentialConfig{
									Value: "value1",
								},
							},
						},
					},
				},
			},
			wantErr: false,
			cfg: &Config{
				AuthProviders: &AuthProviderConfig{
					OIDC: map[string]*OIDCAuthConfig{
						"test": {
							ClientSecret: &CredentialConfig{
								Value: "value1",
							},
						},
					},
				},
			},
			result: []*CredentialConfig{
				{
					Value: "value1",
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := loadAllCredentials(tt.args.out)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadAllCredentials() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(tt.cfg, tt.args.out) {
				t.Errorf("loadAllCredentials() source = %+v, want %+v", tt.cfg, tt.args.out)
			}
			if !reflect.DeepEqual(tt.result, res) {
				t.Errorf("loadAllCredentials() result = %+v, want %+v", res, tt.result)
			}
		})
	}
}
