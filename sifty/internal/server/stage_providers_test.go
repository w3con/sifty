package server_test

// Tests for GET /api/stage-providers and PUT /api/stage-providers.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/server"
)

// TestGetStageProviders verifies that GET /api/stage-providers returns the three stage
// selections (assessment, preparation, extraction) and every configured provider.
func TestGetStageProviders(t *testing.T) {
	h, _ := newReloadHandler(t)
	resp := apiDo(t, h, "GET", "/api/stage-providers", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/stage-providers: got %d, want 200", resp.StatusCode)
	}
	var body map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	stages, ok := body["stages"].(map[string]any)
	if !ok {
		t.Fatalf("stages field missing or wrong type: %v", body["stages"])
	}
	for _, key := range []string{"assessment", "preparation", "extraction"} {
		if _, ok := stages[key]; !ok {
			t.Errorf("stages.%s missing", key)
		}
	}
	providers, ok := body["providers"].([]any)
	if !ok {
		t.Fatalf("providers field missing or wrong type: %v", body["providers"])
	}
	if len(providers) == 0 {
		t.Fatal("providers list is empty")
	}
	// Each entry must have name and model.
	p, ok := providers[0].(map[string]any)
	if !ok {
		t.Fatalf("first provider entry is not an object: %v", providers[0])
	}
	if _, ok := p["name"]; !ok {
		t.Error("provider entry missing name field")
	}
	if _, ok := p["model"]; !ok {
		t.Error("provider entry missing model field")
	}
}

// TestPutStageProvidersAcceptsValid verifies that setting preparation to a known provider
// updates providers.yaml, and the next GET reflects the new selection.
func TestPutStageProvidersAcceptsValid(t *testing.T) {
	h, _ := newReloadHandler(t)

	body := fmt.Sprintf(`{"stage":"preparation","provider":"primary"}`)
	resp := apiDo(t, h, "PUT", "/api/stage-providers", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/stage-providers: got %d, want 200", resp.StatusCode)
	}
	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	stages, ok := result["stages"].(map[string]any)
	if !ok {
		t.Fatalf("stages missing from response: %v", result)
	}
	if stages["preparation"] != "primary" {
		t.Errorf("preparation stage: got %v, want primary", stages["preparation"])
	}

	// Confirm GET also reflects the change.
	resp2 := apiDo(t, h, "GET", "/api/stage-providers", "")
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("GET after PUT: got %d", resp2.StatusCode)
	}
	var body2 map[string]any
	if err := json.NewDecoder(resp2.Body).Decode(&body2); err != nil {
		t.Fatalf("decode GET: %v", err)
	}
	stages2 := body2["stages"].(map[string]any)
	if stages2["preparation"] != "primary" {
		t.Errorf("GET after PUT: preparation = %v, want primary", stages2["preparation"])
	}
}

// TestPutStageProvidersRejectsUnknownProvider verifies that setting a stage to an unknown
// provider name returns a non-200 status.
func TestPutStageProvidersRejectsUnknownProvider(t *testing.T) {
	h, _ := newReloadHandler(t)

	body := `{"stage":"preparation","provider":"nonexistent"}`
	resp := apiDo(t, h, "PUT", "/api/stage-providers", body)
	if resp.StatusCode == http.StatusOK {
		t.Fatal("PUT with unknown provider: expected non-200, got 200")
	}
}

// TestPutStageProvidersRejectsUnknownStage verifies that setting an unknown stage returns
// a non-200 status.
func TestPutStageProvidersRejectsUnknownStage(t *testing.T) {
	h, _ := newReloadHandler(t)

	body := `{"stage":"invalid","provider":"primary"}`
	resp := apiDo(t, h, "PUT", "/api/stage-providers", body)
	if resp.StatusCode == http.StatusOK {
		t.Fatal("PUT with unknown stage: expected non-200, got 200")
	}
}

// newHandlerWithCustomProviders creates a handler backed by on-disk config where providers.yaml
// is set to providersYAML; all other fixtures use configFixtures(). The in-memory config is
// loaded from disk so provider validation in PUT /api/stage-providers works against the actual
// on-disk providers. Returns the handler and the Dirs so callers can read config files after
// PUT operations.
func newHandlerWithCustomProviders(t *testing.T, providersYAML string) (http.Handler, dirs.Dirs) {
	t.Helper()
	s := newTestStore(t)
	d := newTestDirs(t)
	fixtures := configFixtures()
	for _, name := range []string{"search", "runtime", "candidate"} {
		if err := os.WriteFile(d.ConfigFile(name), []byte(fixtures[name]), 0o600); err != nil {
			t.Fatalf("write %s.yaml: %v", name, err)
		}
	}
	if err := os.WriteFile(d.ConfigFile("providers"), []byte(providersYAML), 0o600); err != nil {
		t.Fatalf("write providers.yaml: %v", err)
	}
	cfg, err := config.Load(d)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	deps := &server.Deps{Store: s, Dirs: d, Cfg: config.NewHolder(cfg)}
	mux := server.New(testPort, func(m *http.ServeMux) {
		server.RegisterRoutes(m, deps)
	})
	return server.Handler(testPort, mux), d
}

