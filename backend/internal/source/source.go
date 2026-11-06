// Package source defines the Source interface and the registry that selects an
// implementation by host pattern. Every source implements Discover and Resolve;
// the collection runner and the single-URL import path use the registry to route
// requests without knowing which platform is on the other end.
package source

import (
	"context"
	"regexp"
	"sync"

	"github.com/w3con/sifty/internal/model"
)

// Ref is a reference to a single posting discovered from a listing page. It
// carries enough information for the runner to decide whether to call Resolve or
// to use the partial JobInput directly when the listing page already contained
// all required fields.
type Ref struct {
	// URL is the canonical URL of the individual posting.
	URL string
	// Source is the source name, typically the config site name set by the runner.
	Source string
	// Partial may contain fields already extracted from the listing page. When
	// every required field (Title, Company, Description) is populated, Resolve
	// must return this value without a network fetch.
	Partial model.JobInput
}

// Source is the interface every collection backend implements.
type Source interface {
	// Discover fetches the listing at listingURL and returns a slice of Refs.
	// Returning an empty slice is valid; returning an error signals a retriable
	// failure, not an empty board.
	Discover(ctx context.Context, listingURL string) ([]Ref, error)

	// Resolve fetches and normalises the posting identified by ref. It must
	// return ref.Partial without a network fetch when all required fields are
	// already populated. The returned JobInput has its Source field set by the
	// caller (the runner), not by the implementation.
	Resolve(ctx context.Context, ref Ref) (model.JobInput, error)
}

// Registry maps host patterns to Source implementations and falls back to a
// default when no pattern matches.
type Registry struct {
	mu       sync.RWMutex
	entries  []registryEntry
	fallback Source
}

type registryEntry struct {
	pattern *regexp.Regexp
	source  Source
}

// NewRegistry creates a Registry. fallback is returned for any host that matches
// no registered pattern; it must not be nil.
func NewRegistry(fallback Source) *Registry {
	return &Registry{fallback: fallback}
}

// Register adds a Source for all hosts whose full hostname matches pattern
// (anchored automatically). Patterns are matched in registration order; the
// first match wins.
func (r *Registry) Register(pattern string, s Source) {
	re := regexp.MustCompile(`(?i)` + pattern)
	r.mu.Lock()
	defer r.mu.Unlock()
	r.entries = append(r.entries, registryEntry{pattern: re, source: s})
}

// For returns the Source whose pattern matches host, or the fallback Source.
func (r *Registry) For(host string) Source {
	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, e := range r.entries {
		if e.pattern.MatchString(host) {
			return e.source
		}
	}
	return r.fallback
}
