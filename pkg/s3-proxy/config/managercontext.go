package config

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"emperror.dev/errors"
	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/utils/generalutils"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
)

// Main configuration folder path.
var mainConfigFolderPath = "conf/"

var validate = validator.New()

type managercontext struct {
	cfg                       *Config
	configs                   []*viper.Viper
	onChangeHooks             []func()
	logger                    log.Logger
	internalFileWatchChannels []chan bool
}

func (ctx *managercontext) AddOnChangeHook(hook func()) {
	ctx.onChangeHooks = append(ctx.onChangeHooks, hook)
}

func (ctx *managercontext) Load() error {
	// List files
	files, err := os.ReadDir(mainConfigFolderPath)
	if err != nil {
		return errors.WithStack(err)
	}

	// Generate viper instances for static configs
	ctx.configs = generateViperInstances(files)

	// Load configuration
	err = ctx.loadConfiguration()
	if err != nil {
		return err
	}

	// Loop over config files
	funk.ForEach(ctx.configs, func(vip *viper.Viper) {
		// Add hooks for on change events
		vip.OnConfigChange(func(in fsnotify.Event) {
			ctx.logger.Infof("Reload configuration detected for file %s", vip.ConfigFileUsed())

			// Reload config
			err2 := ctx.loadConfiguration()
			if err2 != nil {
				ctx.logger.Error(err2)
				// Stop here and do not call hooks => configuration is unstable
				return
			}
			// Call all hooks
			funk.ForEach(ctx.onChangeHooks, func(hook func()) { hook() })
		})
		// Watch for configuration changes
		vip.WatchConfig()
	})

	return nil
}

// Imported and modified from viper v1.7.0.
func (ctx *managercontext) watchInternalFile(filePath string, forceStop chan bool, onChange func()) {
	initWG := sync.WaitGroup{}
	initWG.Add(1)

	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			ctx.logger.Fatal(errors.WithStack(err))
		}
		defer watcher.Close()

		configFile := filepath.Clean(filePath)
		configDir, _ := filepath.Split(configFile)
		realConfigFile, _ := filepath.EvalSymlinks(filePath)

		eventsWG := sync.WaitGroup{}
		eventsWG.Add(1)

		go func() {
			for {
				select {
				case <-forceStop:
					eventsWG.Done()

					return
				case event, ok := <-watcher.Events:
					if !ok { // 'Events' channel is closed
						eventsWG.Done()

						return
					}

					currentConfigFile, _ := filepath.EvalSymlinks(filePath)
					// we only care about the config file with the following cases:
					// 1 - if the config file was modified or created
					// 2 - if the real path to the config file changed (eg: k8s ConfigMap replacement)
					const writeOrCreateMask = fsnotify.Write | fsnotify.Create
					if (filepath.Clean(event.Name) == configFile &&
						event.Op&writeOrCreateMask != 0) ||
						(currentConfigFile != "" && currentConfigFile != realConfigFile) {
						realConfigFile = currentConfigFile

						// Call on change
						onChange()
					} else if filepath.Clean(event.Name) == configFile && event.Op&fsnotify.Remove&fsnotify.Remove != 0 {
						eventsWG.Done()

						return
					}

				case err, ok := <-watcher.Errors:
					if ok { // 'Errors' channel is not closed
						ctx.logger.Errorf("watcher error: %v\n", err)
					}

					eventsWG.Done()

					return
				}
			}
		}()

		_ = watcher.Add(configDir)

		initWG.Done()   // done initializing the watch in this go routine, so the parent routine can move on...
		eventsWG.Wait() // now, wait for event loop to end in this go-routine...
	}()
	initWG.Wait() // make sure that the go routine above fully ended before returning
}

