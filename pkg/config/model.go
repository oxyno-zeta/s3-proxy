package config

import "errors"

// MainConfigPath Configuration path
const MainConfigPath = "conf/config.yaml"

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
	Log            *LogConfig          `koanf:"log"`
	Server         *ServerConfig       `koanf:"server"`
	InternalServer *ServerConfig       `koanf:"internalServer"`
	Targets        []*Target           `koanf:"targets" validate:"gte=0,required,dive,required"`
	Templates      *TemplateConfig     `koanf:"templates"`
	AuthProviders  *AuthProviderConfig `koanf:"authProviders"`
	ListTargets    *ListTargetsConfig  `koanf:"listTargets" validate:"required"`
}

// ListTargetsConfig List targets configuration
type ListTargetsConfig struct {
	Enabled  bool         `koanf:"enabled"`
	Mount    *MountConfig `koanf:"mount"`
	Resource *Resource    `koanf:"resource"`
}

// MountConfig Mount configuration
type MountConfig struct {
	Host string   `koanf:"host"`
	Path []string `koanf:"path" validate:"dive,required"`
}

// AuthProviderConfig Authentication provider configurations
type AuthProviderConfig struct {
	Basic map[string]*BasicAuthConfig `koanf:"basic" validate:"omitempty,dive"`
	OIDC  map[string]*OIDCAuthConfig  `koanf:"oidc" validate:"omitempty,dive"`
}

// OIDCAuthConfig OpenID Connect authentication configurations
type OIDCAuthConfig struct {
	ClientID      string            `koanf:"clientID" validate:"required"`
	ClientSecret  *CredentialConfig `koanf:"clientSecret" validate:"omitempty,dive"`
	IssuerURL     string            `koanf:"issuerUrl" validate:"required"`
	RedirectURL   string            `koanf:"redirectUrl" validate:"required"`
	Scopes        []string          `koanf:"scope"`
	State         string            `koanf:"state" validate:"required"`
	GroupClaim    string            `koanf:"groupClaim"`
	EmailVerified bool              `koanf:"emailVerified"`
	CookieName    string            `koanf:"cookieName"`
	CookieSecure  bool              `koanf:"cookieSecure"`
}

// OIDCAuthorizationAccess OpenID Connect authorization accesses
type OIDCAuthorizationAccess struct {
	Group string `koanf:"group" validate:"required_without=Email"`
	Email string `koanf:"email" validate:"required_without=Group"`
}

// BasicAuthConfig Basic auth configurations
type BasicAuthConfig struct {
	Realm string `koanf:"realm" validate:"required"`
}

// BasicAuthUserConfig Basic User auth configuration
type BasicAuthUserConfig struct {
	User     string            `koanf:"user" validate:"required"`
	Password *CredentialConfig `koanf:"password" validate:"required,dive"`
}

// TemplateConfig Templates configuration
type TemplateConfig struct {
	FolderList          string `koanf:"folderList" validate:"required"`
	TargetList          string `koanf:"targetList" validate:"required"`
	NotFound            string `koanf:"notFound" validate:"required"`
	InternalServerError string `koanf:"internalServerError" validate:"required"`
	Unauthorized        string `koanf:"unauthorized" validate:"required"`
	Forbidden           string `koanf:"forbidden" validate:"required"`
	BadRequest          string `koanf:"badRequest" validate:"required"`
}

// ServerConfig Server configuration
type ServerConfig struct {
	ListenAddr string `koanf:"listenAddr"`
	Port       int    `koanf:"port" validate:"required"`
}

// Target Bucket instance configuration
type Target struct {
	Name          string        `koanf:"name" validate:"required"`
	Bucket        *BucketConfig `koanf:"bucket" validate:"required"`
	Resources     []*Resource   `koanf:"resources" validate:"dive"`
	Mount         *MountConfig  `koanf:"mount" validate:"required"`
	IndexDocument string        `koanf:"indexDocument"`
}

// Resource Resource
type Resource struct {
	Path      string         `koanf:"path" validate:"required"`
	WhiteList *bool          `koanf:"whiteList"`
	Provider  string         `koanf:"provider"`
	Basic     *ResourceBasic `koanf:"basic" validate:"omitempty"`
	OIDC      *ResourceOIDC  `koanf:"oidc" validate:"omitempty"`
}

// ResourceBasic Basic auth resource
type ResourceBasic struct {
	Credentials []*BasicAuthUserConfig `koanf:"credentials" validate:"omitempty,dive"`
}

// ResourceOIDC OIDC auth Resource
type ResourceOIDC struct {
	AuthorizationAccesses []*OIDCAuthorizationAccess `koanf:"authorizationAccesses" validate:"dive"`
}

// BucketConfig Bucket configuration
type BucketConfig struct {
	Name        string                  `koanf:"name" validate:"required"`
	Prefix      string                  `koanf:"prefix"`
	Region      string                  `koanf:"region"`
	S3Endpoint  string                  `koanf:"s3Endpoint"`
	Credentials *BucketCredentialConfig `koanf:"credentials" validate:"omitempty,dive"`
}

// BucketCredentialConfig Bucket Credentials configurations
type BucketCredentialConfig struct {
	AccessKey *CredentialConfig `koanf:"accessKey" validate:"omitempty,dive"`
	SecretKey *CredentialConfig `koanf:"secretKey" validate:"omitempty,dive"`
}

// CredentialConfig Credential Configurations
type CredentialConfig struct {
	Path  string `koanf:"path" validate:"required_without_all=Env Value"`
	Env   string `koanf:"env" validate:"required_without_all=Path Value"`
	Value string `koanf:"value" validate:"required_without_all=Path Env"`
}

// LogConfig Log configuration
type LogConfig struct {
	Level  string `koanf:"level"`
	Format string `koanf:"format"`
}
