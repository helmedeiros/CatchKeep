package domain_test

import (
	"errors"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// known returns a Pokémon whose variant is fully known (not shiny etc.).
func known(id, species string, ivs ...string) domain.Pokemon {
	kv := knownVariant(species)
	if len(ivs) == 3 {
		kv = append(kv, "iv_attack", ivs[0], "iv_defense", ivs[1], "iv_stamina", ivs[2])
	}
	return mon(id, kv...)
}

// knownVariant is a species with every variant feature confirmed absent.
func knownVariant(species string) []string {
	return []string{"species", species, "form", "Normal", "shiny", "false", "costume", "false", "background", "false", "dynamax", "false", "gigantamax", "false"}
}

func TestSetup_Validate(t *testing.T) {
	for name, s := range map[string]domain.Setup{
		"no goals":        {Capacity: 300},
		"repeated goal":   {Capacity: 300, Goals: []domain.Goal{domain.GoalCollecting, domain.GoalCollecting}},
		"unknown goal":    {Capacity: 300, Goals: []domain.Goal{"hoarding"}},
		"zero capacity":   {Capacity: 0, Goals: []domain.Goal{domain.GoalCollecting}},
		"absurd capacity": {Capacity: 100000, Goals: []domain.Goal{domain.GoalCollecting}},
	} {
		if err := s.Validate(); !errors.Is(err, domain.ErrInvalidSetup) {
			t.Errorf("%s: err = %v, want ErrInvalidSetup", name, err)
		}
	}
	if err := (domain.Setup{Capacity: 2700, Goals: []domain.Goal{domain.GoalBattling, domain.GoalCollecting}}).Validate(); err != nil {
		t.Fatalf("valid setup rejected: %v", err)
	}
}

func TestMinimums_ScaleWithCapacity(t *testing.T) {
	if small := domain.OrdinaryMinimum(350); small != 1 {
		t.Fatalf("small bag minimum = %d, want 1", small)
	}
	if big := domain.OrdinaryMinimum(2700); big != 2 {
		t.Fatalf("2,700 slots minimum = %d, want 2", big)
	}
	if rare := domain.RareMinimum(2700); rare != 4 {
		t.Fatalf("2,700 slots rare minimum = %d, want 4", rare)
	}
}

func TestSummarizeKeeps_KeepsTheBestCopiesUpToTheMinimum(t *testing.T) {
	pokemon := []domain.Pokemon{
		known("P1", "Pidgey", "1", "1", "1"),
		known("P2", "Pidgey", "15", "15", "15"),
		known("P3", "Pidgey", "10", "10", "10"),
		known("P4", "Mewtwo", "15", "14", "13"),
	}

	s := domain.SummarizeKeeps(pokemon, domain.Setup{Capacity: 2700, Goals: []domain.Goal{domain.GoalCollecting}}, nil)

	if s.Total != 4 || s.Kept != 3 || s.Surplus != 1 || s.NeedsConfirmation != 0 || s.Minimum != 2 {
		t.Fatalf("summary = %+v", s)
	}
	pidgey := s.Groups[0]
	if pidgey.Species != "Pidgey" || pidgey.Count != 3 || pidgey.Keep != 2 || len(pidgey.SurplusIDs) != 1 || pidgey.SurplusIDs[0] != "P1" {
		t.Fatalf("pidgey group = %+v; want the 1/1/1 copy as surplus", pidgey)
	}
}

func TestSummarizeKeeps_UnknownVariantFeaturesWaitForConfirmation(t *testing.T) {
	pokemon := []domain.Pokemon{
		known("P1", "Pidgey"),
		mon("P2", "species", "Pidgey", "form", "Normal"),
	}

	s := domain.SummarizeKeeps(pokemon, domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}}, nil)

	if s.NeedsConfirmation != 1 || s.Kept != 1 || s.Surplus != 0 {
		t.Fatalf("summary = %+v; the unconfirmed Pidgey must never be surplus", s)
	}
}

func TestSummarizeKeeps_DistinctVariantsAreKeptSeparately(t *testing.T) {
	shiny := mon("P2", "species", "Pidgey", "form", "Normal", "shiny", "true", "costume", "false", "background", "false", "dynamax", "false", "gigantamax", "false")

	s := domain.SummarizeKeeps([]domain.Pokemon{known("P1", "Pidgey"), shiny}, domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}}, nil)

	if len(s.Groups) != 2 || s.Surplus != 0 {
		t.Fatalf("summary = %+v; a shiny is its own variant", s)
	}
}

