package bucket

import (
	"net/http"

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

// EntryPath Entry with path
type EntryPath struct {
	Entry *s3client.Entry
	Path  string
}

// bucketListingData Bucket listing data for templating
type bucketListingData struct {
	Entries    []*EntryPath
	BucketName string
	Name       string
	Path       string
}
