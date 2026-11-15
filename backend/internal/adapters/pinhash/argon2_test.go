package pinhash_test

import (
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/pinhash"
)

func TestArgon2_VerifiesTheRightPINOnly(t *testing.T) {
	h := pinhash.NewArgon2()

	hash, err := h.Hash("2468")
	if err != nil {
		t.Fatal(err)
	}

	if ok, err := h.Verify(hash, "2468"); err != nil || !ok {
		t.Fatalf("Verify(right) = %v, %v", ok, err)
	}
	if ok, err := h.Verify(hash, "1357"); err != nil || ok {
		t.Fatalf("Verify(wrong) = %v, %v", ok, err)
	}
}

func TestArgon2_SaltsEachHash(t *testing.T) {
	h := pinhash.NewArgon2()

	a, _ := h.Hash("2468")
	b, _ := h.Hash("2468")

	if a == b {
		t.Fatal("two hashes of the same PIN are identical; want random salt")
	}
}

func TestArgon2_RejectsMalformedHash(t *testing.T) {
	if _, err := pinhash.NewArgon2().Verify("not-a-hash", "2468"); err == nil {
		t.Fatal("Verify(malformed) returned nil error")
	}
}
