package s3client

import (
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

type S3Context struct {
	svcClient      *s3.S3
	BucketInstance *config.BucketInstance
	logger         *logrus.FieldLogger
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

type ObjectOutput struct {
	Body               *io.ReadCloser
	CacheControl       string
	Expires            string
	ContentDisposition string
	ContentEncoding    string
	ContentLanguage    string
	ContentLength      int64
	ContentRange       string
	ContentType        string
	ETag               string
	LastModified       time.Time
}
