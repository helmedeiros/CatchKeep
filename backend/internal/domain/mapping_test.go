package domain_test

import (
	"errors"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func table(header ...string) domain.Table {
	return domain.Table{Header: header, Records: [][]string{
		{"Pidgey", "120", "", "15"},
		{"", "", "", ""},
		{"Rattata", "90", "x", "3"},
		{"Eevee", "300", "y", "10"},
		{"Mew", "1500", "z", "12"},
	}}
}

func TestColumnMapping_Validate(t *testing.T) {
	tbl := table("Pokemon", "CP", "Note", "Attack")
	for name, m := range map[string]domain.ColumnMapping{
		"no species":       {"CP": domain.FieldCP},
		"unknown column":   {"Pokemon": domain.FieldSpecies, "Weight": domain.FieldCP},
		"field twice":      {"Pokemon": domain.FieldSpecies, "CP": domain.FieldCP, "Attack": domain.FieldCP},
		"unmappable field": {"Pokemon": domain.FieldSpecies, "Note": "nickname"},
	} {
		if err := m.Validate(tbl); !errors.Is(err, domain.ErrInvalidMapping) {
			t.Errorf("%s: err = %v, want ErrInvalidMapping", name, err)
		}
	}
	dup := table("Pokemon", "CP", "CP", "Attack")
	if err := (domain.ColumnMapping{"Pokemon": domain.FieldSpecies, "CP": domain.FieldCP}).Validate(dup); !errors.Is(err, domain.ErrInvalidMapping) {
		t.Errorf("a column name used twice in the file: err = %v, want ErrInvalidMapping", err)
	}
	if err := (domain.ColumnMapping{"Pokemon": domain.FieldSpecies, "Attack": domain.FieldIVAttack}).Validate(tbl); err != nil {
		t.Fatalf("valid mapping rejected: %v", err)
	}
}

func TestTable_SignatureIgnoresCaseSpacingAndRows(t *testing.T) {
	a := table("Pokemon", "CP", "Note", "Attack")
	b := domain.Table{Header: []string{" pokemon", "cp ", "NOTE", "attack"}}
	if a.Signature() != b.Signature() {
		t.Fatal("same columns, different case or spacing, should share a signature")
	}
	if a.Signature() == table("Pokemon", "CP", "Attack", "Note").Signature() {
		t.Fatal("different column order should not share a signature")
	}
}

func TestTable_SamplesSkipBlanks(t *testing.T) {
	got := table("Pokemon", "CP", "Note", "Attack").Samples("Note", 2)
	if len(got) != 2 || got[0] != "x" || got[1] != "y" {
		t.Fatalf("samples = %q, want the first two non-blank values", got)
	}
}

func TestSuggestMapping_RecognizesCommonColumnNames(t *testing.T) {
	got := domain.SuggestMapping([]string{"Pokémon", "Dex #", "CP", "Atk IV", "Defense", "STA", "Lvl", "Shiny", "Notes", "Name"})
	want := domain.ColumnMapping{
		"Pokémon": domain.FieldSpecies, "Dex #": domain.FieldDexNumber, "CP": domain.FieldCP,
		"Atk IV": domain.FieldIVAttack, "Defense": domain.FieldIVDefense, "STA": domain.FieldIVStamina,
		"Lvl": domain.FieldLevel, "Shiny": domain.FieldShiny,
	}
	if len(got) != len(want) {
		t.Fatalf("suggested %v, want %v", got, want)
	}
	for col, f := range want {
		if got[col] != f {
			t.Errorf("%q suggested as %q, want %q", col, got[col], f)
		}
	}
}

func TestImportBatch_NeedsMappingUntilMapped(t *testing.T) {
	tbl := table("Pokemon", "CP", "Note", "Attack")
	b := domain.ImportBatch{Table: &tbl}
	if !b.NeedsMapping() {
		t.Fatal("an unmapped table needs mapping")
	}
	b.Mapping = domain.ColumnMapping{"Pokemon": domain.FieldSpecies}
	if b.NeedsMapping() {
		t.Fatal("a mapped table is ready")
	}
	if (domain.ImportBatch{}).NeedsMapping() {
		t.Fatal("a recognized export needs no mapping")
	}
}

func TestColumnMapping_RebindMatchesColumnsIgnoringCaseAndSpacing(t *testing.T) {
	saved := domain.ColumnMapping{"Pokemon": domain.FieldSpecies, "CP": domain.FieldCP, "Gone": domain.FieldHP}
	got := saved.Rebind(domain.Table{Header: []string{"pokemon ", " cp", "Note"}})
	if len(got) != 2 || got["pokemon"] != domain.FieldSpecies || got["cp"] != domain.FieldCP {
		t.Fatalf("rebound = %v", got)
	}
}
