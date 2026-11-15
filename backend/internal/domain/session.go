package domain

import (
	"errors"
	"fmt"
	"time"
)

// Errors returned when signing in or resolving a session.
var (
	ErrWrongPIN  = errors.New("wrong profile or PIN")
	ErrNoSession = errors.New("not signed in")
)

// SessionIdleTimeout is how long a session survives without use (spec SEC-03).
const SessionIdleTimeout = 30 * 24 * time.Hour

// Session links a sign-in token (stored only as a hash) to a trainer.
type Session struct {
	TokenHash  string
	TrainerID  TrainerID
	CreatedAt  time.Time
	LastSeenAt time.Time
}

// Expired reports whether the session has been idle too long at now.
func (s Session) Expired(now time.Time) bool {
	return now.Sub(s.LastSeenAt) > SessionIdleTimeout
}

// LockedOutError means too many wrong PINs were entered for a profile.
type LockedOutError struct {
	Until time.Time
}

func (e *LockedOutError) Error() string {
	return fmt.Sprintf("too many wrong PINs; try again after %s", e.Until.Format(time.RFC3339))
}

// PIN lockout policy (spec SEC-02): 5 failures lock the profile for 1 minute;
// every further failure doubles the lockout, up to 1 hour. Success resets.
const (
	maxFailuresBeforeLockout = 5
	firstLockout             = time.Minute
	maxLockout               = time.Hour
)

// PINAttempts tracks failed PIN entries for one profile.
type PINAttempts struct {
	failures    int
	lockouts    int
	lockedUntil time.Time
}

// Check returns a *LockedOutError if attempts are blocked at now.
func (a *PINAttempts) Check(now time.Time) error {
	if now.Before(a.lockedUntil) {
		return &LockedOutError{Until: a.lockedUntil}
	}
	return nil
}

// Fail records a wrong PIN at now.
func (a *PINAttempts) Fail(now time.Time) {
	a.failures++
	if a.failures < maxFailuresBeforeLockout {
		return
	}
	lockout := firstLockout << a.lockouts
	if lockout > maxLockout || lockout <= 0 {
		lockout = maxLockout
	}
	a.lockouts++
	a.lockedUntil = now.Add(lockout)
}

// Succeed records a correct PIN and clears the history.
func (a *PINAttempts) Succeed() {
	*a = PINAttempts{}
}