func (ctx *managercontext) loadDefaultConfigurationValues(vip *viper.Viper) {
	// Load default configuration
	vip.SetDefault("log.level", DefaultLogLevel)
	vip.SetDefault("log.format", DefaultLogFormat)
	vip.SetDefault("server.port", DefaultPort)
	vip.SetDefault("server.compress.enabled", &DefaultServerCompressEnabled)
	vip.SetDefault("server.compress.level", DefaultServerCompressLevel)
	vip.SetDefault("server.compress.types", DefaultServerCompressTypes)
	vip.SetDefault("server.timeouts.readHeaderTimeout", DefaultServerTimeoutsReadHeaderTimeout)
	vip.SetDefault("internalServer.port", DefaultInternalPort)
	vip.SetDefault("internalServer.compress.enabled", &DefaultServerCompressEnabled)
	vip.SetDefault("internalServer.compress.level", DefaultServerCompressLevel)
	vip.SetDefault("internalServer.compress.types", DefaultServerCompressTypes)
	vip.SetDefault("internalServer.timeouts.readHeaderTimeout", DefaultServerTimeoutsReadHeaderTimeout)
	vip.SetDefault("templates.helpers", []string{DefaultTemplateHelpersPath})
	vip.SetDefault("templates.folderList.path", DefaultTemplateFolderListPath)
	vip.SetDefault("templates.folderList.headers", DefaultTemplateHeaders)
	vip.SetDefault("templates.folderList.status", DefaultTemplateStatusOk)
	vip.SetDefault("templates.targetList.path", DefaultTemplateTargetListPath)
	vip.SetDefault("templates.targetList.headers", DefaultTemplateHeaders)
	vip.SetDefault("templates.targetList.status", DefaultTemplateStatusOk)
	vip.SetDefault("templates.notFoundError.path", DefaultTemplateNotFoundErrorPath)
	vip.SetDefault("templates.notFoundError.headers", DefaultTemplateHeaders)
	vip.SetDefault("templates.notFoundError.status", DefaultTemplateStatusNotFound)
	vip.SetDefault("templates.internalServerError.path", DefaultTemplateInternalServerErrorPath)
	vip.SetDefault("templates.internalServerError.headers", DefaultTemplateHeaders)
	vip.SetDefault("templates.internalServerError.status", DefaultTemplateStatusInternalServerError)
	vip.SetDefault("templates.unauthorizedError.path", DefaultTemplateUnauthorizedErrorPath)
	vip.SetDefault("templates.unauthorizedError.headers", DefaultTemplateHeaders)
	vip.SetDefault("templates.unauthorizedError.status", DefaultTemplateStatusUnauthorized)
	vip.SetDefault("templates.forbiddenError.path", DefaultTemplateForbiddenErrorPath)
	vip.SetDefault("templates.forbiddenError.headers", DefaultTemplateHeaders)
	vip.SetDefault("templates.forbiddenError.status", DefaultTemplateStatusForbidden)
	vip.SetDefault("templates.badRequestError.path", DefaultTemplateBadRequestErrorPath)
	vip.SetDefault("templates.badRequestError.headers", DefaultTemplateHeaders)
	vip.SetDefault("templates.badRequestError.status", DefaultTemplateStatusBadRequest)
	vip.SetDefault("templates.put.path", DefaultTemplatePutPath)
	vip.SetDefault("templates.put.headers", DefaultEmptyTemplateHeaders)
	vip.SetDefault("templates.put.status", DefaultTemplateStatusNoContent)
	vip.SetDefault("templates.delete.path", DefaultTemplateDeletePath)
	vip.SetDefault("templates.delete.headers", DefaultEmptyTemplateHeaders)
	vip.SetDefault("templates.delete.status", DefaultTemplateStatusNoContent)
}

