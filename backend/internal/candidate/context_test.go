package candidate_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/candidate"
	gocfg "github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/records"
)

// stubStore implements candidate.ContextStore for tests.
type stubStore struct {
	saved map[string]any
}

func (s *stubStore) Context() (map[string]any, error) {
	if s.saved == nil {
		return nil, nil
	}
	return s.saved, nil
}

func (s *stubStore) SaveContext(m map[string]any) error {
	s.saved = m
	return nil
}

func tempDirs(t *testing.T) dirs.Dirs {
	t.Helper()
	d := dirs.Dirs{Root: t.TempDir()}
	if err := d.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}
	return d
}

func minimalCfg() *gocfg.Config {
	t := true
	return &gocfg.Config{
		Candidate: gocfg.Candidate{
			Version:    1,
			CacheHours: 24,
			Files:      []string{},
			Profiles:   []gocfg.ProfileRef{},
		},
		Search: gocfg.Search{
			Version: 1,
			Sites: []gocfg.Source{{Name: "LinkedIn", URL: "https://linkedin.com/jobs", Enabled: &t}},
		},
		Runtime: gocfg.Runtime{Version: 1, Port: 7373},
		Providers: gocfg.Providers{
			Version: 1,
			Stages: struct {
				Assessment  string `yaml:"assessment"`
				Extraction  string `yaml:"extraction"`
				Preparation string `yaml:"preparation"`
				Matching    string `yaml:"matching,omitempty"`
			}{Assessment: "p", Extraction: "p", Preparation: "p"},
			Providers: map[string]gocfg.Provider{"p": {Type: "ollama", Model: "llama3", Endpoint: "http://127.0.0.1:11434"}},
		},
	}
}

// TestContextVersion verifies the fingerprint changes when sources change.
func TestContextVersion(t *testing.T) {
	prefs := map[string]any{"key": "golang"}
	notes := []string{"Note one"}

	s1 := candidate.EvidenceSource{ID: "cv.pdf", Reference: "cv.pdf", Hash: "abc", Status: "available", Text: "My CV text"}
	v1 := candidate.ContextVersion(prefs, notes, []candidate.EvidenceSource{s1})

	s2 := candidate.EvidenceSource{ID: "cv.pdf", Reference: "cv.pdf", Hash: "def", Status: "available", Text: "Different text"}
	v2 := candidate.ContextVersion(prefs, notes, []candidate.EvidenceSource{s2})

	if v1 == "" {
		t.Error("ContextVersion returned empty string")
	}
	if v1 == v2 {
		t.Error("ContextVersion: expected different version for different source text, got same")
	}

	// Same input → same version.
	v1b := candidate.ContextVersion(prefs, notes, []candidate.EvidenceSource{s1})
	if v1 != v1b {
		t.Error("ContextVersion: expected stable version for identical input")
	}
}

// TestFeedbackSources verifies that feedback records are turned into EvidenceSource entries.
func TestFeedbackSources(t *testing.T) {
	d := tempDirs(t)
	supersedes := ""
	rec := &records.FeedbackRecord{
		ID:         "fb-001",
		Created:    "2026-11-02T10:00:00.000Z",
		JobID:      42,
		JobURL:     "https://example.com/job/42",
		JobTitle:   "Go Engineer",
		Source:     "owner",
		Scope:      "general",
		Text:       "Strong distributed systems background.",
		Supersedes: nil,
	}
	_ = supersedes

	if _, err := records.WriteFeedback(d, rec); err != nil {
		t.Fatalf("WriteFeedback: %v", err)
	}

	sources, err := candidate.FeedbackSources(d)
	if err != nil {
		t.Fatalf("FeedbackSources: %v", err)
	}
	if len(sources) != 1 {
		t.Fatalf("FeedbackSources: got %d sources, want 1", len(sources))
	}
	s := sources[0]
	if s.Status != "available" {
		t.Errorf("status: got %q, want available", s.Status)
	}
	if s.Category != "owner-confirmed-feedback" {
		t.Errorf("category: got %q, want owner-confirmed-feedback", s.Category)
	}
	if s.Text == "" {
		t.Error("text: expected non-empty feedback text")
	}
	if s.Hash == "" {
		t.Error("hash: expected non-empty hash")
	}
}

// TestRefreshContextLocalFiles verifies that RefreshContext reads a local text file.
func TestRefreshContextLocalFiles(t *testing.T) {
	d := tempDirs(t)
	// Write a candidate evidence file.
	cvPath := filepath.Join(d.Evidence(), "cv.txt")
	if err := os.WriteFile(cvPath, []byte("Go engineer with 8 years experience. Strong in distributed systems, Kubernetes and cloud architecture."), 0o600); err != nil {
		t.Fatalf("write cv: %v", err)
	}

	// Build a relative path under the data dir for the candidate config.
	relCVPath := "evidence/cv.txt"

	cfg := minimalCfg()
	cfg.Candidate.Files = []string{relCVPath}

	store := &stubStore{}
	prefs := map[string]any{"seniority": "senior"}

	ctx, err := candidate.RefreshContext(store, d, cfg, prefs, false, false)
	if err != nil {
		t.Fatalf("RefreshContext: %v", err)
	}

	if ctx.Version == "" {
		t.Error("version: expected non-empty")
	}
	if ctx.CreatedAt == "" {
		t.Error("createdAt: expected non-empty")
	}
	if len(ctx.Sources) != 1 {
		t.Fatalf("sources: got %d, want 1", len(ctx.Sources))
	}
	src := ctx.Sources[0]
	if src.Status != "available" {
		t.Errorf("source status: got %q, want available", src.Status)
	}
	if src.Text == "" {
		t.Error("source text: expected non-empty")
	}

	// Verify it was saved to the store.
	if store.saved == nil {
		t.Error("context was not saved to store")
	}
}

