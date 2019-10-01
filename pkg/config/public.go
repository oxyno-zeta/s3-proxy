package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/sirupsen/logrus"
	"gopkg.in/go-playground/validator.v9"
)

var k = koanf.New(".")
var validate = validator.New()

// Load Load configuration
func Load() (*Config, error) {
	// Load default configuration
	k.Load(confmap.Provider(map[string]interface{}{
		"log.level":            DefaultLogLevel,
		"log.format":           DefaultLogFormat,
		"server.port":          DefaultPort,
		"templates.folderList": DefaultTemplateFolderList,
		"templates.targetList": DefaultTemplateTargetList,
	}, "."), nil)

	// Try to load main configuration file
	err := k.Load(file.Provider(MainConfigPath), yaml.Parser())
	if err != nil {
		return nil, err
	}

	// Prepare configuration object
	var out Config
	// Quick unmarshal.
	k.Unmarshal("", &out)

	// Manage default s3 bucket region
	for _, item := range out.Targets {
		if item.Bucket.Region == "" {
			item.Bucket.Region = DefaultBucketRegion
		}
	}

	// Configuration validation
	err = validate.Struct(out)
	if err != nil {
		return nil, err
	}
	// Validate main bucket path support option
	if out.MainBucketPathSupport && len(out.Targets) > 1 {
		return nil, ErrMainBucketPathSupportNotValid
	}

	// Load credentials from declaration
	for _, item := range out.Targets {
		if item.Bucket.Credentials != nil && item.Bucket.Credentials.AccessKey != nil && item.Bucket.Credentials.SecretKey != nil {
			// Manage access key
			err := loadCredential(item.Bucket.Credentials.AccessKey, item.Name)
			if err != nil {
				return nil, err
			}
			// Manage secret key
			err = loadCredential(item.Bucket.Credentials.SecretKey, item.Name)
			if err != nil {
				return nil, err
			}
		}
	}

	return &out, nil
}

func loadCredential(credKeyCfg *CredentialKeyConfig, targetName string) error {
	if credKeyCfg.Path != "" {
		// Secret file
		databytes, err := ioutil.ReadFile(credKeyCfg.Path)
		if err != nil {
			return err
		}
		credKeyCfg.Value = string(databytes)
	} else {
		// Environment variable
		envValue := os.Getenv(credKeyCfg.Env)
		if envValue == "" {
			return fmt.Errorf(TemplateErrLoadingEnvCredentialEmpty, credKeyCfg.Env, targetName)
		}
		credKeyCfg.Value = envValue
	}
	return nil
}

// ConfigureLogger Configure logger instance
func ConfigureLogger(logger *logrus.Logger, logConfig *LogConfig) error {
	// Manage log format
	if logConfig.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{})
	} else {
		logger.SetFormatter(&logrus.TextFormatter{})
	}

	// Manage log level
	lvl, err := logrus.ParseLevel(logConfig.Level)
	if err != nil {
		return err
	}
	logger.SetLevel(lvl)

	return nil
}

// GetRootPrefix Get bucket root prefix
func (bcfg *BucketConfig) GetRootPrefix() string {
	key := bcfg.Prefix
	// Check if key ends with a /, if key exists and don't ends with / add it
	if key != "" && !strings.HasSuffix(key, "/") {
		key += "/"
	}
	return key
}
