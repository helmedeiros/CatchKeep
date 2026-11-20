package csvimport

import (
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// variantFields stay unknown unless a column is mapped to them.
var variantFields = []domain.Field{
	domain.FieldShiny, domain.FieldCostume, domain.FieldBackground, domain.FieldDynamax, domain.FieldGigantamax,
}

// Generic reads any CSV file as a table and parses it with the trainer's
// column mapping (spec V1-IMP-02).
type Generic struct{}

// ReadTable reads the header and every line as text. A file needs at
// least two columns and one line under the header to be a table.
func (Generic) ReadTable(r io.Reader) (domain.Table, error) {
	reader := csv.NewReader(skipBOM(r))
	reader.FieldsPerRecord = -1
	reader.LazyQuotes = true
	records, err := reader.ReadAll()
	if err != nil || len(records) < 2 || len(records[0]) < 2 || strings.TrimSpace(strings.Join(records[0], "")) == "" {
		return domain.Table{}, domain.ErrUnrecognizedExport
	}
	header := make([]string, len(records[0]))
	for i, h := range records[0] {
		header[i] = strings.TrimSpace(h)
	}
	return domain.Table{Header: header, Records: records[1:]}, nil
}

// ParseTable turns mapped columns into observations made at observedAt.
// Blank lines are skipped quietly. Unlike Poke Genie's flags, a mapped
// yes/no column can establish that a feature is absent: the trainer said
// what the column holds.
func (Generic) ParseTable(t domain.Table, m domain.ColumnMapping, observedAt time.Time) domain.ParsedExport {
	out := domain.ParsedExport{Source: domain.MappedSource}
	type column struct {
		index int
		name  string
		field domain.Field
	}
	var columns []column
	for i, h := range t.Header {
		if f, ok := m[h]; ok {
			columns = append(columns, column{i, h, f})
		} else if h != "" {
			out.IgnoredColumns = append(out.IgnoredColumns, h)
		}
	}
	for _, f := range variantFields {
		if !slices.ContainsFunc(columns, func(c column) bool { return c.field == f }) {
			out.UnknownFields = append(out.UnknownFields, f)
		}
	}

	for i, record := range t.Records {
		line := i + 2
		get := func(c column) string {
			if c.index < len(record) {
				return strings.TrimSpace(record[c.index])
			}
			return ""
		}
		if strings.TrimSpace(strings.Join(record, "")) == "" {
			continue
		}
		species := ""
		for _, c := range columns {
			if c.field == domain.FieldSpecies {
				species = get(c)
			}
		}
		if species == "" {
			out.Problems = append(out.Problems, domain.ImportProblem{Line: line, Message: "Skipped: the row has no species.", Skipped: true})
			continue
		}
		row := domain.ImportRow{Line: line}
		hasForm := false
		for _, c := range columns {
			raw := get(c)
			if raw == "" {
				continue
			}
			value, ok := mappedValue(c.field, raw)
			if !ok {
				out.Problems = append(out.Problems, domain.ImportProblem{
					Line: line, Message: fmt.Sprintf("Kept; %s %q is not a valid %s, so it is left unknown.", c.name, raw, fieldLabel(c.field)),
				})
				continue
			}
			hasForm = hasForm || c.field == domain.FieldForm
			row.Observations = append(row.Observations, domain.Observation{
				Field: c.field, Value: value, Source: domain.SourceCSV,
				SourceRef: fmt.Sprintf("%s line %d", domain.MappedSource, line), ObservedAt: observedAt,
			})
		}
		if !hasForm {
			row.Observations = append(row.Observations, domain.Observation{
				Field: domain.FieldForm, Value: "Normal", Source: domain.SourceCSV,
				SourceRef: fmt.Sprintf("%s line %d", domain.MappedSource, line), ObservedAt: observedAt,
			})
		}
		out.Rows = append(out.Rows, row)
	}
	return out
}

func mappedValue(f domain.Field, raw string) (string, bool) {
	switch f {
	case domain.FieldSpecies:
		return raw, true
	case domain.FieldForm:
		return normalizeForm(raw), true
	case domain.FieldDexNumber:
		return intBetween(raw, 1, 2000)
	case domain.FieldCP:
		return intBetween(raw, 0, 10000)
	case domain.FieldHP:
		return intBetween(raw, 0, 1000)
	case domain.FieldIVAttack, domain.FieldIVDefense, domain.FieldIVStamina:
		return intBetween(raw, 0, 15)
	case domain.FieldLevel:
		lvl, err := strconv.ParseFloat(raw, 64)
		if err != nil || lvl < 1 || lvl > 51 {
			return "", false
		}
		return strconv.FormatFloat(lvl, 'f', -1, 64), true
	case domain.FieldGender:
		switch strings.ToLower(raw) {
		case "m", "male", "♂":
			return "male", true
		case "f", "female", "♀":
			return "female", true
		}
		return "", false
	case domain.FieldCatchDate:
		for _, layout := range []string{time.DateOnly, "1/2/2006", "2006/01/02"} {
			if d, err := time.Parse(layout, raw); err == nil {
				return d.Format(time.DateOnly), true
			}
		}
		return "", false
	default:
		switch strings.ToLower(raw) {
		case "1", "y", "yes", "true", "x":
			return "true", true
		case "0", "n", "no", "false":
			return "false", true
		}
		return "", false
	}
}

func intBetween(raw string, lo, hi int) (string, bool) {
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		return "", false
	}
	return strconv.Itoa(n), true
}

func fieldLabel(f domain.Field) string {
	switch f {
	case domain.FieldIVAttack, domain.FieldIVDefense, domain.FieldIVStamina:
		return "IV (0 to 15)"
	case domain.FieldCatchDate:
		return "date"
	case domain.FieldGender:
		return "gender"
	case domain.FieldLevel:
		return "level"
	case domain.FieldCP, domain.FieldHP, domain.FieldDexNumber:
		return "number"
	}
	return "yes or no"
}
