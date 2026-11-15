package app_test

import (
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/app"
	"github.com/helmedeiros/catchkeep/backend/internal/app/apptest"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// fakeParser turns lines of "species,atk,def,sta" into rows.
type fakeParser struct{}

func (fakeParser) Parse(r io.Reader) (domain.ParsedExport, error) {
	b, _ := io.ReadAll(r)
	if strings.HasPrefix(string(b), "garbage") {
		return domain.ParsedExport{}, domain.ErrUnrecognizedExport
	}
	out := domain.ParsedExport{Source: "poke-genie", IgnoredColumns: []string{"Dust"}, UnknownFields: []domain.Field{domain.FieldShiny}}
	for i, line := range strings.Split(strings.TrimSpace(string(b)), "\n") {
		parts := strings.Split(line, ",")
		if parts[0] == "" {
			out.Problems = append(out.Problems, domain.ImportProblem{Line: i + 2, Message: "Skipped", Skipped: true})
			continue
		}
		var obs []domain.Observation
		for j, f := range []domain.Field{domain.FieldSpecies, domain.FieldIVAttack, domain.FieldIVDefense, domain.FieldIVStamina, domain.FieldCP} {
			if j < len(parts) {
				obs = append(obs, domain.Observation{Field: f, Value: parts[j], Source: domain.SourceCSV})
			}
		}
		obs = append(obs, domain.Observation{Field: domain.FieldForm, Value: "Normal", Source: domain.SourceCSV})
		out.Rows = append(out.Rows, domain.ImportRow{Line: i + 2, Observations: obs})
	}
	return out, nil
}

type importsFixture struct {
	imports    *app.Imports
	collection *app.Collection
}

func newImportsFixture() importsFixture {
	store := apptest.NewCollectionStore()
	clock := apptest.NewFixedClock(time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC))
	ids := apptest.NewSequentialIDs()
	return importsFixture{
		imports:    app.NewImports(fakeParser{}, store, store, ids, clock),
		collection: app.NewCollection(store),
	}
}

const ash = domain.TrainerID("ash")

func TestStage_PreviewsWithoutChangingTheCollection(t *testing.T) {
	f := newImportsFixture()

	preview, err := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nRattata,4,5,6,200\n,,,\n"))
	if err != nil {
		t.Fatalf("Stage: %v", err)
	}

	if preview.Status != app.ImportStaged || preview.TotalRows != 3 || preview.NewCount != 2 || preview.PossibleMatchCount != 0 {
		t.Fatalf("preview = %+v", preview)
	}
	if len(preview.Problems) != 1 || len(preview.IgnoredColumns) != 1 || len(preview.UnknownFields) != 1 {
		t.Fatalf("preview details = %+v", preview)
	}
	if list, _ := f.collection.List(t.Context(), ash); len(list) != 0 {
		t.Fatalf("collection changed before commit: %d", len(list))
	}
}

func TestCommit_AddsNewPokemonStrongestFirst(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nRattata,4,5,6,200\n"))

	committed, err := f.imports.Commit(t.Context(), ash, preview.ID)
	if err != nil {
		t.Fatalf("Commit: %v", err)
	}

	if committed.Status != app.ImportCommitted || committed.NewCount != 2 {
		t.Fatalf("committed = %+v", committed)
	}
	list, _ := f.collection.List(t.Context(), ash)
	if len(list) != 2 || list[0].Current().String(domain.FieldSpecies) != "Rattata" {
		t.Fatalf("collection = %+v", list)
	}
}

func TestCommit_HoldsBackPossibleDuplicatesInsteadOfMergingThem(t *testing.T) {
	f := newImportsFixture()
	first, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\n"))
	if _, err := f.imports.Commit(t.Context(), ash, first.ID); err != nil {
		t.Fatal(err)
	}

	again, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,150\nRattata,4,5,6,200\nRattata,4,5,6,200\n"))
	if again.NewCount != 1 || again.PossibleMatchCount != 2 {
		t.Fatalf("re-import preview = %+v, want 1 new and 2 possible matches", again)
	}
	if _, err := f.imports.Commit(t.Context(), ash, again.ID); err != nil {
		t.Fatal(err)
	}

	if list, _ := f.collection.List(t.Context(), ash); len(list) != 2 {
		t.Fatalf("collection has %d Pokémon, want 2 (no duplicates added)", len(list))
	}
}

func TestCommit_TwiceIsRejected(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\n"))
	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}

	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); !errors.Is(err, domain.ErrImportAlreadyCommitted) {
		t.Fatalf("err = %v, want ErrImportAlreadyCommitted", err)
	}
}

