package config

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

// MainConfigFolderPath Main configuration folder path
const MainConfigFolderPath = "conf/"

// MainConfigFileName Main configuration filename
const MainConfigFileName = "config"

// DefaultPort Default port
const DefaultPort = 8080

// DefaultInternalPort Default internal port
const DefaultInternalPort = 9090

// DefaultLogLevel Default log level
const DefaultLogLevel = "info"

// DefaultLogFormat Default Log format
const DefaultLogFormat = "json"

// DefaultBucketRegion Default bucket region
const DefaultBucketRegion = "us-east-1"

// DefaultTemplateFolderListPath Default template folder list path
const DefaultTemplateFolderListPath = "templates/folder-list.tpl"

// DefaultTemplateTargetListPath Default template target list path
const DefaultTemplateTargetListPath = "templates/target-list.tpl"

// DefaultTemplateNotFoundPath Default template not found path
const DefaultTemplateNotFoundPath = "templates/not-found.tpl"

// DefaultTemplateForbiddenErrorPath Default template forbidden path
const DefaultTemplateForbiddenErrorPath = "templates/forbidden.tpl"

// DefaultTemplateBadRequestErrorPath Default template bad request path
const DefaultTemplateBadRequestErrorPath = "templates/bad-request.tpl"

// DefaultTemplateInternalServerErrorPath Default template Internal server error path
const DefaultTemplateInternalServerErrorPath = "templates/internal-server-error.tpl"

// DefaultTemplateUnauthorizedErrorPath Default template unauthorized error path
const DefaultTemplateUnauthorizedErrorPath = "templates/unauthorized.tpl"

// DefaultOIDCScopes Default OIDC Scopes
var DefaultOIDCScopes = []string{"openid", "profile", "email"}

// DefaultOIDCGroupClaim Default OIDC group claim
const DefaultOIDCGroupClaim = "groups"

// DefaultOIDCCookieName Default OIDC Cookie name
const DefaultOIDCCookieName = "oidc"

// ErrMainBucketPathSupportNotValid Error thrown when main bucket path support option isn't valid
var ErrMainBucketPathSupportNotValid = errors.New("main bucket path support option can be enabled only when only one bucket is configured")

// TemplateErrLoadingEnvCredentialEmpty Template Error when Loading Environment variable Credentials
var TemplateErrLoadingEnvCredentialEmpty = "error loading credentials, environment variable %s is empty"

const oidcLoginPathTemplate = "/auth/%s"
const oidcCallbackPathTemplate = "/auth/%s/callback"

var validate = validator.New()

// Config Application Configuration
type Config struct {
	Log            *LogConfig          `mapstructure:"log"`
	Server         *ServerConfig       `mapstructure:"server"`
	InternalServer *ServerConfig       `mapstructure:"internalServer"`
	Targets        []*TargetConfig     `mapstructure:"targets" validate:"gte=0,required,dive,required"`
	Templates      *TemplateConfig     `mapstructure:"templates"`
	AuthProviders  *AuthProviderConfig `mapstructure:"authProviders"`
	ListTargets    *ListTargetsConfig  `mapstructure:"listTargets"`
}

// ListTargetsConfig List targets configuration
type ListTargetsConfig struct {
	Enabled  bool         `mapstructure:"enabled"`
	Mount    *MountConfig `mapstructure:"mount" validate:"required_with=Enabled"`
	Resource *Resource    `mapstructure:"resource" validate:"omitempty"`
}

// MountConfig Mount configuration
type MountConfig struct {
	Host string   `mapstructure:"host"`
	Path []string `mapstructure:"path" validate:"required,dive,required"`
}

// AuthProviderConfig Authentication provider configurations
type AuthProviderConfig struct {
	Basic map[string]*BasicAuthConfig `mapstructure:"basic" validate:"omitempty,dive"`
	OIDC  map[string]*OIDCAuthConfig  `mapstructure:"oidc" validate:"omitempty,dive"`
}

// OIDCAuthConfig OpenID Connect authentication configurations
type OIDCAuthConfig struct {
	ClientID      string            `mapstructure:"clientID" validate:"required"`
	ClientSecret  *CredentialConfig `mapstructure:"clientSecret" validate:"omitempty,dive"`
	IssuerURL     string            `mapstructure:"issuerUrl" validate:"required"`
	RedirectURL   string            `mapstructure:"redirectUrl" validate:"required"`
	Scopes        []string          `mapstructure:"scope"`
	State         string            `mapstructure:"state" validate:"required"`
	GroupClaim    string            `mapstructure:"groupClaim"`
	CookieName    string            `mapstructure:"cookieName"`
	EmailVerified bool              `mapstructure:"emailVerified"`
	CookieSecure  bool              `mapstructure:"cookieSecure"`
	LoginPath     string            `mapstructure:"loginPath"`
	CallbackPath  string            `mapstructure:"callbackPath"`
}

// OIDCAuthorizationAccess OpenID Connect authorization accesses
type OIDCAuthorizationAccess struct {
	Group       string `mapstructure:"group" validate:"required_without=Email"`
	Email       string `mapstructure:"email" validate:"required_without=Group"`
	Regexp      bool   `mapstructure:"regexp"`
	GroupRegexp *regexp.Regexp
	EmailRegexp *regexp.Regexp
}

// BasicAuthConfig Basic auth configurations
type BasicAuthConfig struct {
	Realm string `mapstructure:"realm" validate:"required"`
}

