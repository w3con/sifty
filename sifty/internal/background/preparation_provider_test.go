package background

import (
	"fmt"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/prepare"
)

// stubInferFn is a prepare.Inference stub that always fails so we can detect provider resolution
// without needing a real candidate context or model backend.
var stubInferFn prepare.Inference = func(p prepare.Provider, prompt string, schema any) (prepare.ModelOutput, error) {
	return prepare.ModelOutput{}, fmt.Errorf("stub inference")
}

// noop is a no-op progress function.
var noop = func(msg string) {}

// TestDefaultPrepareJobProviderResolution verifies that defaultPrepareJob:
//   - defaults to cfg.Providers.Stages.Preparation when no provider is in the request
//   - uses the request provider when one is present
//   - errors clearly when neither the request nor the stage config provides a provider
func TestDefaultPrepareJobProviderResolution(t *testing.T) {
	t.Run("no request provider — uses stages.preparation", func(t *testing.T) {
		s := newTestStore(t)
		d := newTestDirs(t)
		jobID, err := s.Upsert(model.JobInput{
			Title: "Test", Company: "Co", OriginalURL: "https://example.com/x",
			Description: "desc", DescriptionStatus: "full", Source: "manual",
		}, false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}

		cfg := minimalTestCfg()
		cfg.Providers.Stages.Preparation = "p"

		// no "provider" key in request
		request := map[string]any{"generate": true}
		err = defaultPrepareJob(s, d, cfg, jobID, 0, request, noop, stubInferFn)
		// We expect an error (no candidate context), but NOT a provider-resolution error.
		if err == nil {
			t.Fatal("expected an error (no candidate context), got nil")
		}
		if strings.Contains(err.Error(), "no preparation provider") {
			t.Fatalf("got unexpected 'no preparation provider' error: %v", err)
		}
		if strings.Contains(err.Error(), "unknown preparation provider") {
			t.Fatalf("got unexpected 'unknown preparation provider' error: %v", err)
		}
	})

	t.Run("request provider overrides stages.preparation", func(t *testing.T) {
		s := newTestStore(t)
		d := newTestDirs(t)
		jobID, err := s.Upsert(model.JobInput{
			Title: "Test", Company: "Co", OriginalURL: "https://example.com/y",
			Description: "desc", DescriptionStatus: "full", Source: "manual",
		}, false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}

		cfg := minimalTestCfg()
		cfg.Providers.Stages.Preparation = "p"
		cfg.Providers.Providers["other"] = config.Provider{
			Type: "ollama", Model: "other-model",
			Endpoint: "http://127.0.0.1:11434", ContextChars: 4096,
		}

		request := map[string]any{"generate": true, "provider": "other"}
		err = defaultPrepareJob(s, d, cfg, jobID, 0, request, noop, stubInferFn)
		// Not a provider-resolution error
		if err == nil {
			t.Fatal("expected an error (no candidate context), got nil")
		}
		if strings.Contains(err.Error(), "no preparation provider") {
			t.Fatalf("got unexpected 'no preparation provider' error: %v", err)
		}
		if strings.Contains(err.Error(), "unknown preparation provider") {
			t.Fatalf("got unexpected 'unknown preparation provider' error: %v", err)
		}
	})

	t.Run("no request provider and no stages.preparation — clear error", func(t *testing.T) {
		s := newTestStore(t)
		d := newTestDirs(t)
		jobID, err := s.Upsert(model.JobInput{
			Title: "Test", Company: "Co", OriginalURL: "https://example.com/z",
			Description: "desc", DescriptionStatus: "full", Source: "manual",
		}, false)
		if err != nil {
			t.Fatalf("Upsert: %v", err)
		}

		cfg := minimalTestCfg()
		cfg.Providers.Stages.Preparation = "" // explicitly unset

		request := map[string]any{"generate": true} // no provider key
		err = defaultPrepareJob(s, d, cfg, jobID, 0, request, noop, stubInferFn)
		if err == nil {
			t.Fatal("expected error when neither request nor config provides a provider, got nil")
		}
		if !strings.Contains(err.Error(), "no preparation provider") {
			t.Fatalf("expected 'no preparation provider' in error, got: %v", err)
		}
	})
}
