package s3client

import (
	"net/http"

	"github.com/aws/aws-sdk-go-v2/aws"
)

type httpCl struct {
	httpClient aws.HTTPClient
	headers    map[string]string
}

func (h httpCl) Do(req *http.Request) (*http.Response, error) {
	// Add all custom headers
	for k, v := range h.headers {
		req.Header.Add(k, v)
	}

	return h.httpClient.Do(req)
}

func customizeHTTPClient(cli aws.HTTPClient, headers map[string]string) aws.HTTPClient {
	// Init original http client
	oriHC := cli
	// Check if it is nil
	if oriHC == nil {
		// Set global default http client
		oriHC = http.DefaultClient
	}

	// Create custom http client
	h := &httpCl{
		httpClient: oriHC,
		headers:    headers,
	}

	return h
}