func TestImports_AreScopedToTheTrainer(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\n"))

	if _, err := f.imports.Get(t.Context(), "misty", preview.ID); !errors.Is(err, domain.ErrImportNotFound) {
		t.Fatalf("other trainer Get: err = %v, want ErrImportNotFound", err)
	}
	if _, err := f.imports.Commit(t.Context(), "misty", preview.ID); !errors.Is(err, domain.ErrImportNotFound) {
		t.Fatalf("other trainer Commit: err = %v, want ErrImportNotFound", err)
	}
}

func TestStage_RejectsUnrecognizedFiles(t *testing.T) {
	f := newImportsFixture()

	if _, err := f.imports.Stage(t.Context(), ash, strings.NewReader("garbage")); !errors.Is(err, domain.ErrUnrecognizedExport) {
		t.Fatalf("err = %v", err)
	}
}

// reimportPoweredUp commits a Pidgey, then stages it again with more CP.
func reimportPoweredUp(t *testing.T, f importsFixture) app.ImportPreview {
	t.Helper()
	first, err := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.imports.Commit(t.Context(), ash, first.ID); err != nil {
		t.Fatal(err)
	}
	preview, err := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,150\n"))
	if err != nil {
		t.Fatal(err)
	}
	return preview
}

func TestStage_ListsPossibleDuplicatesWithTheirCandidates(t *testing.T) {
	f := newImportsFixture()

	preview := reimportPoweredUp(t, f)

	if len(preview.Matches) != 1 || preview.Matches[0].Row.Line != 2 || len(preview.Matches[0].Candidates) != 1 {
		t.Fatalf("matches = %+v", preview.Matches)
	}
}

func TestResolve_SameUpdatesTheExistingPokemonOnCommit(t *testing.T) {
	f := newImportsFixture()
	preview := reimportPoweredUp(t, f)
	existing := preview.Matches[0].Candidates[0].Ref

	resolved, err := f.imports.Resolve(t.Context(), ash, preview.ID, []domain.Resolution{{Line: 2, SameAs: existing}})
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if resolved.PossibleMatchCount != 0 || resolved.UpdatedCount != 1 || resolved.Matches[0].Resolution == nil {
		t.Fatalf("resolved = %+v", resolved)
	}
	if again, _ := f.imports.Get(t.Context(), ash, preview.ID); again.UpdatedCount != 1 {
		t.Fatalf("answer not kept: %+v", again)
	}

	committed, err := f.imports.Commit(t.Context(), ash, preview.ID)
	if err != nil {
		t.Fatal(err)
	}
	if committed.UpdatedCount != 1 || committed.NewCount != 0 || len(committed.Matches) != 0 {
		t.Fatalf("committed = %+v", committed)
	}
	list, _ := f.collection.List(t.Context(), ash)
	if len(list) != 1 || string(list[0].ID) != existing {
		t.Fatalf("collection = %+v, want only %s", list, existing)
	}
	if cp, _ := list[0].Current().Int(domain.FieldCP); cp != 150 {
		t.Fatalf("CP = %d, want the newer 150", cp)
	}
}

func TestResolve_DifferentAddsTheRowOnCommit(t *testing.T) {
	f := newImportsFixture()
	preview := reimportPoweredUp(t, f)

	if _, err := f.imports.Resolve(t.Context(), ash, preview.ID, []domain.Resolution{{Line: 2}}); err != nil {
		t.Fatal(err)
	}
	committed, err := f.imports.Commit(t.Context(), ash, preview.ID)
	if err != nil {
		t.Fatal(err)
	}

	if list, _ := f.collection.List(t.Context(), ash); committed.NewCount != 1 || len(list) != 2 {
		t.Fatalf("committed = %+v, collection size %d", committed, len(list))
	}
}

func TestResolve_RejectsInvalidAnswersAndCommittedImports(t *testing.T) {
	f := newImportsFixture()
	preview := reimportPoweredUp(t, f)

	if _, err := f.imports.Resolve(t.Context(), ash, preview.ID, []domain.Resolution{{Line: 2, SameAs: "nope"}}); !errors.Is(err, domain.ErrInvalidResolution) {
		t.Fatalf("invalid answer: err = %v", err)
	}
	if _, err := f.imports.Resolve(t.Context(), "misty", preview.ID, []domain.Resolution{{Line: 2}}); !errors.Is(err, domain.ErrImportNotFound) {
		t.Fatalf("other trainer: err = %v", err)
	}
	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := f.imports.Resolve(t.Context(), ash, preview.ID, []domain.Resolution{{Line: 2}}); !errors.Is(err, domain.ErrImportAlreadyCommitted) {
		t.Fatalf("committed import: err = %v", err)
	}
}
