// +build unit

package utils

import (
	"net/http"
	"testing"
)

func TestGetRequestURI(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:989/fake/path", nil)
	if err != nil {
		t.Fatal(err)
	}

	want := "http://localhost:989/fake/path"
	got := GetRequestURI(req)
	if got != want {
		t.Errorf("GetRequestURI() = %v, want %v", got, want)
	}
}

func TestProxiedGetRequestURI(t *testing.T) {
	req, err := http.NewRequest("GET", "http://localhost:989/fake/path", nil)
	if err != nil {
		t.Fatal(err)
	}
	// Add the same header a Load Balancer should set
	req.Header.Set("X-Forwarded-Proto", "https")

	want := "https://localhost:989/fake/path"
	got := GetRequestURI(req)
	if got != want {
		t.Errorf("GetRequestURI() = %v, want %v", got, want)
	}
}

func Test_GetRequestHost(t *testing.T) {
	hXForwardedHost1 := http.Header{
		"X-Forwarded-Host": []string{"fake.host"},
	}
	hXForwardedHost2 := http.Header{
		"X-Forwarded-Host": []string{"fake.host:9090"},
	}
	hXForwarded := http.Header{
		"Forwarded": []string{"for=192.0.2.60;proto=http;by=203.0.113.43;host=fake.host:9090"},
	}

	tests := []struct {
		name     string
		headers  http.Header
		inputURL string
		want     string
	}{
		{
			name:     "request host",
			headers:  nil,
			inputURL: "http://request.host/",
			want:     "request.host",
		},
		{
			name:     "forwarded host",
			headers:  hXForwarded,
			inputURL: "http://fake.host:9090/",
			want:     "fake.host:9090",
		},
		{
			name:     "x-forwarded host 1",
			headers:  hXForwardedHost1,
			inputURL: "http://fake.host/",
			want:     "fake.host",
		},
		{
			name:     "x-forwarded host 2",
			headers:  hXForwardedHost2,
			inputURL: "http://fake.host:9090/",
			want:     "fake.host:9090",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			req, err := http.NewRequest("GET", tt.inputURL, nil)
			if err != nil {
				t.Fatal(err)
			}
			if tt.headers != nil {
				req.Header = tt.headers
			}

			if got := GetRequestHost(req); got != tt.want {
				t.Errorf("RequestHost() = %v, want %v", got, tt.want)
			}
		})
	}
}
