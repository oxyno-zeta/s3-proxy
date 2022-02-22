//go:build unit

package config

import (
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
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
					Targets: map[string]*TargetConfig{
						"test1": {
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
			errorString: "path 0 in target test1 must ends with /",
		},
		{
			name: "Resource is invalid in target",
			args: args{
				out: &Config{
					Targets: map[string]*TargetConfig{
						"test1": {
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
			errorString: "resource 0 from target test1 must have whitelist, basic configuration or oidc configuration",
		},
		{
			name: "No actions are present in target",
			args: args{
				out: &Config{
					Targets: map[string]*TargetConfig{
						"test1": {
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
			errorString: "at least one action must be declared in target test1",
		},
		{
			name: "No actions are enabled in target",
			args: args{
				out: &Config{
					Targets: map[string]*TargetConfig{
						"test1": {
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
			errorString: "at least one action must be enabled in target test1",
		},
		{
			name: "Configuration is valid without list targets",
			args: args{
				out: &Config{
					Targets: map[string]*TargetConfig{
						"test1": {
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
					Targets: map[string]*TargetConfig{
						"test1": {
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
					Targets: map[string]*TargetConfig{
						"test1": {
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
					Targets: map[string]*TargetConfig{
						"test1": {
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
			name: "OIDC provider with wrong state",
			args: args{
				out: &Config{
					AuthProviders: &AuthProviderConfig{
						OIDC: map[string]*OIDCAuthConfig{
							"provider1": {
								State: "fake:fake",
							},
						},
					},
					Targets: map[string]*TargetConfig{
						"test1": {
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
			errorString: "provider provider1 state can't contain ':' character",
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
					Targets: map[string]*TargetConfig{
						"test1": {
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
					Targets: map[string]*TargetConfig{
						"test1": {
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
					Targets: map[string]*TargetConfig{
						"test1": {
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
					Targets: map[string]*TargetConfig{
						"test1": {
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

func Test_validateSSL(t *testing.T) {
	tests := []struct {
		name                 string
		serverConfig         *ServerConfig
		internalServerConfig *ServerConfig
		wantErr              bool
		errorString          string
	}{
		{
			name: "Valid server config with generated certificate",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled:             true,
					SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
				},
			},
		},
		{
			name: "Valid server config with supplied certificate",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							Certificate: &testCertificate,
							PrivateKey:  &testPrivateKey,
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Valid server config with supplied and generated certificates",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							Certificate: &testCertificate,
							PrivateKey:  &testPrivateKey,
						},
					},
					SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid server config with no certificates",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled:             true,
					Certificates:        nil,
					SelfSignedHostnames: nil,
				},
			},
			wantErr:     true,
			errorString: "at least one of server.ssl.certificates or server.ssl.selfSignedHostnames must have values",
		},
		{
			name: "Invalid minTLSVersion",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							Certificate: &testCertificate,
							PrivateKey:  &testPrivateKey,
						},
					},
					MinTLSVersion: aws.String("ssl3.0"),
				},
			},
			wantErr:     true,
			errorString: `server.ssl.minTLSVersion "ssl3.0" must be a valid TLS version: expected "TLSv1.0", "TLSv1.1", "TLSv1.2", or "TLSv1.3"`,
		},
		{
			name: "Invalid maxTLSVersion",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							Certificate: &testCertificate,
							PrivateKey:  &testPrivateKey,
						},
					},
					MinTLSVersion: aws.String("tls1.2"),
					MaxTLSVersion: aws.String("ssl3.0"),
				},
			},
			wantErr:     true,
			errorString: `server.ssl.maxTLSVersion "ssl3.0" must be a valid TLS version: expected "TLSv1.0", "TLSv1.1", "TLSv1.2", or "TLSv1.3"`,
		},
		{
			name: "Invalid cipherSuites",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							Certificate: &testCertificate,
							PrivateKey:  &testPrivateKey,
						},
					},
					MinTLSVersion: aws.String("tls1.2"),
					MaxTLSVersion: aws.String("tls1.3"),
					CipherSuites:  []string{"TLS_NOT_A_VALID_CIPHER"},
				},
			},
			wantErr:     true,
			errorString: `invalid cipher suite "TLS_NOT_A_VALID_CIPHER" in server.ssl.cipherSuites; expected one of `,
		},
		{
			name: "Invalid internalServer config with no certificates",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled:             true,
					SelfSignedHostnames: []string{"localhost", "localhost.localdomain"},
				},
			},
			internalServerConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
				},
			},
			wantErr:     true,
			errorString: "at least one of internalServer.ssl.certificates or internalServer.ssl.selfSignedHostnames must have values",
		},
		{
			name: "Missing certificate/certificateUrl",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							PrivateKey: &testPrivateKey,
						},
					},
				},
			},
			wantErr:     true,
			errorString: "server.ssl.certificates[0] must have either certificate or certificateUrl set",
		},
		{
			name: "Missing privateKey/privateKeyUrl",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							Certificate: &testCertificate,
						},
					},
				},
			},
			wantErr:     true,
			errorString: "server.ssl.certificates[0] must have either privateKey or privateKeyUrl set",
		},
		{
			name: "Both certificate and certificateUrl set",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							Certificate:    &testCertificate,
							CertificateURL: aws.String("s3://test/test.crt"),
							PrivateKey:     &testPrivateKey,
						},
					},
				},
			},
			wantErr:     true,
			errorString: "server.ssl.certificates[0] cannot have both certificate and certificateUrl set",
		},
		{
			name: "Both privateKey and privateKeyUrl set",
			serverConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							Certificate:   &testCertificate,
							PrivateKey:    &testPrivateKey,
							PrivateKeyURL: aws.String("s3://test/test.crt"),
						},
					},
				},
			},
			wantErr:     true,
			errorString: "server.ssl.certificates[0] cannot have both privateKey and privateKeyUrl set",
		},
		{
			name:         "InternalServer missing certificate/certificateUrl",
			serverConfig: &ServerConfig{},
			internalServerConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							PrivateKey: &testPrivateKey,
						},
					},
				},
			},
			wantErr:     true,
			errorString: "internalServer.ssl.certificates[0] must have either certificate or certificateUrl set",
		},
		{
			name:         "InternalServer missing privateKey/privateKeyUrl",
			serverConfig: &ServerConfig{},
			internalServerConfig: &ServerConfig{
				SSL: &ServerSSLConfig{
					Enabled: true,
					Certificates: []ServerSSLCertificate{
						{
							Certificate: &testCertificate,
						},
					},
				},
			},
			wantErr:     true,
			errorString: "internalServer.ssl.certificates[0] must have either privateKey or privateKeyUrl set",
		},
	}

	for _, currentTest := range tests {
		// Capture the current test for parallel processing. Otherwise currentTest will be modified during our test run.
		tt := currentTest

		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			cfg := &Config{
				Server:         tt.serverConfig,
				InternalServer: tt.internalServerConfig,
			}
			err := validateBusinessConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("%s validateBusinessConfig() error = %v, wantErr %v", tt.name, err, tt.wantErr)
			}
			if err != nil && !strings.HasPrefix(err.Error(), tt.errorString) {
				t.Errorf("validateBusinessConfig() error = %v, wantErr %v", err.Error(), tt.errorString)
			}
		})
	}
}

