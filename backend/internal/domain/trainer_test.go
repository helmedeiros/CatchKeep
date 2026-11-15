package domain_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/domain"
)

func TestNewDisplayName_TrimsAndAcceptsUpTo40Characters(t *testing.T) {
	for _, in := range []string{"Ash", "  Misty  ", strings.Repeat("é", 40)} {
		got, err := domain.NewDisplayName(in)
		if err != nil {
			t.Fatalf("NewDisplayName(%q) error: %v", in, err)
		}
		if got != strings.TrimSpace(in) {
			t.Fatalf("NewDisplayName(%q) = %q", in, got)
		}
	}
}

func TestNewDisplayName_RejectsEmptyTooLongOrControlCharacters(t *testing.T) {
	for _, in := range []string{"", "   ", strings.Repeat("a", 41), "A\nsh", "A\x00b"} {
		if _, err := domain.NewDisplayName(in); !errors.Is(err, domain.ErrInvalidDisplayName) {
			t.Fatalf("NewDisplayName(%q) error = %v, want ErrInvalidDisplayName", in, err)
		}
	}
}

func TestValidatePIN_AcceptsFourToEightDigits(t *testing.T) {
	for _, pin := range []string{"0000", "1234", "12345678"} {
		if err := domain.ValidatePIN(pin); err != nil {
			t.Fatalf("ValidatePIN(%q) error: %v", pin, err)
		}
	}
}

func TestValidatePIN_RejectsWrongLengthOrNonDigits(t *testing.T) {
	for _, pin := range []string{"", "123", "123456789", "12a4", "１２３４", "12 4"} {
		if err := domain.ValidatePIN(pin); !errors.Is(err, domain.ErrInvalidPIN) {
			t.Fatalf("ValidatePIN(%q) error = %v, want ErrInvalidPIN", pin, err)
		}
	}
}
