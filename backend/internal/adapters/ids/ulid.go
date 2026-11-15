// Package ids implements ports.IDs and ports.Clock for production use.
package ids

import (
	"crypto/rand"
	"sync"
	"time"

	"github.com/oklog/ulid/v2"
)

// ULID generates monotonic, lexicographically sortable ULIDs.
type ULID struct {
	mu      sync.Mutex
	entropy *ulid.MonotonicEntropy
}

// NewULID returns a ULID generator seeded from crypto/rand.
func NewULID() *ULID {
	return &ULID{entropy: ulid.Monotonic(rand.Reader, 0)}
}

// NewID implements ports.IDs.
func (g *ULID) NewID() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	return ulid.MustNew(ulid.Timestamp(time.Now()), g.entropy).String()
}

// SystemClock is the wall clock.
type SystemClock struct{}

// Now implements ports.Clock.
func (SystemClock) Now() time.Time { return time.Now().UTC() }
