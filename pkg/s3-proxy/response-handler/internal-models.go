package responsehandler

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

// bucketListingData Bucket listing data for templating.
type bucketListingData struct {
	Request    *http.Request
	Entries    []*Entry
	BucketName string
	Name       string
	Path       string // Deprecated
}

// errorData represents the structure used by error templating.
type errorData struct {
	Request *http.Request
	Path    string
	Error   error
}

// targetListData represents the structure used by target list templating.
type targetListData struct {
	Request *http.Request
	Targets map[string]*config.TargetConfig
}