// BasicAuthUserConfig Basic User auth configuration
type BasicAuthUserConfig struct {
	User     string            `mapstructure:"user" validate:"required"`
	Password *CredentialConfig `mapstructure:"password" validate:"required,dive"`
}

// TemplateConfig Templates configuration
type TemplateConfig struct {
	FolderList          string `mapstructure:"folderList" validate:"required"`
	TargetList          string `mapstructure:"targetList" validate:"required"`
	NotFound            string `mapstructure:"notFound" validate:"required"`
	InternalServerError string `mapstructure:"internalServerError" validate:"required"`
	Unauthorized        string `mapstructure:"unauthorized" validate:"required"`
	Forbidden           string `mapstructure:"forbidden" validate:"required"`
	BadRequest          string `mapstructure:"badRequest" validate:"required"`
}

// ServerConfig Server configuration
type ServerConfig struct {
	ListenAddr string `mapstructure:"listenAddr"`
	Port       int    `mapstructure:"port" validate:"required"`
}

// TargetConfig Bucket instance configuration
type TargetConfig struct {
	Name          string         `mapstructure:"name" validate:"required"`
	Bucket        *BucketConfig  `mapstructure:"bucket" validate:"required"`
	Resources     []*Resource    `mapstructure:"resources" validate:"dive"`
	Mount         *MountConfig   `mapstructure:"mount" validate:"required"`
	IndexDocument string         `mapstructure:"indexDocument"`
	Actions       *ActionsConfig `mapstructure:"actions"`
}

// ActionsConfig is dedicated to actions configuration in a target
type ActionsConfig struct {
	GET    *GetActionConfig    `mapstructure:"GET"`
	PUT    *PutActionConfig    `mapstructure:"PUT"`
	DELETE *DeleteActionConfig `mapstructure:"DELETE"`
}

// DeleteActionConfig Delete action configuration
type DeleteActionConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// PutActionConfig Post action configuration
type PutActionConfig struct {
	Enabled bool                   `mapstructure:"enabled"`
	Config  *PutActionConfigConfig `mapstructure:"config"`
}

// PutActionConfigConfig Post action configuration object configuration
type PutActionConfigConfig struct {
	Metadata      map[string]string `mapstructure:"metadata"`
	StorageClass  string            `mapstructure:"storageClass"`
	AllowOverride bool              `mapstructure:"allowOverride"`
}

// GetActionConfig Get action configuration
type GetActionConfig struct {
	Enabled bool `mapstructure:"enabled"`
}

// Resource Resource
type Resource struct {
	Path      string         `mapstructure:"path" validate:"required"`
	Methods   []string       `mapstructure:"methods" validate:"required,dive,required"`
	WhiteList *bool          `mapstructure:"whiteList"`
	Provider  string         `mapstructure:"provider"`
	Basic     *ResourceBasic `mapstructure:"basic" validate:"omitempty"`
	OIDC      *ResourceOIDC  `mapstructure:"oidc" validate:"omitempty"`
}

// ResourceBasic Basic auth resource
type ResourceBasic struct {
	Credentials []*BasicAuthUserConfig `mapstructure:"credentials" validate:"omitempty,dive"`
}

// ResourceOIDC OIDC auth Resource
type ResourceOIDC struct {
	AuthorizationAccesses []*OIDCAuthorizationAccess `mapstructure:"authorizationAccesses" validate:"dive"`
}

// BucketConfig Bucket configuration
type BucketConfig struct {
	Name        string                  `mapstructure:"name" validate:"required"`
	Prefix      string                  `mapstructure:"prefix"`
	Region      string                  `mapstructure:"region"`
	S3Endpoint  string                  `mapstructure:"s3Endpoint"`
	Credentials *BucketCredentialConfig `mapstructure:"credentials" validate:"omitempty,dive"`
}

// BucketCredentialConfig Bucket Credentials configurations
type BucketCredentialConfig struct {
	AccessKey *CredentialConfig `mapstructure:"accessKey" validate:"omitempty,dive"`
	SecretKey *CredentialConfig `mapstructure:"secretKey" validate:"omitempty,dive"`
}

// CredentialConfig Credential Configurations
type CredentialConfig struct {
	Path  string `mapstructure:"path" validate:"required_without_all=Env Value"`
	Env   string `mapstructure:"env" validate:"required_without_all=Path Value"`
	Value string `mapstructure:"value" validate:"required_without_all=Path Env"`
}

// LogConfig Log configuration
type LogConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

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

	// Load default values
	err = loadDefaultValues(&out)
	if err != nil {
		return nil, err
	}

	// Configuration validation
	err = validate.Struct(out)
	if err != nil {
		return nil, err
	}

	// Load all credentials
	err = loadAllCredentials(&out)
	if err != nil {
		return nil, err
	}

	err = validateBusinessConfig(&out)
	if err != nil {
		return nil, err
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

func loadAllCredentials(out *Config) error {
	// Load credentials from declaration
	for _, item := range out.Targets {
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
							return err
						}
					}
				}
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
	// Value case is already managed by koanf
	return nil
}

func loadDefaultValues(out *Config) error {
	// Manage default values for targets
	for _, item := range out.Targets {
		// Manage default configuration for target region
		if item.Bucket.Region == "" {
			item.Bucket.Region = DefaultBucketRegion
		}
		// Manage default configuration for target actions
		if item.Actions == nil {
			item.Actions = &ActionsConfig{GET: &GetActionConfig{Enabled: true}}
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
