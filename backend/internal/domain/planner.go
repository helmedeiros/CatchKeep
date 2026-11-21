package domain

import (
	"cmp"
	"errors"
	"fmt"
	"slices"
)

// Planner errors.
var (
	ErrInvalidTarget = errors.New("invalid cleanup target")
	// ErrInvalidPlan means a plan broke a keep rule; it is a bug, never shown.
	ErrInvalidPlan = errors.New("plan breaks the keep rules")
)

// defaultLimit is the share of storage a trainer aims to stay under.
const defaultLimit = 80

// LimitPercent is the share of storage the trainer wants to stay under.
func (s Setup) LimitPercent() int {
	if s.Limit == 0 {
		return defaultLimit
	}
	return s.Limit
}

// Use is how full storage is (spec §7 target arithmetic).
type Use struct {
	Capacity, Occupied, Free int
	// Estimated is set when Occupied counts imported Pokémon because the
	// trainer gave no in-game count.
	Estimated bool
	// Limit is the number of slots the trainer wants to stay under.
	Limit int
}

// StorageUse works out storage use from the setup, preferring the count
// the trainer reads in the game over the number imported.
func StorageUse(s Setup, imported int) Use {
	u := Use{Capacity: s.Capacity, Occupied: s.InGameCount, Limit: s.Capacity * s.LimitPercent() / 100}
	if u.Occupied == 0 {
		u.Occupied, u.Estimated = imported, true
	}
	u.Free = max(0, u.Capacity-u.Occupied)
	return u
}

// TargetKind is how the trainer says how much room they want.
type TargetKind string

// Targets (spec V1-CLN-01).
const (
	// TargetLimit gets back under the trainer's storage limit.
	TargetLimit TargetKind = "limit"
	// TargetFreeMore frees Value more slots, e.g. before an event.
	TargetFreeMore TargetKind = "free_more"
	// TargetReachFree reaches Value free slots in total.
	TargetReachFree TargetKind = "reach_free"
	// TargetPercent frees Value percent of the Pokémon in storage.
	TargetPercent TargetKind = "percent"
)

// Target is how much room the trainer wants.
type Target struct {
	Kind  TargetKind
	Value int
}

// Slots is how many Pokémon to transfer to reach the target.
func (t Target) Slots(u Use) (int, error) {
	switch {
	case t.Kind == TargetLimit:
		return max(0, u.Occupied-u.Limit), nil
	case t.Value < 1 || t.Value > maxCapacity:
		return 0, fmt.Errorf("%w: choose a number from 1 to %d", ErrInvalidTarget, maxCapacity)
	case t.Kind == TargetFreeMore:
		return t.Value, nil
	case t.Kind == TargetReachFree:
		return max(0, t.Value-u.Free), nil
	case t.Kind == TargetPercent && t.Value <= 100:
		return (t.Value*u.Occupied + 99) / 100, nil
	}
	return 0, fmt.Errorf("%w: %q %d", ErrInvalidTarget, t.Kind, t.Value)
}

// PlanItem is one suggested transfer and why it can go.
type PlanItem struct {
	ID PokemonID
	// KeptInGroup is how many copies of its variant stay; Rule set them.
	KeptInGroup int
	Rule        Rule
	Favorite    TriState
}

// Plan is the suggested cleanup (spec §7.5).
type Plan struct {
	Use    Use
	Target Target
	// Wanted is how many to transfer for the target; MaxFeasible how many
	// the keep rules allow; Shortfall what is still missing.
	Wanted, MaxFeasible, Shortfall int
	Transfers                      []PlanItem
	// Searches find the transfers in the game, one per species.
	Searches []SpeciesSearch
	// Favorites are suggestions starred in the game; FavoritesUnknown
	// those whose star the export could not tell.
	Favorites, FavoritesUnknown int
}

