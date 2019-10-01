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

// DefaultTemplateTargetList Default template target list
const DefaultTemplateTargetList = "templates/target-list.tpl"

// ErrMainBucketPathSupportNotValid Error thrown when main bucket path support option isn't valid
var ErrMainBucketPathSupportNotValid = errors.New("main bucket path support option can be enabled only when only one bucket is configured")

// TemplateErrLoadingEnvCredentialEmpty Template Error when Loading Environment variable Credentials
var TemplateErrLoadingEnvCredentialEmpty = "error loading credentials for target %s environment variable %s is empty"

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
	TargetList string `koanf:"targetList" validate:"required"`
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
	Name        string                  `koanf:"name" validate:"required"`
	Prefix      string                  `koanf:"prefix"`
	Region      string                  `koanf:"region"`
	S3Endpoint  string                  `koanf:"s3Endpoint"`
	Credentials *BucketCredentialConfig `koanf:"credentials" validate:"omitempty,dive"`
}

// BucketCredentialConfig Bucket Credentials configurations
type BucketCredentialConfig struct {
	AccessKey *CredentialKeyConfig `koanf:"accessKey" validate:"omitempty,dive"`
	SecretKey *CredentialKeyConfig `koanf:"secretKey" validate:"omitempty,dive"`
}

type CredentialKeyConfig struct {
	Path  string `koanf:"path" validate:"required_without=Env"`
	Env   string `koanf:"env" validate:"required_without=Path"`
	Value string
}

// LogConfig Log configuration
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}
