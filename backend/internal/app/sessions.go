package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Sessions signs trainers in with their PIN and resolves session tokens.
type Sessions struct {
	trainers ports.Trainers
	sessions ports.Sessions
	hasher   ports.PINHasher
	tokens   ports.Tokens
	clock    ports.Clock

	mu       sync.Mutex
	attempts map[domain.TrainerID]*domain.PINAttempts
}

// NewSessions wires the sign-in use cases. PIN attempt history is kept in
// memory, so a host restart clears lockouts.
func NewSessions(trainers ports.Trainers, sessions ports.Sessions, hasher ports.PINHasher, tokens ports.Tokens, clock ports.Clock) *Sessions {
	return &Sessions{
		trainers: trainers, sessions: sessions, hasher: hasher, tokens: tokens, clock: clock,
		attempts: map[domain.TrainerID]*domain.PINAttempts{},
	}
}

// SignIn checks the PIN for the profile and, if right, starts a session and
// returns its token. A wrong PIN and an unknown profile both return
// domain.ErrWrongPIN; repeated failures return *domain.LockedOutError.
func (s *Sessions) SignIn(ctx context.Context, id domain.TrainerID, pin string) (string, ProfileSummary, error) {
	now := s.clock.Now()
	s.mu.Lock()
	attempts := s.attemptsFor(id)
	if err := attempts.Check(now); err != nil {
		s.mu.Unlock()
		return "", ProfileSummary{}, err
	}
	s.mu.Unlock()

	trainer, err := s.trainers.Get(ctx, id)
	if errors.Is(err, domain.ErrTrainerNotFound) {
		return "", ProfileSummary{}, domain.ErrWrongPIN
	}
	if err != nil {
		return "", ProfileSummary{}, err
	}
	ok, err := s.hasher.Verify(trainer.PINHash, pin)
	if err != nil {
		return "", ProfileSummary{}, fmt.Errorf("verify PIN: %w", err)
	}

	s.mu.Lock()
	if !ok {
		attempts.Fail(now)
		s.mu.Unlock()
		return "", ProfileSummary{}, domain.ErrWrongPIN
	}
	attempts.Succeed()
	s.mu.Unlock()

	token := s.tokens.NewToken()
	session := domain.Session{TokenHash: hashToken(token), TrainerID: trainer.ID, CreatedAt: now, LastSeenAt: now}
	if err := s.sessions.Create(ctx, session); err != nil {
		return "", ProfileSummary{}, fmt.Errorf("create session: %w", err)
	}
	return token, ProfileSummary{ID: trainer.ID, DisplayName: trainer.DisplayName}, nil
}

// Resolve returns the profile a token is signed in to and records the use.
// Unknown or expired tokens return domain.ErrNoSession.
func (s *Sessions) Resolve(ctx context.Context, token string) (ProfileSummary, error) {
	now := s.clock.Now()
	hash := hashToken(token)
	session, err := s.sessions.Get(ctx, hash)
	if err != nil {
		return ProfileSummary{}, err
	}
	if session.Expired(now) {
		_ = s.sessions.Delete(ctx, hash)
		return ProfileSummary{}, domain.ErrNoSession
	}
	trainer, err := s.trainers.Get(ctx, session.TrainerID)
	if errors.Is(err, domain.ErrTrainerNotFound) {
		return ProfileSummary{}, domain.ErrNoSession
	}
	if err != nil {
		return ProfileSummary{}, err
	}
	if err := s.sessions.Touch(ctx, hash, now); err != nil {
		return ProfileSummary{}, fmt.Errorf("touch session: %w", err)
	}
	return ProfileSummary{ID: trainer.ID, DisplayName: trainer.DisplayName}, nil
}

// SignOut ends the session for token. Unknown tokens are ignored.
func (s *Sessions) SignOut(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, hashToken(token))
}

func (s *Sessions) attemptsFor(id domain.TrainerID) *domain.PINAttempts {
	a, ok := s.attempts[id]
	if !ok {
		a = &domain.PINAttempts{}
		s.attempts[id] = a
	}
	return a
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
