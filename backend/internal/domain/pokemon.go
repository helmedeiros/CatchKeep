package domain

import (
	"strconv"
	"time"
)

// PokemonID identifies one Pokémon record for its whole life.
type PokemonID string

// Field names a single observable property of a Pokémon.
type Field string

// Observable fields (spec §5.2).
const (
	FieldSpecies    Field = "species"
	FieldForm       Field = "form"
	FieldDexNumber  Field = "dex_number"
	FieldGender     Field = "gender"
	FieldCP         Field = "cp"
	FieldHP         Field = "hp"
	FieldLevel      Field = "level"
	FieldIVAttack   Field = "iv_attack"
	FieldIVDefense  Field = "iv_defense"
	FieldIVStamina  Field = "iv_stamina"
	FieldCatchDate  Field = "catch_date"
	FieldShiny      Field = "shiny"
	FieldLucky      Field = "lucky"
	FieldShadow     Field = "shadow"
	FieldPurified   Field = "purified"
	FieldFavorite   Field = "in_game_favorite"
	FieldCostume    Field = "costume"
	FieldBackground Field = "background"
	FieldDynamax    Field = "dynamax"
	FieldGigantamax Field = "gigantamax"
)

// Source says where an observation came from.
type Source string

// Observation sources.
const (
	SourceCSV        Source = "csv"
	SourceScreenshot Source = "screenshot"
	SourceManual     Source = "manual"
)

// Observation is one value for one field, with where and when it was seen.
// Values are normalized text: integers, decimals, "true"/"false",
// ISO dates (YYYY-MM-DD) or names.
type Observation struct {
	Field      Field
	Value      string
	Source     Source
	SourceRef  string
	ObservedAt time.Time
}

// Pokemon is a record and everything observed about it.
type Pokemon struct {
	ID           PokemonID
	TrainerID    TrainerID
	Observations []Observation
}

// Snapshot is the current value of each observed field.
type Snapshot map[Field]string

// Current resolves each field: the latest manual observation wins; otherwise
// the latest observation by ObservedAt (spec §5.2).
func (p Pokemon) Current() Snapshot {
	best := map[Field]Observation{}
	for _, o := range p.Observations {
		prev, seen := best[o.Field]
		if !seen || outranks(o, prev) {
			best[o.Field] = o
		}
	}
	out := make(Snapshot, len(best))
	for f, o := range best {
		out[f] = o.Value
	}
	return out
}

func outranks(o, prev Observation) bool {
	oManual, prevManual := o.Source == SourceManual, prev.Source == SourceManual
	if oManual != prevManual {
		return oManual
	}
	return !o.ObservedAt.Before(prev.ObservedAt)
}

// String returns the field's value, or "" if unknown.
func (s Snapshot) String(f Field) string { return s[f] }

// Int returns the field as an integer and whether it is known.
func (s Snapshot) Int(f Field) (int, bool) {
	v, ok := s[f]
	if !ok {
		return 0, false
	}
	n, err := strconv.Atoi(v)
	return n, err == nil
}

// Float returns the field as a decimal and whether it is known.
func (s Snapshot) Float(f Field) (float64, bool) {
	v, ok := s[f]
	if !ok {
		return 0, false
	}
	n, err := strconv.ParseFloat(v, 64)
	return n, err == nil
}

// TriState is yes, no, or not established by any source.
type TriState int

// Tri-state values. The zero value is Unknown.
const (
	Unknown TriState = iota
	Yes
	No
)

// Tri returns a yes/no field. Missing or unparseable values are Unknown —
// never treated as No (spec V1-RUL-07/08).
func (s Snapshot) Tri(f Field) TriState {
	switch s[f] {
	case "true":
		return Yes
	case "false":
		return No
	default:
		return Unknown
	}
}

// IVs are a Pokémon's individual values, each 0–15.
type IVs struct {
	Attack, Defense, Stamina int
}

// IVs returns the IVs if all three are known.
func (s Snapshot) IVs() (IVs, bool) {
	a, okA := s.Int(FieldIVAttack)
	d, okD := s.Int(FieldIVDefense)
	st, okS := s.Int(FieldIVStamina)
	if !okA || !okD || !okS {
		return IVs{}, false
	}
	return IVs{Attack: a, Defense: d, Stamina: st}, true
}
