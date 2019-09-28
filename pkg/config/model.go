package config

// Config Application Configuration
type Config struct {
	Log     *LogConfig        `koanf:"log"`
	Server  *ServerConfig     `koanf:"server"`
	Buckets []*BucketInstance `koanf:"buckets" validate:"gte=0,required,dive,required"`
}

// ServerConfig Server configuration
type ServerConfig struct {
	ListenAddr string `koanf:"listenAddr"`
	Port       int    `koanf:"port"`
}

// BucketInstance Bucket configuration
type BucketInstance struct {
	Name          string        `koanf:"name" validate:"required"`
	Bucket        *BucketConfig `koanf:"bucket" validate:"required"`
	IndexDocument string        `koanf:"indexDocument"`
}

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
