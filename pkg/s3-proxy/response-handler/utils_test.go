//go:build unit

package responsehandler

import (
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/stretchr/testify/assert"
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
			errorString: "template: template-string-loaded:2:3: executing \"template-string-loaded\" at <.WontWork>: can't evaluate field WontWork in type *responsehandler.genericHeaderData",
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
			errorString: "template: template-string-loaded:2:3: executing \"template-string-loaded\" at <.WontWork>: can't evaluate field WontWork in type *responsehandler.genericHeaderData",
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
		{
			name: "empty header must be removed",
			args: args{
				helpersContent: "",
				headersTpl: map[string]string{
					"h1": "fixed",
					"h2": "",
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

			type genericHeaderData struct {
				Request *http.Request
				User    models.GenericUser
			}

			got, err := h.manageHeaders(tt.args.helpersContent, tt.args.headersTpl, &genericHeaderData{Request: req})
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

func Test_setHeadersFromObjectOutput(t *testing.T) {
	// Tests data
	now := time.Now()
	headerFullInput := http.Header{}
	headerFullInput.Add("Cache-Control", "cachecontrol")
	headerFullInput.Add("Expires", "expires")
	headerFullInput.Add("Content-Disposition", "contentdisposition")
	headerFullInput.Add("Content-Encoding", "contentencoding")
	headerFullInput.Add("Content-Language", "contentlanguage")
	headerFullInput.Add("Content-Length", "200")
	headerFullInput.Add("Content-Range", "bytes 200/200")
	headerFullInput.Add("Content-Type", "contenttype")
	headerFullInput.Add("ETag", "etag")
	headerFullInput.Add("Last-Modified", now.UTC().Format(http.TimeFormat))
	headerPartialInput := http.Header{}
	headerPartialInput.Add("Cache-Control", "cachecontrol")
	headerPartialInput.Add("Expires", "expires")
	headerPartialInput.Add("Content-Disposition", "contentdisposition")
	headerPartialInput.Add("Content-Encoding", "contentencoding")
	headerPartialInput.Add("Content-Language", "contentlanguage")
	headerPartialInput.Add("Content-Length", "200")
	headerPartialInput.Add("Content-Range", "bytes 200-1000/10000")
	headerPartialInput.Add("Content-Type", "contenttype")
	headerPartialInput.Add("ETag", "etag")
	headerPartialInput.Add("Last-Modified", now.UTC().Format(http.TimeFormat))
	// Structures
	type args struct {
		obj *StreamInput
	}
	tests := []struct {
		name            string
		args            args
		expectedCode    int
		expectedHeaders http.Header
	}{
		{
			name: "Empty input",
			args: args{
				obj: &StreamInput{},
			},
			expectedHeaders: http.Header{},
			expectedCode:    http.StatusOK,
		},
		{
			name: "Full input",
			args: args{
				obj: &StreamInput{
					CacheControl:       "cachecontrol",
					Expires:            "expires",
					ContentDisposition: "contentdisposition",
					ContentEncoding:    "contentencoding",
					ContentLanguage:    "contentlanguage",
					ContentLength:      200,
					ContentRange:       "bytes 200/200",
					ContentType:        "contenttype",
					ETag:               "etag",
					LastModified:       now,
				},
			},
			expectedHeaders: headerFullInput,
			expectedCode:    http.StatusOK,
		},
		{
			name: "Partial input",
			args: args{
				obj: &StreamInput{
					CacheControl:       "cachecontrol",
					Expires:            "expires",
					ContentDisposition: "contentdisposition",
					ContentEncoding:    "contentencoding",
					ContentLanguage:    "contentlanguage",
					ContentLength:      200,
					ContentRange:       "bytes 200-1000/10000",
					ContentType:        "contenttype",
					ETag:               "etag",
					LastModified:       now,
				},
			},
			expectedHeaders: headerPartialInput,
			expectedCode:    http.StatusPartialContent,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create response
			res := httptest.NewRecorder()

			setHeadersFromObjectOutput(res, tt.args.obj)

			assert.Equal(t, tt.expectedHeaders, res.HeaderMap)
			assert.Equal(t, tt.expectedCode, res.Code)
		})
	}
}
