// Package server builds and runs the HTTP server. All endpoints are under /api, bound to
// 127.0.0.1 only. The security middleware applies Host and Origin allowlisting, Content-Type
// enforcement for non-GET requests, and fixed response headers. Errors are returned in the
// uniform { "error": "<message>" } JSON shape. The built React frontend is served as static
// assets from the configured dist directory, with single-page-application fallback.
package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

const maxBodyBytes = 1 << 20 // 1 MB for ordinary JSON requests

// errorShape is the JSON body returned for all errors.
type errorShape struct {
	Error string `json:"error"`
}

// writeJSON writes a JSON response with the given status code and value.
func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// writeError writes a uniform { "error": msg } JSON response.
func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, errorShape{Error: msg})
}

// securityMiddleware enforces loopback-only access rules:
// - Host must be 127.0.0.1:<port> or localhost:<port>
// - Origin, if present, must be the corresponding http:// origin
// - Non-GET, non-HEAD, non-OPTIONS requests must carry Content-Type: application/json
// - Fixed response headers are set on every response
// - Request bodies are capped at maxBodyBytes
func securityMiddleware(port int, next http.Handler) http.Handler {
	allowedHosts := []string{
		fmt.Sprintf("127.0.0.1:%d", port),
		fmt.Sprintf("localhost:%d", port),
	}
	allowedOrigins := []string{
		fmt.Sprintf("http://127.0.0.1:%d", port),
		fmt.Sprintf("http://localhost:%d", port),
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Fixed security headers on every response.
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")

		// Host allowlist.
		host := r.Header.Get("Host")
		if host == "" {
			if h, _, err := net.SplitHostPort(r.Host); err == nil {
				_, p, _ := net.SplitHostPort(r.Host)
				host = h + ":" + p
			} else {
				host = r.Host
			}
		}
		allowed := false
		for _, a := range allowedHosts {
			if host == a {
				allowed = true
				break
			}
		}
		if !allowed {
			writeError(w, http.StatusForbidden, "Invalid local Host")
			return
		}

		// Origin allowlist (only checked when Origin header is present).
		if origin := r.Header.Get("Origin"); origin != "" {
			originAllowed := false
			for _, a := range allowedOrigins {
				if origin == a {
					originAllowed = true
					break
				}
			}
			if !originAllowed {
				writeError(w, http.StatusForbidden, "Invalid local Origin")
				return
			}
		}

		// Content-Type enforcement for mutation requests.
		m := r.Method
		if m != http.MethodGet && m != http.MethodHead && m != http.MethodOptions {
			ct := r.Header.Get("Content-Type")
			// Accept application/json; charset=... etc.
			base := strings.TrimSpace(strings.SplitN(ct, ";", 2)[0])
			if base != "application/json" {
				writeError(w, http.StatusUnsupportedMediaType, "JSON required")
				return
			}
		}

		// Cap request body size. The profile document upload path is deliberately deferred to its
		// handler, which derives the byte allowance from the configured document limit before
		// decoding base64 content. No second upload ceiling is kept here.
		if r.Body != nil && !(r.Method == http.MethodPost && r.URL.Path == "/api/profile/documents") {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}

		next.ServeHTTP(w, r)
	})
}

// safeHandler wraps a handler function so that panics and returned errors become 400 JSON errors.
func safeHandler(fn func(w http.ResponseWriter, r *http.Request) error) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				writeError(w, http.StatusBadRequest, fmt.Sprintf("%v", rec))
			}
		}()
		if err := fn(w, r); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
		}
	}
}

// readBody reads and returns the JSON-decoded request body into v, enforcing the 1 MB cap.
func readBody(r *http.Request, v any) error {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("read body: %w", err)
	}
	if len(data) == 0 {
		return nil
	}
	return json.Unmarshal(data, v)
}

// pathID extracts the last path segment as an integer.
func pathID(r *http.Request) (int64, error) {
	seg := r.PathValue("id")
	if seg == "" {
		return 0, fmt.Errorf("missing id in path")
	}
	var id int64
	if _, err := fmt.Sscanf(seg, "%d", &id); err != nil {
		return 0, fmt.Errorf("invalid id %q", seg)
	}
	return id, nil
}

// unknownAPIHandler returns the documented 404 response for unknown /api/... paths.
var unknownAPIHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	writeError(w, http.StatusNotFound, "Unknown API endpoint. Refresh the page to load the current catalogue version.")
})

// New builds and returns an *http.ServeMux with the full route table registered.
// The returned mux is not yet bound to a listener; callers wrap it in an *http.Server.
// handlers is a function that registers route-specific handlers onto the mux; it is called
// after the security middleware is in place, letting Phase 4 add endpoints incrementally.
func New(port int, registerRoutes func(mux *http.ServeMux)) *http.ServeMux {
	mux := http.NewServeMux()

	// Catch-all for unknown /api/... paths; must be registered before other /api/ routes so
	// that only genuinely unmatched paths reach it.
	mux.Handle("/api/", unknownAPIHandler)

	// Phase 4 and later register their handlers, which override the catch-all for known paths.
	if registerRoutes != nil {
		registerRoutes(mux)
	}

	return mux
}

// Handler returns an http.Handler wrapping the mux in the security middleware.
func Handler(port int, mux *http.ServeMux) http.Handler {
	return securityMiddleware(port, mux)
}
