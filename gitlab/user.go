package gitlab

import (
	"github.com/xanzy/go-gitlab"
	"sync"
)

type (
	User struct {
		*gitlab.User
		aliases []string
		so      sync.Once
	}
)

func NewUser(user *gitlab.User, aliases []string) *User {
	return &User{User: user, aliases: aliases}
}

func (u *User) GetAliases() []string {
	u.so.Do(func() {
		addUsername, addPublicEmail := true, true

		for _, alias := range u.aliases {
			if alias == u.Username {
				addUsername = false
			}
			if alias == u.PublicEmail {
				addPublicEmail = false
			}
		}

		if addUsername && u.Username != "" {
			u.aliases = append(u.aliases, u.Username)
		}

		if addPublicEmail && u.PublicEmail != "" {
			u.aliases = append(u.aliases, u.PublicEmail)
		}
	})

	return u.aliases
}

func (u *User) GetEmail() string {
	return u.Email
}
