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

// SetPINHash implements ports.Trainers.
func (s *Trainers) SetPINHash(_ context.Context, id domain.TrainerID, hash string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.byID[id]
	if !ok {
		return domain.ErrTrainerNotFound
	}
	t.PINHash = hash
	s.byID[id] = t
	return nil
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

// DeleteForTrainer implements ports.Sessions.
func (s *SessionStore) DeleteForTrainer(_ context.Context, trainer domain.TrainerID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for hash, session := range s.byHash {
		if session.TrainerID == trainer {
			delete(s.byHash, hash)
		}
	}
	return nil
}

// DeleteIdleSince implements ports.Sessions.
func (s *SessionStore) DeleteIdleSince(_ context.Context, cutoff time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for hash, session := range s.byHash {
		if session.LastSeenAt.Before(cutoff) {
			delete(s.byHash, hash)
			n++
		}
	}
	return n, nil
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
	// gone holds transferred Pokémon; events the transfer history.
	gone      map[domain.PokemonID]domain.Pokemon
	events    []domain.TransferEvent
	imports   []domain.ImportEvent
	changes   []domain.Change
	storage   map[string]domain.StorageSnapshot
	revisions map[domain.TrainerID]int64
}

// NewCollectionStore returns an empty store.
func NewCollectionStore() *CollectionStore {
	return &CollectionStore{
		batches: map[string]domain.ImportBatch{}, pokemon: map[domain.TrainerID][]domain.Pokemon{},
		mappings: map[string]domain.ColumnMapping{},
		gone:     map[domain.PokemonID]domain.Pokemon{}, revisions: map[domain.TrainerID]int64{},
		storage: map[string]domain.StorageSnapshot{},
	}
}

// Save implements ports.ImportBatches.
func (s *CollectionStore) Save(_ context.Context, b domain.ImportBatch) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches[b.ID] = b
	return nil
}

// DeleteStagedBefore implements ports.ImportBatches.
func (s *CollectionStore) DeleteStagedBefore(_ context.Context, cutoff time.Time) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for id, b := range s.batches {
		if b.Status == domain.ImportStaged && b.CreatedAt.Before(cutoff) {
			delete(s.batches, id)
			n++
		}
	}
	return n, nil
}

// TrimStaged implements ports.ImportBatches.
func (s *CollectionStore) TrimStaged(_ context.Context, trainer domain.TrainerID, keep int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	var staged []domain.ImportBatch
	for _, b := range s.batches {
		if b.TrainerID == trainer && b.Status == domain.ImportStaged {
			staged = append(staged, b)
		}
	}
	slices.SortFunc(staged, func(a, b domain.ImportBatch) int {
		if c := b.CreatedAt.Compare(a.CreatedAt); c != 0 {
			return c
		}
		return strings.Compare(b.ID, a.ID)
	})
	for _, b := range staged[min(keep, len(staged)):] {
		delete(s.batches, b.ID)
	}
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
func (s *CollectionStore) Commit(_ context.Context, b domain.ImportBatch, add []domain.Pokemon, update []domain.PokemonUpdate, e domain.ImportEvent) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.addObservations(b.TrainerID, update); err != nil {
		return err
	}
	s.batches[b.ID] = b
	s.pokemon[b.TrainerID] = append(s.pokemon[b.TrainerID], add...)
	s.imports = append(s.imports, e)
	return nil
}

// ImportEvents returns the import history recorded for trainer.
func (s *CollectionStore) ImportEvents(trainer domain.TrainerID) []domain.ImportEvent {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.ImportEvent
	for _, e := range s.imports {
		if e.TrainerID == trainer {
			out = append(out, e)
		}
	}
	return out
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
func (s *CollectionStore) SetProtected(_ context.Context, trainer domain.TrainerID, id domain.PokemonID, protected bool, c domain.Change) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	owned := slices.Clone(s.pokemon[trainer])
	i := slices.IndexFunc(owned, func(p domain.Pokemon) bool { return p.ID == id })
	if i < 0 {
		return domain.ErrPokemonNotFound
	}
	owned[i].Protected = protected
	s.pokemon[trainer] = owned
	s.changes = append(s.changes, c)
	return nil
}

// AttachStorage implements ports.History.
func (s *CollectionStore) AttachStorage(_ context.Context, _ domain.TrainerID, eventID string, snapshot domain.StorageSnapshot) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.storage[eventID] = snapshot
	return nil
}

// Storage returns the snapshot attached to an event.
func (s *CollectionStore) Storage(eventID string) domain.StorageSnapshot {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.storage[eventID]
}

// StorageRecorder is an app.StorageRecorder that notes the events.
type StorageRecorder struct {
	mu     sync.Mutex
	events []string
}

// RecordStorage implements app.StorageRecorder.
func (r *StorageRecorder) RecordStorage(_ context.Context, _ domain.TrainerID, eventID string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.events = append(r.events, eventID)
}

// Events returns the events storage was recorded after.
func (r *StorageRecorder) Events() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return slices.Clone(r.events)
}

// record keeps a change made through another fake sharing this store.
func (s *CollectionStore) record(c domain.Change) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.changes = append(s.changes, c)
}

