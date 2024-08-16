package authorization

import (
	"regexp"
	"testing"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

func Test_isHeaderOIDCAuthorizedBasic(t *testing.T) {
	type args struct {
		groups                []string
		email                 string
		authorizationAccesses []*config.HeaderOIDCAuthorizationAccess
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
				authorizationAccesses: make([]*config.HeaderOIDCAuthorizationAccess, 0),
			},
			want: true,
		},
		{
			name: "should be authorized because no authorizations are present (no email)",
			args: args{
				groups:                []string{"group1"},
				email:                 "",
				authorizationAccesses: make([]*config.HeaderOIDCAuthorizationAccess, 0),
			},
			want: true,
		},
		{
			name: "should be authorized because no authorizations are present (no user groups)",
			args: args{
				groups:                make([]string, 0),
				email:                 "email@test.test",
				authorizationAccesses: make([]*config.HeaderOIDCAuthorizationAccess, 0),
			},
			want: true,
		},
		{
			name: "shouldn't be authorized if group isn't in authorized access list",
			args: args{
				groups: []string{"test"},
				email:  "email@test.test",
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
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
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
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
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
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
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
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
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
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
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
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
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
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
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
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
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
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
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
					{
						Regexp:      true,
						Email:       ".*@valid.test",
						EmailRegexp: regexp.MustCompile(".*@valid.test"),
					},
				},
			},
			want: true,
		},
		{
			name: "should be forbidden if email regexp is matching but forbidden",
			args: args{
				groups: make([]string, 0),
				email:  "email@valid.test",
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
					{
						Regexp:      true,
						Email:       ".*@valid.test",
						EmailRegexp: regexp.MustCompile(".*@valid.test"),
						Forbidden:   true,
					},
				},
			},
			want: false,
		},
		{
			name: "should be forbidden if email regexp is matching but forbidden but second have ok for groups",
			args: args{
				groups: []string{"grp1"},
				email:  "email@valid.test",
				authorizationAccesses: []*config.HeaderOIDCAuthorizationAccess{
					{
						Regexp:      true,
						Email:       ".*@valid.test",
						EmailRegexp: regexp.MustCompile(".*@valid.test"),
						Forbidden:   true,
					},
					{
						Regexp: true,
						Group:  "grp1",
					},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isHeaderOIDCAuthorizedBasic(tt.args.groups, tt.args.email, tt.args.authorizationAccesses); got != tt.want {
				t.Errorf("isHeaderOIDCAuthorizedBasic() = %v, want %v", got, tt.want)
			}
		})
	}
}
