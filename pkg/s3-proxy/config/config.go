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
	Log            *LogConfig               `mapstructure:"log"            json:"log"`
	Tracing        *TracingConfig           `mapstructure:"tracing"        json:"tracing"`
	Metrics        *MetricsConfig           `mapstructure:"metrics"        json:"metrics"`
	Server         *ServerConfig            `mapstructure:"server"         json:"server"`
	InternalServer *ServerConfig            `mapstructure:"internalServer" json:"internalServer"`
	Targets        map[string]*TargetConfig `mapstructure:"targets"        json:"targets"`
	Templates      *TemplateConfig          `mapstructure:"templates"      json:"templates"`
	AuthProviders  *AuthProviderConfig      `mapstructure:"authProviders"  json:"authProviders"`
	ListTargets    *ListTargetsConfig       `mapstructure:"listTargets"    json:"listTargets"`
}

// MetricsConfig represents the metrics configuration structure.
type MetricsConfig struct {
	DisableRouterPath bool `mapstructure:"disableRouterPath" json:"disableRouterPath"`
}

// TracingConfig represents the Tracing configuration structure.
type TracingConfig struct {
	FixedTags     map[string]any `mapstructure:"fixedTags"     json:"fixedTags"`
	FlushInterval string         `mapstructure:"flushInterval" json:"flushInterval"`
	UDPHost       string         `mapstructure:"udpHost"       json:"udpHost"`
	QueueSize     int            `mapstructure:"queueSize"     json:"queueSize"`
	Enabled       bool           `mapstructure:"enabled"       json:"enabled"`
	LogSpan       bool           `mapstructure:"logSpan"       json:"logSpan"`
}

// ListTargetsConfig List targets configuration.
type ListTargetsConfig struct {
	Mount    *MountConfig `mapstructure:"mount"    validate:"required_with=Enabled" json:"mount"`
	Resource *Resource    `mapstructure:"resource" validate:"omitempty"             json:"resource"`
	Enabled  bool         `mapstructure:"enabled"                                   json:"enabled"`
}

// MountConfig Mount configuration.
type MountConfig struct {
	Host string   `mapstructure:"host" json:"host"`
	Path []string `mapstructure:"path" json:"path" validate:"required,dive,required"`
}

// AuthProviderConfig Authentication provider configurations.
type AuthProviderConfig struct {
	Basic  map[string]*BasicAuthConfig  `mapstructure:"basic"  validate:"omitempty" json:"basic"`
	OIDC   map[string]*OIDCAuthConfig   `mapstructure:"oidc"   validate:"omitempty" json:"oidc"`
	Header map[string]*HeaderAuthConfig `mapstructure:"header" validate:"omitempty" json:"header"`
}

// OIDCAuthConfig OpenID Connect authentication configurations.
type OIDCAuthConfig struct {
	ClientSecret  *CredentialConfig `mapstructure:"clientSecret"  validate:"omitempty"     json:"clientSecret"`
	GroupClaim    string            `mapstructure:"groupClaim"                             json:"groupClaim"`
	IssuerURL     string            `mapstructure:"issuerUrl"     validate:"required,url"  json:"issuerUrl"`
	RedirectURL   string            `mapstructure:"redirectUrl"   validate:"omitempty,url" json:"redirectUrl"`
	State         string            `mapstructure:"state"         validate:"required"      json:"state"`
	ClientID      string            `mapstructure:"clientID"      validate:"required"      json:"clientID"`
	CookieName    string            `mapstructure:"cookieName"                             json:"cookieName"`
	LoginPath     string            `mapstructure:"loginPath"                              json:"loginPath"`
	CallbackPath  string            `mapstructure:"callbackPath"                           json:"callbackPath"`
	Scopes        []string          `mapstructure:"scopes"                                 json:"scopes"`
	CookieDomains []string          `mapstructure:"cookieDomains"                          json:"cookieDomains"`
	EmailVerified bool              `mapstructure:"emailVerified"                          json:"emailVerified"`
	CookieSecure  bool              `mapstructure:"cookieSecure"                           json:"cookieSecure"`
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
	Realm string `mapstructure:"realm" validate:"required" json:"realm"`
}

