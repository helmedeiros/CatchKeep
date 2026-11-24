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
	if strings.HasPrefix(string(b), "garbage") || strings.HasPrefix(strings.ToLower(string(b)), "mon,") {
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

// fakeTables reads any comma-separated text as a table; values pass
// through as observations of their mapped field.
type fakeTables struct{}

func (fakeTables) ReadTable(r io.Reader) (domain.Table, error) {
	b, _ := io.ReadAll(r)
	if strings.HasPrefix(string(b), "garbage") {
		return domain.Table{}, domain.ErrUnrecognizedExport
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	t := domain.Table{Header: strings.Split(lines[0], ",")}
	for _, l := range lines[1:] {
		t.Records = append(t.Records, strings.Split(l, ","))
	}
	return t, nil
}

func (fakeTables) ParseTable(t domain.Table, m domain.ColumnMapping, at time.Time) domain.ParsedExport {
	out := domain.ParsedExport{Source: "csv"}
	for i, r := range t.Records {
		obs := []domain.Observation{{Field: domain.FieldForm, Value: "Normal", Source: domain.SourceCSV, ObservedAt: at}}
		for j, h := range t.Header {
			if f, ok := m[h]; ok && j < len(r) {
				obs = append(obs, domain.Observation{Field: f, Value: r[j], Source: domain.SourceCSV, ObservedAt: at})
			}
		}
		out.Rows = append(out.Rows, domain.ImportRow{Line: i + 2, Observations: obs})
	}
	return out
}

type importsFixture struct {
	imports    *app.Imports
	collection *app.Collection
	store      *apptest.CollectionStore
}

func newImportsFixture() importsFixture {
	store := apptest.NewCollectionStore()
	clock := apptest.NewFixedClock(time.Date(2026, 11, 15, 12, 0, 0, 0, time.UTC))
	ids := apptest.NewSequentialIDs()
	return importsFixture{
		imports:    app.NewImports(fakeParser{}, fakeTables{}, store, store, ids, clock),
		collection: app.NewCollection(store, store, clock),
		store:      store,
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

func TestConfirmAbsent_SetsUnknownFeaturesOfOneSpeciesToNo(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nPidgey,4,5,6,120\nRattata,4,5,6,200\n"))
	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}

	n, err := f.collection.ConfirmAbsent(t.Context(), ash, "Pidgey", "Normal", []domain.Field{domain.FieldShiny})
	if err != nil {
		t.Fatalf("ConfirmAbsent: %v", err)
	}

	if n != 2 {
		t.Fatalf("updated %d, want 2", n)
	}
	list, _ := f.collection.List(t.Context(), ash)
	for _, p := range list {
		want := domain.No
		if p.Current().String(domain.FieldSpecies) == "Rattata" {
			want = domain.Unknown
		}
		if got := p.Current().Tri(domain.FieldShiny); got != want {
			t.Fatalf("%s shiny = %v, want %v", p.Current().String(domain.FieldSpecies), got, want)
		}
	}
	if again, _ := f.collection.ConfirmAbsent(t.Context(), ash, "Pidgey", "Normal", []domain.Field{domain.FieldShiny}); again != 0 {
		t.Fatalf("confirming again updated %d, want 0", again)
	}
	if _, err := f.collection.ConfirmAbsent(t.Context(), ash, "Pidgey", "Normal", []domain.Field{domain.FieldCP}); !errors.Is(err, domain.ErrNotConfirmable) {
		t.Fatalf("CP: err = %v", err)
	}
}

const spreadsheet = "Mon,Combat Power,Note\nPidgey,120,a\nRattata,90,b\n"

func TestStage_AsksToMapAFileItDoesNotRecognize(t *testing.T) {
	f := newImportsFixture()

	p, err := f.imports.Stage(t.Context(), ash, strings.NewReader(spreadsheet))
	if err != nil {
		t.Fatal(err)
	}
	if !p.NeedsMapping || p.TotalRows != 0 || len(p.Columns) != 3 || p.Columns[1].Name != "Combat Power" || p.Columns[1].Samples[0] != "120" {
		t.Fatalf("preview = %+v; the columns with samples, nothing read yet", p)
	}
	if p.Mapping["Combat Power"] != domain.FieldCP {
		t.Errorf("suggested mapping = %v", p.Mapping)
	}
	if _, err := f.imports.Commit(t.Context(), ash, p.ID); !errors.Is(err, domain.ErrImportNeedsMapping) {
		t.Fatalf("commit before mapping: err = %v", err)
	}
	if _, err := f.imports.Resolve(t.Context(), ash, p.ID, nil); !errors.Is(err, domain.ErrImportNeedsMapping) {
		t.Fatalf("answers before mapping: err = %v", err)
	}
	if list, _ := f.collection.List(t.Context(), ash); len(list) != 0 {
		t.Fatal("staging changed the collection")
	}
}

func TestMap_ReadsTheFileAndRemembersTheMappingForTheNextOne(t *testing.T) {
	f := newImportsFixture()
	p, _ := f.imports.Stage(t.Context(), ash, strings.NewReader(spreadsheet))
	mapping := domain.ColumnMapping{"Mon": domain.FieldSpecies, "Combat Power": domain.FieldCP}

	if _, err := f.imports.Map(t.Context(), ash, p.ID, domain.ColumnMapping{"Combat Power": domain.FieldCP}); !errors.Is(err, domain.ErrInvalidMapping) {
		t.Fatalf("mapping without species: err = %v", err)
	}
	mapped, err := f.imports.Map(t.Context(), ash, p.ID, mapping)
	if err != nil {
		t.Fatal(err)
	}
	if mapped.NeedsMapping || mapped.NewCount != 2 || mapped.Mapping["Mon"] != domain.FieldSpecies || len(mapped.Columns) != 3 {
		t.Fatalf("preview = %+v", mapped)
	}
	if _, err := f.imports.Commit(t.Context(), ash, p.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := f.collection.List(t.Context(), ash); len(list) != 2 {
		t.Fatalf("collection = %d Pokémon, want 2", len(list))
	}
	if _, err := f.imports.Map(t.Context(), ash, p.ID, mapping); !errors.Is(err, domain.ErrImportAlreadyCommitted) {
		t.Fatalf("mapping a committed import: err = %v", err)
	}

	next, err := f.imports.Stage(t.Context(), ash, strings.NewReader("mon, combat power ,NOTE\nEevee,300,c\n"))
	if err != nil {
		t.Fatal(err)
	}
	if next.NeedsMapping || next.NewCount != 1 {
		t.Fatalf("preview = %+v; the same columns reuse the saved mapping", next)
	}
	other, _ := f.imports.Stage(t.Context(), "misty", strings.NewReader(spreadsheet))
	if !other.NeedsMapping {
		t.Fatal("another trainer's saved mapping was applied")
	}
}

func TestMap_RejectsRecognizedExports(t *testing.T) {
	f := newImportsFixture()
	p, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\n"))
	if _, err := f.imports.Map(t.Context(), ash, p.ID, domain.ColumnMapping{"Mon": domain.FieldSpecies}); !errors.Is(err, domain.ErrInvalidMapping) {
		t.Fatalf("err = %v", err)
	}
}

func TestMark_SetsOnePokemonsFeaturesAndOnlyForItsTrainer(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Dialga,1,2,3,100\nDialga,4,5,6,120\n"))
	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}
	list, _ := f.collection.List(t.Context(), ash)
	target, other := list[0], list[1]

	got, err := f.collection.Mark(t.Context(), ash, target.ID, map[domain.Field]bool{domain.FieldPurified: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != target.ID || got.Current().Tri(domain.FieldPurified) != domain.Yes || got.Current().Tri(domain.FieldShadow) != domain.No {
		t.Fatalf("marked = %v", got.Current())
	}
	list, _ = f.collection.List(t.Context(), ash)
	for _, p := range list {
		if p.ID == other.ID && p.Current().Tri(domain.FieldPurified) != domain.Unknown {
			t.Fatal("the other Dialga changed too")
		}
	}
	if _, err := f.collection.Mark(t.Context(), "misty", target.ID, map[domain.Field]bool{domain.FieldShiny: true}); !errors.Is(err, domain.ErrPokemonNotFound) {
		t.Fatalf("another trainer: err = %v", err)
	}
	if _, err := f.collection.Mark(t.Context(), ash, target.ID, map[domain.Field]bool{domain.FieldCP: true}); !errors.Is(err, domain.ErrNotConfirmable) {
		t.Fatalf("not a feature: err = %v", err)
	}
}

func TestImports_ListCollectedPokemonMissingFromTheFile(t *testing.T) {
	f := newImportsFixture()
	first, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nZubat,7,8,9,50\n"))
	if _, err := f.imports.Commit(t.Context(), ash, first.ID); err != nil {
		t.Fatal(err)
	}
	list, _ := f.collection.List(t.Context(), ash)
	zubat := list[1].ID

	again, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,150\nRattata,4,5,6,200\n"))
	if len(again.Missing) != 1 || again.Missing[0] != zubat {
		t.Fatalf("staged missing = %v, want the Zubat %s", again.Missing, zubat)
	}
	committed, err := f.imports.Commit(t.Context(), ash, again.ID)
	if err != nil || len(committed.Missing) != 1 || committed.Missing[0] != zubat {
		t.Fatalf("committed missing = %v, %v; the trainer is offered to mark them after adding", committed.Missing, err)
	}
	if later, _ := f.imports.Get(t.Context(), ash, again.ID); len(later.Missing) != 0 {
		t.Fatalf("a committed import read later lists %v; the offer is made once", later.Missing)
	}
}

func TestStage_KeepsOnlyTheLatestUnfinishedImports(t *testing.T) {
	f := newImportsFixture()
	var ids []string
	for range domain.MaxStagedImports + 1 {
		p, err := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\n"))
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, p.ID)
	}

	if _, err := f.imports.Get(t.Context(), ash, ids[0]); !errors.Is(err, domain.ErrImportNotFound) {
		t.Fatalf("oldest unfinished import: err = %v, want removed", err)
	}
	if _, err := f.imports.Get(t.Context(), ash, ids[1]); err != nil {
		t.Fatalf("second import: %v, want kept", err)
	}
}

func TestCommit_RecordsTheImportInTheHistory(t *testing.T) {
	f := newImportsFixture()
	preview, _ := f.imports.Stage(t.Context(), ash, strings.NewReader("Pidgey,1,2,3,100\nRattata,4,5,6,200\n"))

	if _, err := f.imports.Commit(t.Context(), ash, preview.ID); err != nil {
		t.Fatal(err)
	}

	events := f.store.ImportEvents(ash)
	if len(events) != 1 || events[0].BatchID != preview.ID || len(events[0].Added) != 2 || events[0].At.IsZero() {
		t.Fatalf("history = %+v, want one import adding 2", events)
	}
}
