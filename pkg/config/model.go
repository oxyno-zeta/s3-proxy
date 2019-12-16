package config

import "errors"

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

// Config Application Configuration
type Config struct {
	Log            *LogConfig          `mapstructure:"log"`
	Server         *ServerConfig       `mapstructure:"server"`
	InternalServer *ServerConfig       `mapstructure:"internalServer"`
	Targets        []*Target           `mapstructure:"targets" validate:"gte=0,required,dive,required"`
	Templates      *TemplateConfig     `mapstructure:"templates"`
	AuthProviders  *AuthProviderConfig `mapstructure:"authProviders"`
	ListTargets    *ListTargetsConfig  `mapstructure:"listTargets" validate:"required"`
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
	Path []string `mapstructure:"path" validate:"dive,required"`
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
}

// OIDCAuthorizationAccess OpenID Connect authorization accesses
type OIDCAuthorizationAccess struct {
	Group string `mapstructure:"group" validate:"required_without=Email"`
	Email string `mapstructure:"email" validate:"required_without=Group"`
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

// Target Bucket instance configuration
type Target struct {
	Name          string        `mapstructure:"name" validate:"required"`
	Bucket        *BucketConfig `mapstructure:"bucket" validate:"required"`
	Resources     []*Resource   `mapstructure:"resources" validate:"dive"`
	Mount         *MountConfig  `mapstructure:"mount" validate:"required"`
	IndexDocument string        `mapstructure:"indexDocument"`
}

// Resource Resource
type Resource struct {
	Path      string         `mapstructure:"path" validate:"required"`
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
