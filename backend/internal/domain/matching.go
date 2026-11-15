package domain

import "strings"

// Fields that identify a catch: power-ups and evolutions change CP, HP and
// level but never these (spec §6.4 matching heuristic).
var identityFields = []Field{FieldSpecies, FieldForm, FieldGender, FieldIVAttack, FieldIVDefense, FieldIVStamina, FieldCatchDate}

// MightBeSameCatch reports whether two Pokémon could be the same catch: no
// identity field is known on both sides with different values. Unknown values
// never rule a match out, so ambiguous pairs are always surfaced for review.
func MightBeSameCatch(a, b Snapshot) bool {
	for _, f := range identityFields {
		va, okA := a[f]
		vb, okB := b[f]
		if okA && okB && !strings.EqualFold(va, vb) {
			return false
		}
	}
	return true
}

// ClassifyImport splits rows into new Pokémon and rows that might already be
// in the collection or repeat an earlier row of the same import. Possible
// matches are held back, never merged (spec V1-IMP-07).
func ClassifyImport(existing []Pokemon, rows []ImportRow) (fresh []ImportRow, possibleMatches int) {
	seen := make([]Snapshot, 0, len(existing)+len(rows))
	for _, p := range existing {
		seen = append(seen, p.Current())
	}
	for _, row := range rows {
		snap := Pokemon{Observations: row.Observations}.Current()
		matched := false
		for _, s := range seen {
			if MightBeSameCatch(s, snap) {
				matched = true
				break
			}
		}
		if matched {
			possibleMatches++
			continue
		}
		fresh = append(fresh, row)
		seen = append(seen, snap)
	}
	return fresh, possibleMatches
}