func generateViperInstances(files []os.DirEntry) []*viper.Viper {
	list := make([]*viper.Viper, 0)
	// Loop over static files to create viper instance for them
	funk.ForEach(files, func(file os.DirEntry) {
		filename := file.Name()
		// Create config file name
		cfgFileName := strings.TrimSuffix(filename, path.Ext(filename))
		// Test if config file name is compliant (ignore hidden files like .keep or directory)
		if !strings.HasPrefix(filename, ".") && cfgFileName != "" && !file.IsDir() {
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
	// Load must start by flushing all existing watcher on internal files
	for i := 0; i < len(ctx.internalFileWatchChannels); i++ {
		ch := ctx.internalFileWatchChannels[i]
		// Send the force stop
		ch <- true
	}

	// Create a viper instance for default value and merging
	globalViper := viper.New()

	// Put default values
	ctx.loadDefaultConfigurationValues(globalViper)

	// Loop over configs
	for _, vip := range ctx.configs {
		err := vip.ReadInConfig()
		if err != nil {
			return errors.WithStack(err)
		}

		err = globalViper.MergeConfigMap(vip.AllSettings())
		if err != nil {
			return errors.WithStack(err)
		}
	}

	// Prepare configuration object
	var out Config
	// Quick unmarshal.
	err := globalViper.Unmarshal(&out)
	if err != nil {
		return errors.WithStack(err)
	}

	// Load default values
	err = loadBusinessDefaultValues(&out)
	if err != nil {
		return err
	}

	// Configuration validation
	err = validate.Struct(out)
	if err != nil {
		return errors.WithStack(err)
	}

	// Load all credentials
	credentials, err := loadAllCredentials(&out)
	if err != nil {
		return err
	}
	// Initialize or flush watch internal file channels
	internalFileWatchChannels := make([]chan bool, 0)
	ctx.internalFileWatchChannels = internalFileWatchChannels
	// Loop over all credentials in order to watch file change
	funk.ForEach(credentials, func(cred *CredentialConfig) {
		// Check if credential is about a path
		if cred.Path != "" {
			// Create channel
			ch := make(chan bool)
			// Run the watch file
			ctx.watchInternalFile(cred.Path, ch, func() {
				// File change detected
				ctx.logger.Infof("Reload credential file detected for path %s", cred.Path)

				// Reload config
				err2 := loadCredential(cred)
				if err2 != nil {
					ctx.logger.Error(err2)
					// Stop here and do not call hooks => configuration is unstable
					return
				}
				// Call all hooks
				funk.ForEach(ctx.onChangeHooks, func(hook func()) { hook() })
			})
			// Add channel to list of channels
			ctx.internalFileWatchChannels = append(ctx.internalFileWatchChannels, ch)
		}
	})

	err = validateBusinessConfig(&out)
	if err != nil {
		return err
	}

	ctx.cfg = &out

	return nil
}

// GetConfig allow to get configuration object.
func (ctx *managercontext) GetConfig() *Config {
	return ctx.cfg
}

func loadAllCredentials(out *Config) ([]*CredentialConfig, error) {
	// Initialize answer
	result := make([]*CredentialConfig, 0)

	// Load credentials from declaration
	for _, item := range out.Targets {
		// Check if resources are declared
		if item.Resources != nil {
			for j := 0; j < len(item.Resources); j++ {
				// Store ressource
				res := item.Resources[j]
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
						// Save credential
						result = append(result, it.Password)
					}
				}
			}
		}
		// Check if actions are declared
		if item.Actions != nil {
			// Check if GET actions are declared and webhook configs
			if item.Actions.GET != nil && item.Actions.GET.Config != nil {
				// Load webhook secrets
				res, err := loadWebhookCfgCredentials(item.Actions.GET.Config.Webhooks)
				// Check error
				if err != nil {
					return nil, err
				}
				// Save credential
				result = append(result, res...)
			}

			// Check if PUT actions are declared and webhook configs
			if item.Actions.PUT != nil && item.Actions.PUT.Config != nil {
				// Load webhook secrets
				res, err := loadWebhookCfgCredentials(item.Actions.PUT.Config.Webhooks)
				// Check error
				if err != nil {
					return nil, err
				}
				// Save credential
				result = append(result, res...)
			}

			// Check if DELETE actions are declared and webhook configs
			if item.Actions.DELETE != nil && item.Actions.DELETE.Config != nil {
				// Load webhook secrets
				res, err := loadWebhookCfgCredentials(item.Actions.DELETE.Config.Webhooks)
				// Check error
				if err != nil {
					return nil, err
				}
				// Save credential
				result = append(result, res...)
			}
		}
		// Load credentials for access key and secret key
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
			// Save credential
			result = append(result, item.Bucket.Credentials.AccessKey, item.Bucket.Credentials.SecretKey)
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
						return nil, err
					}
					// Save credential
					result = append(result, v.ClientSecret)
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
			// Save credential
			result = append(result, it.Password)
		}
	}

	// Load SSL S3 credentials from server/internal server
	if out.Server != nil {
		serverCreds, err := loadServerSSLCredentials(out.Server)
		if err != nil {
			return nil, err
		}

		result = append(result, serverCreds...)
	}

	if out.InternalServer != nil {
		serverCreds, err := loadServerSSLCredentials(out.InternalServer)
		if err != nil {
			return nil, err
		}

		result = append(result, serverCreds...)
	}

	return result, nil
}

