package domain

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Setup errors.
var (
	ErrInvalidSetup  = errors.New("invalid setup")
	ErrSetupNotFound = errors.New("setup not done yet")
)

// Goal is what a trainer keeps Pokémon for (spec V1-SET-01).
type Goal string

// Goals.
const (
	GoalCollecting Goal = "collecting"
	GoalTrading    Goal = "trading"
	GoalBattling   Goal = "battling"
)

// maxCapacity bounds storage well above the game's own limit.
const maxCapacity = 20000

// Setup is storage capacity and goals in priority order (step 1), plus
// optional per-species rules (spec V1-SET-02, V1-SET-03).
type Setup struct {
	Capacity int
	Goals    []Goal
	Species  []SpeciesRule
}

// SpeciesRule overrides the generated minimums of one species. Minimum
// counts copies across all of its variants; a favorite keeps every copy.
type SpeciesRule struct {
	Species  string
	Minimum  int
	Favorite bool
}

// Validate checks capacity and that goals are known and not repeated.
func (s Setup) Validate() error {
	if s.Capacity < 1 || s.Capacity > maxCapacity {
		return fmt.Errorf("%w: storage must be between 1 and %d", ErrInvalidSetup, maxCapacity)
	}
	if len(s.Goals) == 0 {
		return fmt.Errorf("%w: choose at least one goal", ErrInvalidSetup)
	}
	seen := map[Goal]bool{}
	for _, g := range s.Goals {
		if g != GoalCollecting && g != GoalTrading && g != GoalBattling {
			return fmt.Errorf("%w: unknown goal %q", ErrInvalidSetup, g)
		}
		if seen[g] {
			return fmt.Errorf("%w: goal %q is listed twice", ErrInvalidSetup, g)
		}
		seen[g] = true
	}
	species := map[string]bool{}
	for _, r := range s.Species {
		switch {
		case strings.TrimSpace(r.Species) == "":
			return fmt.Errorf("%w: a species rule needs a species", ErrInvalidSetup)
		case r.Minimum < 0 || r.Minimum > maxCapacity:
			return fmt.Errorf("%w: minimum for %s must be between 0 and %d", ErrInvalidSetup, r.Species, maxCapacity)
		case r.Minimum == 0 && !r.Favorite:
			return fmt.Errorf("%w: set a minimum or mark %s as a favorite", ErrInvalidSetup, r.Species)
		case species[r.Species]:
			return fmt.Errorf("%w: %s has two rules", ErrInvalidSetup, r.Species)
		}
		species[r.Species] = true
	}
	return nil
}

func (s Setup) has(g Goal) bool { return slices.Contains(s.Goals, g) }

// CapacityBand is one storage size range: up to UpTo slots, keep Ordinary
// copies of each variant and Rare of rare species (spec V1-RUL-02).
type CapacityBand struct{ UpTo, Ordinary, Rare int }

// capacityBands are placeholder defaults until OD-01 is decided.
var capacityBands = []CapacityBand{
	{UpTo: 1000, Ordinary: 1, Rare: 2},
	{UpTo: 4000, Ordinary: 2, Rare: 4},
	{UpTo: maxCapacity, Ordinary: 3, Rare: 6},
}

// CapacityBands returns the bands, smallest first, ending at the largest
// storage a setup allows.
func CapacityBands() []CapacityBand { return slices.Clone(capacityBands) }

func bandFor(capacity int) CapacityBand {
	for _, b := range capacityBands {
		if capacity <= b.UpTo {
			return b
		}
	}
	return capacityBands[len(capacityBands)-1]
}

// OrdinaryMinimum is how many copies of each variant are kept by default.
func OrdinaryMinimum(capacity int) int { return bandFor(capacity).Ordinary }

// RareMinimum is how many copies of each variant of a rare species
// (legendary, mythical, Ultra Beast) are kept by default.
func RareMinimum(capacity int) int { return bandFor(capacity).Rare }

