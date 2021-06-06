package webhook

import (
	"context"
	"time"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/metrics"
)

// PutInputMetadata Put input metadata.
type PutInputMetadata struct {
	Filename    string
	ContentType string
	ContentSize int64
}

// GetInputMetadata Get input metadata.
type GetInputMetadata struct {
	IfModifiedSince   *time.Time
	IfMatch           string
	IfNoneMatch       string
	IfUnmodifiedSince *time.Time
	Range             string
}

// S3Metadata S3 Metadata.
type S3Metadata struct {
	Bucket     string
	Region     string
	S3Endpoint string
	Key        string
}

// Manager client manager.
//go:generate mockgen -destination=./mocks/mock_Manager.go -package=mocks github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/webhook Manager
type Manager interface {
	// ManageGETHooks will manage GET hooks.
	ManageGETHooks(ctx context.Context, targetKey, requestPath string, inputMetadata *GetInputMetadata, s3Metadata *S3Metadata)
	// ManageGETHooks will manage PUT hooks.
	ManagePUTHooks(ctx context.Context, targetKey, requestPath string, inputMetadata *PutInputMetadata, s3Metadata *S3Metadata)
	// ManageGETHooks will manage DELETE hooks.
	ManageDELETEHooks(ctx context.Context, targetKey, requestPath string, s3Metadata *S3Metadata)
	// Load will load all webhooks clients.
	Load() error
}

func NewManager(cfgManager config.Manager, metricsSvc metrics.Client) Manager {
	return &manager{
		cfgManager: cfgManager,
		storageMap: map[string]*hooksCfgStorage{},
		metricsSvc: metricsSvc,
	}
}
