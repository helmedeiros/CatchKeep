// Package refdata serves the reference data bundled with the binary.
package refdata

import (
	_ "embed"
	"strings"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

//go:embed rare.txt
var rareList string

// Species answers questions about species from the bundled lists.
type Species struct {
	category map[string]domain.Category
}

// NewSpecies loads the bundled species lists.
func NewSpecies() *Species {
	s := &Species{category: map[string]domain.Category{}}
	var section domain.Category
	for line := range strings.Lines(rareList) {
		name := strings.TrimSpace(line)
		switch {
		case name == "" || strings.HasPrefix(name, "#"):
		case strings.HasPrefix(name, "[") && strings.HasSuffix(name, "]"):
			section = domain.Category(strings.Trim(name, "[]"))
		default:
			s.category[strings.ToLower(name)] = section
		}
	}
	return s
}

// Category names a species' category: legendary, mythical, Ultra Beast or
// none.
func (s *Species) Category(species string) domain.Category {
	return s.category[strings.ToLower(strings.TrimSpace(species))]
}
