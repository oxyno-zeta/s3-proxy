package responsehandler

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

// bucketListingData Bucket listing data for templating.
type bucketListingData struct {
	Request    *http.Request
	User       models.GenericUser
	Entries    []*Entry
	BucketName string
	Name       string
	Path       string
}

// errorData represents the structure used by error templating.
type errorData struct {
	Request *http.Request
	User    models.GenericUser
	Error   error
}

// targetListData represents the structure used by target list templating.
type targetListData struct {
	Request *http.Request
	User    models.GenericUser
	Targets map[string]*config.TargetConfig
}

// headerData represents the structure used by header templating.
type headerData struct {
	Request *http.Request
	User    models.GenericUser
}
