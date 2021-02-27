package config

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/go-playground/validator/v10"
	"github.com/oxyno-zeta/s3-proxy/pkg/s3-proxy/log"
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
	files, err := ioutil.ReadDir(mainConfigFolderPath)
	if err != nil {
		return err
	}

	// Generate viper instances for static configs
	ctx.configs = generateViperInstances(files)

	// Load configuration
	err = ctx.loadConfiguration()
	if err != nil {
		return err
	}

	// Loop over config files
	funk.ForEach(ctx.configs, func(vv interface{}) {
		// Cast viper object
		vip := vv.(*viper.Viper)

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
			ctx.logger.Fatal(err)
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
	vip.SetDefault("internalServer.port", DefaultInternalPort)
	vip.SetDefault("templates.folderList", DefaultTemplateFolderListPath)
	vip.SetDefault("templates.targetList", DefaultTemplateTargetListPath)
	vip.SetDefault("templates.notFound", DefaultTemplateNotFoundPath)
	vip.SetDefault("templates.internalServerError", DefaultTemplateInternalServerErrorPath)
	vip.SetDefault("templates.unauthorized", DefaultTemplateUnauthorizedErrorPath)
	vip.SetDefault("templates.forbidden", DefaultTemplateForbiddenErrorPath)
	vip.SetDefault("templates.badRequest", DefaultTemplateBadRequestErrorPath)
}

func generateViperInstances(files []os.FileInfo) []*viper.Viper {
	list := make([]*viper.Viper, 0)
	// Loop over static files to create viper instance for them
	funk.ForEach(files, func(file os.FileInfo) {
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
			return err
		}

		err = globalViper.MergeConfigMap(vip.AllSettings())
		if err != nil {
			return err
		}
	}

	// Prepare configuration object
	var out Config
	// Quick unmarshal.
	err := globalViper.Unmarshal(&out)
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
	credentials, err := loadAllCredentials(&out)
	if err != nil {
		return err
	}
	// Initialize or flush watch internal file channels
	internalFileWatchChannels := make([]chan bool, 0)
	ctx.internalFileWatchChannels = internalFileWatchChannels
	// Loop over all credentials in order to watch file change
	funk.ForEach(credentials, func(item interface{}) {
		cred := item.(*CredentialConfig)
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

	return result, nil
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
	for _, item := range out.Targets {
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
		// DEPRECATED
		// Manage default value of index document for deprecated value
		if item.IndexDocument != "" && item.Actions.GET.IndexDocument == "" {
			item.Actions.GET.IndexDocument = item.IndexDocument
		}
		// Manage default for target templates configurations
		if item.Templates == nil {
			item.Templates = &TargetTemplateConfig{}
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
	// Parse source regex
	rs, err := regexp.Compile(item.Source)
	// Check error
	if err != nil {
		return err
	}
	// Save source
	item.SourceRegex = rs

	// Default value
	return nil
}

// Load Regex in OIDC Authorization access objects.
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
