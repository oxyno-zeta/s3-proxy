//go:build unit

package converter

import (
	"net/http"
	"net/url"
	"testing"

	models "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler/models"
	"github.com/stretchr/testify/assert"
)

func TestConvertAndSanitizeHttpRequest(t *testing.T) {
	type args struct {
		buildSource func() (*http.Request, error)
	}
	tests := []struct {
		name string
		args args
		want *models.LightSanitizedRequest
	}{
		{
			name: "sanitize url path with img onerror",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/<img src="x" onerror="alert(1)">`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL: u,
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "/",
				},
			},
		},
		{
			name: "sanitize url path with script",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/text-fake<script src="http://fake.com/fake.js"/>`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL: u,
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/text-fake",
					RawPath: "/text-fake",
				},
			},
		},
		{
			name: "sanitize method with script only",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:    u,
						Method: `<script src="http://fake.com/fake.js"/>`,
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
			},
		},
		{
			name: "sanitize method with script and method",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:    u,
						Method: `GET<script src="http://fake.com/fake.js"/>`,
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				Method: "GET",
			},
		},
		{
			name: "sanitize proto with script and proto",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:    u,
						Method: "GET",
						Proto:  `HTTP1/1<script src="http://fake.com/fake.js"/>`,
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				Method: "GET",
				Proto:  "HTTP1/1",
			},
		},
		{
			name: "sanitize headers with script only has first value",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:    u,
						Method: "GET",
						Header: http.Header{
							"FAKE": []string{`<script src="http://fake.com/fake.js"/>`},
						},
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				Method: "GET",
				Header: http.Header{
					"FAKE": []string{""},
				},
			},
		},
		{
			name: "sanitize headers with script and value has first value",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:    u,
						Method: "GET",
						Header: http.Header{
							"FAKE": []string{`fake<script src="http://fake.com/fake.js"/>`},
						},
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				Method: "GET",
				Header: http.Header{
					"FAKE": []string{"fake"},
				},
			},
		},
		{
			name: "sanitize headers with script only has second value",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:    u,
						Method: "GET",
						Header: http.Header{
							"FAKE": []string{"fake", `<script src="http://fake.com/fake.js"/>`},
						},
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				Method: "GET",
				Header: http.Header{
					"FAKE": []string{"fake", ""},
				},
			},
		},
		{
			name: "sanitize headers with script and value has second value",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:    u,
						Method: "GET",
						Header: http.Header{
							"FAKE": []string{"fake", `fake2<script src="http://fake.com/fake.js"/>`},
						},
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				Method: "GET",
				Header: http.Header{
					"FAKE": []string{"fake", "fake2"},
				},
			},
		},
		{
			name: "sanitize headers with script only has first value on second header",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:    u,
						Method: "GET",
						Header: http.Header{
							"OK":   []string{"fake"},
							"FAKE": []string{`<script src="http://fake.com/fake.js"/>`},
						},
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				Method: "GET",
				Header: http.Header{
					"OK":   []string{"fake"},
					"FAKE": []string{""},
				},
			},
		},
		{
			name: "sanitize headers with script and value has first value on second header",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:    u,
						Method: "GET",
						Header: http.Header{
							"EMAIL": []string{"fake@fake.com"},
							"FAKE":  []string{`fake<script src="http://fake.com/fake.js"/>`},
						},
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				Method: "GET",
				Header: http.Header{
					"EMAIL": []string{"fake@fake.com"},
					"FAKE":  []string{"fake"},
				},
			},
		},
		{
			name: "sanitize transfer encoding with script only",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:              u,
						TransferEncoding: []string{`<script src="http://fake.com/fake.js"/>`},
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				TransferEncoding: []string{""},
			},
		},
		{
			name: "sanitize transfer encoding with script and value",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:              u,
						TransferEncoding: []string{`fake<script src="http://fake.com/fake.js"/>`},
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				TransferEncoding: []string{"fake"},
			},
		},
		{
			name: "sanitize transfer encoding with p and value",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:              u,
						TransferEncoding: []string{`fake<p>fake</p>`},
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				TransferEncoding: []string{"fakefake"},
			},
		},
		{
			name: "sanitize trailers with script and value has first value on second header",
			args: args{
				buildSource: func() (*http.Request, error) {
					u, err := url.Parse(`http://fake.com/`)
					if err != nil {
						return nil, err
					}

					return &http.Request{
						URL:    u,
						Method: "GET",
						Trailer: http.Header{
							"OK":   []string{"fake"},
							"FAKE": []string{`fake<script src="http://fake.com/fake.js"/>`},
						},
					}, nil
				},
			},
			want: &models.LightSanitizedRequest{
				URL: &url.URL{
					Scheme:  "http",
					Host:    "fake.com",
					Path:    "/",
					RawPath: "",
				},
				Method: "GET",
				Trailer: http.Header{
					"OK":   []string{"fake"},
					"FAKE": []string{"fake"},
				},
			},
		},
	}
	t.Parallel()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build req
			req, err := tt.args.buildSource()

			if assert.NoError(t, err) {
				got := ConvertAndSanitizeHTTPRequest(req)

				assert.Equal(t, tt.want, got)
			}
		})
	}
}
