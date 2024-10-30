package models

import "github.com/gaarutyunov/gitstat/types"

type User struct {
	email   string
	aliases []string
}

func NewUser(email string, aliases []string) types.User {
	return &User{email: email, aliases: aliases}
}

func (u *User) GetEmail() string {
	return u.email
}

func (u *User) GetAliases() []string {
	return u.aliases
}
