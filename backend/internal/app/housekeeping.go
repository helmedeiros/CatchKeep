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
