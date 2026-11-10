package profile

import (
	"errors"
	"testing"

	"github.com/w3con/sifty/internal/config"
)

// TestExtractionProviderDisablesThinking catches the hidden failure where a reasoning model
// spends the whole configured call budget on a hidden trace instead of the schema-constrained
// answer. Extraction must ask the ollama adapter to skip thinking; other stages are not changed.
func TestExtractionProviderDisablesThinking(t *testing.T) {
	cfg := &config.Config{}
	cfg.Providers.Stages.Extraction = "local-model"
	cfg.Providers.Providers = map[string]config.Provider{
		"local-model": {
			Type: "ollama", Model: "qwen3.5:9b", Endpoint: "http://127.0.0.1:11434",
			TimeoutMS: 120000, ContextChars: 32000,
		},
	}
	cfg.Runtime.ProfileExtractionTimeoutMS = 120000

	name, provider, err := extractionProvider(cfg)
	if err != nil {
		t.Fatalf("extractionProvider: %v", err)
	}
	if name != "local-model" {
		t.Fatalf("provider name = %q, want local-model", name)
	}
	if provider.Think == nil || *provider.Think {
		t.Fatalf("extraction provider Think = %v, want explicit false", provider.Think)
	}
	if provider.TimeoutMS != 120000 {
		t.Fatalf("timeout = %d, want configured 120000 (not raised)", provider.TimeoutMS)
	}
}

func TestAnalysisErrorKind(t *testing.T) {
	cases := []struct {
		err  error
		want string
	}{
		{errors.New(`extraction provider "qwen3.5-9b" failed: Post "http://127.0.0.1:11434/api/chat": context deadline exceeded`), "timeout"},
		{errors.New("context overflow: 40000 > 32000"), "context_overflow"},
		{errors.New("dial tcp 127.0.0.1:11434: connect: connection refused"), "unavailable"},
		{errors.New(`local provider HTTP 404: model "qwen3.5:9b" not found`), "model_unavailable"},
		{errors.New("model output is not valid JSON: unexpected end of JSON input"), "invalid_output"},
		{errors.New("something unexpected"), "provider_error"},
	}
	for _, c := range cases {
		if got := analysisErrorKind(c.err); got != c.want {
			t.Errorf("analysisErrorKind(%q) = %q, want %q", c.err, got, c.want)
		}
	}
}