// Rarity reports whether a species is rare. A nil Rarity treats none as rare.
type Rarity func(species string) bool

// variantFields complete a variant key with species and form (spec §5.3).
var variantFields = []Field{FieldShiny, FieldCostume, FieldBackground, FieldDynamax, FieldGigantamax}

// Variant is a fully known variant key. Lucky, Shadow and Purified extend
// §5.3 so those copies are never transferred in place of ordinary ones.
type Variant struct {
	Species, Form                                   string
	Shiny, Costume, Background, Dynamax, Gigantamax bool
	Lucky, Shadow, Purified                         bool
}

func (v Variant) special() bool { return v.Shiny || v.Costume || v.Background }

// Rule is what decided a group's minimum.
type Rule string

// Rules, from generated defaults to the trainer's own.
const (
	RuleDefault  Rule = "default"
	RuleRare     Rule = "rare"
	RuleSpecies  Rule = "species"
	RuleFavorite Rule = "favorite"
)

// VariantGroup is the trainer's copies of one variant and what the keep
// rules keep.
type VariantGroup struct {
	Variant
	// Rare is set for legendary, mythical and Ultra Beast species.
	Rare bool
	// Default is the generated minimum (spec V1-RUL-09), including the
	// trading bonus when TradeBonus is set; Rule says what replaced it.
	Default              int
	TradeBonus           bool
	Rule                 Rule
	Count, Minimum, Keep int
	// KeptIDs are the protected copies, then the best others; SurplusIDs
	// the rest, best first. ProtectedIDs are the kept copies the trainer
	// protected.
	KeptIDs, SurplusIDs, ProtectedIDs []PokemonID
}

// KeepSummary is what the default keep rules do with a collection:
// Kept + Surplus + NeedsConfirmation = Total.
type KeepSummary struct {
	Total, Kept, Surplus, NeedsConfirmation int
	// Protected counts kept Pokémon the trainer protected.
	Protected int
	// Minimum and RareMinimum are the per-variant minimums for the setup's
	// capacity.
	Minimum, RareMinimum int
	// Groups are sorted by surplus, largest first.
	Groups []VariantGroup
}

// SummarizeKeeps applies the keep rules (spec V1-RUL-01..06): keep the best
// Minimum copies of each variant (RareMinimum for rare species), one more of
// rare, shiny, costume or background variants when trading is a goal. A
// species rule replaces those defaults for its species: one copy of each
// variant, then the best copies overall up to its minimum, or every copy of
// a favorite. Pokémon with an unknown variant feature could belong to an
// under-filled variant, so they are kept and counted as needing
// confirmation (spec §7.3).
func SummarizeKeeps(pokemon []Pokemon, setup Setup, rare Rarity) KeepSummary {
	band := bandFor(setup.Capacity)
	out := KeepSummary{Total: len(pokemon), Minimum: band.Ordinary, RareMinimum: band.Rare}
	groups := map[Variant][]member{}
	var order []Variant
	for _, p := range pokemon {
		cur := p.Current()
		v, ok := variantOf(cur)
		if !ok {
			out.NeedsConfirmation++
			continue
		}
		if _, seen := groups[v]; !seen {
			order = append(order, v)
		}
		groups[v] = append(groups[v], member{p.ID, rank(cur), p.Protected})
	}
	built := map[Variant]*VariantGroup{}
	bySpecies := map[string][]*VariantGroup{}
	for _, v := range order {
		slices.SortStableFunc(groups[v], member.compare)
		g := &VariantGroup{Variant: v, Rare: rare != nil && rare(v.Species), Rule: RuleDefault, Count: len(groups[v]), Minimum: out.Minimum}
		if g.Rare {
			g.Rule, g.Minimum = RuleRare, out.RareMinimum
		}
		if setup.has(GoalTrading) && (g.Rare || v.special()) {
			g.Minimum++
			g.TradeBonus = true
		}
		g.Default = g.Minimum
		g.Keep = min(g.Minimum, g.Count)
		built[v] = g
		bySpecies[v.Species] = append(bySpecies[v.Species], g)
	}
	for _, r := range setup.Species {
		applySpeciesRule(r, bySpecies[r.Species], groups)
	}
	for _, v := range order {
		g := built[v]
		split(g, groups[v])
		out.Protected += len(g.ProtectedIDs)
		out.Kept += g.Keep
		out.Surplus += len(g.SurplusIDs)
		out.Groups = append(out.Groups, *g)
	}
	slices.SortStableFunc(out.Groups, func(a, b VariantGroup) int {
		return cmp.Or(cmp.Compare(len(b.SurplusIDs), len(a.SurplusIDs)), cmp.Compare(a.Species, b.Species))
	})
	return out
}

