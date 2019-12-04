package s3client

import (
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/oxyno-zeta/s3-proxy/pkg/config"
	"github.com/sirupsen/logrus"
)

// S3ContextInterface S3 Context interface
type S3ContextInterface interface {
	ListFilesAndDirectories(string) ([]*Entry, error)
	GetObject(string) (*ObjectOutput, error)
}

type s3Context struct {
	svcClient *s3.S3
	Target    *config.Target
	logger    *logrus.FieldLogger
}

// FileType File type
const FileType = "FILE"

// FolderType Folder type
const FolderType = "FOLDER"

// Entry Bucket Entry
type Entry struct {
	Type         string
	ETag         string
	Name         string
	LastModified time.Time
	Size         int64
	Key          string
}

// ErrNotFound Error not found
var ErrNotFound = errors.New("not found")

// ObjectOutput Object output for S3 get object
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
