package domain

import "time"

// SignInKind names an entry in the sign-in log.
type SignInKind string

// What the sign-in log records (spec §13 sign-in log, SEC-10).
const (
	SignedIn       SignInKind = "signed_in"
	WrongPIN       SignInKind = "wrong_pin"
	SignInPaused   SignInKind = "paused"
	ProfileCreated SignInKind = "profile_created"
	PINReset       SignInKind = "pin_reset"
)

// SignInLogLifetime is how long the sign-in log keeps an entry.
const SignInLogLifetime = 90 * 24 * time.Hour

// SignInEntry is one entry in the sign-in log: what happened, to which
// profile ("" for a profile that does not exist), from which device and
// when. It never holds a PIN.
type SignInEntry struct {
	Kind      SignInKind
	TrainerID TrainerID
	Device    string
	At        time.Time
}
