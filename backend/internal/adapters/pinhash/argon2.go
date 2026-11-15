// Package pinhash implements ports.PINHasher with Argon2id.
package pinhash

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Parameters follow the OWASP minimum for Argon2id (19 MiB, 2 iterations).
const (
	memoryKiB   = 19 * 1024
	iterations  = 2
	parallelism = 1
	saltLen     = 16
	keyLen      = 32
)

var errMalformed = errors.New("malformed PIN hash")

// Argon2 hashes PINs with Argon2id and a random salt per hash.
type Argon2 struct{}

// NewArgon2 returns an Argon2id hasher.
func NewArgon2() Argon2 { return Argon2{} }

// Hash returns an encoded hash in the PHC string format.
func (Argon2) Hash(pin string) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("salt: %w", err)
	}
	key := argon2.IDKey([]byte(pin), salt, iterations, memoryKiB, parallelism, keyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, memoryKiB, iterations, parallelism, b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// Verify reports whether pin matches the encoded hash.
func (Argon2) Verify(encoded, pin string) (bool, error) {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false, errMalformed
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false, errMalformed
	}
	var memory, time uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads); err != nil {
		return false, errMalformed
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false, errMalformed
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) == 0 {
		return false, errMalformed
	}
	got := argon2.IDKey([]byte(pin), salt, time, memory, threads, uint32(len(want))) //nolint:gosec // len(want) is a small decoded key length
	return subtle.ConstantTimeCompare(got, want) == 1, nil
}
