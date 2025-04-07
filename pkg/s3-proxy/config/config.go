package config

import (
	"regexp"
	"strings"
	"time"

	"emperror.dev/errors"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
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

// Default Upload configurations.
const (
	DefaultS3MaxUploadParts          = s3manager.MaxUploadParts
	DefaultS3UploadPartSize    int64 = 5
	DefaultS3UploadConcurrency       = s3manager.DefaultUploadConcurrency
)

const (
	oidcLoginPathTemplate    = "/auth/%s"
	oidcCallbackPathTemplate = "/auth/%s/callback"
)

// Config Application Configuration.
type Config struct {
	Log            *LogConfig               `json:"log"            mapstructure:"log"`
	Tracing        *TracingConfig           `json:"tracing"        mapstructure:"tracing"`
	Metrics        *MetricsConfig           `json:"metrics"        mapstructure:"metrics"`
	Server         *ServerConfig            `json:"server"         mapstructure:"server"`
	InternalServer *ServerConfig            `json:"internalServer" mapstructure:"internalServer"`
	Targets        map[string]*TargetConfig `json:"targets"        mapstructure:"targets"`
	Templates      *TemplateConfig          `json:"templates"      mapstructure:"templates"`
	AuthProviders  *AuthProviderConfig      `json:"authProviders"  mapstructure:"authProviders"`
	ListTargets    *ListTargetsConfig       `json:"listTargets"    mapstructure:"listTargets"`
}

// MetricsConfig represents the metrics configuration structure.
type MetricsConfig struct {
	DisableRouterPath bool `json:"disableRouterPath" mapstructure:"disableRouterPath"`
}

// TracingConfig represents the Tracing configuration structure.
type TracingConfig struct {
	FixedTags     map[string]interface{} `json:"fixedTags"     mapstructure:"fixedTags"`
	FlushInterval string                 `json:"flushInterval" mapstructure:"flushInterval"`
	UDPHost       string                 `json:"udpHost"       mapstructure:"udpHost"`
	QueueSize     int                    `json:"queueSize"     mapstructure:"queueSize"`
	Enabled       bool                   `json:"enabled"       mapstructure:"enabled"`
	LogSpan       bool                   `json:"logSpan"       mapstructure:"logSpan"`
}

// ListTargetsConfig List targets configuration.
type ListTargetsConfig struct {
	Mount    *MountConfig `json:"mount"    mapstructure:"mount"    validate:"required_with=Enabled"`
	Resource *Resource    `json:"resource" mapstructure:"resource" validate:"omitempty"`
	Enabled  bool         `json:"enabled"  mapstructure:"enabled"`
}

// MountConfig Mount configuration.
type MountConfig struct {
	Host string   `json:"host" mapstructure:"host"`
	Path []string `json:"path" mapstructure:"path" validate:"required,dive,required"`
}

// AuthProviderConfig Authentication provider configurations.
type AuthProviderConfig struct {
	Basic  map[string]*BasicAuthConfig  `json:"basic"  mapstructure:"basic"  validate:"omitempty"`
	OIDC   map[string]*OIDCAuthConfig   `json:"oidc"   mapstructure:"oidc"   validate:"omitempty"`
	Header map[string]*HeaderAuthConfig `json:"header" mapstructure:"header" validate:"omitempty"`
}

// OIDCAuthConfig OpenID Connect authentication configurations.
type OIDCAuthConfig struct {
	ClientSecret  *CredentialConfig `json:"clientSecret"  mapstructure:"clientSecret"  validate:"omitempty"`
	GroupClaim    string            `json:"groupClaim"    mapstructure:"groupClaim"`
	IssuerURL     string            `json:"issuerUrl"     mapstructure:"issuerUrl"     validate:"required,url"`
	RedirectURL   string            `json:"redirectUrl"   mapstructure:"redirectUrl"   validate:"omitempty,url"`
	State         string            `json:"state"         mapstructure:"state"         validate:"required"`
	ClientID      string            `json:"clientID"      mapstructure:"clientID"      validate:"required"`
	CookieName    string            `json:"cookieName"    mapstructure:"cookieName"`
	LoginPath     string            `json:"loginPath"     mapstructure:"loginPath"`
	CallbackPath  string            `json:"callbackPath"  mapstructure:"callbackPath"`
	Scopes        []string          `json:"scopes"        mapstructure:"scopes"`
	CookieDomains []string          `json:"cookieDomains" mapstructure:"cookieDomains"`
	EmailVerified bool              `json:"emailVerified" mapstructure:"emailVerified"`
	CookieSecure  bool              `json:"cookieSecure"  mapstructure:"cookieSecure"`
}

// HeaderOIDCAuthorizationAccess OpenID Connect or Header authorization accesses.
type HeaderOIDCAuthorizationAccess struct {
	GroupRegexp *regexp.Regexp `json:"-"`
	EmailRegexp *regexp.Regexp `json:"-"`
	Group       string         `json:"group"     mapstructure:"group"     validate:"required_without=Email"`
	Email       string         `json:"email"     mapstructure:"email"     validate:"required_without=Group"`
	Regexp      bool           `json:"regexp"    mapstructure:"regexp"`
	Forbidden   bool           `json:"forbidden" mapstructure:"forbidden"`
}

// BasicAuthConfig Basic auth configurations.
type BasicAuthConfig struct {
	Realm string `json:"realm" mapstructure:"realm" validate:"required"`
}

// HeaderAuthConfig Header auth configuration.
type HeaderAuthConfig struct {
	UsernameHeader string `json:"usernameHeader" mapstructure:"usernameHeader" validate:"required"`
	EmailHeader    string `json:"emailHeader"    mapstructure:"emailHeader"    validate:"required"`
	GroupsHeader   string `json:"groupsHeader"   mapstructure:"groupsHeader"`
}

// BasicAuthUserConfig Basic User auth configuration.
type BasicAuthUserConfig struct {
	Password *CredentialConfig `json:"password" mapstructure:"password" validate:"required"`
	User     string            `json:"user"     mapstructure:"user"     validate:"required"`
}

// TemplateConfigItem Template configuration item.
type TemplateConfigItem struct {
	Path    string            `json:"path"    mapstructure:"path"    validate:"required"`
	Headers map[string]string `json:"headers" mapstructure:"headers"`
	Status  string            `json:"status"  mapstructure:"status"`
}

// TemplateConfig Templates configuration.
type TemplateConfig struct {
	FolderList          *TemplateConfigItem `json:"folderList"          mapstructure:"folderList"          validate:"required"`
	TargetList          *TemplateConfigItem `json:"targetList"          mapstructure:"targetList"          validate:"required"`
	NotFoundError       *TemplateConfigItem `json:"notFoundError"       mapstructure:"notFoundError"       validate:"required"`
	InternalServerError *TemplateConfigItem `json:"internalServerError" mapstructure:"internalServerError" validate:"required"`
	UnauthorizedError   *TemplateConfigItem `json:"unauthorizedError"   mapstructure:"unauthorizedError"   validate:"required"`
	ForbiddenError      *TemplateConfigItem `json:"forbiddenError"      mapstructure:"forbiddenError"      validate:"required"`
	BadRequestError     *TemplateConfigItem `json:"badRequestError"     mapstructure:"badRequestError"     validate:"required"`
	Put                 *TemplateConfigItem `json:"put"                 mapstructure:"put"                 validate:"required"`
	Delete              *TemplateConfigItem `json:"delete"              mapstructure:"delete"              validate:"required"`
	Helpers             []string            `json:"helpers"             mapstructure:"helpers"             validate:"required,min=1,dive,required"`
}

// ServerConfig Server configuration.
type ServerConfig struct {
	Timeouts   *ServerTimeoutsConfig `json:"timeouts"   mapstructure:"timeouts"   validate:"required"`
	CORS       *ServerCorsConfig     `json:"cors"       mapstructure:"cors"       validate:"omitempty"`
	Cache      *CacheConfig          `json:"cache"      mapstructure:"cache"      validate:"omitempty"`
	Compress   *ServerCompressConfig `json:"compress"   mapstructure:"compress"   validate:"omitempty"`
	SSL        *ServerSSLConfig      `json:"ssl"        mapstructure:"ssl"        validate:"omitempty"`
	ListenAddr string                `json:"listenAddr" mapstructure:"listenAddr"`
	Port       int                   `json:"port"       mapstructure:"port"       validate:"required"`
}

// ServerTimeoutsConfig Server timeouts configuration.
type ServerTimeoutsConfig struct {
	ReadTimeout       string `json:"readTimeout"       mapstructure:"readTimeout"`
	ReadHeaderTimeout string `json:"readHeaderTimeout" mapstructure:"readHeaderTimeout"`
	WriteTimeout      string `json:"writeTimeout"      mapstructure:"writeTimeout"`
	IdleTimeout       string `json:"idleTimeout"       mapstructure:"idleTimeout"`
}

// ServerCompressConfig Server compress configuration.
type ServerCompressConfig struct {
	Enabled *bool    `json:"enabled" mapstructure:"enabled"`
	Types   []string `json:"types"   mapstructure:"types"   validate:"required,min=1"`
	Level   int      `json:"level"   mapstructure:"level"   validate:"required,min=1"`
}

// ServerSSLConfig Server SSL configuration.
type ServerSSLConfig struct {
	MinTLSVersion       *string                 `json:"minTLSVersion"       mapstructure:"minTLSVersion"`
	MaxTLSVersion       *string                 `json:"maxTLSVersion"       mapstructure:"maxTLSVersion"`
	Certificates        []*ServerSSLCertificate `json:"certificates"        mapstructure:"certificates"`
	SelfSignedHostnames []string                `json:"selfSignedHostnames" mapstructure:"selfSignedHostnames"`
	CipherSuites        []string                `json:"cipherSuites"        mapstructure:"cipherSuites"`
	Enabled             bool                    `json:"enabled"             mapstructure:"enabled"`
}

// ServerSSLCertificate Server SSL certificate.
type ServerSSLCertificate struct {
	Certificate          *string       `json:"certificate"          mapstructure:"certificate"`
	CertificateURL       *string       `json:"certificateUrl"       mapstructure:"certificateUrl"`
	CertificateURLConfig *SSLURLConfig `json:"certificateUrlConfig" mapstructure:"certificateUrlConfig"`
	PrivateKey           *string       `json:"privateKey"           mapstructure:"privateKey"`
	PrivateKeyURL        *string       `json:"privateKeyUrl"        mapstructure:"privateKeyUrl"`
	PrivateKeyURLConfig  *SSLURLConfig `json:"privateKeyUrlConfig"  mapstructure:"privateKeyUrlConfig"`
}

// SSLURLConfig SSL certificate/private key configuration for URLs.
type SSLURLConfig struct {
	AWSCredentials *BucketCredentialConfig `json:"awsCredentials" mapstructure:"awsCredentials" validate:"omitempty"`
	HTTPTimeout    string                  `json:"httpTimeout"    mapstructure:"httpTimeout"`
	AWSRegion      string                  `json:"awsRegion"      mapstructure:"awsRegion"`
	AWSEndpoint    string                  `json:"awsEndpoint"    mapstructure:"awsEndpoint"`
	AWSDisableSSL  bool                    `json:"awsDisableSSL"  mapstructure:"awsDisableSSL"`
}

// CacheConfig Cache configuration.
type CacheConfig struct {
	Expires        string `json:"expires"        mapstructure:"expires"`
	CacheControl   string `json:"cacheControl"   mapstructure:"cacheControl"`
	Pragma         string `json:"pragma"         mapstructure:"pragma"`
	XAccelExpires  string `json:"xAccelExpires"  mapstructure:"xAccelExpires"`
	NoCacheEnabled bool   `json:"noCacheEnabled" mapstructure:"noCacheEnabled"`
}

// ServerCorsConfig Server CORS configuration.
type ServerCorsConfig struct {
	MaxAge             *int     `json:"maxAge"             mapstructure:"maxAge"`
	AllowCredentials   *bool    `json:"allowCredentials"   mapstructure:"allowCredentials"`
	Debug              *bool    `json:"debug"              mapstructure:"debug"`
	OptionsPassthrough *bool    `json:"optionsPassthrough" mapstructure:"optionsPassthrough"`
	AllowOrigins       []string `json:"allowOrigins"       mapstructure:"allowOrigins"`
	AllowMethods       []string `json:"allowMethods"       mapstructure:"allowMethods"`
	AllowHeaders       []string `json:"allowHeaders"       mapstructure:"allowHeaders"`
	ExposeHeaders      []string `json:"exposeHeaders"      mapstructure:"exposeHeaders"`
	Enabled            bool     `json:"enabled"            mapstructure:"enabled"`
	AllowAll           bool     `json:"allowAll"           mapstructure:"allowAll"`
}

// TargetConfig Bucket instance configuration.
type TargetConfig struct {
	Name           string                    `json:"-"              validate:"required"`
	Bucket         *BucketConfig             `json:"bucket"         validate:"required" mapstructure:"bucket"`
	Resources      []*Resource               `json:"resources"      validate:"dive"     mapstructure:"resources"`
	Mount          *MountConfig              `json:"mount"          validate:"required" mapstructure:"mount"`
	Actions        *ActionsConfig            `json:"actions"                            mapstructure:"actions"`
	Templates      *TargetTemplateConfig     `json:"templates"                          mapstructure:"templates"`
	KeyRewriteList []*TargetKeyRewriteConfig `json:"keyRewriteList"                     mapstructure:"keyRewriteList"`
}

// TargetKeyRewriteConfig Target key rewrite configuration.
type TargetKeyRewriteConfig struct {
	Source      string         `json:"source"     mapstructure:"source"     validate:"required,min=1"`
	SourceRegex *regexp.Regexp `json:"-"`
	Target      string         `json:"target"     mapstructure:"target"     validate:"required,min=1"`
	TargetType  string         `json:"targetType" mapstructure:"targetType" validate:"required,oneof=REGEX TEMPLATE"`
}

// TargetTemplateConfig Target templates configuration to override default ones.
type TargetTemplateConfig struct {
	FolderList          *TargetTemplateConfigItem `json:"folderList"          mapstructure:"folderList"`
	NotFoundError       *TargetTemplateConfigItem `json:"notFoundError"       mapstructure:"notFoundError"`
	InternalServerError *TargetTemplateConfigItem `json:"internalServerError" mapstructure:"internalServerError"`
	ForbiddenError      *TargetTemplateConfigItem `json:"forbiddenError"      mapstructure:"forbiddenError"`
	UnauthorizedError   *TargetTemplateConfigItem `json:"unauthorizedError"   mapstructure:"unauthorizedError"`
	BadRequestError     *TargetTemplateConfigItem `json:"badRequestError"     mapstructure:"badRequestError"`
	Put                 *TargetTemplateConfigItem `json:"put"                 mapstructure:"put"`
	Delete              *TargetTemplateConfigItem `json:"delete"              mapstructure:"delete"`
	Helpers             []*TargetHelperConfigItem `json:"helpers"             mapstructure:"helpers"`
}

// TargetHelperConfigItem Target helper configuration item.
type TargetHelperConfigItem struct {
	Path     string `json:"path"     mapstructure:"path"     validate:"required,min=1"`
	InBucket bool   `json:"inBucket" mapstructure:"inBucket"`
}

// TargetTemplateConfigItem Target template configuration item.
type TargetTemplateConfigItem struct {
	Path     string            `json:"path"     mapstructure:"path"     validate:"required,min=1"`
	Headers  map[string]string `json:"headers"  mapstructure:"headers"`
	Status   string            `json:"status"   mapstructure:"status"`
	InBucket bool              `json:"inBucket" mapstructure:"inBucket"`
}

// ActionsConfig is dedicated to actions configuration in a target.
type ActionsConfig struct {
	HEAD   *HeadActionConfig   `json:"HEAD"   mapstructure:"HEAD"`
	GET    *GetActionConfig    `json:"GET"    mapstructure:"GET"`
	PUT    *PutActionConfig    `json:"PUT"    mapstructure:"PUT"`
	DELETE *DeleteActionConfig `json:"DELETE" mapstructure:"DELETE"`
}

// HeadActionConfig Head action configuration.
type HeadActionConfig struct {
	Config  *HeadActionConfigConfig `json:"config"  mapstructure:"config"`
	Enabled bool                    `json:"enabled" mapstructure:"enabled"`
}

// HeadActionConfigConfig Head action configuration object configuration.
type HeadActionConfigConfig struct {
	Webhooks []*WebhookConfig `json:"webhooks" mapstructure:"webhooks" validate:"dive"`
}

// DeleteActionConfig Delete action configuration.
type DeleteActionConfig struct {
	Config  *DeleteActionConfigConfig `json:"config"  mapstructure:"config"`
	Enabled bool                      `json:"enabled" mapstructure:"enabled"`
}

// DeleteActionConfigConfig Delete action configuration object configuration.
type DeleteActionConfigConfig struct {
	Webhooks []*WebhookConfig `json:"webhooks" mapstructure:"webhooks" validate:"dive"`
}

// PutActionConfig Put action configuration.
type PutActionConfig struct {
	Config  *PutActionConfigConfig `json:"config"  mapstructure:"config"`
	Enabled bool                   `json:"enabled" mapstructure:"enabled"`
}

// PutActionConfigConfig Put action configuration object configuration.
type PutActionConfigConfig struct {
	Metadata       map[string]string                    `json:"metadata"       mapstructure:"metadata"`
	SystemMetadata *PutActionConfigSystemMetadataConfig `json:"systemMetadata" mapstructure:"systemMetadata"`
	CannedACL      *string                              `json:"cannedACL"      mapstructure:"cannedACL"`
	StorageClass   string                               `json:"storageClass"   mapstructure:"storageClass"`
	Webhooks       []*WebhookConfig                     `json:"webhooks"       mapstructure:"webhooks"       validate:"dive"`
	AllowOverride  bool                                 `json:"allowOverride"  mapstructure:"allowOverride"`
}

// PutActionConfigSystemMetadataConfig Put action configuration system metadata object configuration.
type PutActionConfigSystemMetadataConfig struct {
	CacheControl       string `json:"cacheControl"       mapstructure:"cacheControl"`
	ContentDisposition string `json:"contentDisposition" mapstructure:"contentDisposition"`
	ContentEncoding    string `json:"contentEncoding"    mapstructure:"contentEncoding"`
	ContentLanguage    string `json:"contentLanguage"    mapstructure:"contentLanguage"`
	Expires            string `json:"expires"            mapstructure:"expires"`
}

// GetActionConfig Get action configuration.
type GetActionConfig struct {
	Config  *GetActionConfigConfig `json:"config"  mapstructure:"config"`
	Enabled bool                   `json:"enabled" mapstructure:"enabled"`
}

// GetActionConfigConfig Get action configuration object configuration.
type GetActionConfigConfig struct {
	StreamedFileHeaders                      map[string]string `json:"streamedFileHeaders"                      mapstructure:"streamedFileHeaders"`
	IndexDocument                            string            `json:"indexDocument"                            mapstructure:"indexDocument"`
	SignedURLExpirationString                string            `json:"signedUrlExpiration"                      mapstructure:"signedUrlExpiration"`
	Webhooks                                 []*WebhookConfig  `json:"webhooks"                                 mapstructure:"webhooks"                                 validate:"dive"`
	SignedURLExpiration                      time.Duration     `json:"-"`
	RedirectWithTrailingSlashForNotFoundFile bool              `json:"redirectWithTrailingSlashForNotFoundFile" mapstructure:"redirectWithTrailingSlashForNotFoundFile"`
	RedirectToSignedURL                      bool              `json:"redirectToSignedUrl"                      mapstructure:"redirectToSignedUrl"`
	DisableListing                           bool              `json:"disableListing"                           mapstructure:"disableListing"`
}

// WebhookConfig Webhook configuration.
type WebhookConfig struct {
	Headers         map[string]string            `json:"headers"         mapstructure:"headers"`
	SecretHeaders   map[string]*CredentialConfig `json:"secretHeaders"   mapstructure:"secretHeaders"   validate:"omitempty"`
	Method          string                       `json:"method"          mapstructure:"method"          validate:"required,oneof=POST PATCH PUT DELETE"`
	URL             string                       `json:"url"             mapstructure:"url"             validate:"required,url"`
	MaxWaitTime     string                       `json:"maxWaitTime"     mapstructure:"maxWaitTime"`
	DefaultWaitTime string                       `json:"defaultWaitTime" mapstructure:"defaultWaitTime"`
	RetryCount      int                          `json:"retryCount"      mapstructure:"retryCount"      validate:"gte=0"`
}

// Resource Resource.
type Resource struct {
	WhiteList *bool               `json:"whiteList" mapstructure:"whiteList"`
	Basic     *ResourceBasic      `json:"basic"     mapstructure:"basic"     validate:"omitempty"`
	OIDC      *ResourceHeaderOIDC `json:"oidc"      mapstructure:"oidc"      validate:"omitempty"`
	Header    *ResourceHeaderOIDC `json:"header"    mapstructure:"header"    validate:"omitempty"`
	Path      string              `json:"path"      mapstructure:"path"      validate:"required"`
	Provider  string              `json:"provider"  mapstructure:"provider"`
	Methods   []string            `json:"methods"   mapstructure:"methods"   validate:"required,dive,required"`
}

// ResourceBasic Basic auth resource.
type ResourceBasic struct {
	Credentials []*BasicAuthUserConfig `json:"credentials" mapstructure:"credentials" validate:"omitempty,dive"`
}

// ResourceHeaderOIDC OIDC or Header auth Resource.
type ResourceHeaderOIDC struct {
	AuthorizationOPAServer *OPAServerAuthorization          `json:"authorizationOPAServer" mapstructure:"authorizationOPAServer" validate:"omitempty"`
	AuthorizationAccesses  []*HeaderOIDCAuthorizationAccess `json:"authorizationAccesses"  mapstructure:"authorizationAccesses"  validate:"omitempty,dive"`
}

// OPAServerAuthorization OPA Server authorization.
type OPAServerAuthorization struct {
	Tags map[string]string `json:"tags" mapstructure:"tags"`
	URL  string            `json:"url"  mapstructure:"url"  validate:"required,url"`
}

// BucketConfig Bucket configuration.
type BucketConfig struct {
	Credentials               *BucketCredentialConfig `json:"credentials"               mapstructure:"credentials"               validate:"omitempty"`
	RequestConfig             *BucketRequestConfig    `json:"requestConfig"             mapstructure:"requestConfig"             validate:"omitempty"`
	Name                      string                  `json:"name"                      mapstructure:"name"                      validate:"required"`
	Prefix                    string                  `json:"prefix"                    mapstructure:"prefix"`
	Region                    string                  `json:"region"                    mapstructure:"region"`
	S3Endpoint                string                  `json:"s3Endpoint"                mapstructure:"s3Endpoint"`
	S3ListMaxKeys             int64                   `json:"s3ListMaxKeys"             mapstructure:"s3ListMaxKeys"             validate:"gt=0"`
	S3MaxUploadParts          int                     `json:"s3MaxUploadParts"          mapstructure:"s3MaxUploadParts"          validate:"required,gte=1"`
	S3UploadPartSize          int64                   `json:"s3UploadPartSize"          mapstructure:"s3UploadPartSize"          validate:"required,gte=5"`
	S3UploadConcurrency       int                     `json:"s3UploadConcurrency"       mapstructure:"s3UploadConcurrency"       validate:"required,gte=1"`
	S3UploadLeavePartsOnError bool                    `json:"s3UploadLeavePartsOnError" mapstructure:"s3UploadLeavePartsOnError"`
	DisableSSL                bool                    `json:"disableSSL"                mapstructure:"disableSSL"`
}

// BucketRequestConfig Bucket request configuration.
type BucketRequestConfig struct {
	ListHeaders   map[string]string `json:"listHeaders"   mapstructure:"listHeaders"`
	GetHeaders    map[string]string `json:"getHeaders"    mapstructure:"getHeaders"`
	PutHeaders    map[string]string `json:"putHeaders"    mapstructure:"putHeaders"`
	DeleteHeaders map[string]string `json:"deleteHeaders" mapstructure:"deleteHeaders"`
}

// BucketCredentialConfig Bucket Credentials configurations.
type BucketCredentialConfig struct {
	AccessKey *CredentialConfig `json:"accessKey" mapstructure:"accessKey" validate:"omitempty"`
	SecretKey *CredentialConfig `json:"secretKey" mapstructure:"secretKey" validate:"omitempty"`
}

// CredentialConfig Credential Configurations.
type CredentialConfig struct {
	Path  string `json:"path" mapstructure:"path"  validate:"required_without_all=Env Value"`
	Env   string `json:"env"  mapstructure:"env"   validate:"required_without_all=Path Value"`
	Value string `json:"-"    mapstructure:"value" validate:"required_without_all=Path Env"` // Ignore this key in json marshal
}

// LogConfig Log configuration.
type LogConfig struct {
	Level    string `json:"level"    mapstructure:"level"    validate:"required"`
	Format   string `json:"format"   mapstructure:"format"   validate:"required"`
	FilePath string `json:"filePath" mapstructure:"filePath"`
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
