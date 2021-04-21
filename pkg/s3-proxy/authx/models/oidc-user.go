package models

const OIDCUserType = "OIDC"

type OIDCUser struct {
	PreferredUsername string   `json:"preferred_username"`
	Name              string   `json:"name"`
	Groups            []string `json:"groups"`
	GivenName         string   `json:"given_name"`
	FamilyName        string   `json:"family_name"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
}

func (u *OIDCUser) GetType() string {
	return OIDCUserType
}

func (u *OIDCUser) GetIdentifier() string {
	if u.PreferredUsername != "" {
		return u.PreferredUsername
	}

	return u.Email
}

// Get username.
func (u *OIDCUser) GetUsername() string {
	return u.PreferredUsername
}

// Get name (only available for OIDC user).
func (u *OIDCUser) GetName() string {
	return u.Name
}

// Get groups (only available for OIDC user).
func (u *OIDCUser) GetGroups() []string {
	return u.Groups
}

// Get given name (only available for OIDC user).
func (u *OIDCUser) GetGivenName() string {
	return u.GivenName
}

// Get family name (only available for OIDC user).
func (u *OIDCUser) GetFamilyName() string {
	return u.FamilyName
}

// Get email (only available for OIDC user).
func (u *OIDCUser) GetEmail() string {
	return u.Email
}

// Is Email Verified ? (only available for OIDC user).
func (u *OIDCUser) IsEmailVerified() bool {
	return u.EmailVerified
}
