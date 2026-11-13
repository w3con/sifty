package assess_test

import (
	"fmt"
	"testing"

	"github.com/w3con/sifty/internal/assess"
)

// TestDiagnosticsShape verifies that Diagnostics returns one entry per provider.
func TestDiagnosticsShape(t *testing.T) {
	providers := map[string]assess.Provider{
		"local": {Type: "ollama", Model: "llama3", Endpoint: "http://127.0.0.1:11434", TimeoutMS: 5000, ContextChars: 32000},
		"cloud": {Type: "claude", Model: "claude-haiku-4-5", TimeoutMS: 30000, ContextChars: 64000},
	}
	result := assess.Diagnostics(providers)
	if len(result) != 2 {
		t.Errorf("Diagnostics: got %d entries, want 2", len(result))
	}
	for _, r := range result {
		if r.Name == "" {
			t.Error("Diagnostics: entry has empty name")
		}
		if r.Status == "" {
			t.Error("Diagnostics: entry has empty status")
		}
		if r.Model == "" {
			t.Error("Diagnostics: entry has empty model")
		}
	}
}

// TestDiagnosticsUnknownType verifies that an unknown provider type produces an error status.
func TestDiagnosticsUnknownType(t *testing.T) {
	providers := map[string]assess.Provider{
		"mystery": {Type: "custom", Model: "gpt-4", TimeoutMS: 5000, ContextChars: 32000},
	}
	result := assess.Diagnostics(providers)
	if len(result) != 1 {
		t.Fatalf("Diagnostics: got %d entries, want 1", len(result))
	}
	if result[0].Status == "" || result[0].Status == "Installed model available" {
		t.Errorf("Diagnostics: expected error status for unknown type, got %q", result[0].Status)
	}
}

// TestSmokeCheckShape verifies that SmokeCheck calls the inference function and returns a RawModel.
func TestSmokeCheckShape(t *testing.T) {
	p := assess.Provider{Type: "ollama", Model: "llama3", Endpoint: "http://127.0.0.1:11434", TimeoutMS: 5000, ContextChars: 32000}

	var capturedPrompt string
	stub := func(prov assess.Provider, requestText string, schema any) (assess.RawModel, error) {
		capturedPrompt = requestText
		return assess.RawModel{
			Score:   0,
			Summary: "Minimal provider connectivity check. No candidate data supplied.",
			Evidence: []assess.RawEvidence{},
			Gaps:    []string{"This is a smoke check, not an assessment."},
		}, nil
	}

	result, err := assess.SmokeCheck(p, stub)
	if err != nil {
		t.Fatalf("SmokeCheck: %v", err)
	}
	if result.Score != 0 {
		t.Errorf("SmokeCheck: score = %v, want 0", result.Score)
	}
	if result.Summary == "" {
		t.Error("SmokeCheck: summary is empty")
	}
	if !contains(capturedPrompt, "smoke check") {
		t.Errorf("SmokeCheck: prompt does not contain 'smoke check', got: %q", capturedPrompt[:min(100, len(capturedPrompt))])
	}
}

// TestSmokeCheckPropagatesError verifies that an inference error is returned as-is.
func TestSmokeCheckPropagatesError(t *testing.T) {
	p := assess.Provider{Type: "ollama", Model: "llama3", Endpoint: "http://127.0.0.1:11434", TimeoutMS: 5000, ContextChars: 32000}

	stub := func(_ assess.Provider, _ string, _ any) (assess.RawModel, error) {
		return assess.RawModel{}, fmt.Errorf("provider unavailable")
	}

	_, err := assess.SmokeCheck(p, stub)
	if err == nil {
		t.Error("SmokeCheck: expected error from failed inference, got nil")
	}
}

func contains(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
