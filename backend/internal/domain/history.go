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
