package domain

import (
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// ErrUnsupportedSearch means a search uses terms CatchKeep cannot check.
var ErrUnsupportedSearch = errors.New("unsupported search")

// Matches evaluates a Pokémon GO search against one Pokémon. Only terms
// whose meaning in the game is certain are supported: a Pokédex number
// ("16"), exact or ranged CP and HP ("cp120", "cp100-130", "hp30"), "!" to
// negate, "," for or and "&" for and, with "&" joining groups of "," terms
// as the game does. A field CatchKeep does not know might match, so it does.
func Matches(query string, s Snapshot) (bool, error) {
	result := yes
	for _, group := range strings.Split(query, "&") {
		either := no
		for _, term := range strings.Split(group, ",") {
			v, err := evalTerm(strings.TrimSpace(term), s)
			if err != nil {
				return false, err
			}
			either = either.or(v)
		}
		result = result.and(either)
	}
	return result != no, nil
}

type maybeBool int

const (
	no maybeBool = iota
	maybe
	yes
)

func (a maybeBool) or(b maybeBool) maybeBool  { return max(a, b) }
func (a maybeBool) and(b maybeBool) maybeBool { return min(a, b) }

func evalTerm(term string, s Snapshot) (maybeBool, error) {
	negate := strings.HasPrefix(term, "!")
	term = strings.TrimPrefix(term, "!")
	var v maybeBool
	var err error
	switch {
	case term == "":
		return no, fmt.Errorf("%w: empty term", ErrUnsupportedSearch)
	case strings.HasPrefix(term, "cp"):
		v, err = inRange(strings.TrimPrefix(term, "cp"), s, FieldCP)
	case strings.HasPrefix(term, "hp"):
		v, err = inRange(strings.TrimPrefix(term, "hp"), s, FieldHP)
	default:
		dex, convErr := strconv.Atoi(term)
		if convErr != nil {
			return no, fmt.Errorf("%w: %q", ErrUnsupportedSearch, term)
		}
		v = compare(s, FieldDexNumber, func(n int) bool { return n == dex })
	}
	if err != nil {
		return no, err
	}
	if negate && v != maybe {
		v = yes - v
	}
	return v, nil
}

func inRange(spec string, s Snapshot, f Field) (maybeBool, error) {
	lo, hi, found := strings.Cut(spec, "-")
	if !found {
		hi = lo
	}
	bound := func(text string, open int) (int, error) {
		if text == "" && found {
			return open, nil
		}
		return strconv.Atoi(text)
	}
	from, err1 := bound(lo, 0)
	to, err2 := bound(hi, maxCapacity)
	if err1 != nil || err2 != nil {
		return no, fmt.Errorf("%w: %q", ErrUnsupportedSearch, spec)
	}
	return compare(s, f, func(n int) bool { return n >= from && n <= to }), nil
}

func compare(s Snapshot, f Field, ok func(int) bool) maybeBool {
	n, known := s.Int(f)
	switch {
	case !known:
		return maybe
	case ok(n):
		return yes
	default:
		return no
	}
}

// SpeciesSearch finds the planned Pokémon of one species in the game.
type SpeciesSearch struct {
	Species string
	// Query finds all of them, or is empty if their Pokédex number is not
	// known; Collisions are other Pokémon it would also find.
	Query      string
	IDs        []PokemonID
	Collisions []PokemonID
	// Items narrows the search for planned Pokémon that share their CP
	// with one to keep.
	Items []ItemSearch
}

// ItemSearch narrows the search for one planned Pokémon. Query is empty
// when the species search already finds it alone; Collisions are Pokémon
// even the narrowed search cannot tell apart, to check in the game.
type ItemSearch struct {
	ID         PokemonID
	Query      string
	Collisions []PokemonID
}

// Searches builds one search per species for the planned Pokémon, in the
// order they first appear, and checks each against the rest of the
// collection (spec V1-CLN-04).
func Searches(pokemon []Pokemon, planned []PokemonID) []SpeciesSearch {
	snap := make(map[PokemonID]Snapshot, len(pokemon))
	for _, p := range pokemon {
		snap[p.ID] = p.Current()
	}
	isPlanned := map[PokemonID]bool{}
	for _, id := range planned {
		isPlanned[id] = true
	}
	var order []string
	bySpecies := map[string][]PokemonID{}
	for _, id := range planned {
		name := snap[id].String(FieldSpecies)
		if _, seen := bySpecies[name]; !seen {
			order = append(order, name)
		}
		bySpecies[name] = append(bySpecies[name], id)
	}
	collide := func(query string, dex int) []PokemonID {
		var out []PokemonID
		for _, p := range pokemon {
			s := snap[p.ID]
			if n, ok := s.Int(FieldDexNumber); isPlanned[p.ID] || (ok && n != dex) {
				continue
			}
			if hit, _ := Matches(query, s); hit {
				out = append(out, p.ID)
			}
		}
		slices.Sort(out)
		return out
	}
	out := make([]SpeciesSearch, 0, len(order))
	for _, name := range order {
		ids := bySpecies[name]
		search := SpeciesSearch{Species: name, IDs: ids}
		dex, ok := snap[ids[0]].Int(FieldDexNumber)
		if !ok {
			out = append(out, search)
			continue
		}
		var cps []int
		for _, id := range ids {
			if cp, ok := snap[id].Int(FieldCP); ok && !slices.Contains(cps, cp) {
				cps = append(cps, cp)
			}
		}
		slices.Sort(cps)
		terms := make([]string, len(cps))
		for i, cp := range cps {
			terms[i] = fmt.Sprintf("cp%d", cp)
		}
		search.Query = fmt.Sprintf("%d&%s", dex, strings.Join(terms, ","))
		search.Collisions = collide(search.Query, dex)
		for _, id := range ids {
			search.Items = append(search.Items, narrow(id, snap[id], dex, collide))
		}
		out = append(out, search)
	}
	return out
}

func narrow(id PokemonID, s Snapshot, dex int, collide func(string, int) []PokemonID) ItemSearch {
	cp, ok := s.Int(FieldCP)
	if !ok {
		return ItemSearch{ID: id, Collisions: collide(strconv.Itoa(dex), dex)}
	}
	query := fmt.Sprintf("%d&cp%d", dex, cp)
	if len(collide(query, dex)) == 0 {
		return ItemSearch{ID: id}
	}
	if hp, ok := s.Int(FieldHP); ok {
		query = fmt.Sprintf("%s&hp%d", query, hp)
	}
	return ItemSearch{ID: id, Query: query, Collisions: collide(query, dex)}
}
