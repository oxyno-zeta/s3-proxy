package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var validate = validator.New()

// Load Load configuration
func Load() (*Config, error) {
	// Set main configuration filename
	viper.SetConfigName(MainConfigFileName)
	// Set main configuration folder path
	viper.AddConfigPath(MainConfigFolderPath)
	viper.AddConfigPath(".")
	// Load default configuration
	viper.SetDefault("log.level", DefaultLogLevel)
	viper.SetDefault("log.format", DefaultLogFormat)
	viper.SetDefault("server.port", DefaultPort)
	viper.SetDefault("internalServer.port", DefaultInternalPort)
	viper.SetDefault("templates.folderList", DefaultTemplateFolderListPath)
	viper.SetDefault("templates.targetList", DefaultTemplateTargetListPath)
	viper.SetDefault("templates.notFound", DefaultTemplateNotFoundPath)
	viper.SetDefault("templates.internalServerError", DefaultTemplateInternalServerErrorPath)
	viper.SetDefault("templates.unauthorized", DefaultTemplateUnauthorizedErrorPath)
	viper.SetDefault("templates.forbidden", DefaultTemplateForbiddenErrorPath)
	viper.SetDefault("templates.badRequest", DefaultTemplateBadRequestErrorPath)

	// Try to load main configuration file
	err := viper.ReadInConfig()
	if err != nil {
		return nil, err
	}

	// Prepare configuration object
	var out Config
	// Quick unmarshal.
	err = viper.Unmarshal(&out)
	if err != nil {
		return nil, err
	}

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
			for k, v := range out.AuthProviders.OIDC {
				// Check if client secret exists
				if v.ClientSecret != nil {
					err := loadCredential(v.ClientSecret)
					if err != nil {
						return nil, err
					}
				}
				// Check if login path is defined
				if v.LoginPath == "" {
					v.LoginPath = fmt.Sprintf(oidcLoginPathTemplate, k)
				}
				// Check if callback path is defined
				if v.CallbackPath == "" {
					v.CallbackPath = fmt.Sprintf(oidcCallbackPathTemplate, k)
				}
			}
		}
	}

	// Load auth credentials from list targets with basic auth
	if out.ListTargets != nil && out.ListTargets.Resource != nil &&
		out.ListTargets.Resource.Basic != nil && out.ListTargets.Resource.Basic.Credentials != nil {
		// Loop over credentials declared
		for i := 0; i < len(out.ListTargets.Resource.Basic.Credentials); i++ {
			// Store item access
			it := out.ListTargets.Resource.Basic.Credentials[i]
			// Load credential
			err := loadCredential(it.Password)
			if err != nil {
				return nil, err
			}
		}
	}
	// Load auth credentials from targets with basic auth
	for i := 0; i < len(out.Targets); i++ {
		target := out.Targets[i]
		// Check if resources are declared
		if target.Resources != nil {
			for j := 0; j < len(target.Resources); j++ {
				res := target.Resources[j]
				// Check if basic auth configuration exists
				if res.Basic != nil && res.Basic.Credentials != nil {
					// Loop over creds
					for k := 0; k < len(res.Basic.Credentials); k++ {
						it := res.Basic.Credentials[k]
						// Load credential
						err := loadCredential(it.Password)
						if err != nil {
							return nil, err
						}
					}
				}
			}
		}
	}

	// Validate resources if they exists in all targets and validate target mount path
	for i := 0; i < len(out.Targets); i++ {
		target := out.Targets[i]
		// Check if resources are declared
		if target.Resources != nil {
			for j := 0; j < len(target.Resources); j++ {
				res := target.Resources[j]
				// Validate resource
				err = validateResource(fmt.Sprintf("resource %d from target %d", j, i), res, out.AuthProviders)
				// Return error if exists
				if err != nil {
					return nil, err
				}
			}
		}
		// Check mount path items
		pathList := target.Mount.Path
		for j := 0; j < len(pathList); j++ {
			path := pathList[j]
			// Check path value
			err := validatePath(fmt.Sprintf("path %d in target %d", j, i), path)
			if err != nil {
				return nil, err
			}
		}
	}

	// Validate list targets object
	if out.ListTargets != nil {
		// Check list targets resource
		if out.ListTargets.Resource != nil {
			res := out.ListTargets.Resource
			// Validate resource
			err = validateResource("resource from list targets", res, out.AuthProviders)
			// Return error if exists
			if err != nil {
				return nil, err
			}
		}
		// Check mount path items
		pathList := out.ListTargets.Mount.Path
		for j := 0; j < len(pathList); j++ {
			path := pathList[j]
			// Check path value
			err := validatePath(fmt.Sprintf("path %d in list targets", j), path)
			if err != nil {
				return nil, err
			}
		}
	}

	return &out, nil
}

func validateResource(beginErrorMessage string, res *Resource, authProviders *AuthProviderConfig) error {
	// Check resource not valid
	if res.WhiteList == nil && res.Basic == nil && res.OIDC == nil {
		return errors.New(beginErrorMessage + " have whitelist, basic configuration or oidc configuration")
	}
	// Check if provider exists
	if res.WhiteList != nil && !*res.WhiteList && res.Provider == "" {
		return errors.New(beginErrorMessage + " must have a provider")
	}
	// Check auth logins are provided in case of no whitelist
	if res.WhiteList != nil && !*res.WhiteList && res.Basic == nil && res.OIDC == nil {
		return errors.New(beginErrorMessage + " must have authentication configuration declared (oidc or basic)")
	}
	// Check that provider is declared is auth providers and correctly linked
	if res.Provider != "" {
		// Check that auth provider exists
		exists := authProviders.Basic[res.Provider] != nil || authProviders.OIDC[res.Provider] != nil
		if !exists {
			return errors.New(beginErrorMessage + " must have a valid provider declared in authentication providers")
		}
		// Check that selected provider is in link with authentication selected
		// Check basic
		if res.Basic != nil && authProviders.Basic[res.Provider] == nil {
			return errors.New(
				beginErrorMessage + " must use a valid authentication configuration with selected authentication provider: basic auth not allowed")
		}
		// Check oidc
		if res.OIDC != nil && authProviders.OIDC[res.Provider] == nil {
			return errors.New(beginErrorMessage + " must use a valid authentication configuration with selected authentication provider: oidc not allowed")
		}
	}
	// Return no error
	return nil
}

func validatePath(beginErrorMessage string, path string) error {
	// Check that path begins with /
	if !strings.HasPrefix(path, "/") {
		return errors.New(beginErrorMessage + " must starts with /")
	}
	// Check that path ends with /
	if !strings.HasSuffix(path, "/") {
		return errors.New(beginErrorMessage + " must ends with /")
	}
	// Return no error
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
	// Set log level
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
	// Return result
	return key
}
