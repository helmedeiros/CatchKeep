package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

// Errors returned for invalid or conflicting trainer data.
var (
	ErrInvalidDisplayName = errors.New("display name must be 1 to 40 characters without control characters")
	ErrInvalidPIN         = errors.New("PIN must be 4 to 8 digits")
	ErrDisplayNameTaken   = errors.New("display name is already used by another profile")
	ErrTrainerNotFound    = errors.New("trainer not found")
)

const maxDisplayNameRunes = 40

// TrainerID identifies a trainer profile for its whole life.
type TrainerID string

// Trainer is one person's isolated profile on the host.
type Trainer struct {
	ID          TrainerID
	DisplayName string
	PINHash     string
	CreatedAt   time.Time
	// Revision increases with every change that can affect recommendations,
	// so a plan can tell it is stale (spec §5.1).
	Revision int64
}

// NewDisplayName trims s and checks it is a usable display name.
func NewDisplayName(s string) (string, error) {
	name := strings.TrimSpace(s)
	if name == "" || utf8.RuneCountInString(name) > maxDisplayNameRunes {
		return "", ErrInvalidDisplayName
	}
	if strings.IndexFunc(name, unicode.IsControl) >= 0 {
		return "", ErrInvalidDisplayName
	}
	return name, nil
}

// ValidatePIN checks pin is 4 to 8 ASCII digits.
func ValidatePIN(pin string) error {
	if len(pin) < 4 || len(pin) > 8 {
		return ErrInvalidPIN
	}
	for _, c := range []byte(pin) {
		if c < '0' || c > '9' {
			return ErrInvalidPIN
		}
	}
	return nil
}
