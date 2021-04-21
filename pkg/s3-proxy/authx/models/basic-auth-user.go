package models

const BasicAuthUserType = "BASIC"

type BasicAuthUser struct {
	Username string
}

func (u *BasicAuthUser) GetType() string {
	return BasicAuthUserType
}

func (u *BasicAuthUser) GetIdentifier() string {
	return u.Username
}

// Get username.
func (u *BasicAuthUser) GetUsername() string {
	return u.Username
}

// Get name (only available for OIDC user).
func (u *BasicAuthUser) GetName() string {
	return ""
}

// Get groups (only available for OIDC user).
func (u *BasicAuthUser) GetGroups() []string {
	return nil
}

// Get given name (only available for OIDC user).
func (u *BasicAuthUser) GetGivenName() string {
	return ""
}

// Get family name (only available for OIDC user).
func (u *BasicAuthUser) GetFamilyName() string {
	return ""
}

// Get email (only available for OIDC user).
func (u *BasicAuthUser) GetEmail() string {
	return ""
}

// Is Email Verified ? (only available for OIDC user).
func (u *BasicAuthUser) IsEmailVerified() bool {
	return false
}
