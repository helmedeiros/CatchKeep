package domain

import (
	"errors"
	"fmt"
	"strings"
	"time"
)

// Transfer errors.
var (
	ErrTransferNotFound = errors.New("transfer not found")
	ErrAlreadyUndone    = errors.New("that transfer was already undone")
	ErrInvalidTransfer  = errors.New("invalid transfer")
	// ErrStaleRevision means the collection or rules changed since the
	// plan was made (spec §10.3).
	ErrStaleRevision = errors.New("the plan is out of date")
)

// NotActiveError lists Pokémon that are no longer in the collection, so a
// whole confirmation is refused (spec §9, atomic or nothing).
type NotActiveError struct{ IDs []PokemonID }

func (e *NotActiveError) Error() string {
	ids := make([]string, len(e.IDs))
	for i, id := range e.IDs {
		ids[i] = string(id)
	}
	return "already transferred or not in the collection: " + strings.Join(ids, ", ")
}

// TransferKind says how a transfer was recorded.
type TransferKind string

// Transfer kinds (spec §5 TransferEvent).
const (
	TransferSingle TransferKind = "single"
	TransferBulk   TransferKind = "bulk"
	TransferUndo   TransferKind = "undo"
)

// maxTransfer bounds one confirmation.
const maxTransfer = 5000

// Transferred describes a Pokémon as it was when transferred, so history
// stays readable.
type Transferred struct {
	ID      PokemonID
	Species string
	CP      int
	HasCP   bool
}

// TransferEvent records Pokémon the trainer transferred in the game, or an
// undo of such a record. Nothing is hard-deleted.
type TransferEvent struct {
	ID        string
	TrainerID TrainerID
	Kind      TransferKind
	Pokemon   []Transferred
	At        time.Time
	// RequestID makes a repeated confirmation count once.
	RequestID string
	// Undoes names the event an undo reverses; UndoneBy the undo that
	// reversed this one.
	Undoes, UndoneBy string
}

// NewTransfer builds the event for confirming Pokémon as transferred,
// described from the trainer's active collection. Every Pokémon must be
// active and listed once.
func NewTransfer(id string, trainer TrainerID, pokemon []Pokemon, ids []PokemonID, requestID string, at time.Time) (TransferEvent, error) {
	switch {
	case len(ids) == 0:
		return TransferEvent{}, fmt.Errorf("%w: choose at least one Pokémon", ErrInvalidTransfer)
	case len(ids) > maxTransfer:
		return TransferEvent{}, fmt.Errorf("%w: at most %d at a time", ErrInvalidTransfer, maxTransfer)
	case strings.TrimSpace(requestID) == "":
		return TransferEvent{}, fmt.Errorf("%w: a request id is needed", ErrInvalidTransfer)
	}
	active := make(map[PokemonID]Pokemon, len(pokemon))
	for _, p := range pokemon {
		active[p.ID] = p
	}
	seen := map[PokemonID]bool{}
	var gone []PokemonID
	e := TransferEvent{ID: id, TrainerID: trainer, Kind: TransferBulk, At: at, RequestID: requestID}
	for _, pid := range ids {
		if seen[pid] {
			return TransferEvent{}, fmt.Errorf("%w: %s is listed twice", ErrInvalidTransfer, pid)
		}
		seen[pid] = true
		p, ok := active[pid]
		if !ok {
			gone = append(gone, pid)
			continue
		}
		cur := p.Current()
		cp, hasCP := cur.Int(FieldCP)
		e.Pokemon = append(e.Pokemon, Transferred{ID: pid, Species: cur.String(FieldSpecies), CP: cp, HasCP: hasCP})
	}
	if len(gone) > 0 {
		return TransferEvent{}, &NotActiveError{IDs: gone}
	}
	if len(ids) == 1 {
		e.Kind = TransferSingle
	}
	return e, nil
}
