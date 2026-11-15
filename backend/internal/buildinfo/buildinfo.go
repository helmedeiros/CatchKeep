// Package buildinfo exposes values stamped into the binary at build time.
package buildinfo

// version is overridden at build time with
// -ldflags "-X github.com/helmedeiros/catchkeep/backend/internal/buildinfo.version=<v>".
var version = "dev"

// Version returns the build version of the running binary.
func Version() string {
	return version
}
