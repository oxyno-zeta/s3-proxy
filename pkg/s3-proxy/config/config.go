package config

import (
	"regexp"
	"strings"

	"github.com/pkg/errors"
)

// DefaultPort Default port.
const DefaultPort = 8080

// DefaultServerCompressEnabled Default server compress enabled.
var DefaultServerCompressEnabled = true

// DefaultServerCompressLevel Default server compress level.
const DefaultServerCompressLevel = 5

// DefaultServerCompressTypes Default server compress types.
var DefaultServerCompressTypes = []string{
	"text/html",
	"text/css",
	"text/plain",
	"text/javascript",
	"application/javascript",
	"application/x-javascript",
	"application/json",
	"application/atom+xml",
	"application/rss+xml",
	"image/svg+xml",
}

// DefaultInternalPort Default internal port.
const DefaultInternalPort = 9090

// DefaultLogLevel Default log level.
const DefaultLogLevel = "info"

// DefaultLogFormat Default Log format.
const DefaultLogFormat = "json"

// DefaultBucketRegion Default bucket region.
const DefaultBucketRegion = "us-east-1"

// DefaultBucketS3ListMaxKeys Default bucket S3 list max keys.
const DefaultBucketS3ListMaxKeys int64 = 1000

// DefaultTemplateFolderListPath Default template folder list path.
const DefaultTemplateFolderListPath = "templates/folder-list.tpl"

// DefaultTemplateTargetListPath Default template target list path.
const DefaultTemplateTargetListPath = "templates/target-list.tpl"

// DefaultTemplateNotFoundErrorPath Default template not found path.
const DefaultTemplateNotFoundErrorPath = "templates/not-found-error.tpl"

// DefaultTemplateForbiddenErrorPath Default template forbidden path.
const DefaultTemplateForbiddenErrorPath = "templates/forbidden-error.tpl"

// DefaultTemplateBadRequestErrorPath Default template bad request path.
const DefaultTemplateBadRequestErrorPath = "templates/bad-request-error.tpl"

// DefaultTemplateInternalServerErrorPath Default template Internal server error path.
const DefaultTemplateInternalServerErrorPath = "templates/internal-server-error.tpl"

// DefaultTemplateUnauthorizedErrorPath Default template unauthorized error path.
const DefaultTemplateUnauthorizedErrorPath = "templates/unauthorized-error.tpl"

// DefaultTemplateHelpersPath Default template helpers path.
const DefaultTemplateHelpersPath = "templates/_helpers.tpl"

// DefaultTemplateHeaders Default template headers.
var DefaultTemplateHeaders = map[string]string{
	"Content-Type": "{{ template \"main.headers.contentType\" . }}",
}

// DefaultOIDCScopes Default OIDC Scopes.
var DefaultOIDCScopes = []string{"openid", "profile", "email"}

// DefaultOIDCGroupClaim Default OIDC group claim.
const DefaultOIDCGroupClaim = "groups"

// DefaultOIDCCookieName Default OIDC Cookie name.
const DefaultOIDCCookieName = "oidc"

// ErrMainBucketPathSupportNotValid Error thrown when main bucket path support option isn't valid.
var ErrMainBucketPathSupportNotValid = errors.New("main bucket path support option can be enabled only when only one bucket is configured")

// TemplateErrLoadingEnvCredentialEmpty Template Error when Loading Environment variable Credentials.
var TemplateErrLoadingEnvCredentialEmpty = "error loading credentials, environment variable %s is empty"

const oidcLoginPathTemplate = "/auth/%s"
const oidcCallbackPathTemplate = "/auth/%s/callback"

// Config Application Configuration.
type Config struct {
	Log            *LogConfig               `mapstructure:"log"`
	Tracing        *TracingConfig           `mapstructure:"tracing"`
	Server         *ServerConfig            `mapstructure:"server"`
	InternalServer *ServerConfig            `mapstructure:"internalServer"`
	Targets        map[string]*TargetConfig `mapstructure:"targets" validate:"required,dive"`
	Templates      *TemplateConfig          `mapstructure:"templates"`
	AuthProviders  *AuthProviderConfig      `mapstructure:"authProviders"`
	ListTargets    *ListTargetsConfig       `mapstructure:"listTargets"`
}

// TracingConfig represents the Tracing configuration structure.
type TracingConfig struct {
	Enabled       bool                   `mapstructure:"enabled"`
	LogSpan       bool                   `mapstructure:"logSpan"`
	FlushInterval string                 `mapstructure:"flushInterval"`
	UDPHost       string                 `mapstructure:"udpHost"`
	QueueSize     int                    `mapstructure:"queueSize"`
	FixedTags     map[string]interface{} `mapstructure:"fixedTags"`
}

