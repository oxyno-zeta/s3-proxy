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
		"log.level":                     DefaultLogLevel,
		"log.format":                    DefaultLogFormat,
		"server.port":                   DefaultPort,
		"internalServer.port":           DefaultInternalPort,
		"templates.folderList":          DefaultTemplateFolderListPath,
		"templates.targetList":          DefaultTemplateTargetListPath,
		"templates.notFound":            DefaultTemplateNotFoundPath,
		"templates.internalServerError": DefaultTemplateInternalServerErrorPath,
		"templates.unauthorized":        DefaultTemplateUnauthorizedErrorPath,
		"templates.forbidden":           DefaultTemplateForbiddenErrorPath,
		"templates.badRequest":          DefaultTemplateBadRequestErrorPath,
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
			err := loadCredential(item.Bucket.Credentials.AccessKey)
			if err != nil {
				return nil, err
			}
			// Manage secret key
			err = loadCredential(item.Bucket.Credentials.SecretKey)
			if err != nil {
				return nil, err
			}
		}
	}

	// Load auth credentials
	if out.Auth != nil {
		// Load credential for basic auth if needed
		if out.Auth.Basic != nil && out.Auth.Basic.Credentials != nil && len(out.Auth.Basic.Credentials) > 0 {
			for _, item := range out.Auth.Basic.Credentials {
				if item.User != "" && item.Password != nil {
					err := loadCredential(item.Password)
					if err != nil {
						return nil, err
					}
				}
			}
		}
		// Load credentials for oidc auth if needed and apply default oidc values
		if out.Auth.OIDC != nil {
			// Load credentials for oidc auth if needed
			if out.Auth.OIDC.ClientSecret != nil {
				err := loadCredential(out.Auth.OIDC.ClientSecret)
				if err != nil {
					return nil, err
				}
			}
			// Manage default scopes
			if out.Auth.OIDC.Scopes == nil || len(out.Auth.OIDC.Scopes) == 0 {
				out.Auth.OIDC.Scopes = DefaultOIDCScopes
			}
			// Manage default group claim
			if out.Auth.OIDC.GroupClaim == "" {
				out.Auth.OIDC.GroupClaim = DefaultOIDCGroupClaim
			}
			// Manage default oidc cookie name
			if out.Auth.OIDC.CookieName == "" {
				out.Auth.OIDC.CookieName = DefaultOIDCCookieName
			}
		}
	}

	return &out, nil
}

func loadCredential(credCfg *CredentialConfig) error {
	if credCfg.Path != "" {
		// Secret file
		databytes, err := ioutil.ReadFile(credCfg.Path)
		if err != nil {
			return err
		}
		credCfg.Value = string(databytes)
	} else if credCfg.Env != "" {
		// Environment variable
		envValue := os.Getenv(credCfg.Env)
		if envValue == "" {
			return fmt.Errorf(TemplateErrLoadingEnvCredentialEmpty, credCfg.Env)
		}
		credCfg.Value = envValue
	}
	// Value case is already managed by koanf
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
