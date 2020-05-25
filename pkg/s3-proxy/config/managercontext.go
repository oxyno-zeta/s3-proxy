package config

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
)

// Main configuration folder path
var mainConfigFolderPath = "conf/"

var validate = validator.New()

type managercontext struct {
	cfg     *Config
	configs []*viper.Viper
	logger  log.Logger
}

func (ctx *managercontext) Load() error {
	// List files
	files, err := ioutil.ReadDir(mainConfigFolderPath)
	if err != nil {
		return err
	}

	// Generate viper instances for static configs
	ctx.configs = generateViperInstances(files)

	// Put default values
	ctx.loadDefaultConfigurationValues()

	// Load configuration
	err = ctx.loadConfiguration()
	if err != nil {
		return err
	}

	return err
}

func (ctx *managercontext) loadDefaultConfigurationValues() {
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
}

func generateViperInstances(files []os.FileInfo) []*viper.Viper {
	list := make([]*viper.Viper, 0)
	// Loop over static files to create viper instance for them
	funk.ForEach(files, func(file os.FileInfo) {
		filename := file.Name()
		// Create config file name
		cfgFileName := strings.TrimSuffix(filename, path.Ext(filename))
		// Test if config file name is compliant (ignore hidden files like .keep)
		if cfgFileName != "" {
			// Create new viper instance
			vip := viper.New()
			// Set config name
			vip.SetConfigName(cfgFileName)
			// Add configuration path
			vip.AddConfigPath(mainConfigFolderPath)
			// Append it
			list = append(list, vip)
		}
	})

	return list
}

func (ctx *managercontext) loadConfiguration() error {
	// Loop over configs
	for _, vip := range ctx.configs {
		err := vip.ReadInConfig()
		if err != nil {
			return err
		}

		err = viper.MergeConfigMap(vip.AllSettings())
		if err != nil {
			return err
		}
	}

	// Prepare configuration object
	var out Config
	// Quick unmarshal.
	err := viper.Unmarshal(&out)
	if err != nil {
		return err
	}

	// Load default values
	err = loadBusinessDefaultValues(&out)
	if err != nil {
		return err
	}

	// Configuration validation
	err = validate.Struct(out)
	if err != nil {
		return err
	}

	// Load all credentials
	err = loadAllCredentials(&out)
	if err != nil {
		return err
	}

	err = validateBusinessConfig(&out)
	if err != nil {
		return err
	}

	ctx.cfg = &out

	return nil
}

// GetConfig allow to get configuration object
func (ctx *managercontext) GetConfig() *Config {
	return ctx.cfg
}

func loadAllCredentials(out *Config) error {
	// Load credentials from declaration
	for _, item := range out.Targets {
		// Check if resources are declared
		if item.Resources != nil {
			for j := 0; j < len(item.Resources); j++ {
				res := item.Resources[j]
				// Check if basic auth configuration exists
				if res.Basic != nil && res.Basic.Credentials != nil {
					// Loop over creds
					for k := 0; k < len(res.Basic.Credentials); k++ {
						it := res.Basic.Credentials[k]
						// Load credential
						err := loadCredential(it.Password)
						if err != nil {
							return err
						}
					}
				}
			}
		}
		// Load credentials for access key and secret key
		if item.Bucket.Credentials != nil && item.Bucket.Credentials.AccessKey != nil && item.Bucket.Credentials.SecretKey != nil {
			// Manage access key
			err := loadCredential(item.Bucket.Credentials.AccessKey)
			if err != nil {
				return err
			}
			// Manage secret key
			err = loadCredential(item.Bucket.Credentials.SecretKey)
			if err != nil {
				return err
			}
		}
	}

	// Load auth credentials
	if out.AuthProviders != nil {
		// Load credentials for oidc auth if needed
		if out.AuthProviders.OIDC != nil {
			// Load credentials for oidc auth if needed
			for _, v := range out.AuthProviders.OIDC {
				// Check if client secret exists
				if v.ClientSecret != nil {
					err := loadCredential(v.ClientSecret)
					if err != nil {
						return err
					}
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
				return err
			}
		}
	}

	return nil
}

func loadCredential(credCfg *CredentialConfig) error {
	if credCfg.Path != "" {
		// Secret file
		databytes, err := ioutil.ReadFile(credCfg.Path)
		if err != nil {
			return err
		}
		// Store value
		credCfg.Value = string(databytes)
	} else if credCfg.Env != "" {
		// Environment variable
		envValue := os.Getenv(credCfg.Env)
		if envValue == "" {
			return fmt.Errorf(TemplateErrLoadingEnvCredentialEmpty, credCfg.Env)
		}
		// Store value
		credCfg.Value = envValue
	}
	// Default value
	return nil
}

func loadBusinessDefaultValues(out *Config) error {
	// Manage default values for targets
	for _, item := range out.Targets {
		// Manage default configuration for target region
		if item.Bucket != nil && item.Bucket.Region == "" {
			item.Bucket.Region = DefaultBucketRegion
		}
		// Manage default configuration for target actions
		if item.Actions == nil {
			item.Actions = &ActionsConfig{GET: &GetActionConfig{Enabled: true}}
		}
		// Manage default for target templates configurations
		if item.Templates == nil {
			item.Templates = &TargetTemplateConfig{}
		}
		// Manage default value for resources methods
		if item.Resources != nil {
			for _, res := range item.Resources {
				// Check if resource has methods
				if res.Methods == nil {
					// Set default values
					res.Methods = []string{http.MethodGet}
				}

				// Check if regexp is enabled in OIDC Authorization groups
				if res.OIDC != nil && res.OIDC.AuthorizationAccesses != nil {
					for _, item := range res.OIDC.AuthorizationAccesses {
						err2 := loadRegexOIDCAuthorizationAccess(item)
						if err2 != nil {
							return err2
						}
					}
				}
			}
		}
	}

	// Manage default value for list targets resources
	if out.ListTargets != nil && out.ListTargets.Resource != nil {
		// Store resource object
		res := out.ListTargets.Resource

		// Manage default values for http methods
		if res.Methods == nil {
			// Set default values
			res.Methods = []string{http.MethodGet}
		}

		// Manage regexp values
		// Check if regexp is enabled in OIDC Authorization groups
		if res.OIDC != nil && res.OIDC.AuthorizationAccesses != nil {
			for _, item := range res.OIDC.AuthorizationAccesses {
				err2 := loadRegexOIDCAuthorizationAccess(item)
				if err2 != nil {
					return err2
				}
			}
		}
	}

	// Manage default values for auth providers
	if out.AuthProviders != nil && out.AuthProviders.OIDC != nil {
		for k, v := range out.AuthProviders.OIDC {
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

	// Manage default value for list targets
	if out.ListTargets == nil {
		out.ListTargets = &ListTargetsConfig{Enabled: false}
	}

	return nil
}

// Load Regex in OIDC Authorization access objects
func loadRegexOIDCAuthorizationAccess(item *OIDCAuthorizationAccess) error {
	if item.Regexp {
		// Try to compile regex for group or email
		// Group case
		if item.Group != "" {
			// Compile Regexp
			reg, err2 := regexp.Compile(item.Group)
			// Check error
			if err2 != nil {
				return err2
			}
			// Save regexp
			item.GroupRegexp = reg
		}

		// Email case
		if item.Email != "" {
			// Compile regexp
			reg, err2 := regexp.Compile(item.Email)
			// Check error
			if err2 != nil {
				return err2
			}
			// Save regexp
			item.EmailRegexp = reg
		}
	}

	return nil
}
