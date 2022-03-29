package models

const HeaderUserType = "HEADER"

type HeaderUser struct {
	// Username example: x-forwarded-preferred-username: "user"
	Username string `json:"username"`
	// Email example: x-forwarded-email: "sample-user@example.com"
	Email string `json:"email"`
	// Groups example: x-forwarded-groups: "group1,group2"
	Groups []string `json:"groups"`
}

func (u *HeaderUser) GetType() string {
	return HeaderUserType
}

func (u *HeaderUser) GetIdentifier() string {
	if u.Username != "" {
		return u.Username
	}

	return u.Email
}

// Get username.
func (u *HeaderUser) GetUsername() string {
	return u.Username
}

// Get name (only available for OIDC user).
func (u *HeaderUser) GetName() string {
	return ""
}

// Get groups (only available for OIDC and Header user).
func (u *HeaderUser) GetGroups() []string {
	return u.Groups
}

// Get given name (only available for OIDC user).
func (u *HeaderUser) GetGivenName() string {
	return ""
}

// Get family name (only available for OIDC user).
func (u *HeaderUser) GetFamilyName() string {
	return ""
}

// Get email (only available for OIDC user).
func (u *HeaderUser) GetEmail() string {
	return u.Email
}

// Is Email Verified ? (only available for OIDC user).
func (u *HeaderUser) IsEmailVerified() bool {
	return false
}
