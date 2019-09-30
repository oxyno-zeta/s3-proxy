package config

import "errors"

// MainConfigPath Configuration path
const MainConfigPath = "config.yaml"

// DefaultPort Default port
const DefaultPort = 8080

// DefaultLogLevel Default log level
const DefaultLogLevel = "info"

// DefaultLogFormat Default Log format
const DefaultLogFormat = "json"

// DefaultBucketRegion Default bucket region
const DefaultBucketRegion = "us-east-1"

// DefaultTemplateFolderList Default template folder list
const DefaultTemplateFolderList = "templates/folder-list.tpl"

// ErrMainBucketPathSupportNotValid Error thrown when main bucket path support option isn't valid
var ErrMainBucketPathSupportNotValid = errors.New("main bucket path support option can be enabled only when only one bucket is configured")

// Config Application Configuration
type Config struct {
	Log                   *LogConfig      `koanf:"log"`
	Server                *ServerConfig   `koanf:"server"`
	Targets               []*Target       `koanf:"targets" validate:"gte=0,required,dive,required"`
	Templates             *TemplateConfig `koanf:"templates"`
	MainBucketPathSupport bool            `koanf:"mainBucketPathSupport"`
}

// TemplateConfig Templates configuration
type TemplateConfig struct {
	FolderList string `koanf:"folderList" validate:"required"`
}

// ServerConfig Server configuration
type ServerConfig struct {
	ListenAddr string `koanf:"listenAddr"`
	Port       int    `koanf:"port" validate:"required"`
}

// Target Bucket instance configuration
type Target struct {
	Name          string        `koanf:"name" validate:"required"`
	Bucket        *BucketConfig `koanf:"bucket" validate:"required"`
	IndexDocument string        `koanf:"indexDocument"`
}

// BucketConfig Bucket configuration
type BucketConfig struct {
	Name   string `koang:"name" validate:"required"`
	Prefix string `koanf:"prefix"`
	Region string `koanf:"region"`
}

// LogConfig Log configuration
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}
