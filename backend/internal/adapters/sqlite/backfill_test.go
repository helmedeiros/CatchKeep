package sqlite_test

import (
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestPastImports_RebuildEachImportFromWhatItSaved(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash")
	coll := sqlite.NewCollection(store)
	ctx := t.Context()
	obs := func(at time.Time, kv ...string) []domain.Observation {
		var out []domain.Observation
		for i := 0; i+1 < len(kv); i += 2 {
			out = append(out, domain.Observation{Field: domain.Field(kv[i]), Value: kv[i+1], Source: domain.SourceCSV, SourceRef: "line", ObservedAt: at})
		}
		return out
	}
	first, second := at, at.Add(5*24*time.Hour)
	commit := func(id string, committed time.Time, add []domain.Pokemon, update []domain.PokemonUpdate) {
		b := domain.ImportBatch{ID: id, TrainerID: "ash", Status: domain.ImportStaged, CreatedAt: committed}
		if err := coll.Save(ctx, b); err != nil {
			t.Fatal(err)
		}
		b.Status, b.CommittedAt = domain.ImportCommitted, committed
		if err := coll.Commit(ctx, b, add, update, imported(b)); err != nil {
			t.Fatal(err)
		}
	}
	commit("B1", first, []domain.Pokemon{{ID: "P1", TrainerID: "ash", Observations: obs(first, "species", "Pidgey", "cp", "120")}}, nil)
	commit("B2", second,
		[]domain.Pokemon{{ID: "P2", TrainerID: "ash", Observations: obs(second, "species", "Zubat", "cp", "80")}},
		[]domain.PokemonUpdate{{PokemonID: "P1", Observations: obs(second, "species", "Pidgey", "cp", "300")}})
	if _, err := store.DB().ExecContext(ctx, `DELETE FROM history_events`); err != nil {
		t.Fatal(err)
	}
	rev, _ := coll.Revision(ctx, "ash")

	past, err := coll.UnrecordedImports(ctx)
	if err != nil || len(past) != 2 || past[0].BatchID != "B1" || !past[1].CommittedAt.Equal(second) {
		t.Fatalf("past = %+v, %v; want B1 then B2", past, err)
	}
	e := past[1].Event("E2")
	if len(e.Added) != 1 || e.Added[0].Species != "Zubat" || len(e.Changed) != 1 || e.Changed[0].ID != "P1" || e.Changed[0].CP != 300 {
		t.Fatalf("B2 rebuilt as %+v; it added Zubat and powered up Pidgey", e)
	}
	if one := past[0].Event("E1"); len(one.Added) != 1 || one.Added[0].CP != 120 || len(one.Changed) != 0 {
		t.Fatalf("B1 rebuilt as %+v; Pidgey as it was then", one)
	}

	for i, p := range past {
		if err := coll.RecordImport(ctx, p.Event([]string{"E1", "E2"}[i])); err != nil {
			t.Fatal(err)
		}
	}
	if again, err := coll.UnrecordedImports(ctx); err != nil || len(again) != 0 {
		t.Fatalf("after recording: %+v, %v", again, err)
	}
	var backfilled int
	if err := store.DB().QueryRowContext(ctx, `SELECT count(*) FROM history_events WHERE json_extract(detail, '$.backfilled')`).Scan(&backfilled); err != nil || backfilled != 2 {
		t.Fatalf("backfilled entries = %d, %v", backfilled, err)
	}
	if now, _ := coll.Revision(ctx, "ash"); now != rev {
		t.Fatalf("revision %d → %d; filling in history changes no Pokémon", rev, now)
	}
}
