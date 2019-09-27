package config

// Config Application Configuration
type Config struct {
	Log     *LogConfig      `koanf:"log"`
	Server  *ServerConfig   `koanf:"server"`
	Buckets []*BucketConfig `koanf:"buckets" validate:"gte=0,required,dive,required"`
}

// ServerConfig Server configuration
type ServerConfig struct {
	ListenAddr string `koanf:"listenAddr"`
	Port       int    `koanf:"port"`
}

// BucketConfig Bucket configuration
type BucketConfig struct {
	Name          string `koanf:"name" validate:"required"`
	Bucket        string `koanf:"bucket" validate:"required"`
	Prefix        string `koanf:"prefix"`
	Region        string `koanf:"region"`
	IndexDocument string `koanf:"indexDocument"`
}

// LogConfig Log configuration
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}