// ListTargetsConfig List targets configuration.
type ListTargetsConfig struct {
	Enabled  bool         `mapstructure:"enabled"`
	Mount    *MountConfig `mapstructure:"mount" validate:"required_with=Enabled"`
	Resource *Resource    `mapstructure:"resource" validate:"omitempty"`
}

// MountConfig Mount configuration.
type MountConfig struct {
	Host string   `mapstructure:"host"`
	Path []string `mapstructure:"path" validate:"required,dive,required"`
}

// AuthProviderConfig Authentication provider configurations.
type AuthProviderConfig struct {
	Basic map[string]*BasicAuthConfig `mapstructure:"basic" validate:"omitempty,dive"`
	OIDC  map[string]*OIDCAuthConfig  `mapstructure:"oidc" validate:"omitempty,dive"`
}

// OIDCAuthConfig OpenID Connect authentication configurations.
type OIDCAuthConfig struct {
	ClientID      string            `mapstructure:"clientID" validate:"required"`
	ClientSecret  *CredentialConfig `mapstructure:"clientSecret" validate:"omitempty,dive"`
	IssuerURL     string            `mapstructure:"issuerUrl" validate:"required,url"`
	RedirectURL   string            `mapstructure:"redirectUrl" validate:"omitempty,url"`
	Scopes        []string          `mapstructure:"scopes"`
	State         string            `mapstructure:"state" validate:"required"`
	GroupClaim    string            `mapstructure:"groupClaim"`
	CookieName    string            `mapstructure:"cookieName"`
	EmailVerified bool              `mapstructure:"emailVerified"`
	CookieSecure  bool              `mapstructure:"cookieSecure"`
	LoginPath     string            `mapstructure:"loginPath"`
	CallbackPath  string            `mapstructure:"callbackPath"`
}

// OIDCAuthorizationAccess OpenID Connect authorization accesses.
type OIDCAuthorizationAccess struct {
	Group       string `mapstructure:"group" validate:"required_without=Email"`
	Email       string `mapstructure:"email" validate:"required_without=Group"`
	Regexp      bool   `mapstructure:"regexp"`
	GroupRegexp *regexp.Regexp
	EmailRegexp *regexp.Regexp
}

// BasicAuthConfig Basic auth configurations.
type BasicAuthConfig struct {
	Realm string `mapstructure:"realm" validate:"required"`
}

// BasicAuthUserConfig Basic User auth configuration.
type BasicAuthUserConfig struct {
	User     string            `mapstructure:"user" validate:"required"`
	Password *CredentialConfig `mapstructure:"password" validate:"required,dive"`
}

// TemplateConfigItem Template configuration item.
type TemplateConfigItem struct {
	Path    string            `mapstructure:"path" validate:"required"`
	Headers map[string]string `mapstructure:"headers"`
}

// TemplateConfig Templates configuration.
type TemplateConfig struct {
	Helpers             []string            `mapstructure:"helpers" validate:"required,min=1,dive,required"`
	FolderList          *TemplateConfigItem `mapstructure:"folderList" validate:"required"`
	TargetList          *TemplateConfigItem `mapstructure:"targetList" validate:"required"`
	NotFoundError       *TemplateConfigItem `mapstructure:"notFoundError" validate:"required"`
	InternalServerError *TemplateConfigItem `mapstructure:"internalServerError" validate:"required"`
	UnauthorizedError   *TemplateConfigItem `mapstructure:"unauthorizedError" validate:"required"`
	ForbiddenError      *TemplateConfigItem `mapstructure:"forbiddenError" validate:"required"`
	BadRequestError     *TemplateConfigItem `mapstructure:"badRequestError" validate:"required"`
}

// ServerConfig Server configuration.
type ServerConfig struct {
	ListenAddr string                `mapstructure:"listenAddr"`
	Port       int                   `mapstructure:"port" validate:"required"`
	CORS       *ServerCorsConfig     `mapstructure:"cors" validate:"omitempty"`
	Cache      *CacheConfig          `mapstructure:"cache" validate:"omitempty"`
	Compress   *ServerCompressConfig `mapstructure:"compress" validate:"omitempty"`
}

// ServerCompressConfig Server compress configuration.
type ServerCompressConfig struct {
	Enabled *bool    `mapstructure:"enabled"`
	Level   int      `mapstructure:"level" validate:"required,min=1"`
	Types   []string `mapstructure:"types" validate:"required,min=1"`
}

