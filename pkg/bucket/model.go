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
	s3Context                 s3client.S3ContextInterface
	logger                    logrus.FieldLogger
	bucketInstance            *config.Target
	tplConfig                 *config.TemplateConfig
	mountPath                 string
	requestPath               string
	httpRW                    http.ResponseWriter
	handleNotFound            func(rw http.ResponseWriter, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
	handleInternalServerError func(rw http.ResponseWriter, err error, requestPath string, logger logrus.FieldLogger, tplCfg *config.TemplateConfig)
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