type member struct {
	id        PokemonID
	score     [2]int
	protected bool
}

// split keeps every protected copy, which also counts toward the group's
// minimum (V1-RUL-05), then the best others up to it.
func split(g *VariantGroup, members []member) {
	for _, m := range members {
		if m.protected {
			g.KeptIDs = append(g.KeptIDs, m.id)
			g.ProtectedIDs = append(g.ProtectedIDs, m.id)
		}
	}
	for _, m := range members {
		switch {
		case m.protected:
		case len(g.KeptIDs) < g.Keep:
			g.KeptIDs = append(g.KeptIDs, m.id)
		default:
			g.SurplusIDs = append(g.SurplusIDs, m.id)
		}
	}
	g.Keep = len(g.KeptIDs)
}

// compare orders the better copy first, then by ID for stability.
func (a member) compare(b member) int {
	return cmp.Or(cmp.Compare(b.score[0], a.score[0]), cmp.Compare(b.score[1], a.score[1]), cmp.Compare(a.id, b.id))
}

// applySpeciesRule replaces the defaults of one species' groups.
func applySpeciesRule(r SpeciesRule, species []*VariantGroup, members map[Variant][]member) {
	if r.Favorite {
		for _, g := range species {
			g.Rule, g.Minimum, g.Keep = RuleFavorite, g.Count, g.Count
		}
		return
	}
	for _, g := range species {
		g.Rule, g.Keep = RuleSpecies, 1
	}
	for left := r.Minimum - len(species); left > 0; left-- {
		var best *VariantGroup
		for _, g := range species {
			if g.Keep < g.Count && (best == nil || members[g.Variant][g.Keep].compare(members[best.Variant][best.Keep]) < 0) {
				best = g
			}
		}
		if best == nil {
			break
		}
		best.Keep++
	}
	for _, g := range species {
		g.Minimum = g.Keep
	}
}

// variantOf returns the variant if every variant feature is known.
func variantOf(s Snapshot) (Variant, bool) {
	v := Variant{Species: s.String(FieldSpecies), Form: formOf(s)}
	flags := []*bool{&v.Shiny, &v.Costume, &v.Background, &v.Dynamax, &v.Gigantamax}
	for i, f := range variantFields {
		switch s.Tri(f) {
		case Yes:
			*flags[i] = true
		case No:
		default:
			return Variant{}, false
		}
	}
	// The export only establishes these as present, so anything else groups
	// with the ordinary copies, exactly as §5.3 alone would.
	v.Lucky, v.Shadow, v.Purified = s.Tri(FieldLucky) == Yes, s.Tri(FieldShadow) == Yes, s.Tri(FieldPurified) == Yes
	return v, true
}

// rank orders copies of a variant: IV total, then CP. Unknown sorts last.
func rank(s Snapshot) [2]int {
	ivs := -1
	if iv, ok := s.IVs(); ok {
		ivs = iv.Attack + iv.Defense + iv.Stamina
	}
	cp, ok := s.Int(FieldCP)
	if !ok {
		cp = -1
	}
	return [2]int{ivs, cp}
}