func TestSummarizeKeeps_TradingKeepsAnExtraSpecialVariant(t *testing.T) {
	shiny := func(id string) domain.Pokemon {
		return mon(id, "species", "Pidgey", "form", "Normal", "shiny", "true", "costume", "false", "background", "false", "dynamax", "false", "gigantamax", "false")
	}
	pokemon := []domain.Pokemon{shiny("P1"), shiny("P2"), shiny("P3")}

	collecting := domain.SummarizeKeeps(pokemon, domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}}, nil)
	trading := domain.SummarizeKeeps(pokemon, domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting, domain.GoalTrading}}, nil)

	if collecting.Surplus != 2 || trading.Surplus != 1 {
		t.Fatalf("surplus collecting=%d trading=%d, want 2 and 1", collecting.Surplus, trading.Surplus)
	}
}

func TestSummarizeKeeps_LuckyShadowAndPurifiedCopiesAreTheirOwnVariants(t *testing.T) {
	kyurem := func(id string, kv ...string) domain.Pokemon {
		return mon(id, append(knownVariant("Kyurem"), kv...)...)
	}
	pokemon := []domain.Pokemon{
		kyurem("P1"), kyurem("P2"),
		kyurem("P3", "lucky", "true"),
		kyurem("P4", "shadow", "true"),
		kyurem("P5", "purified", "true"),
	}

	s := domain.SummarizeKeeps(pokemon, domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}}, nil)

	if len(s.Groups) != 4 || s.Surplus != 1 || s.NeedsConfirmation != 0 {
		t.Fatalf("summary = %+v; want plain, lucky, shadow and purified groups and one plain extra", s)
	}
	if g := s.Groups[0]; g.Lucky || g.Shadow || g.Purified || g.Count != 2 {
		t.Fatalf("largest surplus group = %+v, want the two plain copies", g)
	}
}

func TestSummarizeKeeps_UnestablishedLuckyOrShadowDoesNotWaitForConfirmation(t *testing.T) {
	// The export only says "1" for lucky and shadow; anything else is not
	// established and groups with the ordinary copies, as §5.3 alone would.
	pokemon := []domain.Pokemon{known("P1", "Pidgey"), known("P2", "Pidgey")}

	s := domain.SummarizeKeeps(pokemon, domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}}, nil)

	if s.NeedsConfirmation != 0 || s.Surplus != 1 {
		t.Fatalf("summary = %+v", s)
	}
}

func TestSummarizeKeeps_RareSpeciesKeepMoreCopies(t *testing.T) {
	var pokemon []domain.Pokemon
	for _, id := range []string{"K1", "K2", "K3", "K4", "K5", "K6"} {
		pokemon = append(pokemon, known(id, "Kyurem"))
	}
	rare := func(species string) bool { return species == "Kyurem" }
	collecting := domain.Setup{Capacity: 2700, Goals: []domain.Goal{domain.GoalCollecting}}

	s := domain.SummarizeKeeps(pokemon, collecting, rare)

	if s.RareMinimum != 4 || s.Kept != 4 || s.Surplus != 2 || !s.Groups[0].Rare {
		t.Fatalf("summary = %+v; want 4 Kyurem kept as a rare species", s)
	}
	trading := domain.Setup{Capacity: 2700, Goals: []domain.Goal{domain.GoalTrading}}
	if s := domain.SummarizeKeeps(pokemon, trading, rare); s.Kept != 5 {
		t.Fatalf("trading kept %d, want one more legendary for trades", s.Kept)
	}
}

func TestSetup_ValidateSpeciesRules(t *testing.T) {
	base := domain.Setup{Capacity: 2700, Goals: []domain.Goal{domain.GoalCollecting}}
	for name, rules := range map[string][]domain.SpeciesRule{
		"no species":       {{Minimum: 2}},
		"nothing set":      {{Species: "Kyurem"}},
		"negative minimum": {{Species: "Kyurem", Minimum: -1}},
		"absurd minimum":   {{Species: "Kyurem", Minimum: 100000}},
		"repeated":         {{Species: "Kyurem", Minimum: 2}, {Species: "Kyurem", Favorite: true}},
	} {
		s := base
		s.Species = rules
		if err := s.Validate(); !errors.Is(err, domain.ErrInvalidSetup) {
			t.Errorf("%s: err = %v, want ErrInvalidSetup", name, err)
		}
	}
	base.Species = []domain.SpeciesRule{{Species: "Kyurem", Minimum: 10}, {Species: "Eevee", Favorite: true}}
	if err := base.Validate(); err != nil {
		t.Fatalf("valid species rules rejected: %v", err)
	}
}

