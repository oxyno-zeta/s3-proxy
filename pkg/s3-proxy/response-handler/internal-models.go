package responsehandler

import (
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

// bucketListingData Bucket listing data for templating.
type bucketListingData struct {
	Request    *requestData
	Entries    []*Entry
	BucketName string
	Name       string
	Path       string // Deprecated
}

// errorData represents the structure used by error templating.
type errorData struct {
	Request *requestData
	Path    string
	Error   error
}

// targetListData represents the structure used by target list templating.
type targetListData struct {
	Request *requestData
	Targets map[string]*config.TargetConfig
}

// requestData represents the structure of a request.
type requestData struct {
	Method        string
	Proto         string
	ProtoMajor    int
	ProtoMinor    int
	Headers       map[string]interface{}
	ContentLength int64
	Close         bool
	Host          string
	RemoteAddr    string
	RequestURI    string
	URL           *urlData
}

// urlData represents the structure of an url structure in a request.
type urlData struct { //nolint:maligned // Ignored for visibility
	Scheme      string
	Opaque      string
	Host        string
	Path        string
	RawPath     string
	ForceQuery  bool
	RawQuery    string
	Fragment    string
	RawFragment string
	// After this line, values are coming from functions
	IsAbs           bool
	EscapedFragment string
	EscapedPath     string
	Port            string
	RequestURI      string
	String          string
	QueryParams     map[string]interface{}
}
