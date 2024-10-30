package utils

import (
	"fmt"
	"strings"
)

type AliasMap[T any] map[string][]string

func (m AliasMap[T]) Parse(aliases []string) error {
	for _, alias := range aliases {
		spl := strings.Split(alias, ":")

		if len(spl) != 2 {
			return fmt.Errorf("invalid format: %s", alias)
		}

		if _, ok := m[spl[0]]; !ok {
			m[spl[0]] = []string{}
		}

		m[spl[0]] = append(m[spl[0]], spl[1])
	}

	return nil
}

func (m AliasMap[T]) ToSlice(f func(name string, aliases []string) T) (slice []T) {
	for name, aliases := range m {
		slice = append(slice, f(name, aliases))
	}

	return
}
