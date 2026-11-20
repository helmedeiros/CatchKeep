// Package apptest provides in-memory fakes of the driven ports for use-case
// tests.
package apptest

import (
	"context"
	"fmt"
	"slices"
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

// SessionStore is an in-memory ports.Sessions.
type SessionStore struct {
	mu     sync.Mutex
	byHash map[string]domain.Session
}

// NewSessionStore returns an empty session store.
func NewSessionStore() *SessionStore {
	return &SessionStore{byHash: map[string]domain.Session{}}
}

// Create implements ports.Sessions.
func (s *SessionStore) Create(_ context.Context, session domain.Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byHash[session.TokenHash] = session
	return nil
}

// Get implements ports.Sessions.
func (s *SessionStore) Get(_ context.Context, tokenHash string) (domain.Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.byHash[tokenHash]
	if !ok {
		return domain.Session{}, domain.ErrNoSession
	}
	return session, nil
}

// Touch implements ports.Sessions.
func (s *SessionStore) Touch(_ context.Context, tokenHash string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session, ok := s.byHash[tokenHash]
	if !ok {
		return domain.ErrNoSession
	}
	session.LastSeenAt = at
	s.byHash[tokenHash] = session
	return nil
}

// Delete implements ports.Sessions.
func (s *SessionStore) Delete(_ context.Context, tokenHash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byHash, tokenHash)
	return nil
}

// SequentialTokens is a deterministic ports.Tokens.
type SequentialTokens struct{ SequentialIDs }

// NewSequentialTokens returns tokens id-1, id-2, ...
func NewSequentialTokens() *SequentialTokens { return &SequentialTokens{} }

// NewToken implements ports.Tokens.
func (g *SequentialTokens) NewToken() string { return "token-" + g.NewID() }

// CollectionStore is an in-memory ports.ImportBatches and ports.CollectionReader.
type CollectionStore struct {
	mu       sync.Mutex
	batches  map[string]domain.ImportBatch
	pokemon  map[domain.TrainerID][]domain.Pokemon
	mappings map[string]domain.ColumnMapping
}

// NewCollectionStore returns an empty store.
func NewCollectionStore() *CollectionStore {
	return &CollectionStore{
		batches: map[string]domain.ImportBatch{}, pokemon: map[domain.TrainerID][]domain.Pokemon{},
		mappings: map[string]domain.ColumnMapping{},
	}
}

// Save implements ports.ImportBatches.
func (s *CollectionStore) Save(_ context.Context, b domain.ImportBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches[b.ID] = b
	return nil
}

// Get implements ports.ImportBatches.
func (s *CollectionStore) Get(_ context.Context, trainer domain.TrainerID, id string) (domain.ImportBatch, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.batches[id]
	if !ok || b.TrainerID != trainer {
		return domain.ImportBatch{}, domain.ErrImportNotFound
	}
	return b, nil
}

// SaveResolutions implements ports.ImportBatches.
func (s *CollectionStore) SaveResolutions(_ context.Context, b domain.ImportBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	stored := s.batches[b.ID]
	stored.Resolutions = b.Resolutions
	s.batches[b.ID] = stored
	return nil
}

// SaveMapping implements ports.ImportBatches.
func (s *CollectionStore) SaveMapping(_ context.Context, b domain.ImportBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches[b.ID] = b
	s.mappings[string(b.TrainerID)+"/"+b.Table.Signature()] = b.Mapping
	return nil
}

// SavedMapping implements ports.ImportBatches.
func (s *CollectionStore) SavedMapping(_ context.Context, trainer domain.TrainerID, signature string) (domain.ColumnMapping, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.mappings[string(trainer)+"/"+signature], nil
}

// Commit implements ports.ImportBatches.
func (s *CollectionStore) Commit(_ context.Context, b domain.ImportBatch, add []domain.Pokemon, update []domain.PokemonUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.addObservations(b.TrainerID, update); err != nil {
		return err
	}
	s.batches[b.ID] = b
	s.pokemon[b.TrainerID] = append(s.pokemon[b.TrainerID], add...)
	return nil
}

// AddObservations implements ports.CollectionWriter.
func (s *CollectionStore) AddObservations(_ context.Context, trainer domain.TrainerID, updates []domain.PokemonUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.addObservations(trainer, updates)
}

