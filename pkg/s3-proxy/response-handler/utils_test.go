package responsehandler

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func Test_handler_manageHeaders(t *testing.T) {
	type args struct {
		helpersContent string
		headersTpl     map[string]string
	}
	tests := []struct {
		name        string
		args        args
		want        map[string]string
		wantErr     bool
		errorString string
	}{
		{
			name: "error in first rendering",
			args: args{
				helpersContent: "",
				headersTpl: map[string]string{
					"h1": "{{ .WontWork }}",
				},
			},
			wantErr:     true,
			errorString: "template: template-string-loaded:2:3: executing \"template-string-loaded\" at <.WontWork>: can't evaluate field WontWork in type *responsehandler.headerData",
		},
		{
			name: "error in second rendering",
			args: args{
				helpersContent: "",
				headersTpl: map[string]string{
					"h1": "{{ .Request.Method }}",
					"h2": "{{ .WontWork }}",
				},
			},
			wantErr:     true,
			errorString: "template: template-string-loaded:2:3: executing \"template-string-loaded\" at <.WontWork>: can't evaluate field WontWork in type *responsehandler.headerData",
		},
		{
			name: "clean new lines",
			args: args{
				helpersContent: "",
				headersTpl: map[string]string{
					"h1": `
{{ .Request.Method }}
`,
				},
			},
			want: map[string]string{
				"h1": "GET",
			},
		},
		{
			name: "use helpers",
			args: args{
				helpersContent: `
{{- define "fnc" -}}
{{- .Request.Method -}}
{{- end -}}
`,
				headersTpl: map[string]string{
					"h1": "{{ template \"fnc\" . }}",
				},
			},
			want: map[string]string{
				"h1": "GET",
			},
		},
		{
			name: "multiple headers and use helpers",
			args: args{
				helpersContent: `
{{- define "fnc" -}}
{{- .Request.Method -}}
{{- end -}}
`,
				headersTpl: map[string]string{
					"h1": "{{ template \"fnc\" . }}",
					"h2": "{{ template \"fnc\" . }}-{{ .Request.Host }}",
				},
			},
			want: map[string]string{
				"h1": "GET",
				"h2": "GET-fake.com",
			},
		},
		{
			name: "fixed header",
			args: args{
				helpersContent: "",
				headersTpl: map[string]string{
					"h1": "fixed",
				},
			},
			want: map[string]string{
				"h1": "fixed",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fake request
			req := httptest.NewRequest("GET", "http://fake.com", nil)

			h := &handler{
				req: req,
			}
			got, err := h.manageHeaders(tt.args.helpersContent, tt.args.headersTpl)
			if (err != nil) != tt.wantErr {
				t.Errorf("handler.manageHeaders() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && err.Error() != tt.errorString {
				t.Errorf("handler.manageHeaders() error = %v, wantErr %v", err, tt.errorString)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("handler.manageHeaders() = %v, want %v", got, tt.want)
			}
		})
	}
}