// CacheConfig Cache configuration.
type CacheConfig struct {
	NoCacheEnabled bool   `mapstructure:"noCacheEnabled"`
	Expires        string `mapstructure:"expires"`
	CacheControl   string `mapstructure:"cacheControl"`
	Pragma         string `mapstructure:"pragma"`
	XAccelExpires  string `mapstructure:"xAccelExpires"`
}

// ServerCorsConfig Server CORS configuration.
type ServerCorsConfig struct {
	Enabled            bool     `mapstructure:"enabled"`
	AllowAll           bool     `mapstructure:"allowAll"`
	AllowOrigins       []string `mapstructure:"allowOrigins"`
	AllowMethods       []string `mapstructure:"allowMethods"`
	AllowHeaders       []string `mapstructure:"allowHeaders"`
	ExposeHeaders      []string `mapstructure:"exposeHeaders"`
	MaxAge             *int     `mapstructure:"maxAge"`
	AllowCredentials   *bool    `mapstructure:"allowCredentials"`
	Debug              *bool    `mapstructure:"debug"`
	OptionsPassthrough *bool    `mapstructure:"optionsPassthrough"`
}

// TargetConfig Bucket instance configuration.
type TargetConfig struct {
	Name           string                    `validate:"required"`
	Bucket         *BucketConfig             `mapstructure:"bucket" validate:"required"`
	Resources      []*Resource               `mapstructure:"resources" validate:"dive"`
	Mount          *MountConfig              `mapstructure:"mount" validate:"required"`
	Actions        *ActionsConfig            `mapstructure:"actions"`
	Templates      *TargetTemplateConfig     `mapstructure:"templates"`
	KeyRewriteList []*TargetKeyRewriteConfig `mapstructure:"keyRewriteList"`
}

// TargetKeyRewriteConfig Target key rewrite configuration.
type TargetKeyRewriteConfig struct {
	Source      string `mapstructure:"source" validate:"required,min=1"`
	SourceRegex *regexp.Regexp
	Target      string `mapstructure:"target" validate:"required,min=1"`
}

// TargetTemplateConfig Target templates configuration to override default ones.
type TargetTemplateConfig struct {
	Helpers             []*TargetHelperConfigItem `mapstructure:"helpers"`
	FolderList          *TargetTemplateConfigItem `mapstructure:"folderList"`
	NotFoundError       *TargetTemplateConfigItem `mapstructure:"notFoundError"`
	InternalServerError *TargetTemplateConfigItem `mapstructure:"internalServerError"`
	ForbiddenError      *TargetTemplateConfigItem `mapstructure:"forbiddenError"`
	UnauthorizedError   *TargetTemplateConfigItem `mapstructure:"unauthorizedError"`
	BadRequestError     *TargetTemplateConfigItem `mapstructure:"badRequestError"`
}

// TargetHelperConfigItem Target helper configuration item.
type TargetHelperConfigItem struct {
	Path     string `mapstructure:"path" validate:"required,min=1"`
	InBucket bool   `mapstructure:"inBucket"`
}

// TargetTemplateConfigItem Target template configuration item.
type TargetTemplateConfigItem struct {
	Path     string            `mapstructure:"path" validate:"required,min=1"`
	Headers  map[string]string `mapstructure:"headers"`
	InBucket bool              `mapstructure:"inBucket"`
}

// ActionsConfig is dedicated to actions configuration in a target.
type ActionsConfig struct {
	GET    *GetActionConfig    `mapstructure:"GET"`
	PUT    *PutActionConfig    `mapstructure:"PUT"`
	DELETE *DeleteActionConfig `mapstructure:"DELETE"`
}

// DeleteActionConfig Delete action configuration.
type DeleteActionConfig struct {
	Enabled bool                      `mapstructure:"enabled"`
	Config  *DeleteActionConfigConfig `mapstructure:"config"`
}

// DeleteActionConfigConfig Delete action configuration object configuration.
type DeleteActionConfigConfig struct {
	Webhooks []*WebhookConfig `mapstructure:"webhooks" validate:"dive"`
}

// PutActionConfig Put action configuration.
type PutActionConfig struct {
	Enabled bool                   `mapstructure:"enabled"`
	Config  *PutActionConfigConfig `mapstructure:"config"`
}

// PutActionConfigConfig Put action configuration object configuration.
type PutActionConfigConfig struct {
	Metadata      map[string]string `mapstructure:"metadata"`
	StorageClass  string            `mapstructure:"storageClass"`
	AllowOverride bool              `mapstructure:"allowOverride"`
	Webhooks      []*WebhookConfig  `mapstructure:"webhooks" validate:"dive"`
}

