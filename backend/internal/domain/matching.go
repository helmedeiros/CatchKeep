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
