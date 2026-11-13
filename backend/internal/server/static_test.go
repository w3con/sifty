package server_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/server"
)

func makeDist(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	// Write a minimal index.html.
	if err := os.WriteFile(filepath.Join(dir, "index.html"), []byte("<html><body>App</body></html>"), 0o644); err != nil {
		t.Fatalf("write index.html: %v", err)
	}
	// Write a static asset.
	if err := os.WriteFile(filepath.Join(dir, "app.js"), []byte("// bundle"), 0o644); err != nil {
		t.Fatalf("write app.js: %v", err)
	}
	return dir
}

// TestStaticHandlerRoot verifies that the root serves the index.html.
func TestStaticHandlerRoot(t *testing.T) {
	h, err := server.StaticHandler(makeDist(t))
	if err != nil {
		t.Fatalf("StaticHandler: %v", err)
	}
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("root: got %d, want 200", resp.StatusCode)
	}
}

// TestStaticHandlerKnownFile verifies that a known file is served directly.
func TestStaticHandlerKnownFile(t *testing.T) {
	h, err := server.StaticHandler(makeDist(t))
	if err != nil {
		t.Fatalf("StaticHandler: %v", err)
	}
	req := httptest.NewRequest("GET", "/app.js", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("known file: got %d, want 200", resp.StatusCode)
	}
}

// TestStaticHandlerDeepLink verifies that an unknown path falls back to index.html.
func TestStaticHandlerDeepLink(t *testing.T) {
	h, err := server.StaticHandler(makeDist(t))
	if err != nil {
		t.Fatalf("StaticHandler: %v", err)
	}
	req := httptest.NewRequest("GET", "/jobs/42", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("deep link: got %d, want 200 (SPA fallback)", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if ct == "" || ct[:9] != "text/html" {
		t.Errorf("deep link content type: got %q, want text/html prefix", ct)
	}
}

// TestStaticHandlerMissingDistReturnsError verifies that a missing dist dir returns an error.
func TestStaticHandlerMissingDistReturnsError(t *testing.T) {
	_, err := server.StaticHandler("/nonexistent/dist/dir")
	if err == nil {
		t.Error("missing dist: expected error, got nil")
	}
}
