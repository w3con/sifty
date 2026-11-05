// Package server_test — route drift check.
//
// TestRouteDrift reads the fixture files in testdata/fixtures/ (the documented endpoint list),
// normalises each fixture path into the route pattern the server would register (e.g.
// /api/jobs/1/seen → /api/jobs/{id}/seen), and compares that set against the patterns actually
// registered by RegisterRoutes on a recording mux. The test fails when the two sets diverge.
//
// TestRouteDriftSeedDivergence verifies that the checker itself detects a difference when one
// is introduced, so a silent no-op check cannot pass.
package server_test

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/server"
)

// recordingMux wraps *http.ServeMux and records every pattern registered via HandleFunc.
// It satisfies server.Router so it can be passed directly to RegisterRoutes.
type recordingMux struct {
	*http.ServeMux
	patterns []string
}

func (r *recordingMux) HandleFunc(pattern string, h func(http.ResponseWriter, *http.Request)) {
	r.patterns = append(r.patterns, pattern)
	r.ServeMux.HandleFunc(pattern, h)
}

// normalizeFixturePath converts a concrete fixture path into the route pattern registered
// by RegisterRoutes, substituting path-value placeholders where the server uses them.
//
//	/api/jobs/1              → /api/jobs/{id}
//	/api/jobs/1/seen         → /api/jobs/{id}/seen
//	/api/block-rules/1       → /api/block-rules/{id}
//	/api/stop-words/1        → /api/stop-words/{id}
//	/api/providers/p/check   → /api/providers/{name}/check
//	/api/config/runtime      → /api/config/{name}
//	/api/application-files/cv.pdf → /api/application-files/{name}
func normalizeFixturePath(path string) string {
	segs := strings.Split(strings.TrimPrefix(path, "/"), "/")
	// /api/jobs/<numeric>/...
	if len(segs) >= 3 && segs[0] == "api" && segs[1] == "jobs" {
		if _, err := strconv.Atoi(segs[2]); err == nil {
			segs[2] = "{id}"
		}
	}
	// /api/block-rules/<numeric> and /api/stop-words/<numeric>
	if len(segs) == 3 && segs[0] == "api" && (segs[1] == "block-rules" || segs[1] == "stop-words") {
		if _, err := strconv.Atoi(segs[2]); err == nil {
			segs[2] = "{id}"
		}
	}
	// /api/providers/<name>/check
	if len(segs) == 4 && segs[0] == "api" && segs[1] == "providers" && segs[3] == "check" {
		segs[2] = "{name}"
	}
	// /api/config/<name>
	if len(segs) == 3 && segs[0] == "api" && segs[1] == "config" {
		segs[2] = "{name}"
	}
	// /api/application-files/<name>
	if len(segs) == 3 && segs[0] == "api" && segs[1] == "application-files" {
		segs[2] = "{name}"
	}
	return "/" + strings.Join(segs, "/")
}

// routeDiff returns patterns in want but not in got (missing) and patterns in got but not in
// want (extra). Both slices are sorted for deterministic output.
func routeDiff(want, got []string) (missing, extra []string) {
	wantSet := make(map[string]bool, len(want))
	for _, r := range want {
		wantSet[r] = true
	}
	gotSet := make(map[string]bool, len(got))
	for _, r := range got {
		gotSet[r] = true
	}
	for r := range wantSet {
		if !gotSet[r] {
			missing = append(missing, r)
		}
	}
	for r := range gotSet {
		if !wantSet[r] {
			extra = append(extra, r)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)
	return
}

// fixtureRoutes reads all JSON fixture files and returns the deduplicated, sorted set of
// "METHOD /pattern" strings derived from each fixture's method and normalised path.
func fixtureRoutes(t *testing.T) []string {
	t.Helper()
	dir := "testdata/fixtures"
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Skipf("fixture directory %s not found: %v", dir, err)
		return nil
	}
	seen := map[string]bool{}
	var routes []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("read %s: %v", e.Name(), err)
		}
		var spec struct {
			Method string `json:"method"`
			Path   string `json:"path"`
		}
		if err := json.Unmarshal(data, &spec); err != nil {
			t.Fatalf("parse %s: %v", e.Name(), err)
		}
		key := spec.Method + " " + normalizeFixturePath(spec.Path)
		if !seen[key] {
			seen[key] = true
			routes = append(routes, key)
		}
	}
	sort.Strings(routes)
	return routes
}

// serverRoutes calls RegisterRoutes on a recording mux and returns the sorted slice of patterns
// it registered (in "METHOD /pattern" form as registered by the Go 1.22 ServeMux).
func serverRoutes(t *testing.T) []string {
	t.Helper()
	s := newTestStore(t)
	d := newTestDirs(t)
	cfg := minimalTestCfg()
	deps := &server.Deps{Store: s, Dirs: d, Cfg: config.NewHolder(cfg)}
	rec := &recordingMux{ServeMux: http.NewServeMux()}
	server.RegisterRoutes(rec, deps)
	out := make([]string, len(rec.patterns))
	copy(out, rec.patterns)
	sort.Strings(out)
	return out
}

// TestRouteDrift fails when the documented fixture set and the registered route set diverge.
func TestRouteDrift(t *testing.T) {
	documented := fixtureRoutes(t)
	registered := serverRoutes(t)
	missing, extra := routeDiff(documented, registered)
	for _, r := range missing {
		t.Errorf("in fixtures but not registered by server: %s", r)
	}
	for _, r := range extra {
		t.Errorf("registered by server but missing from fixtures: %s", r)
	}
}

// TestRouteDriftSeedDivergence verifies that routeDiff detects a seeded difference, so a
// silent no-op implementation cannot make the check always pass.
func TestRouteDriftSeedDivergence(t *testing.T) {
	base := serverRoutes(t)
	phantom := "GET /api/phantom-endpoint-that-does-not-exist"
	// The phantom is in the "want" (fixture) side but absent from the server.
	want := append(append([]string{}, base...), phantom)
	missing, _ := routeDiff(want, base)
	found := false
	for _, m := range missing {
		if m == phantom {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("routeDiff did not detect seeded divergence %q in missing list %v", phantom, missing)
	}
}
