package cache

import (
	"context"

	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/config"
)

// Client represents a Cache client.
//go:generate mockgen -destination=./mocks/mock_Client.go -package=mocks github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/cache Client
type Client interface {
	// Initialize cache.
	Initialize() error
	// Reload will reload configuration and cache.
	Reload() error
	// Get will get result from cache if it exists.
	Get(ctx context.Context, key string, result interface{}) error
	// Set will store in cache.
	Set(ctx context.Context, key string, value interface{}) error
}

func NewCache(cfgManager config.Manager) Client {
	return &cac{cfgManager: cfgManager}
}
