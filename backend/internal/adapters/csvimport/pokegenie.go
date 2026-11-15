// Package csvimport reads collection exports. PokeGenie follows the column
// rules in spec §6.5.
package csvimport

import (
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

const sourceName = "poke-genie"

// Columns the importer reads. Everything else is reported as ignored.
const (
	colName      = "Name"
	colForm      = "Form"
	colDex       = "Pokemon Number"
	colGender    = "Gender"
	colCP        = "CP"
	colHP        = "HP"
	colAtk       = "Atk IV"
	colDef       = "Def IV"
	colSta       = "Sta IV"
	colLevelMin  = "Level Min"
	colScanDate  = "Scan Date"
	colCatchDate = "Catch Date"
	colLucky     = "Lucky"
	colShaPur    = "Shadow/Purified"
	colFavorite  = "Favorite"
	colIndex     = "Index"
)

var requiredColumns = []string{colName, colDex, colCP, colHP, colAtk, colDef, colSta, colScanDate}

var usedColumns = map[string]bool{
	colIndex: true, colName: true, colForm: true, colDex: true, colGender: true, colCP: true, colHP: true,
	colAtk: true, colDef: true, colSta: true, colLevelMin: true, colScanDate: true, colCatchDate: true,
	colLucky: true, colShaPur: true, colFavorite: true,
}

// The export has no columns for these; they stay unknown (spec V1-IMP-04).
var cannotEstablish = []domain.Field{
	domain.FieldShiny, domain.FieldCostume, domain.FieldBackground, domain.FieldDynamax, domain.FieldGigantamax,
}

// PokeGenie parses Poke Genie CSV exports.
type PokeGenie struct {
	loc *time.Location
}

// NewPokeGenie returns a parser that reads scan times in loc (the export
// writes local times without a zone).
func NewPokeGenie(loc *time.Location) PokeGenie {
	return PokeGenie{loc: loc}
}

// Parse reads the whole export.
func (p PokeGenie) Parse(r io.Reader) (domain.ParsedExport, error) {
	reader := csv.NewReader(skipBOM(r))
	reader.FieldsPerRecord = -1
	header, err := reader.Read()
	if err != nil {
		return domain.ParsedExport{}, domain.ErrUnrecognizedExport
	}
	index := map[string]int{}
	for i, name := range header {
		index[strings.TrimSpace(name)] = i
	}
	for _, c := range requiredColumns {
		if _, ok := index[c]; !ok {
			return domain.ParsedExport{}, domain.ErrUnrecognizedExport
		}
	}

	out := domain.ParsedExport{Source: sourceName, UnknownFields: cannotEstablish}
	for _, name := range header {
		if n := strings.TrimSpace(name); !usedColumns[n] {
			out.IgnoredColumns = append(out.IgnoredColumns, n)
		}
	}

	for line := 2; ; line++ {
		record, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			out.Problems = append(out.Problems, domain.ImportProblem{Line: line, Message: "Unreadable line: " + err.Error(), Skipped: true})
			continue
		}
		get := func(col string) string {
			if i, ok := index[col]; ok && i < len(record) {
				return strings.TrimSpace(record[i])
			}
			return ""
		}
		row, problems := p.parseRow(line, get)
		out.Problems = append(out.Problems, problems...)
		if row != nil {
			out.Rows = append(out.Rows, *row)
		}
	}
	return out, nil
}

func (p PokeGenie) parseRow(line int, get func(string) string) (*domain.ImportRow, []domain.ImportProblem) {
	skip := func(msg string) (*domain.ImportRow, []domain.ImportProblem) {
		return nil, []domain.ImportProblem{{Line: line, Message: msg, Skipped: true}}
	}
	species := get(colName)
	dex, err := strconv.Atoi(get(colDex))
	if species == "" || err != nil || dex < 1 {
		return skip("Skipped: the row has no valid species name and Pokédex number.")
	}
	observedAt, err := time.ParseInLocation("2006-01-02 15:04", get(colScanDate), p.loc)
	if err != nil {
		return skip(fmt.Sprintf("Skipped: scan date %q is not in the expected format.", get(colScanDate)))
	}

	var (
		obs      []domain.Observation
		problems []domain.ImportProblem
	)
	add := func(f domain.Field, v string) {
		obs = append(obs, domain.Observation{
			Field: f, Value: v, Source: domain.SourceCSV,
			SourceRef: fmt.Sprintf("%s line %d", sourceName, line), ObservedAt: observedAt,
		})
	}
	note := func(format string, args ...any) {
		problems = append(problems, domain.ImportProblem{Line: line, Message: "Kept; " + fmt.Sprintf(format, args...)})
	}
	intIn := func(f domain.Field, col string, maxValue int) {
		raw := get(col)
		if raw == "" {
			return
		}
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 || n > maxValue {
			note("%s %q is not between 0 and %d, so it is left unknown.", col, raw, maxValue)
			return
		}
		add(f, strconv.Itoa(n))
	}

	add(domain.FieldSpecies, species)
	add(domain.FieldDexNumber, strconv.Itoa(dex))
	add(domain.FieldForm, normalizeForm(get(colForm)))
	switch get(colGender) {
	case "♂":
		add(domain.FieldGender, "male")
	case "♀":
		add(domain.FieldGender, "female")
	}
	intIn(domain.FieldCP, colCP, 10000)
	intIn(domain.FieldHP, colHP, 1000)
	intIn(domain.FieldIVAttack, colAtk, 15)
	intIn(domain.FieldIVDefense, colDef, 15)
	intIn(domain.FieldIVStamina, colSta, 15)
	if raw := get(colLevelMin); raw != "" {
		if lvl, err := strconv.ParseFloat(raw, 64); err == nil && lvl >= 1 && lvl <= 51 {
			add(domain.FieldLevel, strconv.FormatFloat(lvl, 'f', -1, 64))
		} else {
			note("level %q is not valid, so it is left unknown.", raw)
		}
	}
	if raw := get(colCatchDate); raw != "" {
		if d, err := time.Parse("1/2/2006", raw); err == nil {
			add(domain.FieldCatchDate, d.Format(time.DateOnly))
		} else {
			note("catch date %q is not M/D/YYYY, so it is left unknown.", raw)
		}
	}
	// "1" establishes the feature; "0" may only mean "not captured" (spec §6.5).
	if get(colLucky) == "1" {
		add(domain.FieldLucky, "true")
	}
	if get(colFavorite) == "1" {
		add(domain.FieldFavorite, "true")
	}
	if get(colShaPur) == "1" {
		add(domain.FieldShadow, "true")
	}
	return &domain.ImportRow{Line: line, Observations: obs}, problems
}

func normalizeForm(raw string) string {
	if raw == "" || strings.EqualFold(raw, "Normal") {
		return "Normal"
	}
	return raw
}

func skipBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	if b, err := br.Peek(3); err == nil && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		_, _ = br.Discard(3)
	}
	return br
}
