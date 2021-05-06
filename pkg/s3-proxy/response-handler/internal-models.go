package responsehandler

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/authx/models"
)

// folderListingData Folder listing data for templating.
type folderListingData struct {
	Request    *http.Request
	User       models.GenericUser
	Entries    []*Entry
	BucketName string
	Name       string
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
	Targets map[string]interface{}
}

// headerData represents the structure used by header templating.
type headerData struct {
	Request *http.Request
	User    models.GenericUser
}

// streamFileHeaderData represents the structure used by stream file header templating.
type streamFileHeaderData struct {
	Request    *http.Request
	User       models.GenericUser
	StreamFile *StreamInput
}
