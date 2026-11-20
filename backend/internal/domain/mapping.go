package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"slices"
	"strings"
)

// Mapping errors.
var (
	ErrInvalidMapping     = errors.New("invalid column mapping")
	ErrImportNeedsMapping = errors.New("import needs a column mapping first")
)

// MappedSource names files read with a column mapping.
const MappedSource = "csv"

// Table is an export as read, column names and data rows, kept while the
// trainer maps its columns (spec V1-IMP-02). Record i is on line i+2.
type Table struct {
	Header  []string
	Records [][]string
}

// ColumnMapping says which field a column holds, by column name. Columns
// it leaves out are ignored.
type ColumnMapping map[string]Field

// MappableFields are the fields a column can be mapped to.
var MappableFields = []Field{
	FieldSpecies, FieldDexNumber, FieldForm, FieldGender, FieldCP, FieldHP, FieldLevel,
	FieldIVAttack, FieldIVDefense, FieldIVStamina, FieldCatchDate,
	FieldShiny, FieldLucky, FieldShadow, FieldPurified, FieldCostume, FieldBackground,
	FieldDynamax, FieldGigantamax, FieldFavorite,
}

// Signature identifies a table's columns regardless of case and spacing,
// so a saved mapping is reused for the next export of the same shape.
func (t Table) Signature() string {
	names := make([]string, len(t.Header))
	for i, h := range t.Header {
		names[i] = strings.ToLower(strings.TrimSpace(h))
	}
	sum := sha256.Sum256([]byte(strings.Join(names, "\n")))
	return hex.EncodeToString(sum[:])
}

// Column returns the index of a column, or -1 when it is missing or its
// name appears more than once.
func (t Table) Column(name string) int {
	at := -1
	for i, h := range t.Header {
		if strings.TrimSpace(h) == name {
			if at >= 0 {
				return -1
			}
			at = i
		}
	}
	return at
}

// Samples returns up to n non-blank values of a column, top first.
func (t Table) Samples(column string, n int) []string {
	i := t.Column(column)
	var out []string
	for _, r := range t.Records {
		if len(out) == n || i < 0 {
			break
		}
		if i < len(r) && strings.TrimSpace(r[i]) != "" {
			out = append(out, strings.TrimSpace(r[i]))
		}
	}
	return out
}

// Validate checks the mapping names each column once, maps species, and
// uses only mappable fields, each once.
func (m ColumnMapping) Validate(t Table) error {
	used := map[Field]string{}
	for col, f := range m {
		if t.Column(col) < 0 {
			return fmt.Errorf("%w: no single column named %q", ErrInvalidMapping, col)
		}
		if !slices.Contains(MappableFields, f) {
			return fmt.Errorf("%w: %q is not a field a column can hold", ErrInvalidMapping, f)
		}
		if other, ok := used[f]; ok {
			return fmt.Errorf("%w: %q and %q both map to %s", ErrInvalidMapping, other, col, f)
		}
		used[f] = col
	}
	if _, ok := used[FieldSpecies]; !ok {
		return fmt.Errorf("%w: choose the column with the species", ErrInvalidMapping)
	}
	return nil
}

// Rebind keys a saved mapping by this table's own column names, matching
// them regardless of case and spacing; columns the table lacks are dropped.
func (m ColumnMapping) Rebind(t Table) ColumnMapping {
	out := ColumnMapping{}
	for col, f := range m {
		for _, h := range t.Header {
			if strings.EqualFold(strings.TrimSpace(h), strings.TrimSpace(col)) {
				out[strings.TrimSpace(h)] = f
				break
			}
		}
	}
	return out
}

// columnAliases are normalized column names that suggest a field.
var columnAliases = map[Field][]string{
	FieldSpecies:    {"species", "pokemon", "name", "pokemonname"},
	FieldDexNumber:  {"dex", "dexnumber", "pokedex", "pokedexnumber", "pokemonnumber", "number"},
	FieldForm:       {"form"},
	FieldGender:     {"gender", "sex"},
	FieldCP:         {"cp", "combatpower"},
	FieldHP:         {"hp"},
	FieldLevel:      {"level", "lvl", "levelmin"},
	FieldIVAttack:   {"atk", "atkiv", "attack", "attackiv", "ivattack", "ivatk"},
	FieldIVDefense:  {"def", "defiv", "defense", "defence", "defenseiv", "ivdefense", "ivdef"},
	FieldIVStamina:  {"sta", "staiv", "stamina", "staminaiv", "ivstamina", "ivsta"},
	FieldCatchDate:  {"catchdate", "caught", "datecaught", "caughton"},
	FieldShiny:      {"shiny"},
	FieldLucky:      {"lucky"},
	FieldShadow:     {"shadow"},
	FieldPurified:   {"purified"},
	FieldCostume:    {"costume"},
	FieldBackground: {"background", "locationbackground"},
	FieldDynamax:    {"dynamax"},
	FieldGigantamax: {"gigantamax", "gmax"},
	FieldFavorite:   {"favorite", "favourite", "fav"},
}

// SuggestMapping guesses fields from common column names. Each field goes
// to the first column that suggests it.
func SuggestMapping(header []string) ColumnMapping {
	out := ColumnMapping{}
	taken := map[Field]bool{}
	for _, h := range header {
		key := normalizeColumn(h)
		for _, f := range MappableFields {
			if !taken[f] && slices.Contains(columnAliases[f], key) {
				out[strings.TrimSpace(h)] = f
				taken[f] = true
				break
			}
		}
	}
	return out
}

func normalizeColumn(s string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(s) {
		switch {
		case r == 'é':
			b.WriteRune('e')
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		}
	}
	return b.String()
}
