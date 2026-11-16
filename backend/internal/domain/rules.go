package domain

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
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

// Setup is step 1 of setup: storage capacity and goals in priority order.
type Setup struct {
	Capacity int
	Goals    []Goal
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
	return nil
}

func (s Setup) has(g Goal) bool { return slices.Contains(s.Goals, g) }

type capacityBand struct{ UpTo, Ordinary, Rare int }

// capacityBands are placeholder defaults until OD-01 is decided: up to
// UpTo slots, keep Ordinary copies of each variant, Rare of rare species
// (spec V1-RUL-02).
var capacityBands = []capacityBand{
	{UpTo: 1000, Ordinary: 1, Rare: 2},
	{UpTo: 4000, Ordinary: 2, Rare: 4},
	{UpTo: maxCapacity, Ordinary: 3, Rare: 6},
}

func bandFor(capacity int) capacityBand {
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

// VariantGroup is the trainer's copies of one variant and what the default
// rule keeps.
type VariantGroup struct {
	Variant
	// Rare is set for legendary, mythical and Ultra Beast species.
	Rare                 bool
	Count, Minimum, Keep int
	// KeptIDs are the best copies, SurplusIDs the rest, best first.
	KeptIDs, SurplusIDs []PokemonID
}

// KeepSummary is what the default keep rules do with a collection:
// Kept + Surplus + NeedsConfirmation = Total.
type KeepSummary struct {
	Total, Kept, Surplus, NeedsConfirmation int
	// Minimum and RareMinimum are the per-variant minimums for the setup's
	// capacity.
	Minimum, RareMinimum int
	// Groups are sorted by surplus, largest first.
	Groups []VariantGroup
}

// SummarizeKeeps applies the default rules (spec V1-RUL-01..03, V1-RUL-06):
// keep the best Minimum copies of each variant (RareMinimum for rare
// species), one more of rare, shiny, costume or background variants when
// trading is a goal. Pokémon with an unknown
// variant feature could belong to an under-filled variant, so they are kept
// and counted as needing confirmation (spec §7.3).
func SummarizeKeeps(pokemon []Pokemon, setup Setup, rare Rarity) KeepSummary {
	band := bandFor(setup.Capacity)
	out := KeepSummary{Total: len(pokemon), Minimum: band.Ordinary, RareMinimum: band.Rare}
	type member struct {
		id    PokemonID
		score [2]int
	}
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
		groups[v] = append(groups[v], member{p.ID, rank(cur)})
	}
	for _, v := range order {
		members := groups[v]
		slices.SortStableFunc(members, func(a, b member) int {
			return cmp.Or(cmp.Compare(b.score[0], a.score[0]), cmp.Compare(b.score[1], a.score[1]), cmp.Compare(a.id, b.id))
		})
		g := VariantGroup{Variant: v, Rare: rare != nil && rare(v.Species), Count: len(members), Minimum: out.Minimum}
		if g.Rare {
			g.Minimum = out.RareMinimum
		}
		if setup.has(GoalTrading) && (g.Rare || v.special()) {
			g.Minimum++
		}
		g.Keep = min(g.Minimum, g.Count)
		for i, m := range members {
			if i < g.Keep {
				g.KeptIDs = append(g.KeptIDs, m.id)
			} else {
				g.SurplusIDs = append(g.SurplusIDs, m.id)
			}
		}
		out.Kept += g.Keep
		out.Surplus += len(g.SurplusIDs)
		out.Groups = append(out.Groups, g)
	}
	slices.SortStableFunc(out.Groups, func(a, b VariantGroup) int {
		return cmp.Or(cmp.Compare(len(b.SurplusIDs), len(a.SurplusIDs)), cmp.Compare(a.Species, b.Species))
	})
	return out
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
