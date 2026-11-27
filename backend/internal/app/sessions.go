package app

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
	"github.com/helmedeiros/catchkeep/backend/internal/ports"
)

// Sessions signs trainers in with their PIN and resolves session tokens.
type Sessions struct {
	trainers ports.Trainers
	sessions ports.Sessions
	log      ports.SignInLog
	hasher   ports.PINHasher
	tokens   ports.Tokens
	clock    ports.Clock

	mu       sync.Mutex
	attempts map[domain.TrainerID]*domain.PINAttempts
	devices  map[string]*domain.DeviceAttempts
}

// maxDevices bounds the devices remembered; idle ones are dropped first.
const maxDevices = 1024

// NewSessions wires the sign-in use cases. PIN attempt history is kept in
// memory, so a host restart clears lockouts.
func NewSessions(trainers ports.Trainers, sessions ports.Sessions, log ports.SignInLog, hasher ports.PINHasher, tokens ports.Tokens, clock ports.Clock) *Sessions {
	return &Sessions{
		trainers: trainers, sessions: sessions, log: log, hasher: hasher, tokens: tokens, clock: clock,
		attempts: map[domain.TrainerID]*domain.PINAttempts{},
		devices:  map[string]*domain.DeviceAttempts{},
	}
}

// SignIn checks the PIN for the profile and, if right, starts a session and
// returns its token. A wrong PIN and an unknown profile both return
// domain.ErrWrongPIN; repeated failures for the profile, or from the device
// on any profiles, return *domain.LockedOutError. Sign-ins, wrong PINs and
// the start of each pause go in the sign-in log.
func (s *Sessions) SignIn(ctx context.Context, device string, id domain.TrainerID, pin string) (string, ProfileSummary, error) {
	now := s.clock.Now()
	if err := s.check(device, id, now); err != nil {
		return "", ProfileSummary{}, err
	}

	trainer, err := s.trainers.Get(ctx, id)
	if errors.Is(err, domain.ErrTrainerNotFound) {
		return "", ProfileSummary{}, s.wrongPIN(ctx, device, "", now)
	}
	if err != nil {
		return "", ProfileSummary{}, err
	}
	ok, err := s.hasher.Verify(trainer.PINHash, pin)
	if err != nil {
		return "", ProfileSummary{}, fmt.Errorf("verify PIN: %w", err)
	}
	if !ok {
		return "", ProfileSummary{}, s.wrongPIN(ctx, device, id, now)
	}
	s.mu.Lock()
	delete(s.attempts, id)
	s.mu.Unlock()

	token := s.tokens.NewToken()
	session := domain.Session{TokenHash: hashToken(token), TrainerID: trainer.ID, CreatedAt: now, LastSeenAt: now}
	if err := s.sessions.Create(ctx, session); err != nil {
		return "", ProfileSummary{}, fmt.Errorf("create session: %w", err)
	}
	if err := s.record(ctx, domain.SignedIn, device, trainer.ID, now); err != nil {
		return "", ProfileSummary{}, err
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

// ResetPIN gives the trainer a new PIN, ends all their sessions and lifts
// any lockout; the sign-in log notes it. Only the host admin may call it.
func (s *Sessions) ResetPIN(ctx context.Context, device string, id domain.TrainerID, pin string) error {
	if err := domain.ValidatePIN(pin); err != nil {
		return err
	}
	hash, err := s.hasher.Hash(pin)
	if err != nil {
		return fmt.Errorf("hash PIN: %w", err)
	}
	if err := s.trainers.SetPINHash(ctx, id, hash); err != nil {
		return err
	}
	if err := s.sessions.DeleteForTrainer(ctx, id); err != nil {
		return fmt.Errorf("end sessions: %w", err)
	}
	s.mu.Lock()
	delete(s.attempts, id)
	s.mu.Unlock()
	return s.record(ctx, domain.PINReset, device, id, s.clock.Now())
}

// SignOut ends the session for token. Unknown tokens are ignored.
func (s *Sessions) SignOut(ctx context.Context, token string) error {
	return s.sessions.Delete(ctx, hashToken(token))
}

// wrongPIN counts a wrong PIN, notes it and any pause it starts in the
// sign-in log, and returns domain.ErrWrongPIN.
func (s *Sessions) wrongPIN(ctx context.Context, device string, id domain.TrainerID, now time.Time) error {
	s.fail(device, id, now)
	if err := s.record(ctx, domain.WrongPIN, device, id, now); err != nil {
		return err
	}
	if s.check(device, id, now) != nil {
		if err := s.record(ctx, domain.SignInPaused, device, id, now); err != nil {
			return err
		}
	}
	return domain.ErrWrongPIN
}

func (s *Sessions) record(ctx context.Context, kind domain.SignInKind, device string, id domain.TrainerID, now time.Time) error {
	if err := s.log.Record(ctx, domain.SignInEntry{Kind: kind, TrainerID: id, Device: device, At: now}); err != nil {
		return fmt.Errorf("sign-in log: %w", err)
	}
	return nil
}

// check refuses a device or profile that is paused at now.
func (s *Sessions) check(device string, id domain.TrainerID, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if d, ok := s.devices[device]; ok {
		if err := d.Check(now); err != nil {
			return err
		}
	}
	if a, ok := s.attempts[id]; ok {
		return a.Check(now)
	}
	return nil
}

// fail counts a wrong PIN against the device and, when id is an existing
// profile, against it, so made-up ids cannot grow the profile map.
func (s *Sessions) fail(device string, id domain.TrainerID, now time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	d, ok := s.devices[device]
	if !ok {
		s.forgetIdleDevices(now)
		d = &domain.DeviceAttempts{}
		s.devices[device] = d
	}
	d.Fail(now)
	if id == "" {
		return
	}
	a, ok := s.attempts[id]
	if !ok {
		a = &domain.PINAttempts{}
		s.attempts[id] = a
	}
	a.Fail(now)
}

func (s *Sessions) forgetIdleDevices(now time.Time) {
	if len(s.devices) < maxDevices {
		return
	}
	for device, d := range s.devices {
		if d.Idle(now) {
			delete(s.devices, device)
		}
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// signInLogLength is how many sign-in log entries the host sees.
const signInLogLength = 200

// SignInRecord is a sign-in log entry with the profile's name now, "" for
// a profile that does not exist.
type SignInRecord struct {
	domain.SignInEntry
	ProfileName string
}

// SignIns returns the latest sign-in log entries, newest first. Only the
// host admin may call it.
func (s *Sessions) SignIns(ctx context.Context) ([]SignInRecord, error) {
	entries, err := s.log.List(ctx, signInLogLength)
	if err != nil {
		return nil, err
	}
	trainers, err := s.trainers.List(ctx)
	if err != nil {
		return nil, err
	}
	names := make(map[domain.TrainerID]string, len(trainers))
	for _, t := range trainers {
		names[t.ID] = t.DisplayName
	}
	out := make([]SignInRecord, len(entries))
	for i, e := range entries {
		out[i] = SignInRecord{SignInEntry: e, ProfileName: names[e.TrainerID]}
	}
	return out, nil
}