// HeaderAuthConfig Header auth configuration.
type HeaderAuthConfig struct {
	UsernameHeader string `mapstructure:"usernameHeader" validate:"required" json:"usernameHeader"`
	EmailHeader    string `mapstructure:"emailHeader"    validate:"required" json:"emailHeader"`
	GroupsHeader   string `mapstructure:"groupsHeader"                       json:"groupsHeader"`
}

// BasicAuthUserConfig Basic User auth configuration.
type BasicAuthUserConfig struct {
	Password *CredentialConfig `mapstructure:"password" validate:"required" json:"password"`
	User     string            `mapstructure:"user"     validate:"required" json:"user"`
}

// TemplateConfigItem Template configuration item.
type TemplateConfigItem struct {
	Path    string            `mapstructure:"path"    validate:"required" json:"path"`
	Headers map[string]string `mapstructure:"headers"                     json:"headers"`
	Status  string            `mapstructure:"status"                      json:"status"`
}

// TemplateConfig Templates configuration.
type TemplateConfig struct {
	FolderList          *TemplateConfigItem `mapstructure:"folderList"          validate:"required"                     json:"folderList"`
	TargetList          *TemplateConfigItem `mapstructure:"targetList"          validate:"required"                     json:"targetList"`
	NotFoundError       *TemplateConfigItem `mapstructure:"notFoundError"       validate:"required"                     json:"notFoundError"`
	InternalServerError *TemplateConfigItem `mapstructure:"internalServerError" validate:"required"                     json:"internalServerError"`
	UnauthorizedError   *TemplateConfigItem `mapstructure:"unauthorizedError"   validate:"required"                     json:"unauthorizedError"`
	ForbiddenError      *TemplateConfigItem `mapstructure:"forbiddenError"      validate:"required"                     json:"forbiddenError"`
	BadRequestError     *TemplateConfigItem `mapstructure:"badRequestError"     validate:"required"                     json:"badRequestError"`
	Put                 *TemplateConfigItem `mapstructure:"put"                 validate:"required"                     json:"put"`
	Delete              *TemplateConfigItem `mapstructure:"delete"              validate:"required"                     json:"delete"`
	Helpers             []string            `mapstructure:"helpers"             validate:"required,min=1,dive,required" json:"helpers"`
}

// ServerConfig Server configuration.
type ServerConfig struct {
	Timeouts   *ServerTimeoutsConfig `mapstructure:"timeouts"   validate:"required"  json:"timeouts"`
	CORS       *ServerCorsConfig     `mapstructure:"cors"       validate:"omitempty" json:"cors"`
	Cache      *CacheConfig          `mapstructure:"cache"      validate:"omitempty" json:"cache"`
	Compress   *ServerCompressConfig `mapstructure:"compress"   validate:"omitempty" json:"compress"`
	SSL        *ServerSSLConfig      `mapstructure:"ssl"        validate:"omitempty" json:"ssl"`
	ListenAddr string                `mapstructure:"listenAddr"                      json:"listenAddr"`
	Port       int                   `mapstructure:"port"       validate:"required"  json:"port"`
}

// ServerTimeoutsConfig Server timeouts configuration.
type ServerTimeoutsConfig struct {
	ReadTimeout       string `mapstructure:"readTimeout"       json:"readTimeout"`
	ReadHeaderTimeout string `mapstructure:"readHeaderTimeout" json:"readHeaderTimeout"`
	WriteTimeout      string `mapstructure:"writeTimeout"      json:"writeTimeout"`
	IdleTimeout       string `mapstructure:"idleTimeout"       json:"idleTimeout"`
}

// ServerCompressConfig Server compress configuration.
type ServerCompressConfig struct {
	Enabled *bool    `mapstructure:"enabled" json:"enabled"`
	Types   []string `mapstructure:"types"   json:"types"   validate:"required,min=1"`
	Level   int      `mapstructure:"level"   json:"level"   validate:"required,min=1"`
}

