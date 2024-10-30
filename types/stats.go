package types

type Stats interface {
	LineCounter
	Err() error
}
