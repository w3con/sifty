package config_test

import (
	"sync"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
)

// tempDirsForHolder creates a minimal temp dirs structure with all required config files.
func tempDirsForHolder(t *testing.T) dirs.Dirs {
	t.Helper()
	d := tempDirs(t)
	writeFixture(t, d, "search", validSearch)
	writeFixture(t, d, "runtime", validRuntime)
	writeFixture(t, d, "providers", validProviders)
	writeFixture(t, d, "candidate", validCandidate)
	return d
}

// TestHolderRace verifies that concurrent reads and swaps on a Holder produce no data race.
// Run with: go test -race ./internal/config/...
func TestHolderRace(t *testing.T) {
	initial := &config.Config{Runtime: config.Runtime{Port: 7373}}
	h := config.NewHolder(initial)

	const (
		numReaders = 8
		numSwaps   = 200
		numReads   = 200
	)
	var wg sync.WaitGroup
	wg.Add(numReaders + 1)

	// Writer goroutine: continuously swaps the config while readers are running.
	go func() {
		defer wg.Done()
		for i := 0; i < numSwaps; i++ {
			h.Set(&config.Config{Runtime: config.Runtime{Port: 7374 + i}})
		}
	}()

	// Reader goroutines: each reads the config snapshot repeatedly.
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numReads; j++ {
				cfg := h.Get()
				_ = cfg.Runtime.Port
			}
		}()
	}

	wg.Wait()
}

// TestHolderReload verifies that after mutating a config file on disk, Reload publishes
// the updated value through the accessor without touching the old snapshot.
func TestHolderReload(t *testing.T) {
	d := tempDirsForHolder(t)

	initial, err := config.Load(d)
	if err != nil {
		t.Fatalf("Load initial: %v", err)
	}
	h := config.NewHolder(initial)

	if got := h.Get().Providers.Providers["primary"].Model; got != "llama3" {
		t.Fatalf("precondition: initial model = %q, want llama3", got)
	}

	// Mutate providers.yaml on disk — change the provider model.
	const updatedProviders = `version: 1
stages:
  assessment: primary
  extraction: primary
providers:
  primary:
    type: ollama
    model: mistral
    endpoint: http://127.0.0.1:11434
`
	writeFixture(t, d, "providers", updatedProviders)

	if err := h.Reload(d); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	got := h.Get().Providers.Providers["primary"].Model
	if got != "mistral" {
		t.Errorf("after Reload: model = %q, want mistral", got)
	}
}
