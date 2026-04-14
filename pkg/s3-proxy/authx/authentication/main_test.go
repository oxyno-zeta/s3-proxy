//go:build unit

package authentication

import (
	"reflect"
	"testing"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

func Test_findResource(t *testing.T) {
	type args struct {
		resL       []*config.Resource
		requestURI string
		httpMethod string
	}
	tests := []struct {
		name    string
		args    args
		want    *config.Resource
		wantErr bool
	}{
		{
			name: "Empty resource list",
			args: args{
				resL:       nil,
				requestURI: "/test",
				httpMethod: "GET",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Should find a valid resource with fixed path",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/test",
						Methods: []string{"GET"},
					},
				},
				requestURI: "/test",
				httpMethod: "GET",
			},
			want: &config.Resource{
				Path:    "/test",
				Methods: []string{"GET"},
			},
			wantErr: false,
		},
		{
			name: "Wildcard * at end matches a single segment",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/test/*",
						Methods: []string{"GET"},
					},
				},
				requestURI: "/test/fake",
				httpMethod: "GET",
			},
			want: &config.Resource{
				Path:    "/test/*",
				Methods: []string{"GET"},
			},
			wantErr: false,
		},
		{
			name: "Wildcard * at end does not match multiple segments",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/test/*",
						Methods: []string{"GET"},
					},
				},
				requestURI: "/test/fake/fake",
				httpMethod: "GET",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Wildcard * between segments matches exactly one segment",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/single/*/segment/",
						Methods: []string{"GET"},
					},
				},
				requestURI: "/single/foo/segment/",
				httpMethod: "GET",
			},
			want: &config.Resource{
				Path:    "/single/*/segment/",
				Methods: []string{"GET"},
			},
			wantErr: false,
		},
		{
			name: "Wildcard * between segments does not match multiple segments",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/single/*/segment/",
						Methods: []string{"GET"},
					},
				},
				requestURI: "/single/foo/bar/segment/",
				httpMethod: "GET",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Wildcard ** at end matches a single segment",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/test/**",
						Methods: []string{"GET"},
					},
				},
				requestURI: "/test/fake",
				httpMethod: "GET",
			},
			want: &config.Resource{
				Path:    "/test/**",
				Methods: []string{"GET"},
			},
			wantErr: false,
		},
		{
			name: "Wildcard ** at end matches multiple segments",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/test/**",
						Methods: []string{"GET"},
					},
				},
				requestURI: "/test/fake/fake",
				httpMethod: "GET",
			},
			want: &config.Resource{
				Path:    "/test/**",
				Methods: []string{"GET"},
			},
			wantErr: false,
		},
		{
			name: "Wildcard ** between segments matches a single intermediate segment",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/multiple/**/segments/",
						Methods: []string{"GET"},
					},
				},
				requestURI: "/multiple/foo/segments/",
				httpMethod: "GET",
			},
			want: &config.Resource{
				Path:    "/multiple/**/segments/",
				Methods: []string{"GET"},
			},
			wantErr: false,
		},
		{
			name: "Wildcard ** between segments matches multiple intermediate segments",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/multiple/**/segments/",
						Methods: []string{"GET"},
					},
				},
				requestURI: "/multiple/foo/bar/segments/",
				httpMethod: "GET",
			},
			want: &config.Resource{
				Path:    "/multiple/**/segments/",
				Methods: []string{"GET"},
			},
			wantErr: false,
		},
		{
			name: "Wildcard ** between segments does not match when suffix differs",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/multiple/**/segments/",
						Methods: []string{"GET"},
					},
				},
				requestURI: "/multiple/foo/other/",
				httpMethod: "GET",
			},
			want:    nil,
			wantErr: false,
		},
		{
			name: "Shouldn't find a valid resource with valid glob path but not http method",
			args: args{
				resL: []*config.Resource{
					{
						Path:    "/test/*",
						Methods: []string{"GET", "PUT"},
					},
				},
				requestURI: "/test/fake",
				httpMethod: "POST",
			},
			want:    nil,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := findResource(tt.args.resL, tt.args.requestURI, tt.args.httpMethod)
			if (err != nil) != tt.wantErr {
				t.Errorf("findResource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("findResource() = %v, want %v", got, tt.want)
			}
		})
	}
}
