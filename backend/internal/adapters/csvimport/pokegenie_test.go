package csvimport_test

import (
	"errors"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/csvimport"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func parseFixture(t *testing.T) domain.ParsedExport {
	t.Helper()
	f, err := os.Open("testdata/poke_genie_sample.csv")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	parsed, err := csvimport.NewPokeGenie(time.UTC).Parse(f)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return parsed
}

func current(row domain.ImportRow) domain.Snapshot {
	return domain.Pokemon{Observations: row.Observations}.Current()
}

func TestPokeGenie_ReadsEveryIdentifiableRow(t *testing.T) {
	parsed := parseFixture(t)

	if len(parsed.Rows) != 4 {
		t.Fatalf("rows = %d, want 4", len(parsed.Rows))
	}
	squirtle := current(parsed.Rows[0])
	if squirtle.String(domain.FieldSpecies) != "Squirtle" || squirtle.String(domain.FieldForm) != "Normal" ||
		squirtle.String(domain.FieldGender) != "male" || squirtle.String(domain.FieldCatchDate) != "2026-09-05" {
		t.Fatalf("squirtle = %v", squirtle)
	}
	if cp, _ := squirtle.Int(domain.FieldCP); cp != 539 {
		t.Fatalf("cp = %d", cp)
	}
	if iv, ok := squirtle.IVs(); !ok || iv != (domain.IVs{Attack: 13, Defense: 9, Stamina: 0}) {
		t.Fatalf("ivs = %+v %v", iv, ok)
	}
	if parsed.Rows[0].Line != 2 || parsed.Rows[0].Observations[0].Source != domain.SourceCSV {
		t.Fatalf("row meta = line %d source %s", parsed.Rows[0].Line, parsed.Rows[0].Observations[0].Source)
	}
	want := time.Date(2026, 9, 5, 22, 30, 0, 0, time.UTC)
	if at := parsed.Rows[0].Observations[0].ObservedAt; !at.Equal(want) {
		t.Fatalf("observed at %v, want scan date %v", at, want)
	}
}

func TestPokeGenie_FlagsOnlyEstablishYesWhileZeroStaysUnknown(t *testing.T) {
	parsed := parseFixture(t)
	squirtle, charizard, vulpix := current(parsed.Rows[0]), current(parsed.Rows[1]), current(parsed.Rows[2])

	if charizard.Tri(domain.FieldLucky) != domain.Yes || charizard.Tri(domain.FieldFavorite) != domain.Yes {
		t.Fatalf("charizard lucky/favorite = %v/%v, want yes", charizard.Tri(domain.FieldLucky), charizard.Tri(domain.FieldFavorite))
	}
	if vulpix.Tri(domain.FieldShadow) != domain.Yes {
		t.Fatal("vulpix shadow not yes")
	}
	for _, f := range []domain.Field{domain.FieldLucky, domain.FieldFavorite, domain.FieldShadow, domain.FieldShiny} {
		if squirtle.Tri(f) != domain.Unknown {
			t.Fatalf("squirtle %s = %v, want Unknown (0 is not proof of absence)", f, squirtle.Tri(f))
		}
	}
}

func TestPokeGenie_NormalizesFormsAndKeepsBlankCatchDateUnknown(t *testing.T) {
	parsed := parseFixture(t)
	vulpix := current(parsed.Rows[2])

	if vulpix.String(domain.FieldForm) != "Alola" || vulpix.String(domain.FieldGender) != "female" {
		t.Fatalf("vulpix = %v", vulpix)
	}
	if _, known := vulpix[domain.FieldCatchDate]; known {
		t.Fatal("blank catch date recorded as a value")
	}
}

func TestPokeGenie_OutOfRangeValueIsLeftUnknownAndReported(t *testing.T) {
	parsed := parseFixture(t)
	mewtwo := current(parsed.Rows[3])

	if _, known := mewtwo[domain.FieldIVAttack]; known {
		t.Fatal("IV 16 recorded")
	}
	if !slices.ContainsFunc(parsed.Problems, func(p domain.ImportProblem) bool {
		return p.Line == 5 && !p.Skipped && strings.Contains(p.Message, "Atk IV")
	}) {
		t.Fatalf("problems = %+v, want a kept-row note for line 5", parsed.Problems)
	}
}

func TestPokeGenie_SkipsRowsThatDoNotIdentifyAPokemon(t *testing.T) {
	parsed := parseFixture(t)

	if !slices.ContainsFunc(parsed.Problems, func(p domain.ImportProblem) bool { return p.Line == 6 && p.Skipped }) {
		t.Fatalf("problems = %+v, want line 6 skipped", parsed.Problems)
	}
}

func TestPokeGenie_ReportsIgnoredColumnsAndFieldsItCannotEstablish(t *testing.T) {
	parsed := parseFixture(t)

	for _, col := range []string{"IV Avg", "Dust", "Rank % (G)"} {
		if !slices.Contains(parsed.IgnoredColumns, col) {
			t.Fatalf("ignored columns %v missing %q", parsed.IgnoredColumns, col)
		}
	}
	for _, f := range []domain.Field{domain.FieldShiny, domain.FieldCostume, domain.FieldBackground, domain.FieldDynamax, domain.FieldGigantamax} {
		if !slices.Contains(parsed.UnknownFields, f) {
			t.Fatalf("unknown fields %v missing %s", parsed.UnknownFields, f)
		}
	}
}

func TestPokeGenie_RejectsFilesWithoutTheExpectedHeader(t *testing.T) {
	_, err := csvimport.NewPokeGenie(time.UTC).Parse(strings.NewReader("name,cp\nPikachu,500\n"))

	if !errors.Is(err, domain.ErrUnrecognizedExport) {
		t.Fatalf("err = %v, want ErrUnrecognizedExport", err)
	}
}