// keptOf counts the kept copies of a species and returns its groups' rule.
func keptOf(s domain.KeepSummary, species string) (int, domain.Rule) {
	kept, rule := 0, domain.Rule("")
	for _, g := range s.Groups {
		if g.Species == species {
			kept += g.Keep
			rule = g.Rule
		}
	}
	return kept, rule
}

func TestSummarizeKeeps_SpeciesMinimumCountsAcrossVariantsAndCanLowerTheDefault(t *testing.T) {
	var pokemon []domain.Pokemon
	for i, iv := range []string{"1", "2", "3", "4", "5", "15"} {
		pokemon = append(pokemon, known("K"+string(rune('1'+i)), "Kyurem", iv, iv, iv))
	}
	pokemon = append(pokemon,
		mon("S1", append(knownVariant("Kyurem"), "shadow", "true", "iv_attack", "0", "iv_defense", "0", "iv_stamina", "0")...),
		mon("S2", append(knownVariant("Kyurem"), "shadow", "true", "iv_attack", "14", "iv_defense", "14", "iv_stamina", "14")...),
	)
	rare := func(string) bool { return true }
	setup := domain.Setup{Capacity: 2700, Goals: []domain.Goal{domain.GoalCollecting}}

	if kept, rule := keptOf(domain.SummarizeKeeps(pokemon, setup, rare), "Kyurem"); kept != 6 || rule != domain.RuleRare {
		t.Fatalf("default kept %d by %q, want 4 plain + 2 shadow by the rare rule", kept, rule)
	}

	setup.Species = []domain.SpeciesRule{{Species: "Kyurem", Minimum: 3}}
	s := domain.SummarizeKeeps(pokemon, setup, rare)
	kept, rule := keptOf(s, "Kyurem")
	if kept != 3 || rule != domain.RuleSpecies {
		t.Fatalf("kept %d by %q, want 3 by the species rule", kept, rule)
	}
	for _, g := range s.Groups {
		if g.Shadow && (g.Keep != 1 || g.KeptIDs[0] != "S2") {
			t.Fatalf("shadow group %+v: one of each variant, then the best copies overall", g)
		}
		if !g.Shadow && (g.Keep != 2 || g.KeptIDs[1] != "K5") {
			t.Fatalf("plain group %+v: the 5/5/5 copy beats the 0/0/0 shadow for the third slot", g)
		}
	}

	setup.Species = []domain.SpeciesRule{{Species: "Kyurem", Minimum: 7}}
	if kept, _ := keptOf(domain.SummarizeKeeps(pokemon, setup, rare), "Kyurem"); kept != 7 {
		t.Fatalf("kept %d, want 7", kept)
	}
}

func TestSummarizeKeeps_FavoriteSpeciesKeepEveryCopy(t *testing.T) {
	pokemon := []domain.Pokemon{known("P1", "Eevee"), known("P2", "Eevee"), known("P3", "Eevee"), known("P4", "Pidgey"), known("P5", "Pidgey")}
	setup := domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}, Species: []domain.SpeciesRule{{Species: "Eevee", Minimum: 1, Favorite: true}}}

	s := domain.SummarizeKeeps(pokemon, setup, nil)

	if kept, rule := keptOf(s, "Eevee"); kept != 3 || rule != domain.RuleFavorite {
		t.Fatalf("Eevee kept %d by %q, want all 3 as a favorite", kept, rule)
	}
	if s.Surplus != 1 {
		t.Fatalf("surplus = %d, want one extra Pidgey", s.Surplus)
	}
}

func TestSummarizeKeeps_ProtectedCopiesAreAlwaysKeptAndCountTowardTheMinimum(t *testing.T) {
	weak := known("P1", "Pidgey", "0", "0", "0")
	weak.Protected = true
	pokemon := []domain.Pokemon{weak, known("P2", "Pidgey", "15", "15", "15"), known("P3", "Pidgey", "10", "10", "10")}
	setup := domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}}

	s := domain.SummarizeKeeps(pokemon, setup, nil)

	g := s.Groups[0]
	if s.Protected != 1 || g.Keep != 1 || len(g.KeptIDs) != 1 || g.KeptIDs[0] != "P1" || len(g.ProtectedIDs) != 1 {
		t.Fatalf("group = %+v; the protected 0/0/0 copy fills the one slot", g)
	}

	strong := pokemon[1]
	strong.Protected = true
	pokemon[1] = strong
	s = domain.SummarizeKeeps(pokemon, setup, nil)
	if g := s.Groups[0]; g.Keep != 2 || s.Surplus != 1 || g.SurplusIDs[0] != "P3" {
		t.Fatalf("group = %+v; both protected copies stay even above the minimum", g)
	}
}
