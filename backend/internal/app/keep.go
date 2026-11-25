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
	teams      ports.Teams
	ids        ports.IDs
	clock      ports.Clock
}

// NewKeep wires the keep-rule use cases.
func NewKeep(setups ports.Setups, collection ports.CollectionReader, species ports.Species, teams ports.Teams, ids ports.IDs, clock ports.Clock) *Keep {
	return &Keep{setups: setups, collection: collection, species: species, teams: teams, ids: ids, clock: clock}
}

// CapacityBands are the default minimums for each storage size.
func (k *Keep) CapacityBands() []domain.CapacityBand { return domain.CapacityBands() }

// Setup returns the trainer's setup, or domain.ErrSetupNotFound.
func (k *Keep) Setup(ctx context.Context, trainer domain.TrainerID) (domain.Setup, error) {
	return k.setups.GetSetup(ctx, trainer)
}

// SaveSetup validates and stores the trainer's storage and goals, keeping
// any species and category rules they already have.
func (k *Keep) SaveSetup(ctx context.Context, trainer domain.TrainerID, s domain.Setup) error {
	current, err := k.setups.GetSetup(ctx, trainer)
	switch {
	case err == nil:
		s.Species, s.Categories = current.Species, current.Categories
		return k.save(ctx, trainer, &current, s, domain.ChangeSetup)
	case errors.Is(err, domain.ErrSetupNotFound):
		return k.save(ctx, trainer, nil, s, domain.ChangeSetup)
	default:
		return err
	}
}

// SaveSpeciesRules replaces the trainer's species rules. It needs a setup.
func (k *Keep) SaveSpeciesRules(ctx context.Context, trainer domain.TrainerID, rules []domain.SpeciesRule) error {
	current, err := k.setups.GetSetup(ctx, trainer)
	if err != nil {
		return err
	}
	s := current
	s.Species = rules
	return k.save(ctx, trainer, &current, s, domain.ChangeRules)
}

// SaveCategoryRules replaces the trainer's category rules. It needs a setup.
func (k *Keep) SaveCategoryRules(ctx context.Context, trainer domain.TrainerID, rules []domain.CategoryRule) error {
	current, err := k.setups.GetSetup(ctx, trainer)
	if err != nil {
		return err
	}
	s := current
	s.Categories = rules
	return k.save(ctx, trainer, &current, s, domain.ChangeRules)
}

// save stores s over current (nil if there is none) and records it in the
// history as kind; saving it unchanged changes nothing.
func (k *Keep) save(ctx context.Context, trainer domain.TrainerID, current *domain.Setup, s domain.Setup, kind domain.ChangeKind) error {
	if err := s.Validate(); err != nil {
		return err
	}
	if current != nil && domain.SameSetup(*current, s) {
		return nil
	}
	change := domain.NewChange(k.ids.NewID(), trainer, kind, nil, k.clock.Now())
	if err := k.setups.SaveSetup(ctx, trainer, s, change); err != nil {
		return fmt.Errorf("save setup: %w", err)
	}
	return nil
}

// Plan suggests what to transfer for the target (spec §7).
func (k *Keep) Plan(ctx context.Context, trainer domain.TrainerID, target domain.Target) (domain.Plan, error) {
	setup, pokemon, err := k.inputs(ctx, trainer)
	if err != nil {
		return domain.Plan{}, err
	}
	return domain.PlanTransfers(pokemon, domain.SummarizeKeeps(pokemon, setup, k.species.Category), setup, target)
}

// Summary applies the trainer's keep rules to their collection; team
// members are protected.
func (k *Keep) Summary(ctx context.Context, trainer domain.TrainerID) (domain.KeepSummary, error) {
	setup, pokemon, err := k.inputs(ctx, trainer)
	if err != nil {
		return domain.KeepSummary{}, err
	}
	return domain.SummarizeKeeps(pokemon, setup, k.species.Category), nil
}

// inputs are the trainer's setup and Pokémon, with team members protected.
func (k *Keep) inputs(ctx context.Context, trainer domain.TrainerID) (domain.Setup, []domain.Pokemon, error) {
	setup, err := k.setups.GetSetup(ctx, trainer)
	if err != nil {
		return domain.Setup{}, nil, err
	}
	pokemon, err := k.collection.ListPokemon(ctx, trainer)
	if err != nil {
		return domain.Setup{}, nil, err
	}
	teams, err := k.teams.ListTeams(ctx, trainer)
	if err != nil {
		return domain.Setup{}, nil, err
	}
	return setup, domain.ProtectTeamMembers(pokemon, teams), nil
}
