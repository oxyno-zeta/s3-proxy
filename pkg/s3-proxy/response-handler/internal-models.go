package responsehandler

import "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"

// bucketListingData Bucket listing data for templating.
type bucketListingData struct {
	Entries    []*Entry
	BucketName string
	Name       string
	Path       string
}

// errorData represents the structure used by error templating.
type errorData struct {
	Path  string
	Error error
}

// targetListData represents the structure used by target list templating.
type targetListData struct {
	Targets map[string]*config.TargetConfig
}
