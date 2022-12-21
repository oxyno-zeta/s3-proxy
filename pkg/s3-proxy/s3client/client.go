package s3client

import (
	"context"
	"io"
	"time"

	"emperror.dev/errors"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
)

// Manager S3 client manager.
//
//go:generate mockgen -destination=./mocks/mock_Manager.go -package=mocks github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client Manager
type Manager interface {
	// GetClientForTarget will return a S3 client for a target.
	GetClientForTarget(name string) Client
	// Load will load all S3 clients.
	Load() error
}

// Client S3 Context interface.
//
//go:generate mockgen -destination=./mocks/mock_Client.go -package=mocks github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/s3client Client
type Client interface {
	// ListFilesAndDirectories will list files and directories in S3.
	ListFilesAndDirectories(ctx context.Context, key string) ([]*ListElementOutput, *ResultInfo, error)
	// HeadObject will head a key.
	HeadObject(ctx context.Context, key string) (*HeadOutput, error)
	// GetObject will get an object.
	GetObject(ctx context.Context, input *GetInput) (*GetOutput, *ResultInfo, error)
	// PutObject will put an object.
	PutObject(ctx context.Context, input *PutInput) (*ResultInfo, error)
	// DeleteObject will delete an object.
	DeleteObject(ctx context.Context, key string) (*ResultInfo, error)
	// GetObjectSignedURL will return a signed url for a get object.
	GetObjectSignedURL(ctx context.Context, input *GetInput, expiration time.Duration) (string, error)
}

// ResultInfo ResultInfo structure.
type ResultInfo struct {
	Bucket     string
	Key        string
	Region     string
	S3Endpoint string
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
	Metadata           map[string]string
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
