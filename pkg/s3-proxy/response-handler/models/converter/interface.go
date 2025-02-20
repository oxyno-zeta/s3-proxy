package converter

import (
	"net/http"
	"net/url"

	"github.com/microcosm-cc/bluemonday"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler/models"
)

//go:generate goverter gen .

// goverter:variables
// goverter:output:format assign-variable
// goverter:output:file ./generated.go
// goverter:output:package github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler/models/converter
// goverter:extend sanitizeString
var (
	ConvertAndSanitizeHTTPRequest func(source *http.Request) *models.LightSanitizedRequest
	// goverter:map User | sanitizeURLUserInfo
	sanitizeURL func(source *url.URL) *url.URL
)

// goverter:variables
// goverter:output:format assign-variable
// goverter:output:file ./generated.go
// goverter:output:package github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/response-handler/models/converter
var (
	// goverter:map URL | identity
	// goverter:map Header | identity
	// goverter:map Trailer | identity
	// goverter:map TransferEncoding | identity
	// goverter:ignore Body
	// goverter:ignore GetBody
	// goverter:ignore Close
	// goverter:ignore Form
	// goverter:ignore PostForm
	// goverter:ignore MultipartForm
	// goverter:ignore TLS
	// goverter:ignore Cancel
	// goverter:ignore Response
	// goverter:ignore ctx
	// goverter:ignore pat
	// goverter:ignore matches
	// goverter:ignore otherValues
	ConvertSanitizedToHTTPRequest func(source *models.LightSanitizedRequest) *http.Request
)

// Need to do this by hand since fields are private
// Note: Only manage username to avoid potential password leak.
func sanitizeURLUserInfo(user *url.Userinfo) *url.Userinfo {
	// Check nil case
	if user == nil {
		return nil
	}

	return url.User(sanitizeString(user.Username()))
}

func identity[T any](input T) T {
	return input
}

func sanitizeString(s string) string {
	return bluemonday.StrictPolicy().Sanitize(s)
}
