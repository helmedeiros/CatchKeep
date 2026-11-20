package refdata_test

import (
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/refdata"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestSpecies_Category(t *testing.T) {
	s := refdata.NewSpecies()
	for name, want := range map[string]domain.Category{
		"Kyurem":     domain.CategoryLegendary,
		"Mewtwo":     domain.CategoryLegendary,
		"Ho-Oh":      domain.CategoryLegendary,
		"Type: Null": domain.CategoryLegendary,
		"Tapu Koko":  domain.CategoryLegendary,
		"mew":        domain.CategoryMythical,
		"Pecharunt":  domain.CategoryMythical,
		"Meltan":     domain.CategoryMythical,
		"Nihilego":   domain.CategoryUltraBeast,
		"Poipole":    domain.CategoryUltraBeast,
		"Pidgey":     "",
		"Mr. Mime":   "",
		"":           "",
		"# Kanto":    "",
		"[mythical]": "",
	} {
		if got := s.Category(name); got != want {
			t.Errorf("%q: category %q, want %q", name, got, want)
		}
	}
}
