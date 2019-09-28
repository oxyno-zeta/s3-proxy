package s3client

import (
	"errors"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

type S3Context struct {
	svcClient    *s3.S3
	BucketConfig *config.BucketConfig
	logger       *logrus.FieldLogger
}

const FileType = "FILE"
const FolderType = "FOLDER"

type Entry struct {
	Type         string
	ETag         string
	Name         string
	LastModified time.Time
	Size         int64
	Key          string
}

var ErrNotFound = errors.New("not found")
