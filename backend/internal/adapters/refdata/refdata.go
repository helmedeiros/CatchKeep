// Package refdata serves the reference data bundled with the binary.
package refdata

import (
	_ "embed"
	"strings"
)

//go:embed rare.txt
var rareList string

// Species answers questions about species from the bundled lists.
type Species struct {
	rare map[string]bool
}

// NewSpecies loads the bundled species lists.
func NewSpecies() *Species {
	s := &Species{rare: map[string]bool{}}
	for line := range strings.Lines(rareList) {
		name := strings.TrimSpace(line)
		if name == "" || strings.HasPrefix(name, "#") {
			continue
		}
		s.rare[strings.ToLower(name)] = true
	}
	return s
}

// Rare reports whether a species is legendary, mythical or an Ultra Beast.
func (s *Species) Rare(species string) bool {
	return s.rare[strings.ToLower(strings.TrimSpace(species))]
}
