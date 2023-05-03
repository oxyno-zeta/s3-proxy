package models

const BasicAuthUserType = "BASIC"

type BasicAuthUser struct {
	Username string
}

func (*BasicAuthUser) GetType() string {
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
func (*BasicAuthUser) GetName() string {
	return ""
}

// Get groups (only available for OIDC user).
func (*BasicAuthUser) GetGroups() []string {
	return nil
}

// Get given name (only available for OIDC user).
func (*BasicAuthUser) GetGivenName() string {
	return ""
}

// Get family name (only available for OIDC user).
func (*BasicAuthUser) GetFamilyName() string {
	return ""
}

// Get email (only available for OIDC user).
func (*BasicAuthUser) GetEmail() string {
	return ""
}

// Is Email Verified ? (only available for OIDC user).
func (*BasicAuthUser) IsEmailVerified() bool {
	return false
}
