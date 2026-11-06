// Package version carries the build identity of the sifty binary.
package version

// Version is the human-readable build version. It is overridable at link time with
// -ldflags "-X github.com/w3con/sifty/internal/version.Version=...".
var Version = "0.1.0-dev"
