package app

import (
	"context"
	"fmt"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Housekeeping clears out data nobody can use any more.
type Housekeeping struct {
	batches  ports.ImportBatches
	sessions ports.Sessions
	clock    ports.Clock
}

// NewHousekeeping wires the clean-up.
func NewHousekeeping(batches ports.ImportBatches, sessions ports.Sessions, clock ports.Clock) *Housekeeping {
	return &Housekeeping{batches: batches, sessions: sessions, clock: clock}
}

// Swept counts what a sweep removed.
type Swept struct {
	Imports  int
	Sessions int
}

// Sweep removes unfinished imports older than a week and sessions idle
// past their timeout (spec SEC-03).
func (h *Housekeeping) Sweep(ctx context.Context) (Swept, error) {
	now := h.clock.Now()
	imports, err := h.batches.DeleteStagedBefore(ctx, now.Add(-domain.StagedImportLifetime))
	if err != nil {
		return Swept{}, fmt.Errorf("sweep imports: %w", err)
	}
	sessions, err := h.sessions.DeleteIdleSince(ctx, now.Add(-domain.SessionIdleTimeout))
	if err != nil {
		return Swept{Imports: imports}, fmt.Errorf("sweep sessions: %w", err)
	}
	return Swept{Imports: imports, Sessions: sessions}, nil
}

// Backfill gives imports committed before the history log their entries.
type Backfill struct {
	past ports.PastImports
	ids  ports.IDs
}

// NewBackfill wires the history backfill.
func NewBackfill(past ports.PastImports, ids ports.IDs) *Backfill {
	return &Backfill{past: past, ids: ids}
}

// Run records each import the history log is missing, once, and returns
// how many it recorded.
func (b *Backfill) Run(ctx context.Context) (int, error) {
	past, err := b.past.UnrecordedImports(ctx)
	if err != nil {
		return 0, fmt.Errorf("find past imports: %w", err)
	}
	for i, p := range past {
		if err := b.past.RecordImport(ctx, p.Event(b.ids.NewID())); err != nil {
			return i, fmt.Errorf("record past import: %w", err)
		}
	}
	return len(past), nil
}