// loadServerSSLCredentials is used for any bucket-specific credentials under the
// {server/internalServer}.ssl.certificates[*].certificateUrlConfig / privateKeyUrlConfig branches.
func loadServerSSLCredentials(serverConfig *ServerConfig) ([]*CredentialConfig, error) {
	if serverConfig.SSL == nil {
		return nil, nil
	}

	res := make([]*CredentialConfig, 0)

	for _, cert := range serverConfig.SSL.Certificates {
		if cert.CertificateURLConfig != nil && cert.CertificateURLConfig.AWSCredentials != nil {
			s3Creds := cert.CertificateURLConfig.AWSCredentials

			if s3Creds.AccessKey != nil {
				err := loadCredential(s3Creds.AccessKey)
				if err != nil {
					return nil, err
				}
			}

			if s3Creds.SecretKey != nil {
				err := loadCredential(s3Creds.SecretKey)
				if err != nil {
					return nil, err
				}
			}
		}

		if cert.PrivateKeyURLConfig != nil && cert.PrivateKeyURLConfig.AWSCredentials != nil {
			s3Creds := cert.PrivateKeyURLConfig.AWSCredentials

			if s3Creds.AccessKey != nil {
				err := loadCredential(s3Creds.AccessKey)
				if err != nil {
					return nil, err
				}
			}

			if s3Creds.SecretKey != nil {
				err := loadCredential(s3Creds.SecretKey)
				if err != nil {
					return nil, err
				}
			}
		}
	}

	return res, nil
}

func loadWebhookCfgCredentials(cfgList []*WebhookConfig) ([]*CredentialConfig, error) {
	// Create result
	res := make([]*CredentialConfig, 0)
	// Loop over the list
	for _, wbCfg := range cfgList {
		// Loop over the secret header
		for _, secCre := range wbCfg.SecretHeaders {
			// Loop credential
			err := loadCredential(secCre)
			// Check error
			if err != nil {
				return nil, err
			}

			// Save
			res = append(res, secCre)
		}
	}

	// Default
	return res, nil
}

func loadCredential(credCfg *CredentialConfig) error {
	if credCfg.Path != "" {
		// Secret file
		databytes, err := os.ReadFile(credCfg.Path)
		if err != nil {
			return errors.WithStack(err)
		}
		// Store val
		val := string(databytes)
		// Clean new lines
		val = generalutils.NewLineMatcherRegex.ReplaceAllString(val, "")

		credCfg.Value = val
	} else if credCfg.Env != "" {
		// Environment variable
		envValue := os.Getenv(credCfg.Env)
		if envValue == "" {
			err := fmt.Errorf(TemplateErrLoadingEnvCredentialEmpty, credCfg.Env)

			return errors.WithStack(err)
		}
		// Store value
		credCfg.Value = envValue
	}
	// Default value
	return nil
}

func loadResourceValues(res *Resource) error {
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

	// Check if tags are set in OPA server authorizations
	if res.OIDC != nil && res.OIDC.AuthorizationOPAServer != nil && res.OIDC.AuthorizationOPAServer.Tags == nil {
		res.OIDC.AuthorizationOPAServer.Tags = map[string]string{}
	}

	return nil
}

