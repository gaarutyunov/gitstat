package models

import (
	"github.com/gaarutyunov/gitstat/types"
	"strings"
)

type Language struct {
	name string
	ext  []string
}

func (l *Language) Name() string {
	return l.name
}

func (l *Language) Ext() []string {
	return l.ext
}

func NewLanguage(name string, ext []string) types.Language {
	for i, s := range ext {
		if !strings.HasPrefix(s, ".") {
			ext[i] = "." + s
		}
	}
	return &Language{name: name, ext: ext}
}
