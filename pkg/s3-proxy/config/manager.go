package config

import "github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"

// Manager
type Manager interface {
	// Load configuration
	Load() error
	// Get configuration object
	GetConfig() *Config
}

func NewManager(logger log.Logger) Manager {
	return &managercontext{logger: logger}
}
