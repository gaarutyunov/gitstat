package types

type TotalCounter interface {
	Total() int
}

type PerUserCounter interface {
	PerUser() map[User]PerLanguageCounter
}

type PerLanguageCounter interface {
	PerLanguage() map[Language]int
	TotalCounter
}

type LineCounter interface {
	PerUserCounter
	PerLanguageCounter
}
