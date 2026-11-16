package refdata_test

import (
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/refdata"
)

func TestSpecies_Rare(t *testing.T) {
	s := refdata.NewSpecies()
	for _, name := range []string{"Kyurem", "Mewtwo", "mew", "Ho-Oh", "Type: Null", "Tapu Koko", "Nihilego", "Pecharunt"} {
		if !s.Rare(name) {
			t.Errorf("%s should be rare", name)
		}
	}
	for _, name := range []string{"Pidgey", "Charmander", "Mr. Mime", "", "# Kanto"} {
		if s.Rare(name) {
			t.Errorf("%q should not be rare", name)
		}
	}
}
