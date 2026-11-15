package domain

import (
	"errors"
	"fmt"
	"strings"
)

// ErrInvalidResolution means an answer does not fit the import's review.
var ErrInvalidResolution = errors.New("invalid duplicate review answer")

// RowRef refers to an earlier row of the same import as a match candidate.
func RowRef(line int) string { return fmt.Sprintf("row-%d", line) }

// Candidate is something an import row might already be: a Pokémon in the
// collection (Ref is its ID) or an earlier row of the same import (Ref is
// RowRef(Line)).
type Candidate struct {
	Ref     string
	Line    int
	Pokemon Pokemon
}

// Resolution is the trainer's answer for one possible match: the same
// Pokémon as candidate SameAs, or a different Pokémon when SameAs is empty.
type Resolution struct {
	Line   int
	SameAs string
}

// PossibleMatch is an import row held back until the trainer answers.
type PossibleMatch struct {
	Row        ImportRow
	Candidates []Candidate
	// Exact means one candidate, with every identity field known and equal.
	Exact bool
	// Resolution is the trainer's answer, if any.
	Resolution *Resolution
}

// PokemonUpdate adds newer observations to a Pokémon already collected.
type PokemonUpdate struct {
	PokemonID    PokemonID
	Observations []Observation
}

// ImportReview is what committing an import would do now.
type ImportReview struct {
	Add     []ImportRow
	Update  []PokemonUpdate
	Matches []PossibleMatch
	// Repeats are rows answered as the same as an earlier row; they are dropped.
	Repeats int
}

// Pending counts possible matches still waiting for an answer.
func (r ImportReview) Pending() int {
	n := 0
	for _, m := range r.Matches {
		if m.Resolution == nil {
			n++
		}
	}
	return n
}

// Validate checks that an answer refers to a possible match and, when it says
// "same", to one of that match's candidates.
func (r ImportReview) Validate(res Resolution) error {
	for _, m := range r.Matches {
		if m.Row.Line != res.Line {
			continue
		}
		if res.SameAs == "" || m.candidate(res.SameAs) != nil {
			return nil
		}
		return fmt.Errorf("%w: line %d cannot be the same as %q", ErrInvalidResolution, res.Line, res.SameAs)
	}
	return fmt.Errorf("%w: line %d is not a possible duplicate", ErrInvalidResolution, res.Line)
}

func (m PossibleMatch) candidate(ref string) *Candidate {
	for i := range m.Candidates {
		if m.Candidates[i].Ref == ref {
			return &m.Candidates[i]
		}
	}
	return nil
}

// ReviewImport splits rows into new Pokémon and possible matches, then
// applies the trainer's answers (keyed by line). Possible matches are never
// merged without an answer (spec V1-IMP-07); an answer naming a candidate
// that no longer matches is ignored and the row stays pending.
func ReviewImport(existing []Pokemon, rows []ImportRow, answers map[int]Resolution) ImportReview {
	type seenEntry struct {
		cand Candidate
		snap Snapshot
	}
	seen := make([]seenEntry, 0, len(existing)+len(rows))
	for _, p := range existing {
		seen = append(seen, seenEntry{Candidate{Ref: string(p.ID), Pokemon: p}, p.Current()})
	}
	var out ImportReview
	for _, row := range rows {
		incoming := Pokemon{Observations: row.Observations}
		snap := incoming.Current()
		var m PossibleMatch
		for _, s := range seen {
			if MightBeSameCatch(s.snap, snap) {
				m.Candidates = append(m.Candidates, s.cand)
			}
		}
		if len(m.Candidates) == 0 {
			out.Add = append(out.Add, row)
			seen = append(seen, seenEntry{Candidate{Ref: RowRef(row.Line), Line: row.Line, Pokemon: incoming}, snap})
			continue
		}
		m.Row = row
		m.Exact = len(m.Candidates) == 1 && identityKnownAndEqual(m.Candidates[0].Pokemon.Current(), snap)
		if a, ok := answers[row.Line]; ok && (a.SameAs == "" || m.candidate(a.SameAs) != nil) {
			m.Resolution = &a
			switch c := m.candidate(a.SameAs); {
			case c == nil:
				out.Add = append(out.Add, row)
				seen = append(seen, seenEntry{Candidate{Ref: RowRef(row.Line), Line: row.Line, Pokemon: incoming}, snap})
			case c.Line != 0:
				out.Repeats++
			default:
				out.Update = append(out.Update, PokemonUpdate{PokemonID: c.Pokemon.ID, Observations: row.Observations})
			}
		}
		out.Matches = append(out.Matches, m)
	}
	return out
}

func identityKnownAndEqual(a, b Snapshot) bool {
	for _, f := range identityFields {
		va, okA := a[f]
		vb, okB := b[f]
		if !okA || !okB || !strings.EqualFold(va, vb) {
			return false
		}
	}
	return true
}
