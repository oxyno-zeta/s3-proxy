package middlewares

import (
	"net/http"
	"regexp"
	"testing"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

func Test_isAuthorized(t *testing.T) {
	type args struct {
		groups                []string
		email                 string
		authorizationAccesses []*config.OIDCAuthorizationAccess
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should be authorized because no authorizations are present (no user groups or email)",
			args: args{
				groups:                make([]string, 0),
				email:                 "",
				authorizationAccesses: make([]*config.OIDCAuthorizationAccess, 0),
			},
			want: true,
		},
		{
			name: "should be authorized because no authorizations are present (no email)",
			args: args{
				groups:                []string{"group1"},
				email:                 "",
				authorizationAccesses: make([]*config.OIDCAuthorizationAccess, 0),
			},
			want: true,
		},
		{
			name: "should be authorized because no authorizations are present (no user groups)",
			args: args{
				groups:                make([]string, 0),
				email:                 "email@test.test",
				authorizationAccesses: make([]*config.OIDCAuthorizationAccess, 0),
			},
			want: true,
		},
		{
			name: "shouldn't be authorized if group isn't in authorized access list",
			args: args{
				groups: []string{"test"},
				email:  "email@test.test",
				authorizationAccesses: []*config.OIDCAuthorizationAccess{
					&config.OIDCAuthorizationAccess{Group: "valid1"},
				},
			},
			want: false,
		},
		{
			name: "should be authorized if group is in authorized access list",
			args: args{
				groups: []string{"valid2"},
				email:  "email@test.test",
				authorizationAccesses: []*config.OIDCAuthorizationAccess{
					&config.OIDCAuthorizationAccess{Group: "valid1"},
					&config.OIDCAuthorizationAccess{Group: "valid2"},
				},
			},
			want: true,
		},
		{
			name: "should be authorized if group is in authorized access list (2)",
			args: args{
				groups: []string{"valid2"},
				email:  "email@test.test",
				authorizationAccesses: []*config.OIDCAuthorizationAccess{
					&config.OIDCAuthorizationAccess{Email: "valid@test.test"},
					&config.OIDCAuthorizationAccess{Group: "valid1"},
					&config.OIDCAuthorizationAccess{Group: "valid2"},
				},
			},
			want: true,
		},
		{
			name: "shouldn't be authorized if email isn't in authorized access list",
			args: args{
				groups: make([]string, 0),
				email:  "email@test.test",
				authorizationAccesses: []*config.OIDCAuthorizationAccess{
					&config.OIDCAuthorizationAccess{Email: "valid@test.test"},
				},
			},
			want: false,
		},
		{
			name: "should be authorized if email is in authorized access list",
			args: args{
				groups: []string{"valid2"},
				email:  "email@test.test",
				authorizationAccesses: []*config.OIDCAuthorizationAccess{
					&config.OIDCAuthorizationAccess{Email: "email@test.test"},
				},
			},
			want: true,
		},
		{
			name: "should be authorized if email is in authorized access list (2)",
			args: args{
				groups: []string{"valid2"},
				email:  "email@test.test",
				authorizationAccesses: []*config.OIDCAuthorizationAccess{
					&config.OIDCAuthorizationAccess{Email: "email@test.test"},
					&config.OIDCAuthorizationAccess{Group: "valid1"},
					&config.OIDCAuthorizationAccess{Group: "valid2"},
				},
			},
			want: true,
		},
		{
			name: "shouldn't be authorized if group regexp isn't valid",
			args: args{
				groups: []string{"test"},
				email:  "email@test.test",
				authorizationAccesses: []*config.OIDCAuthorizationAccess{
					&config.OIDCAuthorizationAccess{
						Regexp:      true,
						Group:       "valid.*",
						GroupRegexp: regexp.MustCompile("valid.*"),
					},
				},
			},
			want: false,
		},
		{
			name: "should be authorized if group regexp is valid",
			args: args{
				groups: []string{"test", "valid2"},
				email:  "email@test.test",
				authorizationAccesses: []*config.OIDCAuthorizationAccess{
					&config.OIDCAuthorizationAccess{
						Regexp:      true,
						Group:       "valid.*",
						GroupRegexp: regexp.MustCompile("valid.*"),
					},
				},
			},
			want: true,
		},
		{
			name: "shouldn't be authorized if email regexp isn't valid",
			args: args{
				groups: make([]string, 0),
				email:  "email@test.test",
				authorizationAccesses: []*config.OIDCAuthorizationAccess{
					&config.OIDCAuthorizationAccess{
						Regexp:      true,
						Email:       ".*@valid.test",
						EmailRegexp: regexp.MustCompile(".*@valid.test"),
					},
				},
			},
			want: false,
		},
		{
			name: "should be authorized if email regexp is valid",
			args: args{
				groups: make([]string, 0),
				email:  "email@valid.test",
				authorizationAccesses: []*config.OIDCAuthorizationAccess{
					&config.OIDCAuthorizationAccess{
						Regexp:      true,
						Email:       ".*@valid.test",
						EmailRegexp: regexp.MustCompile(".*@valid.test"),
					},
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isAuthorized(tt.args.groups, tt.args.email, tt.args.authorizationAccesses); got != tt.want {
				t.Errorf("isAuthorized() = %v, want %v", got, tt.want)
			}
		})
	}
}

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
		logEntry   logrus.FieldLogger
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
				logEntry: &logrus.Logger{},
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
				logEntry: &logrus.Logger{},
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
				logEntry: &logrus.Logger{},
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
				logEntry: &logrus.Logger{},
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
				logEntry: &logrus.Logger{},
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
				logEntry: &logrus.Logger{},
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
				logEntry: &logrus.Logger{},
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
			got, err := getJWTToken(tt.args.logEntry, tt.args.r, tt.args.cookieName)
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
