package domain_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestSetup_ValidatesInGameCountAndLimit(t *testing.T) {
	base := domain.Setup{Capacity: 4000, Goals: []domain.Goal{domain.GoalCollecting}}
	for name, s := range map[string]domain.Setup{
		"negative count":  {Capacity: 4000, Goals: base.Goals, InGameCount: -1},
		"absurd count":    {Capacity: 4000, Goals: base.Goals, InGameCount: 30000},
		"limit too low":   {Capacity: 4000, Goals: base.Goals, Limit: 10},
		"limit above 100": {Capacity: 4000, Goals: base.Goals, Limit: 101},
	} {
		if err := s.Validate(); !errors.Is(err, domain.ErrInvalidSetup) {
			t.Errorf("%s: err = %v", name, err)
		}
	}
	if got := base.LimitPercent(); got != 80 {
		t.Fatalf("default limit = %d%%, want 80%%", got)
	}
}

func TestTarget_Slots(t *testing.T) {
	setup := domain.Setup{Capacity: 4000, Goals: []domain.Goal{domain.GoalCollecting}, InGameCount: 3500}
	use := domain.StorageUse(setup, 484)
	if use.Occupied != 3500 || use.Estimated || use.Free != 500 || use.Limit != 3200 {
		t.Fatalf("use = %+v; the in-game count wins over the 484 imported", use)
	}
	for _, tc := range []struct {
		target domain.Target
		want   int
	}{
		{domain.Target{Kind: domain.TargetLimit}, 300},
		{domain.Target{Kind: domain.TargetFreeMore, Value: 400}, 400},
		{domain.Target{Kind: domain.TargetReachFree, Value: 800}, 300},
		{domain.Target{Kind: domain.TargetReachFree, Value: 200}, 0},
		{domain.Target{Kind: domain.TargetPercent, Value: 10}, 350},
	} {
		if got, err := tc.target.Slots(use); err != nil || got != tc.want {
			t.Errorf("%+v: slots = %d, %v; want %d", tc.target, got, err, tc.want)
		}
	}
	for _, bad := range []domain.Target{{Kind: "magic", Value: 1}, {Kind: domain.TargetFreeMore}, {Kind: domain.TargetPercent, Value: 101}} {
		if _, err := bad.Slots(use); !errors.Is(err, domain.ErrInvalidTarget) {
			t.Errorf("%+v: err = %v", bad, err)
		}
	}

	estimated := domain.StorageUse(domain.Setup{Capacity: 4000}, 484)
	if estimated.Occupied != 484 || !estimated.Estimated {
		t.Fatalf("use = %+v; without an in-game count, the import is an estimate", estimated)
	}
}

func planFixture() ([]domain.Pokemon, domain.KeepSummary, domain.Setup) {
	var pokemon []domain.Pokemon
	for i, iv := range []string{"15", "1", "10", "3", "12"} {
		pokemon = append(pokemon, known(fmt.Sprint("P", i), "Pidgey", iv, iv, iv))
	}
	fav := known("R1", "Rattata", "0", "0", "0")
	fav.Observations = append(fav.Observations, domain.Observation{Field: domain.FieldFavorite, Value: "true", Source: domain.SourceCSV})
	pokemon = append(pokemon, known("R0", "Rattata", "9", "9", "9"), fav)
	weak := known("P9", "Eevee", "0", "0", "0")
	weak.Protected = true
	pokemon = append(pokemon, weak, mon("U1", "species", "Pidgey", "form", "Normal"))
	setup := domain.Setup{Capacity: 350, Goals: []domain.Goal{domain.GoalCollecting}}
	return pokemon, domain.SummarizeKeeps(pokemon, setup, nil), setup
}

func TestPlanTransfers_LeastValuableFirstUpToTheTarget(t *testing.T) {
	pokemon, summary, setup := planFixture()

	plan, err := domain.PlanTransfers(pokemon, summary, setup, domain.Target{Kind: domain.TargetFreeMore, Value: 3})
	if err != nil {
		t.Fatal(err)
	}
	var got []domain.PokemonID
	for _, it := range plan.Transfers {
		got = append(got, it.ID)
	}
	if plan.Wanted != 3 || plan.MaxFeasible != 5 || plan.Shortfall != 0 || fmt.Sprint(got) != "[R1 P1 P3]" {
		t.Fatalf("plan = %+v, transfers %v; want the 0, 1 and 3 IV copies", plan, got)
	}
	if plan.Favorites != 1 || plan.Transfers[0].KeptInGroup != 1 {
		t.Fatalf("plan = %+v; the starred Rattata is counted, with 1 copy kept", plan)
	}

	all, _ := domain.PlanTransfers(pokemon, summary, setup, domain.Target{Kind: domain.TargetFreeMore, Value: 50})
	if len(all.Transfers) != 5 || all.Shortfall != 45 {
		t.Fatalf("plan = %+v; at most 5, 45 short", all)
	}
	if err := domain.VerifyPlan(pokemon, summary, all.Transfers); err != nil {
		t.Fatalf("a real plan failed verification: %v", err)
	}
}

func TestVerifyPlan_RejectsKeptProtectedAndUnconfirmedPokemon(t *testing.T) {
	pokemon, summary, _ := planFixture()
	for _, id := range []domain.PokemonID{"P0", "P9", "U1", "ghost"} {
		if err := domain.VerifyPlan(pokemon, summary, []domain.PlanItem{{ID: id}}); !errors.Is(err, domain.ErrInvalidPlan) {
			t.Errorf("%s: err = %v, want ErrInvalidPlan", id, err)
		}
	}
	if err := domain.VerifyPlan(pokemon, summary, []domain.PlanItem{{ID: "P1"}, {ID: "P1"}}); !errors.Is(err, domain.ErrInvalidPlan) {
		t.Errorf("the same Pokémon twice: err = %v", err)
	}
}

func TestPlanTransfers_IsStable(t *testing.T) {
	pokemon, summary, setup := planFixture()
	a, _ := domain.PlanTransfers(pokemon, summary, setup, domain.Target{Kind: domain.TargetFreeMore, Value: 5})
	for range 20 {
		b, _ := domain.PlanTransfers(pokemon, summary, setup, domain.Target{Kind: domain.TargetFreeMore, Value: 5})
		if fmt.Sprint(a.Transfers) != fmt.Sprint(b.Transfers) {
			t.Fatal("the same input gave different plans")
		}
	}
}

func BenchmarkPlanTransfers(b *testing.B) {
	for _, n := range []int{2700, 5400} {
		var pokemon []domain.Pokemon
		for i := range n {
			iv := fmt.Sprint(i % 16)
			pokemon = append(pokemon, known(fmt.Sprint("P", i), fmt.Sprint("Species", i%400), iv, iv, iv))
		}
		setup := domain.Setup{Capacity: 4000, Goals: []domain.Goal{domain.GoalCollecting}}
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			for b.Loop() {
				summary := domain.SummarizeKeeps(pokemon, setup, nil)
				if _, err := domain.PlanTransfers(pokemon, summary, setup, domain.Target{Kind: domain.TargetFreeMore, Value: 1000}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