// GetActionConfig Get action configuration.
type GetActionConfig struct {
	Enabled bool                   `mapstructure:"enabled"`
	Config  *GetActionConfigConfig `mapstructure:"config"`
}

// GetActionConfigConfig Get action configuration object configuration.
type GetActionConfigConfig struct {
	RedirectWithTrailingSlashForNotFoundFile bool              `mapstructure:"redirectWithTrailingSlashForNotFoundFile"`
	IndexDocument                            string            `mapstructure:"indexDocument"`
	StreamedFileHeaders                      map[string]string `mapstructure:"streamedFileHeaders"`
	Webhooks                                 []*WebhookConfig  `mapstructure:"webhooks" validate:"dive"`
}

// WebhookConfig Webhook configuration.
type WebhookConfig struct {
	Method          string                       `mapstructure:"method" validate:"required,oneof=POST PATCH PUT DELETE"`
	URL             string                       `mapstructure:"url" validate:"required,url"`
	Headers         map[string]string            `mapstructure:"headers"`
	SecretHeaders   map[string]*CredentialConfig `mapstructure:"secretHeaders" validate:"omitempty,dive"`
	RetryCount      int                          `mapstructure:"retryCount" validate:"gte=0"`
	MaxWaitTime     string                       `mapstructure:"maxWaitTime"`
	DefaultWaitTime string                       `mapstructure:"defaultWaitTime"`
}

// Resource Resource.
type Resource struct {
	Path      string         `mapstructure:"path" validate:"required"`
	Methods   []string       `mapstructure:"methods" validate:"required,dive,required"`
	WhiteList *bool          `mapstructure:"whiteList"`
	Provider  string         `mapstructure:"provider"`
	Basic     *ResourceBasic `mapstructure:"basic" validate:"omitempty"`
	OIDC      *ResourceOIDC  `mapstructure:"oidc" validate:"omitempty"`
}

// ResourceBasic Basic auth resource.
type ResourceBasic struct {
	Credentials []*BasicAuthUserConfig `mapstructure:"credentials" validate:"omitempty,dive"`
}

// ResourceOIDC OIDC auth Resource.
type ResourceOIDC struct {
	AuthorizationAccesses  []*OIDCAuthorizationAccess `mapstructure:"authorizationAccesses" validate:"omitempty,dive"`
	AuthorizationOPAServer *OPAServerAuthorization    `mapstructure:"authorizationOPAServer" validate:"omitempty,dive"`
}

// OPAServerAuthorization OPA Server authorization.
type OPAServerAuthorization struct {
	URL  string            `mapstructure:"url" validate:"required,url"`
	Tags map[string]string `mapstructure:"tags"`
}

// BucketConfig Bucket configuration.
type BucketConfig struct {
	Name          string                  `mapstructure:"name" validate:"required"`
	Prefix        string                  `mapstructure:"prefix"`
	Region        string                  `mapstructure:"region"`
	S3Endpoint    string                  `mapstructure:"s3Endpoint"`
	Credentials   *BucketCredentialConfig `mapstructure:"credentials" validate:"omitempty,dive"`
	DisableSSL    bool                    `mapstructure:"disableSSL"`
	S3ListMaxKeys int64                   `mapstructure:"s3ListMaxKeys" validate:"gt=0"`
}

// BucketCredentialConfig Bucket Credentials configurations.
type BucketCredentialConfig struct {
	AccessKey *CredentialConfig `mapstructure:"accessKey" validate:"omitempty,dive"`
	SecretKey *CredentialConfig `mapstructure:"secretKey" validate:"omitempty,dive"`
}

// CredentialConfig Credential Configurations.
type CredentialConfig struct {
	Path  string `mapstructure:"path" validate:"required_without_all=Env Value"`
	Env   string `mapstructure:"env" validate:"required_without_all=Path Value"`
	Value string `mapstructure:"value" validate:"required_without_all=Path Env"`
}

// LogConfig Log configuration.
type LogConfig struct {
	Level    string `mapstructure:"level" validate:"required"`
	Format   string `mapstructure:"format" validate:"required"`
	FilePath string `mapstructure:"filePath"`
}

// GetRootPrefix Get bucket root prefix.
func (bcfg *BucketConfig) GetRootPrefix() string {
	key := bcfg.Prefix
	// Check if key ends with a /, if key exists and don't ends with / add it
	if key != "" && !strings.HasSuffix(key, "/") {
		key += "/"
	}
	// Return result
	return key
}