// ServerSSLConfig Server SSL configuration.
type ServerSSLConfig struct {
	MinTLSVersion       *string                 `mapstructure:"minTLSVersion"       json:"minTLSVersion"`
	MaxTLSVersion       *string                 `mapstructure:"maxTLSVersion"       json:"maxTLSVersion"`
	Certificates        []*ServerSSLCertificate `mapstructure:"certificates"        json:"certificates"`
	SelfSignedHostnames []string                `mapstructure:"selfSignedHostnames" json:"selfSignedHostnames"`
	CipherSuites        []string                `mapstructure:"cipherSuites"        json:"cipherSuites"`
	Enabled             bool                    `mapstructure:"enabled"             json:"enabled"`
}

// ServerSSLCertificate Server SSL certificate.
type ServerSSLCertificate struct {
	Certificate          *string       `mapstructure:"certificate"          json:"certificate"`
	CertificateURL       *string       `mapstructure:"certificateUrl"       json:"certificateUrl"`
	CertificateURLConfig *SSLURLConfig `mapstructure:"certificateUrlConfig" json:"certificateUrlConfig"`
	PrivateKey           *string       `mapstructure:"privateKey"           json:"privateKey"`
	PrivateKeyURL        *string       `mapstructure:"privateKeyUrl"        json:"privateKeyUrl"`
	PrivateKeyURLConfig  *SSLURLConfig `mapstructure:"privateKeyUrlConfig"  json:"privateKeyUrlConfig"`
}

// SSLURLConfig SSL certificate/private key configuration for URLs.
type SSLURLConfig struct {
	AWSCredentials *BucketCredentialConfig `mapstructure:"awsCredentials" validate:"omitempty" json:"awsCredentials"`
	HTTPTimeout    string                  `mapstructure:"httpTimeout"                         json:"httpTimeout"`
	AWSRegion      string                  `mapstructure:"awsRegion"                           json:"awsRegion"`
	AWSEndpoint    string                  `mapstructure:"awsEndpoint"                         json:"awsEndpoint"`
	AWSDisableSSL  bool                    `mapstructure:"awsDisableSSL"                       json:"awsDisableSSL"`
}

// CacheConfig Cache configuration.
type CacheConfig struct {
	Expires        string `mapstructure:"expires"        json:"expires"`
	CacheControl   string `mapstructure:"cacheControl"   json:"cacheControl"`
	Pragma         string `mapstructure:"pragma"         json:"pragma"`
	XAccelExpires  string `mapstructure:"xAccelExpires"  json:"xAccelExpires"`
	NoCacheEnabled bool   `mapstructure:"noCacheEnabled" json:"noCacheEnabled"`
}

// ServerCorsConfig Server CORS configuration.
type ServerCorsConfig struct {
	MaxAge             *int     `mapstructure:"maxAge"             json:"maxAge"`
	AllowCredentials   *bool    `mapstructure:"allowCredentials"   json:"allowCredentials"`
	Debug              *bool    `mapstructure:"debug"              json:"debug"`
	OptionsPassthrough *bool    `mapstructure:"optionsPassthrough" json:"optionsPassthrough"`
	AllowOrigins       []string `mapstructure:"allowOrigins"       json:"allowOrigins"`
	AllowMethods       []string `mapstructure:"allowMethods"       json:"allowMethods"`
	AllowHeaders       []string `mapstructure:"allowHeaders"       json:"allowHeaders"`
	ExposeHeaders      []string `mapstructure:"exposeHeaders"      json:"exposeHeaders"`
	Enabled            bool     `mapstructure:"enabled"            json:"enabled"`
	AllowAll           bool     `mapstructure:"allowAll"           json:"allowAll"`
}

// TargetConfig Bucket instance configuration.
type TargetConfig struct {
	Name           string                    `validate:"required" json:"-"`
	Bucket         *BucketConfig             `validate:"required" json:"bucket"         mapstructure:"bucket"`
	Resources      []*Resource               `validate:"dive"     json:"resources"      mapstructure:"resources"`
	Mount          *MountConfig              `validate:"required" json:"mount"          mapstructure:"mount"`
	Actions        *ActionsConfig            `                    json:"actions"        mapstructure:"actions"`
	Templates      *TargetTemplateConfig     `                    json:"templates"      mapstructure:"templates"`
	KeyRewriteList []*TargetKeyRewriteConfig `                    json:"keyRewriteList" mapstructure:"keyRewriteList"`
}

