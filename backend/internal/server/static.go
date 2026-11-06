package server

import (
	"net/http"
	"os"
	"path/filepath"
)

// StaticHandler returns an http.Handler that serves the React build from distDir.
// Any request for a path that is not a known file falls back to index.html (SPA routing).
// The handler only serves files it finds; it never lists directories.
// Returns an error if index.html is not present in distDir.
func StaticHandler(distDir string) (http.Handler, error) {
	if _, err := os.Stat(filepath.Join(distDir, "index.html")); err != nil {
		return nil, err
	}
	fs := http.Dir(distDir)
	return spaHandler{root: fs, distDir: distDir}, nil
}

type spaHandler struct {
	root    http.FileSystem
	distDir string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to find the exact requested path in the dist directory.
	path := r.URL.Path
	// http.FileServer cleans the path itself, but we need to stat first.
	cleanPath := filepath.Clean(path)

	f, err := h.root.Open(cleanPath)
	if err == nil {
		stat, statErr := f.Stat()
		_ = f.Close()
		if statErr == nil && !stat.IsDir() {
			// Exact file exists — serve it normally.
			http.FileServer(h.root).ServeHTTP(w, r)
			return
		}
	}

	// Fall back to index.html for all other paths (SPA deep-link routing).
	index, err := os.Open(filepath.Join(h.distDir, "index.html"))
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer index.Close()
	stat, err := index.Stat()
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeContent(w, r, "index.html", stat.ModTime(), index)
}