// Changes returns the hand-made changes recorded for trainer.
func (s *CollectionStore) Changes(trainer domain.TrainerID) []domain.Change {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.Change
	for _, c := range s.changes {
		if c.TrainerID == trainer {
			out = append(out, c)
		}
	}
	return out
}

// ListPokemon implements ports.CollectionReader.
func (s *CollectionStore) ListPokemon(_ context.Context, trainer domain.TrainerID) ([]domain.Pokemon, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]domain.Pokemon(nil), s.pokemon[trainer]...), nil
}

// Setups is an in-memory ports.Setups.
type Setups struct {
	mu      sync.Mutex
	setups  map[domain.TrainerID]domain.Setup
	changes []domain.Change
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
func (s *Setups) SaveSetup(_ context.Context, trainer domain.TrainerID, setup domain.Setup, c domain.Change) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.setups[trainer] = setup
	s.changes = append(s.changes, c)
	return nil
}

// Changes returns the setup and rule changes recorded for trainer.
func (s *Setups) Changes(trainer domain.TrainerID) []domain.Change {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.Change
	for _, c := range s.changes {
		if c.TrainerID == trainer {
			out = append(out, c)
		}
	}
	return out
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
func (s *Teams) SaveTeam(ctx context.Context, trainer domain.TrainerID, t domain.Team, c domain.Change) error {
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
	s.collection.record(c)
	return nil
}

// DeleteTeam implements ports.Teams.
func (s *Teams) DeleteTeam(_ context.Context, trainer domain.TrainerID, id domain.TeamID, c domain.Change) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	before := len(s.teams[trainer])
	s.teams[trainer] = slices.DeleteFunc(slices.Clone(s.teams[trainer]), func(x domain.Team) bool { return x.ID == id })
	if len(s.teams[trainer]) == before {
		return domain.ErrTeamNotFound
	}
	s.collection.record(c)
	return nil
}

// Autostart is an in-memory ports.Autostart.
type Autostart struct {
	mu sync.Mutex
	on bool
}

// Enabled implements ports.Autostart.
func (a *Autostart) Enabled() (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.on, nil
}

// Enable implements ports.Autostart.
func (a *Autostart) Enable() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.on = true
	return nil
}

// Disable implements ports.Autostart.
func (a *Autostart) Disable() error {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.on = false
	return nil
}

// Revision implements ports.Revisions; it counts recorded transfers and undos.
func (s *CollectionStore) Revision(_ context.Context, trainer domain.TrainerID) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.revisions[trainer], nil
}

// FindByRequest implements ports.Transfers.
func (s *CollectionStore) FindByRequest(_ context.Context, trainer domain.TrainerID, requestID string) (domain.TransferEvent, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.events {
		if e.TrainerID == trainer && e.RequestID != "" && e.RequestID == requestID {
			return e, true, nil
		}
	}
	return domain.TransferEvent{}, false, nil
}

// Record implements ports.Transfers.
func (s *CollectionStore) Record(_ context.Context, e domain.TransferEvent, expected int64) (domain.TransferEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if expected >= 0 && expected != s.revisions[e.TrainerID] {
		return domain.TransferEvent{}, domain.ErrStaleRevision
	}
	list := s.pokemon[e.TrainerID]
	var gone []domain.PokemonID
	for _, t := range e.Pokemon {
		if !slices.ContainsFunc(list, func(p domain.Pokemon) bool { return p.ID == t.ID }) {
			gone = append(gone, t.ID)
		}
	}
	if len(gone) > 0 {
		return domain.TransferEvent{}, &domain.NotActiveError{IDs: gone}
	}
	for _, t := range e.Pokemon {
		i := slices.IndexFunc(list, func(p domain.Pokemon) bool { return p.ID == t.ID })
		s.gone[t.ID] = list[i]
		list = slices.Delete(list, i, i+1)
	}
	s.pokemon[e.TrainerID] = list
	s.events = append(s.events, e)
	s.revisions[e.TrainerID]++
	return e, nil
}

// Undo implements ports.Transfers.
func (s *CollectionStore) Undo(_ context.Context, trainer domain.TrainerID, eventID string, undo domain.TransferEvent) (domain.TransferEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	i := slices.IndexFunc(s.events, func(e domain.TransferEvent) bool { return e.ID == eventID && e.TrainerID == trainer })
	switch {
	case i < 0:
		return domain.TransferEvent{}, domain.ErrTransferNotFound
	case s.events[i].UndoneBy != "":
		return domain.TransferEvent{}, domain.ErrAlreadyUndone
	}
	for _, t := range s.events[i].Pokemon {
		if p, ok := s.gone[t.ID]; ok {
			s.pokemon[trainer] = append(s.pokemon[trainer], p)
			delete(s.gone, t.ID)
		}
	}
	undo.Pokemon, undo.Undoes = s.events[i].Pokemon, eventID
	s.events[i].UndoneBy = undo.ID
	s.events = append(s.events, undo)
	s.revisions[trainer]++
	return undo, nil
}

// ListTransfers implements ports.Transfers.
func (s *CollectionStore) ListTransfers(_ context.Context, trainer domain.TrainerID, limit int) ([]domain.TransferEvent, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []domain.TransferEvent
	for i := len(s.events) - 1; i >= 0 && len(out) < limit; i-- {
		if s.events[i].TrainerID == trainer {
			out = append(out, s.events[i])
		}
	}
	return out, nil
}
