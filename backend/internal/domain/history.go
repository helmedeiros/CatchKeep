package domain

import (
	"slices"
	"time"
)

// ImportEvent records what committing an import did (spec §13 history
// log): the Pokémon it added and those whose current values it changed.
type ImportEvent struct {
	ID        string
	TrainerID TrainerID
	BatchID   string
	Added     []Transferred
	Changed   []ChangedPokemon
	At        time.Time
}

// ChangedPokemon is a Pokémon as an import left it, and the fields it
// changed.
type ChangedPokemon struct {
	Transferred
	Fields []Field
}

// NewImportEvent describes an import that adds add and updates existing
// Pokémon. Updates that leave every current value as it was are left out.
func NewImportEvent(id string, trainer TrainerID, batchID string, existing, add []Pokemon, update []PokemonUpdate, at time.Time) ImportEvent {
	e := ImportEvent{ID: id, TrainerID: trainer, BatchID: batchID, At: at}
	for _, p := range add {
		e.Added = append(e.Added, described(p.ID, p.Current()))
	}
	byID := make(map[PokemonID]Pokemon, len(existing))
	for _, p := range existing {
		byID[p.ID] = p
	}
	for _, u := range update {
		p := byID[u.PokemonID]
		before := p.Current()
		p.Observations = append(slices.Clone(p.Observations), u.Observations...)
		after := p.Current()
		var fields []Field
		for f, v := range after {
			if before[f] != v {
				fields = append(fields, f)
			}
		}
		if len(fields) == 0 {
			continue
		}
		slices.Sort(fields)
		e.Changed = append(e.Changed, ChangedPokemon{Transferred: described(u.PokemonID, after), Fields: fields})
	}
	return e
}

func described(id PokemonID, s Snapshot) Transferred {
	cp, hasCP := s.Int(FieldCP)
	return Transferred{ID: id, Species: s.String(FieldSpecies), CP: cp, HasCP: hasCP}
}

// ChangeKind names a change a trainer makes by hand.
type ChangeKind string

// Changes kept in the history log.
const (
	ChangeProtection ChangeKind = "protection"
	ChangeTeam       ChangeKind = "team"
	ChangeRules      ChangeKind = "rules"
	ChangeSetup      ChangeKind = "setup"
)

// Change records a change a trainer made by hand (spec §13 history log):
// what kind, when, and the Pokémon it touched. The new values are saved
// with it.
type Change struct {
	ID        string
	TrainerID TrainerID
	Kind      ChangeKind
	Pokemon   []Transferred
	At        time.Time
}

// NewChange describes a change of kind that touched pokemon.
func NewChange(id string, trainer TrainerID, kind ChangeKind, pokemon []Pokemon, at time.Time) Change {
	c := Change{ID: id, TrainerID: trainer, Kind: kind, Pokemon: []Transferred{}, At: at}
	for _, p := range pokemon {
		c.Pokemon = append(c.Pokemon, described(p.ID, p.Current()))
	}
	return c
}
