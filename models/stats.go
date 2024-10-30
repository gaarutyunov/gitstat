package models

import (
	"encoding/json"
	"fmt"
	"github.com/gaarutyunov/gitstat/types"
)

type (
	PerLangMap map[types.Language]int

	StatsPerLang struct {
		PerLang PerLangMap `json:"per_lang"`
		Total   int        `json:"total"`
	}

	StatsPerUser map[types.User]types.PerLanguageCounter

	Stats struct {
		StatsPerLang
		PerUser StatsPerUser `json:"per_user"`
	}
)

func NewStats(g types.Stats) *Stats {
	return &Stats{
		StatsPerLang: StatsPerLang{
			PerLang: g.PerLanguage(),
			Total:   g.Total(),
		},
		PerUser: g.PerUser(),
	}
}

func (s StatsPerUser) MarshalJSON() ([]byte, error) {
	m := make(map[string]map[string]int)

	for k, v := range s {
		if _, ok := m[k.GetEmail()]; !ok {
			m[k.GetEmail()] = make(map[string]int)
		}

		for language, n := range v.PerLanguage() {
			m[k.GetEmail()][language.Name()] = n
		}
	}

	return json.Marshal(m)
}

func (p PerLangMap) MarshalJSON() ([]byte, error) {
	m := make(map[string]int)

	for k, v := range p {
		m[k.Name()] = v
	}

	return json.Marshal(m)
}

func (s Stats) String() (txt string) {
	if s.Total == 0 {
		return "Empty statistics, try changing --query, --lang or --user"
	}

	txt += "Languages:\n"

	for k, v := range s.PerLang {
		txt += fmt.Sprintf("  - %s: %d\n", k.Name(), v)
	}

	txt += fmt.Sprintf("  - Total: %d\n", s.Total)

	txt += "Users:\n"

	for user, counter := range s.PerUser {
		txt += fmt.Sprintf("  - %s:\n", user.GetEmail())

		for language, n := range counter.PerLanguage() {
			txt += fmt.Sprintf("    - %s: %d\n", language.Name(), n)
		}

		txt += fmt.Sprintf("    - Total: %d\n", counter.Total())
	}

	return
}
