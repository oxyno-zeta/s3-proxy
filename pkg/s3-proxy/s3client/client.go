package s3client

import (
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/tracing"
)

// Client S3 Context interface.
type Client interface {
	ListFilesAndDirectories(key string) ([]*ListElementOutput, error)
	HeadObject(key string) (*HeadOutput, error)
	GetObject(key string) (*GetOutput, error)
	PutObject(input *PutInput) error
	DeleteObject(key string) error
}

// FileType File type.
const FileType = "FILE"

// FolderType Folder type.
const FolderType = "FOLDER"

// ListElementOutput Bucket ListElementOutput.
type ListElementOutput struct {
	Type         string
	ETag         string
	Name         string
	LastModified time.Time
	Size         int64
	Key          string
}

// HeadOutput represents output of Head.
type HeadOutput struct {
	Type string
	Key  string
}

// ErrNotFound Error not found.
var ErrNotFound = errors.New("not found")

// GetOutput Object output for S3 get object.
type GetOutput struct {
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

// PutInput Put input object for PUT request.
type PutInput struct {
	Key          string
	Body         io.ReadSeeker
	ContentType  string
	ContentSize  int64
	Metadata     map[string]string
	StorageClass string
}

// NewS3Context New S3 Context.
func NewS3Context(tgt *config.TargetConfig, logger log.Logger, metricsCtx metrics.Client, parentTrace tracing.Trace) (Client, error) {
	sessionConfig := &aws.Config{
		Region: aws.String(tgt.Bucket.Region),
	}
	// Load credentials if they exists
	if tgt.Bucket.Credentials != nil && tgt.Bucket.Credentials.AccessKey != nil && tgt.Bucket.Credentials.SecretKey != nil {
		sessionConfig.Credentials = credentials.NewStaticCredentials(tgt.Bucket.Credentials.AccessKey.Value, tgt.Bucket.Credentials.SecretKey.Value, "")
	}
	// Load custom endpoint if it exists
	if tgt.Bucket.S3Endpoint != "" {
		sessionConfig.Endpoint = aws.String(tgt.Bucket.S3Endpoint)
		sessionConfig.S3ForcePathStyle = aws.Bool(true)
	}
	// Check if ssl needs to be disabled
	if tgt.Bucket.DisableSSL {
		sessionConfig.DisableSSL = aws.Bool(true)
	}
	// Create session
	sess, err := session.NewSession(sessionConfig)
	if err != nil {
		return nil, err
	}
	// Create s3 client
	svcClient := s3.New(sess)

	return &s3Context{
		svcClient:   svcClient,
		logger:      logger,
		target:      tgt,
		metricsCtx:  metricsCtx,
		parentTrace: parentTrace,
	}, nil
}
