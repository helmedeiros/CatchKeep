package domain_test

import (
	"fmt"
	"slices"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func dexMon(id, species, dex, cp, hp string) domain.Pokemon {
	return mon(id, append(knownVariant(species), "dex_number", dex, "cp", cp, "hp", hp)...)
}

func TestMatches_EvaluatesTheSearchGrammarSubset(t *testing.T) {
	pidgey := dexMon("P1", "Pidgey", "16", "120", "30").Current()
	for query, want := range map[string]bool{
		"16":                  true,
		"19":                  false,
		"cp120":               true,
		"16&cp120":            true,
		"16&cp130":            false,
		"16&cp90,cp120,cp130": true,
		"16&cp100-130":        true,
		"16&cp121-":           false,
		"16&cp-120&hp30":      true,
		"16&hp31":             false,
		"16&!cp120":           false,
		"!19":                 true,
	} {
		if got, err := domain.Matches(query, pidgey); err != nil || got != want {
			t.Errorf("%q: %v, %v; want %v", query, got, err, want)
		}
	}
	for _, bad := range []string{"shiny", "pidgey", "16&", "cpx"} {
		if _, err := domain.Matches(bad, pidgey); err == nil {
			t.Errorf("%q: unsupported term accepted", bad)
		}
	}
	unknownCP := mon("U", append(knownVariant("Pidgey"), "dex_number", "16")...).Current()
	if got, _ := domain.Matches("16&cp120", unknownCP); !got {
		t.Error("a Pokémon with unknown CP might match, so it counts as a match")
	}
}

func TestSearches_OnePerSpeciesNarrowedWhenTheyWouldCatchKeptPokemon(t *testing.T) {
	pokemon := []domain.Pokemon{
		dexMon("P1", "Pidgey", "16", "90", "29"),
		dexMon("P2", "Pidgey", "16", "120", "30"),
		dexMon("P3", "Pidgey", "16", "140", "32"),
		dexMon("K1", "Pidgey", "16", "120", "31"),
		dexMon("K2", "Pidgey", "16", "90", "29"),
		dexMon("R1", "Rattata", "19", "50", "20"),
		dexMon("R2", "Rattata", "19", "60", "22"),
	}
	planned := []domain.PokemonID{"P1", "P2", "P3", "R1"}

	got := domain.Searches(pokemon, planned)
	if len(got) != 2 {
		t.Fatalf("searches = %+v; one for Pidgey, one for Rattata", got)
	}
	rattata := got[1]
	if rattata.Species != "Rattata" || rattata.Query != "19&cp50" || len(rattata.Collisions) != 0 {
		t.Fatalf("Rattata = %+v", rattata)
	}
	pidgey := got[0]
	if pidgey.Species != "Pidgey" || pidgey.Query != "16&cp90,cp120,cp140" {
		t.Fatalf("Pidgey = %+v", pidgey)
	}
	if fmt.Sprint(pidgey.Collisions) != "[K1 K2]" {
		t.Fatalf("collisions = %v; K1 shares CP 120 and K2 shares CP 90", pidgey.Collisions)
	}
	byID := map[domain.PokemonID]domain.ItemSearch{}
	for _, it := range pidgey.Items {
		byID[it.ID] = it
	}
	if it := byID["P2"]; it.Query != "16&cp120&hp30" || len(it.Collisions) != 0 {
		t.Errorf("P2 = %+v; HP 30 tells it apart from the kept CP 120", it)
	}
	if it := byID["P1"]; it.Query != "16&cp90&hp29" || !slices.Equal(it.Collisions, []domain.PokemonID{"K2"}) {
		t.Errorf("P1 = %+v; same CP and HP as K2, so check it in the game", it)
	}
	if it := byID["P3"]; it.Query != "" {
		t.Errorf("P3 = %+v; the species search already finds it safely", it)
	}
}
