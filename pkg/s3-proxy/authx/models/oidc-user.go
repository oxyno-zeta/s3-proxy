package models

const OIDCUserType = "OIDC"

type OIDCUser struct {
	Email  string
	Groups []string
}

func (u *OIDCUser) GetType() string {
	return OIDCUserType
}
