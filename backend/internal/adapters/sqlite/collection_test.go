package sqlite_test

import (
	"errors"
	"strings"
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

// imported is an empty history entry for committing b.
func imported(b domain.ImportBatch) domain.ImportEvent {
	return domain.ImportEvent{ID: "I-" + b.ID, TrainerID: b.TrainerID, BatchID: b.ID, At: at}
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

	if err := coll.Commit(t.Context(), b, []domain.Pokemon{pidgey}, nil, imported(b)); err != nil {
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

	if err := coll.Commit(t.Context(), b, []domain.Pokemon{dup, dup}, nil, imported(b)); err == nil {
		t.Fatal("Commit with duplicate ids succeeded")
	}

	if list, _ := coll.ListPokemon(t.Context(), "ash"); len(list) != 0 {
		t.Fatalf("%d Pokémon left after failed commit", len(list))
	}
	if got, _ := coll.Get(t.Context(), "ash", "B1"); got.Status != domain.ImportStaged {
		t.Fatalf("status = %s after failed commit, want staged", got.Status)
	}
}

func TestImportBatches_SaveResolutionsKeepsTheAnswers(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash")
	coll := sqlite.NewCollection(store)
	b := stagedBatch()
	if err := coll.Save(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	b.Resolutions = map[int]domain.Resolution{2: {Line: 2, SameAs: "P1"}, 5: {Line: 5}}

	if err := coll.SaveResolutions(t.Context(), b); err != nil {
		t.Fatalf("SaveResolutions: %v", err)
	}

	got, _ := coll.Get(t.Context(), "ash", "B1")
	if len(got.Resolutions) != 2 || got.Resolutions[2].SameAs != "P1" || got.Resolutions[5].SameAs != "" {
		t.Fatalf("Resolutions = %+v", got.Resolutions)
	}
}

func TestImportBatches_CommitAddsObservationsToExistingPokemon(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	coll := sqlite.NewCollection(store)
	first := stagedBatch()
	if err := coll.Save(t.Context(), first); err != nil {
		t.Fatal(err)
	}
	first.Status, first.CommittedAt = domain.ImportCommitted, at
	pidgey := domain.Pokemon{ID: "P1", TrainerID: "ash", Observations: first.Parsed.Rows[0].Observations}
	if err := coll.Commit(t.Context(), first, []domain.Pokemon{pidgey}, nil, imported(first)); err != nil {
		t.Fatal(err)
	}
	second := stagedBatch()
	second.ID = "B2"
	if err := coll.Save(t.Context(), second); err != nil {
		t.Fatal(err)
	}
	second.Status, second.CommittedAt, second.UpdatedCount = domain.ImportCommitted, at.Add(time.Hour), 1
	later := []domain.Observation{{Field: domain.FieldCP, Value: "150", Source: domain.SourceCSV, SourceRef: "poke-genie line 2", ObservedAt: at.Add(time.Hour)}}

	if err := coll.Commit(t.Context(), second, nil, []domain.PokemonUpdate{{PokemonID: "P1", Observations: later}}, imported(second)); err != nil {
		t.Fatalf("Commit: %v", err)
	}

	list, _ := coll.ListPokemon(t.Context(), "ash")
	if cp, _ := list[0].Current().Int(domain.FieldCP); len(list) != 1 || cp != 150 {
		t.Fatalf("collection = %+v", list)
	}
	if got, _ := coll.Get(t.Context(), "ash", "B2"); got.UpdatedCount != 1 {
		t.Fatalf("UpdatedCount = %d, want 1", got.UpdatedCount)
	}
}

func TestImportBatches_CommitRefusesToUpdateAnotherTrainersPokemon(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	coll := sqlite.NewCollection(store)
	mine := stagedBatch()
	if err := coll.Save(t.Context(), mine); err != nil {
		t.Fatal(err)
	}
	mine.Status = domain.ImportCommitted
	if err := coll.Commit(t.Context(), mine, []domain.Pokemon{{ID: "P1", TrainerID: "ash"}}, nil, imported(mine)); err != nil {
		t.Fatal(err)
	}
	theirs := stagedBatch()
	theirs.ID, theirs.TrainerID = "B2", "misty"
	if err := coll.Save(t.Context(), theirs); err != nil {
		t.Fatal(err)
	}
	theirs.Status = domain.ImportCommitted

	update := []domain.PokemonUpdate{{PokemonID: "P1", Observations: []domain.Observation{{Field: domain.FieldCP, Value: "1", Source: domain.SourceCSV, ObservedAt: at}}}}
	if err := coll.Commit(t.Context(), theirs, nil, update, imported(theirs)); err == nil {
		t.Fatal("updated another trainer's Pokémon")
	}

	if got, _ := coll.Get(t.Context(), "misty", "B2"); got.Status != domain.ImportStaged {
		t.Fatalf("status = %s after refused commit, want staged", got.Status)
	}
}

func TestCollection_AddObservationsIsAllOrNothingAndScopedToTheTrainer(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	coll := sqlite.NewCollection(store)
	b := stagedBatch()
	if err := coll.Save(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	b.Status = domain.ImportCommitted
	if err := coll.Commit(t.Context(), b, []domain.Pokemon{{ID: "P1", TrainerID: "ash", Observations: b.Parsed.Rows[0].Observations}}, nil, imported(b)); err != nil {
		t.Fatal(err)
	}
	notShiny := []domain.Observation{{Field: domain.FieldShiny, Value: "false", Source: domain.SourceManual, SourceRef: "bulk confirmation", ObservedAt: at}}

	if err := coll.AddObservations(t.Context(), "misty", []domain.PokemonUpdate{{PokemonID: "P1", Observations: notShiny}}); err == nil {
		t.Fatal("misty changed ash's Pokémon")
	}
	err := coll.AddObservations(t.Context(), "ash", []domain.PokemonUpdate{{PokemonID: "P1", Observations: notShiny}, {PokemonID: "P9", Observations: notShiny}})
	if err == nil {
		t.Fatal("update naming a missing Pokémon succeeded")
	}
	if list, _ := coll.ListPokemon(t.Context(), "ash"); list[0].Current().Tri(domain.FieldShiny) != domain.Unknown {
		t.Fatal("failed update left a change behind")
	}

	if err := coll.AddObservations(t.Context(), "ash", []domain.PokemonUpdate{{PokemonID: "P1", Observations: notShiny}}); err != nil {
		t.Fatalf("AddObservations: %v", err)
	}
	list, _ := coll.ListPokemon(t.Context(), "ash")
	if o := list[0].Observations[len(list[0].Observations)-1]; list[0].Current().Tri(domain.FieldShiny) != domain.No || o.Source != domain.SourceManual {
		t.Fatalf("observations = %+v", list[0].Observations)
	}
}

func TestCollection_SetProtectedIsScopedToTheTrainer(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	coll := sqlite.NewCollection(store)
	b := stagedBatch()
	if err := coll.Save(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	b.Status = domain.ImportCommitted
	if err := coll.Commit(t.Context(), b, []domain.Pokemon{{ID: "P1", TrainerID: "ash", Observations: b.Parsed.Rows[0].Observations}}, nil, imported(b)); err != nil {
		t.Fatal(err)
	}

	if err := coll.SetProtected(t.Context(), "misty", "P1", true, protecting("misty", "C0")); !errors.Is(err, domain.ErrPokemonNotFound) {
		t.Fatalf("misty protecting ash's Pokémon: err = %v", err)
	}
	if err := coll.SetProtected(t.Context(), "ash", "P1", true, protecting("ash", "C1")); err != nil {
		t.Fatal(err)
	}
	if list, _ := coll.ListPokemon(t.Context(), "ash"); !list[0].Protected {
		t.Fatal("P1 is not protected")
	}
	if err := coll.SetProtected(t.Context(), "ash", "P1", false, protecting("ash", "C2")); err != nil {
		t.Fatal(err)
	}
	if list, _ := coll.ListPokemon(t.Context(), "ash"); list[0].Protected {
		t.Fatal("P1 is still protected")
	}
	var kinds, details string
	if err := store.DB().QueryRowContext(t.Context(), `
		SELECT group_concat(kind), group_concat(detail, ' ') FROM (SELECT kind, detail FROM history_events WHERE kind = 'protection' ORDER BY rowid)`).Scan(&kinds, &details); err != nil {
		t.Fatal(err)
	}
	if kinds != "protection,protection" || details != `{"protected":true} {"protected":false}` {
		t.Fatalf("history = %s %s; misty's refused attempt is not recorded", kinds, details)
	}
}

func protecting(trainer domain.TrainerID, id string) domain.Change {
	return domain.Change{ID: id, TrainerID: trainer, Kind: domain.ChangeProtection, Pokemon: []domain.Transferred{{ID: "P1", Species: "Pidgey"}}, At: at}
}

func TestImportBatches_KeepTheTableAndRememberMappingsPerTrainer(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	coll := sqlite.NewCollection(store)
	tbl := domain.Table{Header: []string{"Mon", "CP"}, Records: [][]string{{"Pidgey", "120"}}}
	b := domain.ImportBatch{ID: "B1", TrainerID: "ash", Status: domain.ImportStaged, CreatedAt: at, Table: &tbl}
	if err := coll.Save(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	got, err := coll.Get(t.Context(), "ash", "B1")
	if err != nil || !got.NeedsMapping() || got.Table.Records[0][1] != "120" {
		t.Fatalf("Get = %+v, %v; the table waits for a mapping", got, err)
	}

	b.Mapping = domain.ColumnMapping{"Mon": domain.FieldSpecies}
	b.Parsed = stagedBatch().Parsed
	b.NewCount = 1
	b.Resolutions = map[int]domain.Resolution{2: {Line: 2}}
	if err := coll.SaveMapping(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	got, _ = coll.Get(t.Context(), "ash", "B1")
	if got.NeedsMapping() || got.Mapping["Mon"] != domain.FieldSpecies || len(got.Parsed.Rows) != 1 || got.NewCount != 1 || len(got.Resolutions) != 1 {
		t.Fatalf("Get = %+v", got)
	}
	if m, err := coll.SavedMapping(t.Context(), "ash", tbl.Signature()); err != nil || m["Mon"] != domain.FieldSpecies {
		t.Fatalf("saved mapping = %v, %v", m, err)
	}
	if m, _ := coll.SavedMapping(t.Context(), "misty", tbl.Signature()); m != nil {
		t.Fatalf("misty sees ash's mapping %v", m)
	}

	b.Mapping = domain.ColumnMapping{"Mon": domain.FieldSpecies, "CP": domain.FieldCP}
	if err := coll.SaveMapping(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	if m, _ := coll.SavedMapping(t.Context(), "ash", tbl.Signature()); m["CP"] != domain.FieldCP {
		t.Fatalf("a newer mapping replaces the saved one: %v", m)
	}
}

func batchAt(id string, trainer domain.TrainerID, status domain.ImportStatus, created time.Time) domain.ImportBatch {
	b := stagedBatch()
	b.ID, b.TrainerID, b.Status, b.CreatedAt = id, trainer, status, created
	return b
}

func saveBatches(t *testing.T, coll *sqlite.Collection, batches ...domain.ImportBatch) {
	t.Helper()
	for _, b := range batches {
		if err := coll.Save(t.Context(), b); err != nil {
			t.Fatal(err)
		}
	}
}

func TestImportBatches_DeleteStagedBeforeLeavesCommittedAndRecentOnes(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash")
	coll := sqlite.NewCollection(store)
	saveBatches(t, coll,
		batchAt("old", "ash", domain.ImportStaged, at),
		batchAt("done", "ash", domain.ImportCommitted, at),
		batchAt("recent", "ash", domain.ImportStaged, at.Add(24*time.Hour+time.Millisecond)))

	n, err := coll.DeleteStagedBefore(t.Context(), at.Add(24*time.Hour))
	if err != nil || n != 1 {
		t.Fatalf("DeleteStagedBefore = %d, %v; want 1", n, err)
	}
	if _, err := coll.Get(t.Context(), "ash", "old"); !errors.Is(err, domain.ErrImportNotFound) {
		t.Fatalf("old staged import: err = %v, want removed", err)
	}
	for _, id := range []string{"done", "recent"} {
		if _, err := coll.Get(t.Context(), "ash", id); err != nil {
			t.Fatalf("%s: %v, want kept", id, err)
		}
	}
}

func TestImportBatches_TrimStagedKeepsTheTrainersNewest(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	coll := sqlite.NewCollection(store)
	saveBatches(t, coll,
		batchAt("first", "ash", domain.ImportStaged, at.Add(500*time.Millisecond)),
		batchAt("second", "ash", domain.ImportStaged, at.Add(time.Second)),
		batchAt("third", "ash", domain.ImportStaged, at.Add(2*time.Second)),
		batchAt("done", "ash", domain.ImportCommitted, at),
		batchAt("hers", "misty", domain.ImportStaged, at))

	if err := coll.TrimStaged(t.Context(), "ash", 2); err != nil {
		t.Fatal(err)
	}
	if _, err := coll.Get(t.Context(), "ash", "first"); !errors.Is(err, domain.ErrImportNotFound) {
		t.Fatalf("oldest staged import: err = %v, want removed", err)
	}
	for _, b := range []struct {
		trainer domain.TrainerID
		id      string
	}{{"ash", "second"}, {"ash", "third"}, {"ash", "done"}, {"misty", "hers"}} {
		if _, err := coll.Get(t.Context(), b.trainer, b.id); err != nil {
			t.Fatalf("%s: %v, want kept", b.id, err)
		}
	}
}

func TestImportBatches_CommitRecordsTheImportInTheHistory(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash")
	coll := sqlite.NewCollection(store)
	b := stagedBatch()
	saveBatches(t, coll, b)
	e := imported(b)
	e.Added = []domain.Transferred{{ID: "P1", Species: "Pidgey", CP: 120, HasCP: true}}
	e.Changed = []domain.ChangedPokemon{{Transferred: domain.Transferred{ID: "P0", Species: "Zubat"}, Fields: []domain.Field{domain.FieldCP}}}
	b.Status, b.CommittedAt = domain.ImportCommitted, at

	if err := coll.Commit(t.Context(), b, []domain.Pokemon{{ID: "P1", TrainerID: "ash"}}, nil, e); err != nil {
		t.Fatal(err)
	}

	var kind, pokemon, detail string
	if err := store.DB().QueryRowContext(t.Context(), `SELECT kind, pokemon, detail FROM history_events WHERE id = ?`, e.ID).Scan(&kind, &pokemon, &detail); err != nil {
		t.Fatal(err)
	}
	if kind != "import" || !strings.Contains(pokemon, `"Pidgey"`) || !strings.Contains(detail, `"batchId":"B1"`) || !strings.Contains(detail, `"fields":["cp"]`) {
		t.Fatalf("history row = %s %s %s", kind, pokemon, detail)
	}
	transfers, err := sqlite.NewTransfers(store).ListTransfers(t.Context(), "ash", 10)
	if err != nil || len(transfers) != 0 {
		t.Fatalf("transfers = %+v, %v; an import is not a transfer", transfers, err)
	}
}
