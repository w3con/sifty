package config

import (
	"sync"
	"sync/atomic"

	"github.com/w3con/sifty/internal/dirs"
)

// Holder holds a *Config behind an atomic pointer, allowing concurrent reads
// and lock-free swaps without data races. All callers share one Holder; each
// config reload calls Set with the freshly loaded *Config.
type Holder struct {
	p  atomic.Pointer[Config]
	mu sync.RWMutex
}

// NewHolder returns a Holder initialised with cfg. cfg must not be nil.
func NewHolder(cfg *Config) *Holder {
	h := &Holder{}
	h.p.Store(cfg)
	return h
}

// Get returns the current *Config snapshot. Callers should snapshot the pointer
// once at the start of a request or worker tick rather than calling Get repeatedly.
func (h *Holder) Get() *Config {
	return h.p.Load()
}

// Set atomically replaces the held config with cfg. cfg must not be nil.
func (h *Holder) Set(cfg *Config) {
	h.mu.Lock()
	h.p.Store(cfg)
	h.mu.Unlock()
}

// WithSnapshot runs fn against the active configuration while blocking Set/Reload for the
// duration. It lets an effect derive identities and write the dependent result under one
// stable configuration snapshot without introducing a second config framework.
func (h *Holder) WithSnapshot(fn func(*Config) error) error {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return fn(h.p.Load())
}

// Reload reads and validates all configuration files from d, and if successful
// atomically replaces the held config with the freshly loaded *Config. If the
// new config has a different port than the current one it still replaces the
// in-memory config (the caller is responsible for surfacing the port-change
// notice). Returns an error if loading or validation fails; the existing config
// is left unchanged.
func (h *Holder) Reload(d dirs.Dirs) error {
	cfg, err := Load(d)
	if err != nil {
		return err
	}
	h.Set(cfg)
	return nil
}
