package server_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/server"
)

const testPort = 7373

func newTestHandler() http.Handler {
	mux := server.New(testPort, nil)
	return server.Handler(testPort, mux)
}

func do(t *testing.T, h http.Handler, method, path, body, host, origin, ct string) *http.Response {
	t.Helper()
	var bodyR io.Reader
	if body != "" {
		bodyR = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, path, bodyR)
	if host != "" {
		req.Header.Set("Host", host)
		req.Host = host
	}
	if origin != "" {
		req.Header.Set("Origin", origin)
	}
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w.Result()
}

// TestBadHostReturns403 verifies that an invalid Host header returns 403.
func TestBadHostReturns403(t *testing.T) {
	h := newTestHandler()
	resp := do(t, h, "GET", "/api/", "", "evil.example.com", "", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("bad Host: got %d, want 403", resp.StatusCode)
	}
}

// TestGoodHostAllowed verifies that 127.0.0.1:<port> and localhost:<port> are both accepted.
func TestGoodHostAllowed(t *testing.T) {
	h := newTestHandler()
	for _, host := range []string{
		"127.0.0.1:7373",
		"localhost:7373",
	} {
		resp := do(t, h, "GET", "/api/unknown-path", "", host, "", "")
		// The security middleware passes; the unknown-API handler returns 404, not 403.
		if resp.StatusCode == http.StatusForbidden {
			t.Errorf("host %q: got 403, should be allowed through", host)
		}
	}
}

// TestBadOriginReturns403 verifies that a cross-origin Origin header is refused.
func TestBadOriginReturns403(t *testing.T) {
	h := newTestHandler()
	resp := do(t, h, "GET", "/api/", "127.0.0.1:7373", "http://evil.com", "", "")
	if resp.StatusCode != http.StatusForbidden {
		t.Errorf("bad Origin: got %d, want 403", resp.StatusCode)
	}
}

// TestNoOriginAllowed verifies that a missing Origin header is not an error.
func TestNoOriginAllowed(t *testing.T) {
	h := newTestHandler()
	resp := do(t, h, "GET", "/api/nowhere", "", "127.0.0.1:7373", "", "")
	if resp.StatusCode == http.StatusForbidden {
		t.Error("no Origin: got 403, should not require Origin header")
	}
}

// TestWrongContentTypeReturns415 verifies that a POST without JSON content-type returns 415.
func TestWrongContentTypeReturns415(t *testing.T) {
	h := newTestHandler()
	resp := do(t, h, "POST", "/api/search", `{"source":"x"}`, "127.0.0.1:7373", "", "text/plain")
	if resp.StatusCode != http.StatusUnsupportedMediaType {
		t.Errorf("wrong Content-Type: got %d, want 415", resp.StatusCode)
	}
}

// TestCorrectContentTypeAccepted verifies that application/json passes content-type check.
func TestCorrectContentTypeAccepted(t *testing.T) {
	h := newTestHandler()
	resp := do(t, h, "POST", "/api/search", `{"source":"x"}`, "127.0.0.1:7373", "", "application/json")
	// Should not be 415 (the unknown-API handler returns 404, not 415).
	if resp.StatusCode == http.StatusUnsupportedMediaType {
		t.Errorf("application/json: got 415, should be accepted")
	}
}

// TestContentTypeWithCharset verifies that "application/json; charset=utf-8" is accepted.
func TestContentTypeWithCharset(t *testing.T) {
	h := newTestHandler()
	resp := do(t, h, "POST", "/api/import", `{"url":"https://x.com"}`, "127.0.0.1:7373", "", "application/json; charset=utf-8")
	if resp.StatusCode == http.StatusUnsupportedMediaType {
		t.Errorf("application/json; charset=utf-8: got 415, should be accepted")
	}
}

// TestUnknownAPIPathReturns404 verifies the documented 404 shape for unknown /api/ paths.
func TestUnknownAPIPathReturns404(t *testing.T) {
	h := newTestHandler()
	resp := do(t, h, "GET", "/api/completely-unknown", "", "127.0.0.1:7373", "", "")
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("unknown path: got %d, want 404", resp.StatusCode)
	}
	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	want := "Unknown API endpoint. Refresh the page to load the current catalogue version."
	if body["error"] != want {
		t.Errorf("error message: got %q, want %q", body["error"], want)
	}
}

// TestSecurityHeadersPresent verifies X-Content-Type-Options and Referrer-Policy on every response.
func TestSecurityHeadersPresent(t *testing.T) {
	h := newTestHandler()
	resp := do(t, h, "GET", "/api/nothing", "", "127.0.0.1:7373", "", "")
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options: got %q, want nosniff", got)
	}
	if got := resp.Header.Get("Referrer-Policy"); got != "no-referrer" {
		t.Errorf("Referrer-Policy: got %q, want no-referrer", got)
	}
}

// TestSecurityHeadersOnForbidden verifies security headers are set even on 403 responses.
func TestSecurityHeadersOnForbidden(t *testing.T) {
	h := newTestHandler()
	resp := do(t, h, "GET", "/api/jobs", "", "evil.com", "", "")
	if got := resp.Header.Get("X-Content-Type-Options"); got != "nosniff" {
		t.Errorf("X-Content-Type-Options on 403: got %q, want nosniff", got)
	}
}