var (
	// Test certificate, self-signed, for testhost.example.com
	testCertificate = `-----BEGIN CERTIFICATE-----
MIIDeDCCAmACCQDbKC6SZoxWRTANBgkqhkiG9w0BAQUFADB9MQswCQYDVQQGEwJV
UzETMBEGA1UECAwKV2FzaGluZ3RvbjEQMA4GA1UEBwwHU2VhdHRsZTEdMBsGA1UE
AwwUdGVzdGhvc3QuZXhhbXBsZS5jb20xKDAmBgkqhkiG9w0BCQEWGXRlc3RAdGVz
dGhvc3QuZXhhbXBsZS5jb20wIBcNMjIwMjE2MTYzNjU0WhgPMjEyMjAxMjMxNjM2
NTRaMH0xCzAJBgNVBAYTAlVTMRMwEQYDVQQIDApXYXNoaW5ndG9uMRAwDgYDVQQH
DAdTZWF0dGxlMR0wGwYDVQQDDBR0ZXN0aG9zdC5leGFtcGxlLmNvbTEoMCYGCSqG
SIb3DQEJARYZdGVzdEB0ZXN0aG9zdC5leGFtcGxlLmNvbTCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAL/yQZn2ZDxvtos+CDScWS7YKqlNgV0L2dAF/9SZ
WkhM6+vwrl0AP25+Xf6U50va8Ux2RUC7MCnhsmMq3dp8t1fUxs/WpViX30BE4tLJ
47OuvhSY05aDsUf902dQuTg0HaKxXYjuW8FvaaF9GaR3eu4eVU8ahm09D5YFtz5D
i/wsKkVqikzOsKvBi0dVHZ+fxBmf/1t4Mqualq4YqjWU2DGf7lfsdv6cCDKCmkgg
AWJ3yDA70fiUGq5nigBLE+5bPSTFE/PZOFK+WtQZV2//ykwkE/bk+UOTRkdZPZP0
TqgfkuQub2m3F8JhkzGPtfnQ5S9C+fsndCOd4OBfzcPCldkCAwEAATANBgkqhkiG
9w0BAQUFAAOCAQEAncN7syI1+HcuCEKxS7EArp9fA+bOQX6EIJhSuOeyNXKhHdlm
RFToPkoMRwsCnonmD44lNXjQ2LbTRE0ySCqIm6H9Ha9C7sLZAWnbOB2Iz65YbqyD
zJq0pnhb6TN9jiVO7kXIvcPWrrA1TwBo6Y7dx6Svy3WLlKbQWGwQx9q2Hr209s0L
GO9TXExY6u0fNFJDyh7KFeTablSIH+oDLAytZrjzBOyPqe8aZI2SXAcJjz3Hp9hv
V6sfsRW0PfYOsUxvMglI5LXHGflkM4tRC/WzNUhei6TJKxLhyk8FkSpkRvbsLVQn
JYwisSNsLosVijV7XU2AlvuoWQlNEkY8bPJx3Q==
-----END CERTIFICATE-----`

	testPrivateKey = `-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAv/JBmfZkPG+2iz4INJxZLtgqqU2BXQvZ0AX/1JlaSEzr6/Cu
XQA/bn5d/pTnS9rxTHZFQLswKeGyYyrd2ny3V9TGz9alWJffQETi0snjs66+FJjT
loOxR/3TZ1C5ODQdorFdiO5bwW9poX0ZpHd67h5VTxqGbT0PlgW3PkOL/CwqRWqK
TM6wq8GLR1Udn5/EGZ//W3gyq5qWrhiqNZTYMZ/uV+x2/pwIMoKaSCABYnfIMDvR
+JQarmeKAEsT7ls9JMUT89k4Ur5a1BlXb//KTCQT9uT5Q5NGR1k9k/ROqB+S5C5v
abcXwmGTMY+1+dDlL0L5+yd0I53g4F/Nw8KV2QIDAQABAoIBAQCGkJbPEj55ZDQM
cCOehpG7Vo6p/I0Zpyo/PUV6TTxO/aZT1XrX9kmB9BN/W/K/ajHKUgwA8no0kmbW
QQIhn1eFusTahneKoYZA70o5TpJUsMfPdsi3d4G8n8UqZBxFu7ufCEszqS8ocCwU
q7hjZeQHtbpG56igQrN/kGhDvWURFsmAhi9763/wEgpDYWdLmw2hc7wPmuqg68r7
1Lk1CmcS7ZoQpx/QdhYtyG281f8lWOWQa/SL3VUQQl/J3U9GyCzSjHRy+ESSloYm
uzORowvexWB23324pAca6QYJPf5HqhzkLrfG3xTXI2xJPgoGiBMJqY84zxPaHJlm
mp8Laa4VAoGBAPBzskgH6t274P4slBML78M+E8zKM0amcEtWN+JgT7a1YKM3+3Wo
vwb/Y3RmHBN9Tget4shv2Gifm1zi4HmWgymt6ZTLnV9VfIrQXkC5PblDVCoAaxCL
ytWuLO5q+acq5iiVv5bB6mN0qm7GUl/dfClrHQ0bGb1V1l5BeRQnEdxnAoGBAMxb
oCHwwp3KDL7Xoxa08x7y0cEHAjyEnTFL/UIdZ+Bb/78HkxVAaYBq5fuw7bbcG8oT
Bjpn9FnOnNZXuy1ULNwl8OdkvYqOA5N8XwXcIA+yvIRTIwX/VTb8Rhie/FymStuT
UgA8HNoRjHy2eCP3VUmYI1t4KgmvOejB+HZZIJO/AoGAV7xPe/rvlvKb2QKZEQ4U
8S+wd9P7u7a1WLff8kjkLS2nUkb2COuGsF31gx5S9kWNeD3ZdvtggmRigxUBhTwH
JekgRru483U02U3IZmNxAy1vA1hduI7Zdvhzypbb+0Qq8PobCz48cQe7vGm+2t3t
FQvRcNvHm487he7r6A+Nc9cCgYEAtgwRlOqzlHj/7aqPYJUF19YcQUaLGXpRxi6Z
iCJF/To3k+edgVsGIR4ZjqPIwBNItjVIYRNmO/KxCMjSt8i6xcsO1jOKHjnwuZwb
0k6MSS/CfGbLVnZlZTxK/Xfz/F0vZnfQnuDuGt1zN04drHyS/6KGLN/ZIxN0FQNm
4Zb4TGUCgYEAl6eGVe+cZ5cIdwvNV49+X800BuZjSDSKNYBTaeIJWXeI9H+7b0qL
So0HeYWx9ixaRgxZ8yxGmB/CVOka/M5w06i0cwofTMWsiFYzPd6uPe2Mz6hcIPuE
csZ8PbpqNkbcznkfy8BDRhwanNsvzsXWyX/0LxU+CdZGQ9jDOZwItyY=
-----END RSA PRIVATE KEY-----`
)