// addObservations applies all updates or none; s.mu must be held.
func (s *CollectionStore) addObservations(trainer domain.TrainerID, updates []domain.PokemonUpdate) error {
	owned := slices.Clone(s.pokemon[trainer])
	for _, u := range updates {
		i := slices.IndexFunc(owned, func(p domain.Pokemon) bool { return p.ID == u.PokemonID })
		if i < 0 {
			return fmt.Errorf("update %s: not in the collection", u.PokemonID)
		}
		owned[i].Observations = append(slices.Clone(owned[i].Observations), u.Observations...)
	}
	s.pokemon[trainer] = owned
	return nil
}

// SetProtected implements ports.CollectionWriter.
func (s *CollectionStore) SetProtected(_ context.Context, trainer domain.TrainerID, id domain.PokemonID, protected bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	owned := slices.Clone(s.pokemon[trainer])
	i := slices.IndexFunc(owned, func(p domain.Pokemon) bool { return p.ID == id })
	if i < 0 {
		return domain.ErrPokemonNotFound
	}
	owned[i].Protected = protected
	s.pokemon[trainer] = owned
	return nil
}

// ListPokemon implements ports.CollectionReader.
func (s *CollectionStore) ListPokemon(_ context.Context, trainer domain.TrainerID) ([]domain.Pokemon, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.Pokemon(nil), s.pokemon[trainer]...), nil
}

// Setups is an in-memory ports.Setups.
type Setups struct {
	mu     sync.Mutex
	setups map[domain.TrainerID]domain.Setup
}

// NewSetups returns an empty store.
func NewSetups() *Setups { return &Setups{setups: map[domain.TrainerID]domain.Setup{}} }

// GetSetup implements ports.Setups.
func (s *Setups) GetSetup(_ context.Context, trainer domain.TrainerID) (domain.Setup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	setup, ok := s.setups[trainer]
	if !ok {
		return domain.Setup{}, domain.ErrSetupNotFound
	}
	return setup, nil
}

// SaveSetup implements ports.Setups.
func (s *Setups) SaveSetup(_ context.Context, trainer domain.TrainerID, setup domain.Setup) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setups[trainer] = setup
	return nil
}

// SpeciesCategories is a ports.Species fake naming each listed species'
// category.
type SpeciesCategories map[string]domain.Category

// Category implements ports.Species.
func (c SpeciesCategories) Category(species string) domain.Category { return c[species] }

// Teams is an in-memory ports.Teams that checks members against a
// CollectionStore.
type Teams struct {
	mu         sync.Mutex
	collection *CollectionStore
	teams      map[domain.TrainerID][]domain.Team
}

// NewTeams returns an empty store whose members must be in collection.
func NewTeams(collection *CollectionStore) *Teams {
	return &Teams{collection: collection, teams: map[domain.TrainerID][]domain.Team{}}
}

// ListTeams implements ports.Teams.
func (s *Teams) ListTeams(_ context.Context, trainer domain.TrainerID) ([]domain.Team, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := slices.Clone(s.teams[trainer])
	slices.SortFunc(out, func(a, b domain.Team) int { return strings.Compare(strings.ToLower(a.Name), strings.ToLower(b.Name)) })
	return out, nil
}

// SaveTeam implements ports.Teams.
func (s *Teams) SaveTeam(ctx context.Context, trainer domain.TrainerID, t domain.Team) error {
	owned, _ := s.collection.ListPokemon(ctx, trainer)
	for _, m := range t.Members {
		if !slices.ContainsFunc(owned, func(p domain.Pokemon) bool { return p.ID == m }) {
			return domain.ErrPokemonNotFound
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for other, teams := range s.teams {
		if other != trainer && slices.ContainsFunc(teams, func(x domain.Team) bool { return x.ID == t.ID }) {
			return domain.ErrTeamNotFound
		}
	}
	teams := slices.DeleteFunc(slices.Clone(s.teams[trainer]), func(x domain.Team) bool { return x.ID == t.ID })
	s.teams[trainer] = append(teams, t)
	return nil
}

// DeleteTeam implements ports.Teams.
func (s *Teams) DeleteTeam(_ context.Context, trainer domain.TrainerID, id domain.TeamID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := len(s.teams[trainer])
	s.teams[trainer] = slices.DeleteFunc(slices.Clone(s.teams[trainer]), func(x domain.Team) bool { return x.ID == id })
	if len(s.teams[trainer]) == before {
		return domain.ErrTeamNotFound
	}
	return nil
}
