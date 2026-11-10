package profile_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
)

// TestLiveExtractionProbe is an explicitly gated, single-call diagnostic. It runs the real
// configured extraction stage from SIFTY_PROBE_CONFIG_DIR through the application service against
// a disposable store/data directory and a short synthetic document. It never reads or writes the
// live store or profile. Set the environment variable only when a measured provider check is
// intended; every ordinary test run skips it.
func TestLiveExtractionProbe(t *testing.T) {
	configDir := os.Getenv("SIFTY_PROBE_CONFIG_DIR")
	if configDir == "" {
		t.Skip("set SIFTY_PROBE_CONFIG_DIR to run the one-call live extraction probe")
	}
	cfgDirs, err := dirs.Resolve(configDir)
	if err != nil {
		t.Fatalf("resolve config dir: %v", err)
	}
	cfg, err := config.Load(cfgDirs)
	if err != nil {
		t.Fatalf("load configured providers: %v", err)
	}
	providerName := cfg.Providers.Stages.Extraction
	provider, ok := cfg.Providers.Providers[providerName]
	if !ok {
		t.Fatalf("configured extraction stage %q is not defined", providerName)
	}

	scratch, err := dirs.Resolve(t.TempDir())
	if err != nil {
		t.Fatalf("resolve scratch dir: %v", err)
	}
	if err := scratch.Ensure(); err != nil {
		t.Fatalf("prepare scratch dir: %v", err)
	}
	s, err := store.Open(filepath.Join(scratch.Root, "store.sqlite"))
	if err != nil {
		t.Fatalf("open scratch store: %v", err)
	}
	defer s.Close()

	svc := profile.NewService(s, config.NewHolder(cfg), profile.WithDirs(scratch))
	doc, err := svc.CreateDocument("probe.txt", []byte("Name: Probe Candidate\nSkills: Go, PostgreSQL\nExperience: 6 years as a backend engineer.\n"))
	if err != nil {
		t.Fatalf("create scratch document: %v", err)
	}
	t.Logf("configured stage=%s type=%s model=%s timeout_ms=%d context_chars=%d scratch=%s",
		providerName, provider.Type, provider.Model, provider.TimeoutMS, provider.ContextChars, scratch.Root)

	started := time.Now()
	if _, err := svc.AnalyzeDocument(doc.ID); err != nil {
		t.Fatalf("start analysis: %v", err)
	}
	var final *profile.Document
	for time.Since(started) < 150*time.Second {
		current, err := svc.GetDocument(doc.ID)
		if err != nil {
			t.Fatalf("poll document: %v", err)
		}
		if current.ProcessingState != profile.DocumentAnalyzing && current.ProcessingState != profile.DocumentExtracting {
			final = current
			break
		}
		time.Sleep(time.Second)
	}
	if final == nil {
		t.Fatalf("analysis did not reach a terminal state within 150s")
	}
	elapsed := time.Since(started)
	facts, _ := svc.ListFacts(doc.ID)
	t.Logf("elapsed=%s state=%s facts=%d error=%q metadata=%v",
		elapsed.Round(time.Millisecond), final.ProcessingState, len(facts), final.Error, final.AnalysisMetadata)
	if final.ProcessingState == profile.DocumentAnalyzed {
		t.Logf("proposal_count=%d", len(facts))
	}
}