// TargetKeyRewriteConfig Target key rewrite configuration.
type TargetKeyRewriteConfig struct {
	Source      string         `mapstructure:"source"     validate:"required,min=1"                json:"source"`
	SourceRegex *regexp.Regexp `                                                                   json:"-"`
	Target      string         `mapstructure:"target"     validate:"required,min=1"                json:"target"`
	TargetType  string         `mapstructure:"targetType" validate:"required,oneof=REGEX TEMPLATE" json:"targetType"`
}

// TargetTemplateConfig Target templates configuration to override default ones.
type TargetTemplateConfig struct {
	FolderList          *TargetTemplateConfigItem `mapstructure:"folderList"          json:"folderList"`
	NotFoundError       *TargetTemplateConfigItem `mapstructure:"notFoundError"       json:"notFoundError"`
	InternalServerError *TargetTemplateConfigItem `mapstructure:"internalServerError" json:"internalServerError"`
	ForbiddenError      *TargetTemplateConfigItem `mapstructure:"forbiddenError"      json:"forbiddenError"`
	UnauthorizedError   *TargetTemplateConfigItem `mapstructure:"unauthorizedError"   json:"unauthorizedError"`
	BadRequestError     *TargetTemplateConfigItem `mapstructure:"badRequestError"     json:"badRequestError"`
	Put                 *TargetTemplateConfigItem `mapstructure:"put"                 json:"put"`
	Delete              *TargetTemplateConfigItem `mapstructure:"delete"              json:"delete"`
	Helpers             []*TargetHelperConfigItem `mapstructure:"helpers"             json:"helpers"`
}

// TargetHelperConfigItem Target helper configuration item.
type TargetHelperConfigItem struct {
	Path     string `mapstructure:"path"     validate:"required,min=1" json:"path"`
	InBucket bool   `mapstructure:"inBucket"                           json:"inBucket"`
}

// TargetTemplateConfigItem Target template configuration item.
type TargetTemplateConfigItem struct {
	Path     string            `mapstructure:"path"     validate:"required,min=1" json:"path"`
	Headers  map[string]string `mapstructure:"headers"                            json:"headers"`
	Status   string            `mapstructure:"status"                             json:"status"`
	InBucket bool              `mapstructure:"inBucket"                           json:"inBucket"`
}

// ActionsConfig is dedicated to actions configuration in a target.
type ActionsConfig struct {
	HEAD   *HeadActionConfig   `mapstructure:"HEAD"   json:"HEAD"`
	GET    *GetActionConfig    `mapstructure:"GET"    json:"GET"`
	PUT    *PutActionConfig    `mapstructure:"PUT"    json:"PUT"`
	DELETE *DeleteActionConfig `mapstructure:"DELETE" json:"DELETE"`
}

// HeadActionConfig Head action configuration.
type HeadActionConfig struct {
	Config  *HeadActionConfigConfig `mapstructure:"config"  json:"config"`
	Enabled bool                    `mapstructure:"enabled" json:"enabled"`
}

// HeadActionConfigConfig Head action configuration object configuration.
type HeadActionConfigConfig struct {
	Webhooks []*WebhookConfig `mapstructure:"webhooks" validate:"dive" json:"webhooks"`
}

// DeleteActionConfig Delete action configuration.
type DeleteActionConfig struct {
	Config  *DeleteActionConfigConfig `mapstructure:"config"  json:"config"`
	Enabled bool                      `mapstructure:"enabled" json:"enabled"`
}

// DeleteActionConfigConfig Delete action configuration object configuration.
type DeleteActionConfigConfig struct {
	Webhooks []*WebhookConfig `mapstructure:"webhooks" validate:"dive" json:"webhooks"`
}

// PutActionConfig Put action configuration.
type PutActionConfig struct {
	Config  *PutActionConfigConfig `mapstructure:"config"  json:"config"`
	Enabled bool                   `mapstructure:"enabled" json:"enabled"`
}

