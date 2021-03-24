package s3client

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
)

// Manager S3 client manager.
//go:generate mockgen -destination=./mocks/mock_Manager.go -package=mocks github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client Manager
type Manager interface {
	// GetClientForTarget will return a S3 client for a target.
	GetClientForTarget(name string) Client
	// Load will load all S3 clients.
	Load() error
}

// Client S3 Context interface.
//go:generate mockgen -destination=./mocks/mock_Client.go -package=mocks github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client Client
type Client interface {
	ListFilesAndDirectories(ctx context.Context, key string) ([]*ListElementOutput, error)
	HeadObject(ctx context.Context, key string) (*HeadOutput, error)
	GetObject(ctx context.Context, input *GetInput) (*GetOutput, error)
	PutObject(ctx context.Context, input *PutInput) error
	DeleteObject(ctx context.Context, key string) error
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

// ErrNotModified Error not modified.
var ErrNotModified = errors.New("not modified")

// ErrPreconditionFailed Error precondition failed.
var ErrPreconditionFailed = errors.New("precondition failed")

// GetInput Input object for get requests.
type GetInput struct {
	Key               string
	IfModifiedSince   *time.Time
	IfMatch           string
	IfNoneMatch       string
	IfUnmodifiedSince *time.Time
	Range             string
}

// GetOutput Object output for S3 get object.
type GetOutput struct {
	Body               io.ReadCloser
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

// NewManager will return a new S3 client manager.
func NewManager(cfgManager config.Manager, metricsCl metrics.Client) Manager {
	return &manager{
		targetClient: map[string]Client{},
		cfgManager:   cfgManager,
		metricCl:     metricsCl,
	}
}

func newClient(tgt *config.TargetConfig, metricsCtx metrics.Client) (Client, error) {
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
		svcClient:  svcClient,
		target:     tgt,
		metricsCtx: metricsCtx,
	}, nil
}
