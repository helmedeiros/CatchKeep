package app

import (
	"cmp"
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// ProfileSummary is what the profile picker shows.
type ProfileSummary struct {
	ID          domain.TrainerID
	DisplayName string
}

// Profiles creates and lists trainer profiles.
type Profiles struct {
	trainers ports.Trainers
	hasher   ports.PINHasher
	ids      ports.IDs
	clock    ports.Clock
}

// NewProfiles wires the profile use cases.
func NewProfiles(trainers ports.Trainers, hasher ports.PINHasher, ids ports.IDs, clock ports.Clock) *Profiles {
	return &Profiles{trainers: trainers, hasher: hasher, ids: ids, clock: clock}
}

// Create validates the name and PIN and stores a new trainer with a hashed PIN.
func (p *Profiles) Create(ctx context.Context, displayName, pin string) (ProfileSummary, error) {
	name, err := domain.NewDisplayName(displayName)
	if err != nil {
		return ProfileSummary{}, err
	}
	if err := domain.ValidatePIN(pin); err != nil {
		return ProfileSummary{}, err
	}
	hash, err := p.hasher.Hash(pin)
	if err != nil {
		return ProfileSummary{}, fmt.Errorf("hash PIN: %w", err)
	}
	t := domain.Trainer{ID: domain.TrainerID(p.ids.NewID()), DisplayName: name, PINHash: hash, CreatedAt: p.clock.Now()}
	if err := p.trainers.Create(ctx, t); err != nil {
		return ProfileSummary{}, err
	}
	return ProfileSummary{ID: t.ID, DisplayName: t.DisplayName}, nil
}

// List returns every profile ordered by display name, ignoring case.
func (p *Profiles) List(ctx context.Context) ([]ProfileSummary, error) {
	trainers, err := p.trainers.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]ProfileSummary, 0, len(trainers))
	for _, t := range trainers {
		out = append(out, ProfileSummary{ID: t.ID, DisplayName: t.DisplayName})
	}
	slices.SortFunc(out, func(a, b ProfileSummary) int {
		return cmp.Compare(strings.ToLower(a.DisplayName), strings.ToLower(b.DisplayName))
	})
	return out, nil
}
