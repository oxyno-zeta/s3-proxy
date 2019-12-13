package config

import (
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/file"
	"github.com/sirupsen/logrus"
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

	if out.AuthProviders != nil && out.AuthProviders.OIDC != nil {
		for _, v := range out.AuthProviders.OIDC {
			// Manage default scopes
			if v.Scopes == nil || len(v.Scopes) == 0 {
				v.Scopes = DefaultOIDCScopes
			}
			// Manage default group claim
			if v.GroupClaim == "" {
				v.GroupClaim = DefaultOIDCGroupClaim
			}
			// Manage default oidc cookie name
			if v.CookieName == "" {
				v.CookieName = DefaultOIDCCookieName
			}
		}
	}

	// Configuration validation
	err = validate.Struct(out)
	if err != nil {
		return nil, err
	}
	// Validate resources if they exists
	// noGlobalAuth := out.AuthProviders == nil || (out.AuthProviders != nil && out.AuthProviders.Basic == nil && out.AuthProviders.OIDC == nil)
	// if out.Resources != nil && len(out.Resources) != 0 {
	// 	for i := 0; i < len(out.Resources); i++ {
	// 		res := out.Resources[i]
	// 		// Check resource not valid
	// 		if res.WhiteList == nil && res.Basic == nil && res.OIDC == nil {
	// 			return nil, fmt.Errorf("Resource %d must have whitelist, basic configuration or oidc configuration", i)
	// 		}
	// 		// Check no global auth and not in white list
	// 		if noGlobalAuth &&
	// 			res.Basic == nil &&
	// 			res.OIDC == nil &&
	// 			res.WhiteList != nil &&
	// 			!*res.WhiteList {
	// 			return nil, fmt.Errorf("Resource %d is not declared in whitelist and global authentication were not found", i)
	// 		}
	// 		// Check OIDC but no OIDC configuration
	// 		if (out.AuthProviders == nil || (out.AuthProviders != nil && out.AuthProviders.OIDC == nil)) && res.OIDC != nil {
	// 			return nil, fmt.Errorf("Resource %d have an OIDC configuration but no global authentication were found", i)
	// 		}
	// 	}
	// }

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
	if out.AuthProviders != nil {
		// Load credentials for oidc auth if needed
		if out.AuthProviders.OIDC != nil {
			// Load credentials for oidc auth if needed
			for _, v := range out.AuthProviders.OIDC {
				if v.ClientSecret != nil {
					err := loadCredential(v.ClientSecret)
					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return &out, nil
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
