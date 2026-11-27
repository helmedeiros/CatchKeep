package app

import (
	"context"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// historyLength is how many transfer events the history shows.
const historyLength = 100

// Transfers records what the trainer transferred in the game, with undo.
type Transfers struct {
	store      ports.Transfers
	collection ports.CollectionReader
	revisions  ports.Revisions
	storage    StorageRecorder
	ids        ports.IDs
	clock      ports.Clock
}

// NewTransfers wires the transfer use cases.
func NewTransfers(store ports.Transfers, collection ports.CollectionReader, revisions ports.Revisions, storage StorageRecorder, ids ports.IDs, clock ports.Clock) *Transfers {
	return &Transfers{store: store, collection: collection, revisions: revisions, storage: storage, ids: ids, clock: clock}
}

// AnyRevision skips the stale-plan check, for single confirmations from
// the collection.
const AnyRevision = -1

// Confirm marks Pokémon as transferred in the game, all or none. A plan's
// revision guards bulk confirmations against plans made before a change
// (spec §7.6); the same requestID counts once.
func (t *Transfers) Confirm(ctx context.Context, trainer domain.TrainerID, ids []domain.PokemonID, requestID string, revision int64) (domain.TransferEvent, error) {
	if earlier, ok, err := t.store.FindByRequest(ctx, trainer, requestID); err != nil || ok {
		return earlier, err
	}
	pokemon, err := t.collection.ListPokemon(ctx, trainer)
	if err != nil {
		return domain.TransferEvent{}, err
	}
	e, err := domain.NewTransfer(t.ids.NewID(), trainer, pokemon, ids, requestID, t.clock.Now())
	if err != nil {
		return domain.TransferEvent{}, err
	}
	e, err = t.store.Record(ctx, e, revision)
	if err == nil {
		t.storage.RecordStorage(ctx, trainer, e.ID)
	}
	return e, err
}

// Undo puts the Pokémon of a transfer back in the collection. It cannot
// bring back a Pokémon transferred in the game.
func (t *Transfers) Undo(ctx context.Context, trainer domain.TrainerID, eventID string) (domain.TransferEvent, error) {
	undo := domain.TransferEvent{ID: t.ids.NewID(), TrainerID: trainer, Kind: domain.TransferUndo, At: t.clock.Now()}
	undo, err := t.store.Undo(ctx, trainer, eventID, undo)
	if err == nil {
		t.storage.RecordStorage(ctx, trainer, undo.ID)
	}
	return undo, err
}

// History returns the latest transfers and undos, newest first.
func (t *Transfers) History(ctx context.Context, trainer domain.TrainerID) ([]domain.TransferEvent, error) {
	return t.store.ListTransfers(ctx, trainer, historyLength)
}

// Revision is the trainer's current revision, which plans carry.
func (t *Transfers) Revision(ctx context.Context, trainer domain.TrainerID) (int64, error) {
	return t.revisions.Revision(ctx, trainer)
}
