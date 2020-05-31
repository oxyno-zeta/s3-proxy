package authorization

import (
	"regexp"
	"testing"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
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
					{Group: "valid1"},
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
					{Group: "valid1"},
					{Group: "valid2"},
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
					{Email: "valid@test.test"},
					{Group: "valid1"},
					{Group: "valid2"},
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
					{Email: "valid@test.test"},
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
					{Email: "email@test.test"},
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
					{Email: "email@test.test"},
					{Group: "valid1"},
					{Group: "valid2"},
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
					{
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
					{
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
					{
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
					{
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
