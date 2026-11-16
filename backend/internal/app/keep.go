package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Keep holds a trainer's setup and applies the keep rules to the collection.
type Keep struct {
	setups     ports.Setups
	collection ports.CollectionReader
	species    ports.Species
}

// NewKeep wires the keep-rule use cases.
func NewKeep(setups ports.Setups, collection ports.CollectionReader, species ports.Species) *Keep {
	return &Keep{setups: setups, collection: collection, species: species}
}

// Setup returns the trainer's setup, or domain.ErrSetupNotFound.
func (k *Keep) Setup(ctx context.Context, trainer domain.TrainerID) (domain.Setup, error) {
	return k.setups.GetSetup(ctx, trainer)
}

// SaveSetup validates and stores the trainer's storage and goals, keeping
// any species rules they already have.
func (k *Keep) SaveSetup(ctx context.Context, trainer domain.TrainerID, s domain.Setup) error {
	current, err := k.setups.GetSetup(ctx, trainer)
	switch {
	case err == nil:
		s.Species = current.Species
	case !errors.Is(err, domain.ErrSetupNotFound):
		return err
	}
	return k.save(ctx, trainer, s)
}

// SaveSpeciesRules replaces the trainer's species rules. It needs a setup.
func (k *Keep) SaveSpeciesRules(ctx context.Context, trainer domain.TrainerID, rules []domain.SpeciesRule) error {
	s, err := k.setups.GetSetup(ctx, trainer)
	if err != nil {
		return err
	}
	s.Species = rules
	return k.save(ctx, trainer, s)
}

func (k *Keep) save(ctx context.Context, trainer domain.TrainerID, s domain.Setup) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if err := k.setups.SaveSetup(ctx, trainer, s); err != nil {
		return fmt.Errorf("save setup: %w", err)
	}
	return nil
}

// Summary applies the trainer's keep rules to their collection.
func (k *Keep) Summary(ctx context.Context, trainer domain.TrainerID) (domain.KeepSummary, error) {
	setup, err := k.setups.GetSetup(ctx, trainer)
	if err != nil {
		return domain.KeepSummary{}, err
	}
	pokemon, err := k.collection.ListPokemon(ctx, trainer)
	if err != nil {
		return domain.KeepSummary{}, err
	}
	return domain.SummarizeKeeps(pokemon, setup, k.species.Rare), nil
}
