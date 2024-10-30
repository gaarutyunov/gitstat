package types

type User interface {
	GetEmail() string
	GetAliases() []string
}
