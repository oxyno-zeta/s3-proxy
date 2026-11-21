//go:build unit

package authentication

import (
	"net/http"
	"testing"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
)

func Test_getJWTToken(t *testing.T) {
	validAuthorizationHeader := http.Header{}
	validAuthorizationHeader.Add("Authorization", "Bearer TOKEN")
	invalidAuthorizationHeader1 := http.Header{}
	invalidAuthorizationHeader1.Add("Authorization", "TOKEN")
	invalidAuthorizationHeader2 := http.Header{}
	invalidAuthorizationHeader2.Add("Authorization", " TOKEN")
	invalidAuthorizationHeader3 := http.Header{}
	invalidAuthorizationHeader3.Add("Authorization", "Basic TOKEN")
	noHeader := http.Header{}
	validCookie := http.Header{}
	validCookie.Add("Cookie", "oidc=TOKEN")
	type args struct {
		r          *http.Request
		cookieName string
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Get token from Authorization header",
			args: args{
				r: &http.Request{
					Header: validAuthorizationHeader,
				},
				cookieName: "oidc",
			},
			want:    "TOKEN",
			wantErr: false,
		},
		{
			name: "Get token from Authorization header (invalid 1)",
			args: args{
				r: &http.Request{
					Header: invalidAuthorizationHeader1,
				},
				cookieName: "oidc",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Get token from Authorization header (invalid 2)",
			args: args{
				r: &http.Request{
					Header: invalidAuthorizationHeader2,
				},
				cookieName: "oidc",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Get token from Authorization header (invalid 3)",
			args: args{
				r: &http.Request{
					Header: invalidAuthorizationHeader3,
				},
				cookieName: "oidc",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Get token from cookie without any cookie",
			args: args{
				r: &http.Request{
					Header: noHeader,
				},
				cookieName: "oidc",
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "Get token from cookie without any cookie",
			args: args{
				r: &http.Request{
					Header: noHeader,
				},
				cookieName: "oidc",
			},
			want:    "",
			wantErr: false,
		},
		{
			name: "Get token from cookie with valid cookie",
			args: args{
				r: &http.Request{
					Header: validCookie,
				},
				cookieName: "oidc",
			},
			want:    "TOKEN",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getJWTToken(log.NewLogger(), tt.args.r, tt.args.cookieName)
			if (err != nil) != tt.wantErr {
				t.Errorf("getJWTToken() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getJWTToken() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_isValidRedirect(t *testing.T) {
	type args struct {
		redirectURLStr string
		reqURLStr      string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		{
			name: "empty redirect",
			args: args{redirectURLStr: ""},
			want: false,
		},
		{
			name: "singleSlash",
			args: args{redirectURLStr: "/redirect"},
			want: false,
		},
		{
			name: "doubleSlash",
			args: args{redirectURLStr: "//redirect"},
			want: false,
		},
		{
			name: "validHTTP",
			args: args{redirectURLStr: "http://foo.bar/redirect", reqURLStr: "http://foo.bar/"},
			want: true,
		},
		{
			name: "validHTTPS",
			args: args{redirectURLStr: "https://foo.bar/redirect", reqURLStr: "http://foo.bar/"},
			want: true,
		},
		{
			name: "not same domain http",
			args: args{redirectURLStr: "http://foo.bar/redirect", reqURLStr: "http://fake.com/"},
			want: false,
		},
		{
			name: "not same domain https",
			args: args{redirectURLStr: "https://foo.bar/redirect", reqURLStr: "http://fake.com/"},
			want: false,
		},
		{
			name: "openRedirect1",
			args: args{redirectURLStr: "/\\evil.com"},
			want: false,
		},
		{
			name: "openRedirectSpace1",
			args: args{redirectURLStr: "/ /evil.com"},
			want: false,
		},
		{
			name: "openRedirectSpace2",
			args: args{redirectURLStr: "/ \\evil.com"},
			want: false,
		},
		{
			name: "openRedirectTab1",
			args: args{redirectURLStr: "/\t/evil.com"},
			want: false,
		},
		{
			name: "openRedirectTab2",
			args: args{redirectURLStr: "/\t\\evil.com"},
			want: false,
		},
		{
			name: "openRedirectVerticalTab1",
			args: args{redirectURLStr: "/\v/evil.com"},
			want: false,
		},
		{
			name: "openRedirectVerticalTab2",
			args: args{redirectURLStr: "/\v\\evil.com"},
			want: false,
		},
		{
			name: "openRedirectNewLine1",
			args: args{redirectURLStr: "/\n/evil.com"},
			want: false,
		},
		{
			name: "openRedirectNewLine2",
			args: args{redirectURLStr: "/\n\\evil.com"},
			want: false,
		},
		{
			name: "openRedirectCarriageReturn1",
			args: args{redirectURLStr: "/\r/evil.com"},
			want: false,
		},
		{
			name: "openRedirectCarriageReturn2",
			args: args{redirectURLStr: "/\r\\evil.com"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := isValidRedirect(tt.args.redirectURLStr, tt.args.reqURLStr)
			if (err != nil) != tt.wantErr {
				t.Errorf("isValidRedirect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isValidRedirect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_buildOIDCUID(t *testing.T) {
	tests := []struct {
		name              string
		oidcAuthCfg       *config.OIDCAuthConfig
		claims            map[string]any
		preferredUsername string
		email             string
		want              string
		wantErr           bool
	}{
		{
			name: "default claim uses preferred username when present",
			oidcAuthCfg: &config.OIDCAuthConfig{
				UIDClaim: "",
			},
			claims:            map[string]any{},
			preferredUsername: "alice",
			email:             "alice@example.com",
			want:              "alice",
		},
		{
			name: "default claim falls back to email when preferred username is empty",
			oidcAuthCfg: &config.OIDCAuthConfig{
				UIDClaim: "",
			},
			claims:            map[string]any{},
			preferredUsername: "",
			email:             "alice@example.com",
			want:              "alice@example.com",
		},
		{
			name: "custom claim uses claim value",
			oidcAuthCfg: &config.OIDCAuthConfig{
				UIDClaim: "sub",
			},
			claims: map[string]any{
				"sub": "123456",
			},
			preferredUsername: "alice",
			email:             "alice@example.com",
			want:              "123456",
		},
		{
			name: "custom claim missing is rejected without fallback",
			oidcAuthCfg: &config.OIDCAuthConfig{
				UIDClaim: "sub",
			},
			claims:            map[string]any{},
			preferredUsername: "alice",
			email:             "alice@example.com",
			want:              "",
			wantErr:           true,
		},
		{
			name: "custom claim non string is rejected without fallback",
			oidcAuthCfg: &config.OIDCAuthConfig{
				UIDClaim: "sub",
			},
			claims: map[string]any{
				"sub": 42,
			},
			preferredUsername: "alice",
			email:             "alice@example.com",
			want:              "",
			wantErr:           true,
		},
		{
			name:        "missing default identity is rejected",
			oidcAuthCfg: &config.OIDCAuthConfig{},
			claims:      map[string]any{"sub": "valid-subject"},
			wantErr:     true,
		},
		{
			name:              "blank preferred username is rejected without switching to email",
			oidcAuthCfg:       &config.OIDCAuthConfig{},
			preferredUsername: " \t\n",
			email:             "alice@example.com",
			wantErr:           true,
		},
		{
			name:        "blank fallback email is rejected",
			oidcAuthCfg: &config.OIDCAuthConfig{},
			email:       " \t\n",
			wantErr:     true,
		},
		{
			name:              "empty configured claim is rejected without fallback",
			oidcAuthCfg:       &config.OIDCAuthConfig{UIDClaim: "tenant_uid"},
			claims:            map[string]any{"tenant_uid": ""},
			preferredUsername: "alice",
			email:             "alice@example.com",
			wantErr:           true,
		},
		{
			name:        "whitespace configured claim is rejected",
			oidcAuthCfg: &config.OIDCAuthConfig{UIDClaim: "tenant_uid"},
			claims:      map[string]any{"tenant_uid": " \t\n"},
			wantErr:     true,
		},
		{
			name:        "padded configured identity is rejected rather than normalized",
			oidcAuthCfg: &config.OIDCAuthConfig{UIDClaim: "tenant_uid"},
			claims:      map[string]any{"tenant_uid": " alice "},
			wantErr:     true,
		},
		{
			name:              "padded default username is rejected without email fallback",
			oidcAuthCfg:       &config.OIDCAuthConfig{},
			preferredUsername: " alice",
			email:             "alice@example.com",
			wantErr:           true,
		},
		{
			name:        "padded fallback email is rejected",
			oidcAuthCfg: &config.OIDCAuthConfig{},
			email:       "alice@example.com\t",
			wantErr:     true,
		},
		{
			name:        "unicode padded admin identity is rejected",
			oidcAuthCfg: &config.OIDCAuthConfig{UIDClaim: "tenant_uid"},
			claims:      map[string]any{"tenant_uid": "\u00a0admin\u00a0"},
			wantErr:     true,
		},
		{
			name:        "internal spaces are preserved",
			oidcAuthCfg: &config.OIDCAuthConfig{UIDClaim: "tenant_uid"},
			claims:      map[string]any{"tenant_uid": "alice smith"},
			want:        "alice smith",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildOIDCUID(tt.oidcAuthCfg, tt.claims, tt.preferredUsername, tt.email)
			if (err != nil) != tt.wantErr {
				t.Fatalf("buildOIDCUID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("buildOIDCUID() = %v, want %v", got, tt.want)
			}
		})
	}
}