func loadBusinessDefaultValues(out *Config) error {
	// Manage default values for targets
	for key, item := range out.Targets {
		// Put target name in structure with key as value
		item.Name = key

		// Manage default configuration for target region
		if item.Bucket != nil && item.Bucket.Region == "" {
			item.Bucket.Region = DefaultBucketRegion
		}
		// Manage default configuration for bucket S3 List Max Keys
		if item.Bucket != nil && item.Bucket.S3ListMaxKeys == 0 {
			item.Bucket.S3ListMaxKeys = DefaultBucketS3ListMaxKeys
		}
		// Manage default configuration for target actions
		if item.Actions == nil {
			item.Actions = &ActionsConfig{GET: &GetActionConfig{Enabled: true}}
		}
		// Manage values for signed url
		if item.Actions != nil && item.Actions.GET != nil && item.Actions.GET.Config != nil {
			// Check if expiration is set
			if item.Actions.GET.Config.SignedURLExpirationString != "" {
				// Parse it
				dur, err := time.ParseDuration(item.Actions.GET.Config.SignedURLExpirationString)
				// Check error
				if err != nil {
					return errors.WithStack(err)
				}
				// Save
				item.Actions.GET.Config.SignedURLExpiration = dur
			} else {
				// Set default one
				item.Actions.GET.Config.SignedURLExpiration = DefaultTargetActionsGETConfigSignedURLExpiration
			}
		}
		// Manage default for target templates configurations
		// Else put default headers for template override
		if item.Templates == nil {
			item.Templates = &TargetTemplateConfig{}
		} else {
			// Check if folder list template have been override and not headers
			if item.Templates.FolderList != nil && item.Templates.FolderList.Headers == nil {
				item.Templates.FolderList.Headers = DefaultTemplateHeaders
			}
			// Check if not found error template have been override and not headers
			if item.Templates.NotFoundError != nil && item.Templates.NotFoundError.Headers == nil {
				item.Templates.NotFoundError.Headers = DefaultTemplateHeaders
			}
			// Check if internal server error template have been override and not headers
			if item.Templates.InternalServerError != nil && item.Templates.InternalServerError.Headers == nil {
				item.Templates.InternalServerError.Headers = DefaultTemplateHeaders
			}
			// Check if forbidden error template have been override and not headers
			if item.Templates.ForbiddenError != nil && item.Templates.ForbiddenError.Headers == nil {
				item.Templates.ForbiddenError.Headers = DefaultTemplateHeaders
			}
			// Check if unauthorized error template have been override and not headers
			if item.Templates.UnauthorizedError != nil && item.Templates.UnauthorizedError.Headers == nil {
				item.Templates.UnauthorizedError.Headers = DefaultTemplateHeaders
			}
			// Check if bad request error template have been override and not headers
			if item.Templates.BadRequestError != nil && item.Templates.BadRequestError.Headers == nil {
				item.Templates.BadRequestError.Headers = DefaultTemplateHeaders
			}

			// Check if put template have been override and not headers
			if item.Templates.Put != nil && item.Templates.Put.Headers == nil {
				item.Templates.Put.Headers = DefaultEmptyTemplateHeaders
			}

			// Check if delete template have been override and not headers
			if item.Templates.Delete != nil && item.Templates.Delete.Headers == nil {
				item.Templates.Delete.Headers = DefaultEmptyTemplateHeaders
			}
		}
		// Manage default value for resources methods
		if item.Resources != nil {
			for _, res := range item.Resources {
				// Load default resource values
				err := loadResourceValues(res)
				if err != nil {
					return err
				}
			}
		}
		// Manage key write list
		if item.KeyRewriteList != nil {
			// Loop over keys
			for _, it := range item.KeyRewriteList {
				// Load key rewrite
				err := loadKeyRewriteValues(it)
				// Check error
				if err != nil {
					return err
				}
			}
		}
	}

	// Manage default value for list targets resources
	if out.ListTargets != nil && out.ListTargets.Resource != nil {
		// Store resource object
		res := out.ListTargets.Resource

		// Load default resource values
		err := loadResourceValues(res)
		if err != nil {
			return err
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

	// Manage default value for tracing
	if out.Tracing == nil {
		out.Tracing = &TracingConfig{Enabled: false}
	}

	return nil
}

func loadKeyRewriteValues(item *TargetKeyRewriteConfig) error {
	// Check if target type is set, if not, put REGEX type as default
	if item.TargetType == "" {
		item.TargetType = RegexTargetKeyRewriteTargetType
	}

	// Parse source regex
	rs, err := regexp.Compile(item.Source)
	// Check error
	if err != nil {
		return errors.WithStack(err)
	}
	// Save source
	item.SourceRegex = rs

	// Default value
	return nil
}

// Load Regex in OIDC Authorization access objects.
func loadRegexOIDCAuthorizationAccess(item *HeaderOIDCAuthorizationAccess) error {
	if item.Regexp {
		// Try to compile regex for group or email
		// Group case
		if item.Group != "" {
			// Compile Regexp
			reg, err2 := regexp.Compile(item.Group)
			// Check error
			if err2 != nil {
				return errors.WithStack(err2)
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
				return errors.WithStack(err2)
			}
			// Save regexp
			item.EmailRegexp = reg
		}
	}

	return nil
}
