// Package apptest provides in-memory fakes of the driven ports for use-case
// tests.
package apptest

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

// Trainers is an in-memory ports.Trainers.
type Trainers struct {
	mu   sync.Mutex
	byID map[domain.TrainerID]domain.Trainer
}

// NewTrainers returns an empty trainer store.
func NewTrainers() *Trainers {
	return &Trainers{byID: map[domain.TrainerID]domain.Trainer{}}
}

// Create implements ports.Trainers.
func (s *Trainers) Create(_ context.Context, t domain.Trainer) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, existing := range s.byID {
		if strings.EqualFold(existing.DisplayName, t.DisplayName) {
			return domain.ErrDisplayNameTaken
		}
	}
	s.byID[t.ID] = t
	return nil
}

// Get implements ports.Trainers.
func (s *Trainers) Get(_ context.Context, id domain.TrainerID) (domain.Trainer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.byID[id]
	if !ok {
		return domain.Trainer{}, domain.ErrTrainerNotFound
	}
	return t, nil
}

// List implements ports.Trainers.
func (s *Trainers) List(context.Context) ([]domain.Trainer, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]domain.Trainer, 0, len(s.byID))
	for _, t := range s.byID {
		out = append(out, t)
	}
	return out, nil
}

// PlainHasher is a reversible, insecure ports.PINHasher for tests only.
type PlainHasher struct{}

// Hash implements ports.PINHasher.
func (PlainHasher) Hash(pin string) (string, error) { return "hashed:" + pin, nil }

// Verify implements ports.PINHasher.
func (PlainHasher) Verify(hash, pin string) (bool, error) { return hash == "hashed:"+pin, nil }

// SequentialIDs is a deterministic ports.IDs.
type SequentialIDs struct {
	mu sync.Mutex
	n  int
}

// NewSequentialIDs returns IDs id-1, id-2, ...
func NewSequentialIDs() *SequentialIDs { return &SequentialIDs{} }

// NewID implements ports.IDs.
func (g *SequentialIDs) NewID() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.n++
	return fmt.Sprintf("id-%d", g.n)
}

// FixedClock is a ports.Clock that only moves when told to.
type FixedClock struct {
	mu  sync.Mutex
	now time.Time
}

// NewFixedClock returns a clock set to now.
func NewFixedClock(now time.Time) *FixedClock { return &FixedClock{now: now} }

// Now implements ports.Clock.
func (c *FixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Advance moves the clock forward by d.
func (c *FixedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
