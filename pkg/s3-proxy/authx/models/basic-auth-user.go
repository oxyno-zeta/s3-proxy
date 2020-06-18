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
