package bucket

import (
	"net/http"

	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3client"
	"github.com/sirupsen/logrus"
)

type BucketRequestContext struct {
	s3Context      *s3client.S3Context
	logger         *logrus.FieldLogger
	bucketInstance *config.BucketInstance
	mountPath      string
	requestPath    string
	httpRW         *http.ResponseWriter
}

type EntryPath struct {
	Entry *s3client.Entry
	Path  string
}

type BucketListingData struct {
	Entries    []*EntryPath
	BucketName string
	Name       string
	Path       string
}
