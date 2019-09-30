package config

// MainConfigPath Configuration path
const MainConfigPath = "config.yaml"

// DefaultPort Default port
const DefaultPort = 8080

// DefaultLogLevel Default log level
const DefaultLogLevel = "info"

// DefaultLogFormat Default Log format
const DefaultLogFormat = "json"

// DefaultTemplateFolderList Default template folder list
const DefaultTemplateFolderList = "templates/folder-list.tpl"

// Config Application Configuration
type Config struct {
	Log       *LogConfig        `koanf:"log"`
	Server    *ServerConfig     `koanf:"server"`
	Buckets   []*BucketInstance `koanf:"buckets" validate:"gte=0,required,dive,required"`
	Templates *TemplateConfig   `koanf:"templates"`
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

// BucketInstance Bucket instance configuration
type BucketInstance struct {
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