// PutActionConfigConfig Put action configuration object configuration.
type PutActionConfigConfig struct {
	Metadata       map[string]string                    `mapstructure:"metadata"       json:"metadata"`
	SystemMetadata *PutActionConfigSystemMetadataConfig `mapstructure:"systemMetadata" json:"systemMetadata"`
	CannedACL      *string                              `mapstructure:"cannedACL"      json:"cannedACL"`
	StorageClass   string                               `mapstructure:"storageClass"   json:"storageClass"`
	Webhooks       []*WebhookConfig                     `mapstructure:"webhooks"       json:"webhooks"       validate:"dive"`
	AllowOverride  bool                                 `mapstructure:"allowOverride"  json:"allowOverride"`
}

// PutActionConfigSystemMetadataConfig Put action configuration system metadata object configuration.
type PutActionConfigSystemMetadataConfig struct {
	CacheControl       string `mapstructure:"cacheControl"       json:"cacheControl"`
	ContentDisposition string `mapstructure:"contentDisposition" json:"contentDisposition"`
	ContentEncoding    string `mapstructure:"contentEncoding"    json:"contentEncoding"`
	ContentLanguage    string `mapstructure:"contentLanguage"    json:"contentLanguage"`
	Expires            string `mapstructure:"expires"            json:"expires"`
}

// GetActionConfig Get action configuration.
type GetActionConfig struct {
	Config  *GetActionConfigConfig `mapstructure:"config"  json:"config"`
	Enabled bool                   `mapstructure:"enabled" json:"enabled"`
}

// GetActionConfigConfig Get action configuration object configuration.
type GetActionConfigConfig struct {
	StreamedFileHeaders                      map[string]string `mapstructure:"streamedFileHeaders"                      json:"streamedFileHeaders"`
	IndexDocument                            string            `mapstructure:"indexDocument"                            json:"indexDocument"`
	SignedURLExpirationString                string            `mapstructure:"signedUrlExpiration"                      json:"signedUrlExpiration"`
	Webhooks                                 []*WebhookConfig  `mapstructure:"webhooks"                                 json:"webhooks"                                 validate:"dive"`
	SignedURLExpiration                      time.Duration     `                                                        json:"-"`
	RedirectWithTrailingSlashForNotFoundFile bool              `mapstructure:"redirectWithTrailingSlashForNotFoundFile" json:"redirectWithTrailingSlashForNotFoundFile"`
	RedirectToSignedURL                      bool              `mapstructure:"redirectToSignedUrl"                      json:"redirectToSignedUrl"`
	DisableListing                           bool              `mapstructure:"disableListing"                           json:"disableListing"`
}

// WebhookConfig Webhook configuration.
type WebhookConfig struct {
	Headers         map[string]string            `mapstructure:"headers"         json:"headers"`
	SecretHeaders   map[string]*CredentialConfig `mapstructure:"secretHeaders"   json:"secretHeaders"   validate:"omitempty"`
	Method          string                       `mapstructure:"method"          json:"method"          validate:"required,oneof=POST PATCH PUT DELETE"`
	URL             string                       `mapstructure:"url"             json:"url"             validate:"required,url"`
	MaxWaitTime     string                       `mapstructure:"maxWaitTime"     json:"maxWaitTime"`
	DefaultWaitTime string                       `mapstructure:"defaultWaitTime" json:"defaultWaitTime"`
	RetryCount      int                          `mapstructure:"retryCount"      json:"retryCount"      validate:"gte=0"`
}

// Resource Resource.
type Resource struct {
	WhiteList *bool               `mapstructure:"whiteList" json:"whiteList"`
	Basic     *ResourceBasic      `mapstructure:"basic"     json:"basic"     validate:"omitempty"`
	OIDC      *ResourceHeaderOIDC `mapstructure:"oidc"      json:"oidc"      validate:"omitempty"`
	Header    *ResourceHeaderOIDC `mapstructure:"header"    json:"header"    validate:"omitempty"`
	Path      string              `mapstructure:"path"      json:"path"      validate:"required"`
	Provider  string              `mapstructure:"provider"  json:"provider"`
	Methods   []string            `mapstructure:"methods"   json:"methods"   validate:"required,dive,required"`
}

