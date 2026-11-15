package ids

import (
	"crypto/rand"
	"encoding/base64"
)

// Tokens generates 256-bit random session tokens, base64url-encoded.
type Tokens struct{}

// NewToken implements ports.Tokens.
func (Tokens) NewToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b) // crypto/rand.Read never returns an error
	return base64.RawURLEncoding.EncodeToString(b)
}