// PlanTransfers suggests the least valuable possible transfers, up to the
// target (spec §7.4): lowest IV total, then CP, then oldest catch, then ID.
// imported is the number of active Pokémon in the app.
func PlanTransfers(pokemon []Pokemon, summary KeepSummary, setup Setup, target Target) (Plan, error) {
	use := StorageUse(setup, len(pokemon))
	wanted, err := target.Slots(use)
	if err != nil {
		return Plan{}, err
	}
	byID := make(map[PokemonID]Snapshot, len(pokemon))
	for _, p := range pokemon {
		byID[p.ID] = p.Current()
	}
	var items []PlanItem
	for _, g := range summary.Groups {
		for _, id := range g.SurplusIDs {
			items = append(items, PlanItem{ID: id, KeptInGroup: len(g.KeptIDs), Rule: g.Rule, Favorite: byID[id].Tri(FieldFavorite)})
		}
	}
	slices.SortStableFunc(items, func(a, b PlanItem) int {
		x, y := byID[a.ID], byID[b.ID]
		rx, ry := rank(x), rank(y)
		return cmp.Or(cmp.Compare(rx[0], ry[0]), cmp.Compare(rx[1], ry[1]),
			cmp.Compare(catchOrder(x), catchOrder(y)), cmp.Compare(a.ID, b.ID))
	})
	plan := Plan{Use: use, Target: target, Wanted: wanted, MaxFeasible: len(items)}
	plan.Transfers = items[:min(wanted, len(items))]
	plan.Shortfall = wanted - len(plan.Transfers)
	for _, it := range plan.Transfers {
		switch it.Favorite {
		case Yes:
			plan.Favorites++
		case Unknown:
			plan.FavoritesUnknown++
		}
	}
	if err := VerifyPlan(pokemon, summary, plan.Transfers); err != nil {
		return Plan{}, err
	}
	ids := make([]PokemonID, len(plan.Transfers))
	for i, it := range plan.Transfers {
		ids[i] = it.ID
	}
	plan.Searches = Searches(pokemon, ids)
	return plan, nil
}

// catchOrder sorts older catches first and unknown dates last.
func catchOrder(s Snapshot) string {
	if d := s.String(FieldCatchDate); d != "" {
		return d
	}
	return "9999-99-99"
}

// VerifyPlan checks a plan independently of how it was chosen: each
// Pokémon is the trainer's, has a known variant, is not protected, is not
// one the rules keep, appears once, and every group still keeps
// min(minimum, count).
func VerifyPlan(pokemon []Pokemon, summary KeepSummary, items []PlanItem) error {
	byID := make(map[PokemonID]Pokemon, len(pokemon))
	for _, p := range pokemon {
		byID[p.ID] = p
	}
	planned := map[PokemonID]bool{}
	for _, it := range items {
		p, ok := byID[it.ID]
		switch {
		case !ok:
			return fmt.Errorf("%w: %s is not in the collection", ErrInvalidPlan, it.ID)
		case planned[it.ID]:
			return fmt.Errorf("%w: %s is planned twice", ErrInvalidPlan, it.ID)
		case p.Protected:
			return fmt.Errorf("%w: %s is protected", ErrInvalidPlan, it.ID)
		}
		if _, known := variantOf(p.Current()); !known {
			return fmt.Errorf("%w: %s has unconfirmed features", ErrInvalidPlan, it.ID)
		}
		planned[it.ID] = true
	}
	inGroup := map[PokemonID]bool{}
	for _, g := range summary.Groups {
		gone := 0
		for _, id := range g.KeptIDs {
			if planned[id] {
				return fmt.Errorf("%w: %s is kept by the rules", ErrInvalidPlan, id)
			}
			inGroup[id] = true
		}
		for _, id := range g.SurplusIDs {
			if planned[id] {
				gone++
			}
			inGroup[id] = true
		}
		if g.Count-gone < min(g.Minimum, g.Count) {
			return fmt.Errorf("%w: %s would keep %d of %d", ErrInvalidPlan, g.Species, g.Count-gone, g.Minimum)
		}
	}
	for id := range planned {
		if !inGroup[id] {
			return fmt.Errorf("%w: %s is in no keep group", ErrInvalidPlan, id)
		}
	}
	return nil
}
