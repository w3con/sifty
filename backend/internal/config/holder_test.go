package config_test

import (
	"sync"
	"testing"
	"time"

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

// TestHolderSnapshotBlocksSet verifies that a Quick Match recomputation guard cannot have the
// active configuration swapped between identity derivation and its dependent result write.
func TestHolderSnapshotBlocksSet(t *testing.T) {
	h := config.NewHolder(&config.Config{Runtime: config.Runtime{Port: 1111}})
	entered := make(chan struct{})
	release := make(chan struct{})
	snapshotDone := make(chan struct{})
	go func() {
		_ = h.WithSnapshot(func(cfg *config.Config) error {
			if cfg.Runtime.Port != 1111 {
				t.Errorf("snapshot port = %d, want 1111", cfg.Runtime.Port)
			}
			close(entered)
			<-release
			return nil
		})
		close(snapshotDone)
	}()
	<-entered

	setDone := make(chan struct{})
	go func() {
		h.Set(&config.Config{Runtime: config.Runtime{Port: 2222}})
		close(setDone)
	}()
	select {
	case <-setDone:
		t.Fatal("Set completed while a configuration snapshot was held")
	case <-time.After(20 * time.Millisecond):
	}
	close(release)
	<-snapshotDone
	<-setDone
	if got := h.Get().Runtime.Port; got != 2222 {
		t.Fatalf("post-snapshot port = %d, want 2222", got)
	}
}
