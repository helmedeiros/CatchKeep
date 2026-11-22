package sqlite_test

import (
	"errors"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/sqlite"
	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func withPokemon(t *testing.T, store *sqlite.Store, trainer domain.TrainerID, ids ...domain.PokemonID) {
	t.Helper()
	coll := sqlite.NewCollection(store)
	b := stagedBatch()
	b.ID, b.TrainerID = "B-"+string(trainer), trainer
	if err := coll.Save(t.Context(), b); err != nil {
		t.Fatal(err)
	}
	b.Status = domain.ImportCommitted
	var add []domain.Pokemon
	for _, id := range ids {
		add = append(add, domain.Pokemon{ID: id, TrainerID: trainer, Observations: b.Parsed.Rows[0].Observations})
	}
	if err := coll.Commit(t.Context(), b, add, nil); err != nil {
		t.Fatal(err)
	}
}

func event(id, request string, ids ...domain.PokemonID) domain.TransferEvent {
	e := domain.TransferEvent{ID: id, TrainerID: "ash", Kind: domain.TransferBulk, RequestID: request, At: at}
	for _, pid := range ids {
		e.Pokemon = append(e.Pokemon, domain.Transferred{ID: pid, Species: "Pidgey", CP: 120, HasCP: true})
	}
	return e
}

func TestTransfers_RecordUndoAndHistory(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	withPokemon(t, store, "ash", "P1", "P2", "P3")
	withPokemon(t, store, "misty", "M1")
	coll := sqlite.NewCollection(store)
	transfers := sqlite.NewTransfers(store)
	ctx := t.Context()
	active := func() int {
		list, _ := coll.ListPokemon(ctx, "ash")
		return len(list)
	}
	rev, _ := coll.Revision(ctx, "ash")

	if _, err := transfers.Record(ctx, event("E0", "r0", "P1"), rev+5); !errors.Is(err, domain.ErrStaleRevision) {
		t.Fatalf("stale plan: err = %v", err)
	}
	var gone *domain.NotActiveError
	if _, err := transfers.Record(ctx, event("E0", "r0", "P1", "M1"), -1); !errors.As(err, &gone) || len(gone.IDs) != 1 || gone.IDs[0] != "M1" {
		t.Fatalf("misty's Pokémon: err = %v", err)
	}
	if active() != 3 {
		t.Fatal("a refused confirmation changed the collection")
	}

	first, err := transfers.Record(ctx, event("E1", "r1", "P1", "P2"), rev)
	if err != nil || first.ID != "E1" || active() != 1 {
		t.Fatalf("record: %+v, %v, %d active", first, err, active())
	}
	if again, err := transfers.Record(ctx, event("E9", "r1", "P1", "P2"), -1); err != nil || again.ID != "E1" {
		t.Fatalf("repeat: %+v, %v; the same request counts once", again, err)
	}
	if _, err := transfers.Record(ctx, event("E2", "r2", "P2"), -1); !errors.As(err, &gone) {
		t.Fatalf("already transferred: err = %v", err)
	}
	if now, _ := coll.Revision(ctx, "ash"); now != rev+1 {
		t.Fatalf("revision = %d, want %d", now, rev+1)
	}

	undo := domain.TransferEvent{ID: "U1", TrainerID: "ash", Kind: domain.TransferUndo, At: at}
	if _, err := transfers.Undo(ctx, "misty", "E1", undo); !errors.Is(err, domain.ErrTransferNotFound) {
		t.Fatalf("misty undoing ash's: err = %v", err)
	}
	undone, err := transfers.Undo(ctx, "ash", "E1", undo)
	if err != nil || undone.Undoes != "E1" || len(undone.Pokemon) != 2 || active() != 3 {
		t.Fatalf("undo: %+v, %v, %d active", undone, err, active())
	}
	if _, err := transfers.Undo(ctx, "ash", "E1", domain.TransferEvent{ID: "U2", TrainerID: "ash", Kind: domain.TransferUndo, At: at}); !errors.Is(err, domain.ErrAlreadyUndone) {
		t.Fatalf("undo twice: err = %v", err)
	}

	history, err := transfers.ListTransfers(ctx, "ash", 10)
	if err != nil || len(history) != 2 || history[0].ID != "U1" || history[1].UndoneBy != "U1" || history[1].Pokemon[0].Species != "Pidgey" {
		t.Fatalf("history = %+v, %v; newest first", history, err)
	}
	if theirs, _ := transfers.ListTransfers(ctx, "misty", 10); len(theirs) != 0 {
		t.Fatalf("misty sees %+v", theirs)
	}
}

func TestTransfers_KeepTheInGameCountCurrent(t *testing.T) {
	store, _ := openTemp(t)
	withTrainers(t, store, "ash", "misty")
	withPokemon(t, store, "ash", "P1", "P2", "P3")
	setups := sqlite.NewSetups(store)
	transfers := sqlite.NewTransfers(store)
	ctx := t.Context()
	if err := setups.SaveSetup(ctx, "ash", domain.Setup{Capacity: 4000, Goals: []domain.Goal{domain.GoalCollecting}, InGameCount: 3300}); err != nil {
		t.Fatal(err)
	}
	if err := setups.SaveSetup(ctx, "misty", domain.Setup{Capacity: 400, Goals: []domain.Goal{domain.GoalCollecting}}); err != nil {
		t.Fatal(err)
	}
	inGame := func(trainer domain.TrainerID) int {
		s, err := setups.GetSetup(ctx, trainer)
		if err != nil {
			t.Fatal(err)
		}
		return s.InGameCount
	}

	if _, err := transfers.Record(ctx, event("E1", "r1", "P1", "P2"), -1); err != nil {
		t.Fatal(err)
	}
	if n := inGame("ash"); n != 3298 {
		t.Fatalf("after transferring 2: in-game count = %d, want 3298", n)
	}
	if _, err := transfers.Undo(ctx, "ash", "E1", domain.TransferEvent{ID: "U1", TrainerID: "ash", Kind: domain.TransferUndo, At: at}); err != nil {
		t.Fatal(err)
	}
	if n := inGame("ash"); n != 3300 {
		t.Fatalf("after the undo: in-game count = %d, want 3300", n)
	}
	if n := inGame("misty"); n != 0 {
		t.Fatalf("misty's unknown count became %d", n)
	}
}