// TestPutStageProvidersPreservesYAMLStructure verifies that PUT /api/stage-providers rewrites
// providers.yaml using surgical yaml.Node editing rather than marshalling from the in-memory
// config struct. The test starts from a hand-crafted YAML with comments, non-alphabetical
// provider order, and a provider that omits the optional endpoint and api_key_env fields, then
// asserts that none of these are corrupted after a stage update.
func TestPutStageProvidersPreservesYAMLStructure(t *testing.T) {
	// Non-alphabetical order: zeta before alpha. Alpha has no endpoint or api_key_env.
	// Two comments: one top-of-file and one trailing.
	const customProviders = `version: 1
# Top comment: credentials stay with the CLI.
stages:
  assessment: zeta
  extraction: zeta
providers:
  zeta:
    type: ollama
    model: zeta-model
    endpoint: http://127.0.0.1:11434
    timeout_ms: 60000
    context_chars: 8192
  alpha:
    type: ollama
    model: alpha-model
    timeout_ms: 60000
    context_chars: 8192
# Trailing comment.
`

	h, d := newHandlerWithCustomProviders(t, customProviders)

	// --- Part 1: set preparation (key absent in file; must be appended). ---

	resp := apiDo(t, h, "PUT", "/api/stage-providers", `{"stage":"preparation","provider":"zeta"}`)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT preparation: got %d, want 200", resp.StatusCode)
	}

	v1, err := os.ReadFile(d.ConfigFile("providers"))
	if err != nil {
		t.Fatalf("read providers.yaml after PUT preparation: %v", err)
	}
	v1s := string(v1)

	// Comments must survive.
	if !strings.Contains(v1s, "# Top comment: credentials stay with the CLI.") {
		t.Error("top comment missing after PUT preparation")
	}
	if !strings.Contains(v1s, "# Trailing comment.") {
		t.Error("trailing comment missing after PUT preparation")
	}

	// No empty optional fields must be emitted.
	if strings.Contains(v1s, "endpoint: \"\"") || strings.Contains(v1s, "endpoint: ''") {
		t.Error("empty endpoint field emitted for alpha provider")
	}
	if strings.Contains(v1s, "api_key_env: \"\"") || strings.Contains(v1s, "api_key_env: ''") {
		t.Error("empty api_key_env field emitted for alpha provider")
	}

	// Provider order must be preserved: zeta before alpha.
	zetaIdx := strings.Index(v1s, "  zeta:")
	alphaIdx := strings.Index(v1s, "  alpha:")
	if zetaIdx < 0 || alphaIdx < 0 {
		t.Errorf("provider keys missing in output:\n%s", v1s)
	} else if zetaIdx > alphaIdx {
		t.Errorf("provider order changed: zeta should appear before alpha\n%s", v1s)
	}

	// stages.preparation must be set.
	if !strings.Contains(v1s, "preparation: zeta") {
		t.Errorf("stages.preparation not set in output:\n%s", v1s)
	}

	// --- Part 2: change assessment (existing key; must be a one-line diff vs V1). ---

	resp2 := apiDo(t, h, "PUT", "/api/stage-providers", `{"stage":"assessment","provider":"alpha"}`)
	if resp2.StatusCode != http.StatusOK {
		t.Fatalf("PUT assessment: got %d, want 200", resp2.StatusCode)
	}

	v2, err := os.ReadFile(d.ConfigFile("providers"))
	if err != nil {
		t.Fatalf("read providers.yaml after PUT assessment: %v", err)
	}
	v2s := string(v2)

	// Diff V1 → V2: only the assessment line may differ.
	v1Lines := strings.Split(v1s, "\n")
	v2Lines := strings.Split(v2s, "\n")
	if len(v1Lines) != len(v2Lines) {
		t.Errorf("line count changed after assessment update: %d → %d\nV1:\n%s\nV2:\n%s",
			len(v1Lines), len(v2Lines), v1s, v2s)
	} else {
		var changedLines []int
		for i := range v1Lines {
			if v1Lines[i] != v2Lines[i] {
				changedLines = append(changedLines, i+1)
			}
		}
		if len(changedLines) != 1 {
			t.Errorf("expected exactly 1 changed line, got %d (lines %v)\nV1:\n%s\nV2:\n%s",
				len(changedLines), changedLines, v1s, v2s)
		} else if !strings.Contains(v2Lines[changedLines[0]-1], "assessment") {
			t.Errorf("changed line %d does not contain 'assessment': %q",
				changedLines[0], v2Lines[changedLines[0]-1])
		}
	}
}