// ResourceBasic Basic auth resource.
type ResourceBasic struct {
	Credentials []*BasicAuthUserConfig `mapstructure:"credentials" validate:"omitempty,dive" json:"credentials"`
}

// ResourceHeaderOIDC OIDC or Header auth Resource.
type ResourceHeaderOIDC struct {
	AuthorizationOPAServer *OPAServerAuthorization          `mapstructure:"authorizationOPAServer" validate:"omitempty"      json:"authorizationOPAServer"`
	AuthorizationAccesses  []*HeaderOIDCAuthorizationAccess `mapstructure:"authorizationAccesses"  validate:"omitempty,dive" json:"authorizationAccesses"`
}

// OPAServerAuthorization OPA Server authorization.
type OPAServerAuthorization struct {
	Tags map[string]string `mapstructure:"tags" json:"tags"`
	URL  string            `mapstructure:"url"  json:"url"  validate:"required,url"`
}

// BucketConfig Bucket configuration.
type BucketConfig struct {
	Credentials               *BucketCredentialConfig `mapstructure:"credentials"               validate:"omitempty"      json:"credentials"`
	RequestConfig             *BucketRequestConfig    `mapstructure:"requestConfig"             validate:"omitempty"      json:"requestConfig"`
	Name                      string                  `mapstructure:"name"                      validate:"required"       json:"name"`
	Prefix                    string                  `mapstructure:"prefix"                                              json:"prefix"`
	Region                    string                  `mapstructure:"region"                                              json:"region"`
	S3Endpoint                string                  `mapstructure:"s3Endpoint"                                          json:"s3Endpoint"`
	S3ListMaxKeys             int64                   `mapstructure:"s3ListMaxKeys"             validate:"gt=0"           json:"s3ListMaxKeys"`
	S3MaxUploadParts          int                     `mapstructure:"s3MaxUploadParts"          validate:"required,gte=1" json:"s3MaxUploadParts"`
	S3UploadPartSize          int64                   `mapstructure:"s3UploadPartSize"          validate:"required,gte=5" json:"s3UploadPartSize"`
	S3UploadConcurrency       int                     `mapstructure:"s3UploadConcurrency"       validate:"required,gte=1" json:"s3UploadConcurrency"`
	S3UploadLeavePartsOnError bool                    `mapstructure:"s3UploadLeavePartsOnError"                           json:"s3UploadLeavePartsOnError"`
	DisableSSL                bool                    `mapstructure:"disableSSL"                                          json:"disableSSL"`
}

// BucketRequestConfig Bucket request configuration.
type BucketRequestConfig struct {
	ListHeaders   map[string]string `mapstructure:"listHeaders"   json:"listHeaders"`
	GetHeaders    map[string]string `mapstructure:"getHeaders"    json:"getHeaders"`
	PutHeaders    map[string]string `mapstructure:"putHeaders"    json:"putHeaders"`
	DeleteHeaders map[string]string `mapstructure:"deleteHeaders" json:"deleteHeaders"`
}

// BucketCredentialConfig Bucket Credentials configurations.
type BucketCredentialConfig struct {
	AccessKey *CredentialConfig `mapstructure:"accessKey" validate:"omitempty" json:"accessKey"`
	SecretKey *CredentialConfig `mapstructure:"secretKey" validate:"omitempty" json:"secretKey"`
}

// CredentialConfig Credential Configurations.
type CredentialConfig struct {
	Path  string `mapstructure:"path"  validate:"required_without_all=Env Value"  json:"path"`
	Env   string `mapstructure:"env"   validate:"required_without_all=Path Value" json:"env"`
	Value string `mapstructure:"value" validate:"required_without_all=Path Env"   json:"-"` // Ignore this key in json marshal
}

// LogConfig Log configuration.
type LogConfig struct {
	Level    string `mapstructure:"level"    validate:"required" json:"level"`
	Format   string `mapstructure:"format"   validate:"required" json:"format"`
	FilePath string `mapstructure:"filePath"                     json:"filePath"`
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
