package server_test

// Tests for the config live-reload behaviour: a saved config takes effect on the running server
// without a restart.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/server"
)

// configFixtures returns the minimal YAML content for each config file needed to boot the server.
func configFixtures() map[string]string {
	return map[string]string{
		"search": `version: 1
sites:
  - name: LinkedIn
    url: https://linkedin.com/jobs
search:
  titles: [CTO]
  keywords: [golang]
`,
		"runtime": `version: 1
port: 7373
max_pages: 5
max_jobs_per_source: 100
`,
		"providers": `version: 1
stages:
  assessment: primary
  extraction: primary
providers:
  primary:
    type: ollama
    model: llama3
    endpoint: http://127.0.0.1:11434
`,
		"candidate": `version: 1
cache_hours: 24
files: []
profiles: []
`,
	}
}

// newReloadHandler creates a handler where the config is loaded from actual files on disk so
// the reload path works end-to-end. Returns the handler and the deps so callers can PUT saves.
func newReloadHandler(t *testing.T) (http.Handler, *server.Deps) {
	t.Helper()
	s := newTestStore(t)
	d := newTestDirs(t)

	// Write all required config files so config.Load succeeds.
	for name, content := range configFixtures() {
		if err := os.WriteFile(d.ConfigFile(name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s.yaml: %v", name, err)
		}
	}

	cfg, err := config.Load(d)
	if err != nil {
		t.Fatalf("config.Load: %v", err)
	}
	deps := &server.Deps{Store: s, Dirs: d, Cfg: config.NewHolder(cfg)}
	mux := server.New(testPort, func(m *http.ServeMux) {
		server.RegisterRoutes(m, deps)
	})
	return server.Handler(testPort, mux), deps
}

// --- Step 2.1 ---

// TestProviderDiagnosticsReflectsNewModelAfterSave verifies that saving a changed providers
// config causes the provider panel to report the new model on the next /api/providers request,
// without a server restart.
func TestProviderDiagnosticsReflectsNewModelAfterSave(t *testing.T) {
	h, _ := newReloadHandler(t)

	// Confirm the initial model is "llama3".
	resp := apiDo(t, h, "GET", "/api/providers", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/providers before save: %d", resp.StatusCode)
	}
	var before []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&before); err != nil {
		t.Fatalf("decode before: %v", err)
	}
	if len(before) == 0 {
		t.Fatal("no providers before save")
	}
	if before[0]["model"] != "llama3" {
		t.Fatalf("initial model: got %q, want llama3", before[0]["model"])
	}

	// Save updated providers with model "mistral".
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
	body := fmt.Sprintf(`{"content":%q}`, updatedProviders)
	resp = apiDo(t, h, "PUT", "/api/config/providers", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/config/providers: %d", resp.StatusCode)
	}

	// The next GET /api/providers must report the new model.
	resp = apiDo(t, h, "GET", "/api/providers", "")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/providers after save: %d", resp.StatusCode)
	}
	var after []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&after); err != nil {
		t.Fatalf("decode after: %v", err)
	}
	if len(after) == 0 {
		t.Fatal("no providers after save")
	}
	if after[0]["model"] != "mistral" {
		t.Errorf("model after save: got %q, want mistral", after[0]["model"])
	}
}

// --- Step 2.2 ---

// TestPortChangeSaveReturnsRestartNotice verifies that saving a runtime config with a different
// port returns a 200 response carrying a restartRequired notice, without failing the save.
func TestPortChangeSaveReturnsRestartNotice(t *testing.T) {
	h, _ := newReloadHandler(t)

	// Save a runtime config with a different port.
	const updatedRuntime = `version: 1
port: 7374
max_pages: 5
max_jobs_per_source: 100
`
	body := fmt.Sprintf(`{"content":%q}`, updatedRuntime)
	resp := apiDo(t, h, "PUT", "/api/config/runtime", body)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT /api/config/runtime: %d (save must succeed even with a port change)", resp.StatusCode)
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode save response: %v", err)
	}

	if _, ok := result["restartRequired"]; !ok {
		t.Errorf("save response missing restartRequired; got: %v", result)
	}
	if result["saved"] != "runtime" {
		t.Errorf("saved field: got %q, want runtime", result["saved"])
	}

	// The server must still respond on the original port (7373), not on 7374.
	resp = apiDo(t, h, "GET", "/api/health", "")
	if resp.StatusCode != http.StatusOK {
		t.Errorf("GET /api/health after port-change save: %d (server must stay on original port)", resp.StatusCode)
	}
}

// TestQuickMatchOrderEndpoint verifies that the Inbox sort toggle can be saved on the server and
// read back, that the value reaches runtime.yaml and the running configuration, and that an
// unknown order is refused.
func TestQuickMatchOrderEndpoint(t *testing.T) {
	h, _ := newReloadHandler(t)
	template, err := os.ReadFile("../../../config/runtime.yaml")
	if err != nil {
		t.Fatal(err)
	}
	body := fmt.Sprintf(`{"content":%q}`, strings.Replace(string(template), "port: 8081", "port: 7373", 1))
	if resp := apiDo(t, h, "PUT", "/api/config/runtime", body); resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT runtime template: %d", resp.StatusCode)
	}
	read := func() string {
		resp := apiDo(t, h, "GET", "/api/quick-match/order", "")
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("GET order: %d", resp.StatusCode)
		}
		var out map[string]string
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		return out["order"]
	}
	if got := read(); got != "fit" {
		t.Fatalf("default order = %q, want fit", got)
	}
	if resp := apiDo(t, h, "PUT", "/api/quick-match/order", `{"order":"newest"}`); resp.StatusCode != http.StatusOK {
		t.Fatalf("PUT order newest: %d", resp.StatusCode)
	}
	if got := read(); got != "newest" {
		t.Fatalf("order after save = %q, want newest", got)
	}
	resp := apiDo(t, h, "GET", "/api/config/runtime", "")
	var saved map[string]string
	_ = json.NewDecoder(resp.Body).Decode(&saved)
	if !strings.Contains(saved["content"], "order: newest") {
		t.Errorf("runtime.yaml does not carry the saved order:\n%s", saved["content"])
	}
	if resp := apiDo(t, h, "PUT", "/api/quick-match/order", `{"order":"random"}`); resp.StatusCode == http.StatusOK {
		t.Error("an unknown order was accepted")
	}
	if got := read(); got != "newest" {
		t.Fatalf("a refused save changed the order to %q", got)
	}
}
