package domain

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
)

// ErrNotConfirmable means a bulk confirmation named no field, or a field that
// cannot be confirmed in bulk.
var ErrNotConfirmable = errors.New("only shiny, costume, background, dynamax and gigantamax can be confirmed in bulk")

// ConfirmableFields are the features an export cannot tell and a trainer can
// check in the game (spec §7.3).
var ConfirmableFields = []Field{FieldShiny, FieldCostume, FieldBackground, FieldDynamax, FieldGigantamax}

// ConfirmAbsent records the trainer's word that none of their Pokémon of one
// species and form (or of the whole collection, when species is empty) has
// the given features. Only unknown values are set, as manual "false"
// observations; a feature already known either way is kept.
func ConfirmAbsent(pokemon []Pokemon, species, form string, fields []Field, at time.Time) ([]PokemonUpdate, error) {
	if len(fields) == 0 {
		return nil, ErrNotConfirmable
	}
	for _, f := range fields {
		if !slices.Contains(ConfirmableFields, f) {
			return nil, fmt.Errorf("%w: %q", ErrNotConfirmable, f)
		}
	}
	var out []PokemonUpdate
	for _, p := range pokemon {
		cur := p.Current()
		if species != "" && (!strings.EqualFold(cur.String(FieldSpecies), species) || !strings.EqualFold(formOf(cur), form)) {
			continue
		}
		u := PokemonUpdate{PokemonID: p.ID}
		for _, f := range fields {
			if cur.Tri(f) == Unknown {
				u.Observations = append(u.Observations, Observation{
					Field: f, Value: "false", Source: SourceManual, SourceRef: "bulk confirmation", ObservedAt: at,
				})
			}
		}
		if len(u.Observations) > 0 {
			out = append(out, u)
		}
	}
	return out, nil
}

// formOf returns the form, "Normal" when none was observed.
func formOf(s Snapshot) string {
	if f := s.String(FieldForm); f != "" {
		return f
	}
	return "Normal"
}
