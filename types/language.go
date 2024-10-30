package types

type Language interface {
	Name() string
	Ext() []string
}
