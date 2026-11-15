package sqlite_test

import (
	"errors"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

var at = time.Date(2026, 9, 5, 22, 30, 0, 0, time.UTC)

func withTrainers(t *testing.T, store *sqlite.Store, ids ...domain.TrainerID) {
	t.Helper()
	for _, id := range ids {
		if err := sqlite.NewTrainers(store).Create(t.Context(), domain.Trainer{ID: id, DisplayName: string(id), PINHash: "h"}); err != nil {
			t.Fatal(err)
		}
	}
}

func stagedBatch() domain.ImportBatch {
	return domain.ImportBatch{
		ID: "B1", TrainerID: "ash", Status: domain.ImportStaged, NewCount: 1, CreatedAt: at,
		Parsed: domain.ParsedExport{
			Source: "poke-genie",
			Rows: []domain.ImportRow{{Line: 2, Observations: []domain.Observation{
				{Field: domain.FieldSpecies, Value: "Pidgey", Source: domain.SourceCSV, SourceRef: "poke-genie line 2", ObservedAt: at},
			}}},
			Problems:       []domain.ImportProblem{{Line: 3, Message: "Skipped", Skipped: true}},
			IgnoredColumns: []string{"Dust"},
			UnknownFields:  []domain.Field{domain.FieldShiny},
		},
	}
}

func TestImportBatches_SaveAndGetForTheOwningTrainerOnly(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	coll := sqlite.NewCollection(store)
	b := stagedBatch()

	if err := coll.Save(t.Context(), b); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := coll.Get(t.Context(), "ash", "B1")
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.Status != domain.ImportStaged || len(got.Parsed.Rows) != 1 || got.Parsed.Problems[0].Line != 3 ||
		got.Parsed.Rows[0].Observations[0].Value != "Pidgey" || !got.CreatedAt.Equal(at) {
		t.Fatalf("Get = %+v", got)
	}
	if _, err := coll.Get(t.Context(), "misty", "B1"); !errors.Is(err, domain.ErrImportNotFound) {
		t.Fatalf("other trainer: err = %v, want ErrImportNotFound", err)
	}
}

func TestImportBatches_CommitAddsPokemonWithObservations(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	coll := sqlite.NewCollection(store)
	b := stagedBatch()
	if err := coll.Save(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	b.Status, b.CommittedAt = domain.ImportCommitted, at.Add(time.Minute)
	pidgey := domain.Pokemon{ID: "P1", TrainerID: "ash", Observations: b.Parsed.Rows[0].Observations}

	if err := coll.Commit(t.Context(), b, []domain.Pokemon{pidgey}); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	list, err := coll.ListPokemon(t.Context(), "ash")
	if err != nil || len(list) != 1 || list[0].ID != "P1" || list[0].Current().String(domain.FieldSpecies) != "Pidgey" {
		t.Fatalf("ListPokemon = %+v, %v", list, err)
	}
	if obs := list[0].Observations[0]; obs.Source != domain.SourceCSV || obs.SourceRef != "poke-genie line 2" || !obs.ObservedAt.Equal(at) {
		t.Fatalf("observation = %+v", obs)
	}
	if got, _ := coll.Get(t.Context(), "ash", "B1"); got.Status != domain.ImportCommitted {
		t.Fatalf("status = %s, want committed", got.Status)
	}
	if other, _ := coll.ListPokemon(t.Context(), "misty"); len(other) != 0 {
		t.Fatalf("misty sees %d Pokémon", len(other))
	}
}

func TestImportBatches_CommitIsAllOrNothing(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash")
	coll := sqlite.NewCollection(store)
	b := stagedBatch()
	if err := coll.Save(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	b.Status = domain.ImportCommitted
	dup := domain.Pokemon{ID: "P1", TrainerID: "ash"}

	if err := coll.Commit(t.Context(), b, []domain.Pokemon{dup, dup}); err == nil {
		t.Fatal("Commit with duplicate ids succeeded")
	}

	if list, _ := coll.ListPokemon(t.Context(), "ash"); len(list) != 0 {
		t.Fatalf("%d Pokémon left after failed commit", len(list))
	}
	if got, _ := coll.Get(t.Context(), "ash", "B1"); got.Status != domain.ImportStaged {
		t.Fatalf("status = %s after failed commit, want staged", got.Status)
	}
}
