package ids_test

import (
	"regexp"
	"testing"

	"github.com/helmedeiros/catchkeep/backend/internal/adapters/ids"
)

// Same pattern as ProfileId in api/openapi.yaml.
var ulidPattern = regexp.MustCompile(`^[0-9A-HJKMNP-TV-Z]{26}$`)

func TestULID_GeneratesUniqueContractShapedIDs(t *testing.T) {
	gen := ids.NewULID()
	seen := map[string]bool{}

	for range 1000 {
		id := gen.NewID()
		if !ulidPattern.MatchString(id) {
			t.Fatalf("id %q does not match the contract pattern", id)
		}
		if seen[id] {
			t.Fatalf("duplicate id %q", id)
		}
		seen[id] = true
	}
}
