package models

const BasicAuthUserType = "BASIC"

type BasicAuthUser struct {
	Username string
}

func (u *BasicAuthUser) GetType() string {
	return BasicAuthUserType
}
