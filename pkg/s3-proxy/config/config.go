package config

import (
	"regexp"
	"strings"
	"time"

	"emperror.dev/errors"
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

// DefaultTemplatePutPath Default template put path.
const DefaultTemplatePutPath = "templates/put.tpl"

// DefaultTemplateDeletePath Default template delete path.
const DefaultTemplateDeletePath = "templates/delete.tpl"

// DefaultTemplateHelpersPath Default template helpers path.
const DefaultTemplateHelpersPath = "templates/_helpers.tpl"

// DefaultTemplateHeaders Default template headers.
var DefaultTemplateHeaders = map[string]string{
	"Content-Type": "{{ template \"main.headers.contentType\" . }}",
}

// DefaultEmptyTemplateHeaders Default empty template headers.
var DefaultEmptyTemplateHeaders = map[string]string{}

// DefaultTemplateStatusOk Default template for status ok.
const DefaultTemplateStatusOk = "200"

// DefaultTemplateStatusNoContent Default template for status no content.
const DefaultTemplateStatusNoContent = "204"

// DefaultTemplateStatusNotFound Default template for status not found.
const DefaultTemplateStatusNotFound = "404"

// DefaultTemplateStatusForbidden Default template for status forbidden.
const DefaultTemplateStatusForbidden = "403"

// DefaultTemplateStatusBadRequest Default template for status bad request.
const DefaultTemplateStatusBadRequest = "400"

// DefaultTemplateStatusUnauthorized Default template for status Unauthorized.
const DefaultTemplateStatusUnauthorized = "401"

// DefaultTemplateStatusInternalServerError Default template for status Internal Server Error.
const DefaultTemplateStatusInternalServerError = "500"

// DefaultOIDCScopes Default OIDC Scopes.
var DefaultOIDCScopes = []string{"openid", "profile", "email"}

// DefaultOIDCGroupClaim Default OIDC group claim.
const DefaultOIDCGroupClaim = "groups"

// DefaultOIDCCookieName Default OIDC Cookie name.
const DefaultOIDCCookieName = "oidc"

// RegexTargetKeyRewriteTargetType Regex target key rewrite Target type.
const RegexTargetKeyRewriteTargetType = "REGEX"

// TemplateTargetKeyRewriteTargetType Template Target key rewrite Target type.
const TemplateTargetKeyRewriteTargetType = "TEMPLATE"

// DefaultServerTimeoutsReadHeaderTimeout Server timeouts ReadHeaderTimeout.
const DefaultServerTimeoutsReadHeaderTimeout = "60s"

// DefaultTargetActionsGETConfigSignedURLExpiration default signed url expiration.
const DefaultTargetActionsGETConfigSignedURLExpiration = 15 * time.Minute

// ErrMainBucketPathSupportNotValid Error thrown when main bucket path support option isn't valid.
var ErrMainBucketPathSupportNotValid = errors.New("main bucket path support option can be enabled only when only one bucket is configured")

// TemplateErrLoadingEnvCredentialEmpty Template Error when Loading Environment variable Credentials.
var TemplateErrLoadingEnvCredentialEmpty = "error loading credentials, environment variable %s is empty" //nolint: gosec // No credentials here, false positive

const oidcLoginPathTemplate = "/auth/%s"
const oidcCallbackPathTemplate = "/auth/%s/callback"

// Config Application Configuration.
type Config struct {
	Log            *LogConfig               `mapstructure:"log"`
	Tracing        *TracingConfig           `mapstructure:"tracing"`
	Server         *ServerConfig            `mapstructure:"server"`
	InternalServer *ServerConfig            `mapstructure:"internalServer"`
	Targets        map[string]*TargetConfig `mapstructure:"targets"        validate:"dive"`
	Templates      *TemplateConfig          `mapstructure:"templates"`
	AuthProviders  *AuthProviderConfig      `mapstructure:"authProviders"`
	ListTargets    *ListTargetsConfig       `mapstructure:"listTargets"`
}

// TracingConfig represents the Tracing configuration structure.
type TracingConfig struct {
	FixedTags     map[string]interface{} `mapstructure:"fixedTags"`
	FlushInterval string                 `mapstructure:"flushInterval"`
	UDPHost       string                 `mapstructure:"udpHost"`
	QueueSize     int                    `mapstructure:"queueSize"`
	Enabled       bool                   `mapstructure:"enabled"`
	LogSpan       bool                   `mapstructure:"logSpan"`
}

// ListTargetsConfig List targets configuration.
type ListTargetsConfig struct {
	Mount    *MountConfig `mapstructure:"mount"    validate:"required_with=Enabled"`
	Resource *Resource    `mapstructure:"resource" validate:"omitempty"`
	Enabled  bool         `mapstructure:"enabled"`
}

// MountConfig Mount configuration.
type MountConfig struct {
	Host string   `mapstructure:"host"`
	Path []string `mapstructure:"path" validate:"required,dive,required"`
}

// AuthProviderConfig Authentication provider configurations.
type AuthProviderConfig struct {
	Basic  map[string]*BasicAuthConfig  `mapstructure:"basic"  validate:"omitempty,dive"`
	OIDC   map[string]*OIDCAuthConfig   `mapstructure:"oidc"   validate:"omitempty,dive"`
	Header map[string]*HeaderAuthConfig `mapstructure:"header" validate:"omitempty,dive"`
}

// OIDCAuthConfig OpenID Connect authentication configurations.
type OIDCAuthConfig struct {
	ClientSecret  *CredentialConfig `mapstructure:"clientSecret"  validate:"omitempty,dive"`
	GroupClaim    string            `mapstructure:"groupClaim"`
	IssuerURL     string            `mapstructure:"issuerUrl"     validate:"required,url"`
	RedirectURL   string            `mapstructure:"redirectUrl"   validate:"omitempty,url"`
	State         string            `mapstructure:"state"         validate:"required"`
	ClientID      string            `mapstructure:"clientID"      validate:"required"`
	CookieName    string            `mapstructure:"cookieName"`
	LoginPath     string            `mapstructure:"loginPath"`
	CallbackPath  string            `mapstructure:"callbackPath"`
	Scopes        []string          `mapstructure:"scopes"`
	CookieDomains []string          `mapstructure:"cookieDomains"`
	EmailVerified bool              `mapstructure:"emailVerified"`
	CookieSecure  bool              `mapstructure:"cookieSecure"`
}

// HeaderOIDCAuthorizationAccess OpenID Connect or Header authorization accesses.
type HeaderOIDCAuthorizationAccess struct {
	GroupRegexp *regexp.Regexp
	EmailRegexp *regexp.Regexp
	Group       string `mapstructure:"group"  validate:"required_without=Email"`
	Email       string `mapstructure:"email"  validate:"required_without=Group"`
	Regexp      bool   `mapstructure:"regexp"`
}

// BasicAuthConfig Basic auth configurations.
type BasicAuthConfig struct {
	Realm string `mapstructure:"realm" validate:"required"`
}

// HeaderAuthConfig Header auth configuration.
type HeaderAuthConfig struct {
	UsernameHeader string `mapstructure:"usernameHeader" validate:"required"`
	EmailHeader    string `mapstructure:"emailHeader"    validate:"required"`
	GroupsHeader   string `mapstructure:"groupsHeader"`
}

// BasicAuthUserConfig Basic User auth configuration.
type BasicAuthUserConfig struct {
	Password *CredentialConfig `mapstructure:"password" validate:"required,dive"`
	User     string            `mapstructure:"user"     validate:"required"`
}

// TemplateConfigItem Template configuration item.
type TemplateConfigItem struct {
	Path    string            `mapstructure:"path"    validate:"required"`
	Headers map[string]string `mapstructure:"headers"`
	Status  string            `mapstructure:"status"`
}

// TemplateConfig Templates configuration.
type TemplateConfig struct {
	FolderList          *TemplateConfigItem `mapstructure:"folderList"          validate:"required"`
	TargetList          *TemplateConfigItem `mapstructure:"targetList"          validate:"required"`
	NotFoundError       *TemplateConfigItem `mapstructure:"notFoundError"       validate:"required"`
	InternalServerError *TemplateConfigItem `mapstructure:"internalServerError" validate:"required"`
	UnauthorizedError   *TemplateConfigItem `mapstructure:"unauthorizedError"   validate:"required"`
	ForbiddenError      *TemplateConfigItem `mapstructure:"forbiddenError"      validate:"required"`
	BadRequestError     *TemplateConfigItem `mapstructure:"badRequestError"     validate:"required"`
	Put                 *TemplateConfigItem `mapstructure:"put"                 validate:"required"`
	Delete              *TemplateConfigItem `mapstructure:"delete"              validate:"required"`
	Helpers             []string            `mapstructure:"helpers"             validate:"required,min=1,dive,required"`
}

// ServerConfig Server configuration.
type ServerConfig struct {
	Timeouts   *ServerTimeoutsConfig `mapstructure:"timeouts"   validate:"required"`
	CORS       *ServerCorsConfig     `mapstructure:"cors"       validate:"omitempty"`
	Cache      *CacheConfig          `mapstructure:"cache"      validate:"omitempty"`
	Compress   *ServerCompressConfig `mapstructure:"compress"   validate:"omitempty"`
	SSL        *ServerSSLConfig      `mapstructure:"ssl"        validate:"omitempty"`
	ListenAddr string                `mapstructure:"listenAddr"`
	Port       int                   `mapstructure:"port"       validate:"required"`
}

// ServerTimeoutsConfig Server timeouts configuration.
type ServerTimeoutsConfig struct {
	ReadTimeout       string `mapstructure:"readTimeout"`
	ReadHeaderTimeout string `mapstructure:"readHeaderTimeout"`
	WriteTimeout      string `mapstructure:"writeTimeout"`
	IdleTimeout       string `mapstructure:"idleTimeout"`
}

// ServerCompressConfig Server compress configuration.
type ServerCompressConfig struct {
	Enabled *bool    `mapstructure:"enabled"`
	Types   []string `mapstructure:"types"   validate:"required,min=1"`
	Level   int      `mapstructure:"level"   validate:"required,min=1"`
}

// ServerSSLConfig Server SSL configuration.
type ServerSSLConfig struct {
	MinTLSVersion       *string                 `mapstructure:"minTLSVersion"`
	MaxTLSVersion       *string                 `mapstructure:"maxTLSVersion"`
	Certificates        []*ServerSSLCertificate `mapstructure:"certificates"`
	SelfSignedHostnames []string                `mapstructure:"selfSignedHostnames"`
	CipherSuites        []string                `mapstructure:"cipherSuites"`
	Enabled             bool                    `mapstructure:"enabled"`
}

// ServerSSLCertificate Server SSL certificate.
type ServerSSLCertificate struct {
	Certificate          *string       `mapstructure:"certificate"`
	CertificateURL       *string       `mapstructure:"certificateUrl"`
	CertificateURLConfig *SSLURLConfig `mapstructure:"certificateUrlConfig"`
	PrivateKey           *string       `mapstructure:"privateKey"`
	PrivateKeyURL        *string       `mapstructure:"privateKeyUrl"`
	PrivateKeyURLConfig  *SSLURLConfig `mapstructure:"privateKeyUrlConfig"`
}

// SSLURLConfig SSL certificate/private key configuration for URLs.
type SSLURLConfig struct {
	AWSCredentials *BucketCredentialConfig `mapstructure:"awsCredentials" validate:"omitempty,dive"`
	HTTPTimeout    string                  `mapstructure:"httpTimeout"`
	AWSRegion      string                  `mapstructure:"awsRegion"`
	AWSEndpoint    string                  `mapstructure:"awsEndpoint"`
	AWSDisableSSL  bool                    `mapstructure:"awsDisableSSL"`
}

// CacheConfig Cache configuration.
type CacheConfig struct {
	Expires        string `mapstructure:"expires"`
	CacheControl   string `mapstructure:"cacheControl"`
	Pragma         string `mapstructure:"pragma"`
	XAccelExpires  string `mapstructure:"xAccelExpires"`
	NoCacheEnabled bool   `mapstructure:"noCacheEnabled"`
}

// ServerCorsConfig Server CORS configuration.
type ServerCorsConfig struct {
	MaxAge             *int     `mapstructure:"maxAge"`
	AllowCredentials   *bool    `mapstructure:"allowCredentials"`
	Debug              *bool    `mapstructure:"debug"`
	OptionsPassthrough *bool    `mapstructure:"optionsPassthrough"`
	AllowOrigins       []string `mapstructure:"allowOrigins"`
	AllowMethods       []string `mapstructure:"allowMethods"`
	AllowHeaders       []string `mapstructure:"allowHeaders"`
	ExposeHeaders      []string `mapstructure:"exposeHeaders"`
	Enabled            bool     `mapstructure:"enabled"`
	AllowAll           bool     `mapstructure:"allowAll"`
}

// TargetConfig Bucket instance configuration.
type TargetConfig struct {
	Name           string                    `validate:"required"`
	Bucket         *BucketConfig             `mapstructure:"bucket"         validate:"required"`
	Resources      []*Resource               `mapstructure:"resources"      validate:"dive"`
	Mount          *MountConfig              `mapstructure:"mount"          validate:"required"`
	Actions        *ActionsConfig            `mapstructure:"actions"`
	Templates      *TargetTemplateConfig     `mapstructure:"templates"`
	KeyRewriteList []*TargetKeyRewriteConfig `mapstructure:"keyRewriteList"`
}

// TargetKeyRewriteConfig Target key rewrite configuration.
type TargetKeyRewriteConfig struct {
	Source      string `mapstructure:"source" validate:"required,min=1"`
	SourceRegex *regexp.Regexp
	Target      string `mapstructure:"target"     validate:"required,min=1"`
	TargetType  string `mapstructure:"targetType" validate:"required,oneof=REGEX TEMPLATE"`
}

// TargetTemplateConfig Target templates configuration to override default ones.
type TargetTemplateConfig struct {
	FolderList          *TargetTemplateConfigItem `mapstructure:"folderList"`
	NotFoundError       *TargetTemplateConfigItem `mapstructure:"notFoundError"`
	InternalServerError *TargetTemplateConfigItem `mapstructure:"internalServerError"`
	ForbiddenError      *TargetTemplateConfigItem `mapstructure:"forbiddenError"`
	UnauthorizedError   *TargetTemplateConfigItem `mapstructure:"unauthorizedError"`
	BadRequestError     *TargetTemplateConfigItem `mapstructure:"badRequestError"`
	Put                 *TargetTemplateConfigItem `mapstructure:"put"`
	Delete              *TargetTemplateConfigItem `mapstructure:"delete"`
	Helpers             []*TargetHelperConfigItem `mapstructure:"helpers"`
}

// TargetHelperConfigItem Target helper configuration item.
type TargetHelperConfigItem struct {
	Path     string `mapstructure:"path"     validate:"required,min=1"`
	InBucket bool   `mapstructure:"inBucket"`
}

// TargetTemplateConfigItem Target template configuration item.
type TargetTemplateConfigItem struct {
	Path     string            `mapstructure:"path"     validate:"required,min=1"`
	Headers  map[string]string `mapstructure:"headers"`
	Status   string            `mapstructure:"status"`
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
	Config  *DeleteActionConfigConfig `mapstructure:"config"`
	Enabled bool                      `mapstructure:"enabled"`
}

// DeleteActionConfigConfig Delete action configuration object configuration.
type DeleteActionConfigConfig struct {
	Webhooks []*WebhookConfig `mapstructure:"webhooks" validate:"dive"`
}

// PutActionConfig Put action configuration.
type PutActionConfig struct {
	Config  *PutActionConfigConfig `mapstructure:"config"`
	Enabled bool                   `mapstructure:"enabled"`
}

// PutActionConfigConfig Put action configuration object configuration.
type PutActionConfigConfig struct {
	Metadata       map[string]string                    `mapstructure:"metadata"`
	SystemMetadata *PutActionConfigSystemMetadataConfig `mapstructure:"systemMetadata"`
	CannedACL      *string                              `mapstructure:"cannedACL"`
	StorageClass   string                               `mapstructure:"storageClass"`
	Webhooks       []*WebhookConfig                     `mapstructure:"webhooks"       validate:"dive"`
	AllowOverride  bool                                 `mapstructure:"allowOverride"`
}

// PutActionConfigSystemMetadataConfig Put action configuration system metadata object configuration.
type PutActionConfigSystemMetadataConfig struct {
	CacheControl       string `mapstructure:"cacheControl"`
	ContentDisposition string `mapstructure:"contentDisposition"`
	ContentEncoding    string `mapstructure:"contentEncoding"`
	ContentLanguage    string `mapstructure:"contentLanguage"`
	Expires            string `mapstructure:"expires"`
}

// GetActionConfig Get action configuration.
type GetActionConfig struct {
	Config  *GetActionConfigConfig `mapstructure:"config"`
	Enabled bool                   `mapstructure:"enabled"`
}

// GetActionConfigConfig Get action configuration object configuration.
type GetActionConfigConfig struct {
	StreamedFileHeaders                      map[string]string `mapstructure:"streamedFileHeaders"`
	IndexDocument                            string            `mapstructure:"indexDocument"`
	SignedURLExpirationString                string            `mapstructure:"signedUrlExpiration"`
	Webhooks                                 []*WebhookConfig  `mapstructure:"webhooks"            validate:"dive"`
	SignedURLExpiration                      time.Duration
	RedirectWithTrailingSlashForNotFoundFile bool `mapstructure:"redirectWithTrailingSlashForNotFoundFile"`
	RedirectToSignedURL                      bool `mapstructure:"redirectToSignedUrl"`
}

// WebhookConfig Webhook configuration.
type WebhookConfig struct {
	Headers         map[string]string            `mapstructure:"headers"`
	SecretHeaders   map[string]*CredentialConfig `mapstructure:"secretHeaders"   validate:"omitempty,dive"`
	Method          string                       `mapstructure:"method"          validate:"required,oneof=POST PATCH PUT DELETE"`
	URL             string                       `mapstructure:"url"             validate:"required,url"`
	MaxWaitTime     string                       `mapstructure:"maxWaitTime"`
	DefaultWaitTime string                       `mapstructure:"defaultWaitTime"`
	RetryCount      int                          `mapstructure:"retryCount"      validate:"gte=0"`
}

// Resource Resource.
type Resource struct {
	WhiteList *bool               `mapstructure:"whiteList"`
	Basic     *ResourceBasic      `mapstructure:"basic"     validate:"omitempty"`
	OIDC      *ResourceHeaderOIDC `mapstructure:"oidc"      validate:"omitempty"`
	Header    *ResourceHeaderOIDC `mapstructure:"header"    validate:"omitempty"`
	Path      string              `mapstructure:"path"      validate:"required"`
	Provider  string              `mapstructure:"provider"`
	Methods   []string            `mapstructure:"methods"   validate:"required,dive,required"`
}

// ResourceBasic Basic auth resource.
type ResourceBasic struct {
	Credentials []*BasicAuthUserConfig `mapstructure:"credentials" validate:"omitempty,dive"`
}

// ResourceHeaderOIDC OIDC or Header auth Resource.
type ResourceHeaderOIDC struct {
	AuthorizationOPAServer *OPAServerAuthorization          `mapstructure:"authorizationOPAServer" validate:"omitempty,dive"`
	AuthorizationAccesses  []*HeaderOIDCAuthorizationAccess `mapstructure:"authorizationAccesses"  validate:"omitempty,dive"`
}

// OPAServerAuthorization OPA Server authorization.
type OPAServerAuthorization struct {
	Tags map[string]string `mapstructure:"tags"`
	URL  string            `mapstructure:"url"  validate:"required,url"`
}

// BucketConfig Bucket configuration.
type BucketConfig struct {
	Credentials   *BucketCredentialConfig `mapstructure:"credentials"   validate:"omitempty,dive"`
	RequestConfig *BucketRequestConfig    `mapstructure:"requestConfig" validate:"omitempty,dive"`
	Name          string                  `mapstructure:"name"          validate:"required"`
	Prefix        string                  `mapstructure:"prefix"`
	Region        string                  `mapstructure:"region"`
	S3Endpoint    string                  `mapstructure:"s3Endpoint"`
	S3ListMaxKeys int64                   `mapstructure:"s3ListMaxKeys" validate:"gt=0"`
	DisableSSL    bool                    `mapstructure:"disableSSL"`
}

// BucketRequestConfig Bucket request configuration.
type BucketRequestConfig struct {
	ListHeaders   map[string]string `mapstructure:"listHeaders"`
	GetHeaders    map[string]string `mapstructure:"getHeaders"`
	PutHeaders    map[string]string `mapstructure:"putHeaders"`
	DeleteHeaders map[string]string `mapstructure:"deleteHeaders"`
}

// BucketCredentialConfig Bucket Credentials configurations.
type BucketCredentialConfig struct {
	AccessKey *CredentialConfig `mapstructure:"accessKey" validate:"omitempty,dive"`
	SecretKey *CredentialConfig `mapstructure:"secretKey" validate:"omitempty,dive"`
}

// CredentialConfig Credential Configurations.
type CredentialConfig struct {
	Path  string `mapstructure:"path"  validate:"required_without_all=Env Value"`
	Env   string `mapstructure:"env"   validate:"required_without_all=Path Value"`
	Value string `mapstructure:"value" validate:"required_without_all=Path Env"`
}

// LogConfig Log configuration.
type LogConfig struct {
	Level    string `mapstructure:"level"    validate:"required"`
	Format   string `mapstructure:"format"   validate:"required"`
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