// TestRefreshContextMissingFile verifies that a missing file produces a "missing" source.
func TestRefreshContextMissingFile(t *testing.T) {
	d := tempDirs(t)
	cfg := minimalCfg()
	cfg.Candidate.Files = []string{"evidence/nonexistent.txt"}

	store := &stubStore{}
	ctx, err := candidate.RefreshContext(store, d, cfg, nil, false, false)
	if err != nil {
		t.Fatalf("RefreshContext: %v", err)
	}
	if len(ctx.Sources) != 1 {
		t.Fatalf("sources: got %d, want 1", len(ctx.Sources))
	}
	if ctx.Sources[0].Status != "missing" {
		t.Errorf("missing file status: got %q, want missing", ctx.Sources[0].Status)
	}
}

// TestContextForJobShape verifies the shape of ContextForJob output.
func TestContextForJobShape(t *testing.T) {
	c := &candidate.CandidateContext{
		Version:     "v1",
		CreatedAt:   "2026-11-02T10:00:00.000Z",
		Preferences: map[string]any{"seniority": "senior"},
		ReviewNotes: []string{"Strong Go background preferred"},
		Sources: []candidate.EvidenceSource{
			{
				ID: "cv.txt", Reference: "cv.txt", Hash: "abc",
				Status: "available", Category: "public-profile",
				Scope: "Public source snapshot; unverified statements remain attributed",
				Text: "Go engineer with 8 years experience.\n\nStrong in distributed systems.\n\nKubernetes and cloud architecture.\n\nSome irrelevant hobbies here.",
			},
		},
	}

	result, err := candidate.ContextForJob(c, "We need a Go engineer with Kubernetes experience", 50000)
	if err != nil {
		t.Fatalf("ContextForJob: %v", err)
	}
	if result.Overview == nil {
		t.Error("overview: expected non-nil")
	}
	// Should have selected at least one source.
	if len(result.Sources) != 1 {
		t.Fatalf("sources: got %d, want 1", len(result.Sources))
	}
	src := result.Sources[0]
	if src.ID != "cv.txt" {
		t.Errorf("source id: got %q, want cv.txt", src.ID)
	}
	if src.TotalChars == 0 {
		t.Error("totalChars: expected > 0")
	}
	// Verify the overview has the expected fields.
	overviewJSON, _ := json.Marshal(result.Overview)
	if !contains(string(overviewJSON), "preferences") {
		t.Error("overview: missing preferences")
	}
	if !contains(string(overviewJSON), "sourceStatus") {
		t.Error("overview: missing sourceStatus")
	}
}

// TestContextForJobBudgetError verifies that a too-small budget returns an error.
func TestContextForJobBudgetError(t *testing.T) {
	c := &candidate.CandidateContext{
		Version:   "v1",
		CreatedAt: "2026-11-02T10:00:00.000Z",
		Sources: []candidate.EvidenceSource{
			{ID: "cv.txt", Status: "available", Text: "This is a source with some text"},
		},
	}
	// Budget of 50 bytes is too small.
	_, err := candidate.ContextForJob(c, "job text", 50)
	if err == nil {
		t.Error("expected error for too-small budget, got nil")
	}
}

// TestMapToContext verifies round-trip serialisation through the store map format.
func TestMapToContext(t *testing.T) {
	c := &candidate.CandidateContext{
		Version:     "v1abc",
		CreatedAt:   "2026-11-02T10:00:00.000Z",
		Preferences: map[string]any{"key": "value"},
		ReviewNotes: []string{"Note"},
		Sources: []candidate.EvidenceSource{
			{ID: "cv.txt", Reference: "cv.txt", Hash: "abc", Status: "available", Text: "text"},
		},
	}
	data, _ := json.Marshal(c)
	var m map[string]any
	_ = json.Unmarshal(data, &m)

	back, err := candidate.MapToContext(m)
	if err != nil {
		t.Fatalf("MapToContext: %v", err)
	}
	if back.Version != c.Version {
		t.Errorf("version: got %q, want %q", back.Version, c.Version)
	}
	if len(back.Sources) != 1 {
		t.Errorf("sources: got %d, want 1", len(back.Sources))
	}
	if back.Sources[0].Text != "text" {
		t.Errorf("source text: got %q, want text", back.Sources[0].Text)
	}
}

func contains(s, substr string) bool { return len(s) >= len(substr) && (s == substr || len(substr) == 0 || (len(s) > 0 && containsStr(s, substr))) }
func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
