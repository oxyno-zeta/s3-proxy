package models

// Generic user interface used to know specific type of user.
type GenericUser interface {
	// Get type of user (OIDC or BASIC).
	GetType() string
	// Get identifier (Username for basic auth user or Username or email for OIDC user).
	GetIdentifier() string
	// Get username.
	GetUsername() string
	// Get name (only available for OIDC user).
	GetName() string
	// Get groups (only available for OIDC user).
	GetGroups() []string
	// Get given name (only available for OIDC user).
	GetGivenName() string
	// Get family name (only available for OIDC user).
	GetFamilyName() string
	// Get email (only available for OIDC user).
	GetEmail() string
	// Is Email Verified ? (only available for OIDC user).
	IsEmailVerified() bool
}
