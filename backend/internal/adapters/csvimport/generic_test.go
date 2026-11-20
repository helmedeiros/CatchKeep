package csvimport_test

import (
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/csvimport"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

const spreadsheet = "\xEF\xBB\xBFPokémon,Dex,CP,Attack,Defense,Stamina,Level,Shiny,Gender,Caught,Notes\n" +
	"Pidgey,16,120,15,14,13,20,no,F,2026-10-01,buddy\n" +
	",,,,,,,,,,\n" +
	"Rattata,19,abc,3,20,1,99,yes,m,10/2/2026,\n" +
	",19,90,1,1,1,10,no,,,\n" +
	"Mew,151,1500,15,15,15,30,maybe,,someday,\n"

var mapping = domain.ColumnMapping{
	"Pokémon": domain.FieldSpecies, "Dex": domain.FieldDexNumber, "CP": domain.FieldCP,
	"Attack": domain.FieldIVAttack, "Defense": domain.FieldIVDefense, "Stamina": domain.FieldIVStamina,
	"Level": domain.FieldLevel, "Shiny": domain.FieldShiny, "Gender": domain.FieldGender, "Caught": domain.FieldCatchDate,
}

func readSpreadsheet(t *testing.T) domain.Table {
	t.Helper()
	tbl, err := csvimport.Generic{}.ReadTable(strings.NewReader(spreadsheet))
	if err != nil {
		t.Fatal(err)
	}
	return tbl
}

func TestGeneric_ReadTableKeepsTheHeaderAndEveryLine(t *testing.T) {
	tbl := readSpreadsheet(t)
	if tbl.Header[0] != "Pokémon" || len(tbl.Header) != 11 || len(tbl.Records) != 5 {
		t.Fatalf("table = %q with %d records", tbl.Header, len(tbl.Records))
	}
	if _, err := (csvimport.Generic{}).ReadTable(strings.NewReader("")); !errors.Is(err, domain.ErrUnrecognizedExport) {
		t.Fatalf("empty file: err = %v", err)
	}
}

func TestGeneric_ParseTableReadsMappedColumns(t *testing.T) {
	at := time.Date(2026, 11, 20, 9, 0, 0, 0, time.UTC)
	parsed := csvimport.Generic{}.ParseTable(readSpreadsheet(t), mapping, at)

	if len(parsed.Rows) != 3 {
		t.Fatalf("rows = %d, want Pidgey, Rattata and Mew", len(parsed.Rows))
	}
	pidgey := current(parsed.Rows[0])
	if iv, ok := pidgey.IVs(); !ok || iv.Attack != 15 || iv.Defense != 14 || iv.Stamina != 13 {
		t.Errorf("Pidgey IVs = %+v", iv)
	}
	if pidgey.Tri(domain.FieldShiny) != domain.No || pidgey.String(domain.FieldGender) != "female" ||
		pidgey.String(domain.FieldCatchDate) != "2026-10-01" || pidgey.String(domain.FieldForm) != "Normal" {
		t.Errorf("Pidgey = %+v; a mapped shiny column can say no", pidgey)
	}
	if parsed.Rows[0].Line != 2 || parsed.Rows[0].Observations[0].ObservedAt != at {
		t.Errorf("row = %+v; lines count the header, observed at import time", parsed.Rows[0])
	}

	rattata := current(parsed.Rows[1])
	if _, ok := rattata.Int(domain.FieldCP); ok || rattata.Tri(domain.FieldShiny) != domain.Yes || rattata.String(domain.FieldGender) != "male" ||
		rattata.String(domain.FieldCatchDate) != "2026-10-02" {
		t.Errorf("Rattata = %+v; bad CP left unknown, shiny yes, M/D/YYYY dates", rattata)
	}
	if mew := current(parsed.Rows[2]); mew.Tri(domain.FieldShiny) != domain.Unknown {
		t.Errorf("an unreadable yes/no stays unknown")
	}

	var lines []int
	for _, p := range parsed.Problems {
		lines = append(lines, p.Line)
	}
	if !slices.Equal(lines, []int{4, 4, 4, 5, 6, 6}) {
		t.Errorf("problems on lines %v: Rattata CP, defense and level; the row without species; Mew shiny and date", lines)
	}
	if !slices.Equal(parsed.IgnoredColumns, []string{"Notes"}) {
		t.Errorf("ignored = %q", parsed.IgnoredColumns)
	}
	if slices.Contains(parsed.UnknownFields, domain.FieldShiny) || !slices.Contains(parsed.UnknownFields, domain.FieldCostume) {
		t.Errorf("unknown fields = %q; only unmapped variant features stay unknown", parsed.UnknownFields)
	}
}
