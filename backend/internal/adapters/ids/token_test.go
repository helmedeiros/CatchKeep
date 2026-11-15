package ids_test

import (
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/ids"
)

func TestTokens_Are256BitURLSafeAndUnique(t *testing.T) {
	gen := ids.Tokens{}
	seen := map[string]bool{}

	for range 1000 {
		tok := gen.NewToken()
		if len(tok) != 43 {
			t.Fatalf("token %q has length %d, want 43 (32 bytes, base64url)", tok, len(tok))
		}
		if seen[tok] {
			t.Fatal("duplicate token")
		}
		seen[tok] = true
	}
}
