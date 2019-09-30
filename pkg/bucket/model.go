package bucket

import (
	"net/http"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
	"github.com/sirupsen/logrus"
)

// RequestContext Bucket request context
type RequestContext struct {
	s3Context      *s3client.S3Context
	logger         *logrus.FieldLogger
	bucketInstance *config.BucketInstance
	tplConfig      *config.TemplateConfig
	mountPath      string
	requestPath    string
	httpRW         *http.ResponseWriter
}

// Entry Entry with path
type Entry struct {
	Type         string
	ETag         string
	Name         string
	LastModified time.Time
	Size         int64
	Key          string
	Path         string
}

// bucketListingData Bucket listing data for templating
type bucketListingData struct {
	Entries    []*Entry
	BucketName string
	Name       string
	Path       string
}
