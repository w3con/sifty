package profile_test

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/matchsvc"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
)

// TestProfileSaveUpdatesDerivedContextAndFastRanking catches the hidden failure where a saved
// confirmed skill/target role is shown in the editor but never reaches the derived context or
// the existing BM25 score, or where the context/score is attributed to the wrong revision.
func TestProfileSaveUpdatesDerivedContextAndFastRanking(t *testing.T) {
	root := t.TempDir()
	d := dirs.Dirs{Root: root}
	if err := d.Ensure(); err != nil {
		t.Fatalf("ensure dirs: %v", err)
	}
	s, err := store.Open(filepath.Join(root, "profile.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	rubJob, err := s.Upsert(model.JobInput{
		Title: "Senior Rust Engineer", Company: "Ledger Labs",
		OriginalURL: "https://example.test/rust", Source: "fixture",
		Description:       "Build Rust protocol infrastructure and blockchain systems.",
		DescriptionStatus: "full",
	}, false)
	if err != nil {
		t.Fatalf("upsert rust job: %v", err)
	}
	designJob, err := s.Upsert(model.JobInput{
		Title: "Product Designer", Company: "Studio",
		OriginalURL: "https://example.test/design", Source: "fixture",
		Description:       "Own design systems and user research for a SaaS product.",
		DescriptionStatus: "full",
	}, false)
	if err != nil {
		t.Fatalf("upsert design job: %v", err)
	}

	// No configured search expressions: any observed score change must come from the profile.
	cfg := &config.Config{}
	holder := config.NewHolder(cfg)
	svc := profile.NewService(s, holder, profile.WithDirs(d))
	svc.SetQuickMatchRefresher(matchsvc.New(s, holder))
	boot, err := svc.Get()
	if err != nil {
		t.Fatalf("bootstrap profile: %v", err)
	}

	next := boot.Data
	next.Identity.DisplayName = "Owner"
	next.TargetRoles = []string{"Rust Engineer"}
	next.Skills = []profile.Skill{{Key: "rust", Name: "Rust", Level: ""}}
	saved, err := svc.Update(boot.Revision, next)
	if err != nil {
		t.Fatalf("save profile: %v", err)
	}
	if saved.Revision != boot.Revision+1 {
		t.Fatalf("saved revision = %d, want %d", saved.Revision, boot.Revision+1)
	}
	if !saved.ContextCurrent {
		t.Fatalf("derived context is not current after save: %q", saved.ContextError)
	}
	if saved.ContextProfileRevision != saved.Revision {
		t.Errorf("context profile revision = %d, want %d", saved.ContextProfileRevision, saved.Revision)
	}

	rust, err := s.Get(rubJob)
	if err != nil {
		t.Fatalf("get rust job: %v", err)
	}
	design, err := s.Get(designJob)
	if err != nil {
		t.Fatalf("get design job: %v", err)
	}
	if rust.MatchScore <= 0 {
		t.Errorf("rust job match score = %d, want > 0 after saving Rust signal", rust.MatchScore)
	}
	if design.MatchScore != 0 {
		t.Errorf("unrelated design job match score = %d, want 0", design.MatchScore)
	}
	if rust.MatchProfileRevision != saved.Revision || design.MatchProfileRevision != saved.Revision {
		t.Errorf("score revisions = %d/%d, want saved revision %d",
			rust.MatchProfileRevision, design.MatchProfileRevision, saved.Revision)
	}
	if rust.MatchStale {
		t.Error("fresh score is marked stale")
	}

	raw, err := s.Context()
	if err != nil {
		t.Fatalf("read context: %v", err)
	}
	if raw["profileId"] != saved.ID || int(raw["profileRevision"].(float64)) != saved.Revision {
		t.Errorf("context profile identity = %v/%v, want %s/%d",
			raw["profileId"], raw["profileRevision"], saved.ID, saved.Revision)
	}
	prefs, _ := raw["preferences"].(map[string]any)
	if int(prefs["profileRevision"].(float64)) != saved.Revision {
		t.Errorf("context preferences profileRevision = %v, want %d", prefs["profileRevision"], saved.Revision)
	}
	sources, _ := raw["sources"].([]any)
	foundProfile := false
	for _, rawSource := range sources {
		source, _ := rawSource.(map[string]any)
		if source["id"] != "confirmed-profile" {
			continue
		}
		foundProfile = true
		text, _ := source["text"].(string)
		if text == "" || !strings.Contains(text, "Rust") || !strings.Contains(text, "Revision:") {
			t.Errorf("confirmed-profile source text missing confirmed data: %q", text)
		}
	}
	if !foundProfile {
		t.Error("derived context has no confirmed-profile source")
	}

	// An unknown skill level is preserved as unknown in the derived data, never promoted.
	skills, _ := prefs["skills"].([]any)
	if len(skills) != 1 {
		t.Fatalf("derived skills = %v, want one", skills)
	}
	skill := skills[0].(map[string]any)
	if skill["level"] != "" {
		t.Errorf("unknown skill level became %q in derived context", skill["level"])
	}

	// Force the next Quick Match recomputation to fail after the context save. GET must not infer
	// "current" from the context revision alone: the persisted results still carry the older
	// revision.
	bad := config.DefaultQuickMatch()
	bad.Weights.Role = -1
	holder.Set(&config.Config{Runtime: config.Runtime{QuickMatch: bad}})
	next = saved.Data
	next.TargetRoles = []string{"Product Designer"}
	second, err := svc.Update(saved.Revision, next)
	if err != nil {
		t.Fatalf("save after forced rescore failure: %v", err)
	}
	if !second.ContextCurrent {
		t.Errorf("context should be current after successful context save: %q", second.ContextError)
	}
	if second.FastRankingCurrent {
		t.Error("fast ranking reported current after forced rescore failure")
	}
	if second.DerivedCurrent {
		t.Error("combined derived state reported current after forced rescore failure")
	}
	if second.FastRankingError == "" {
		t.Error("fast ranking failure has no visible error")
	}
	fetched, err := svc.Get()
	if err != nil {
		t.Fatalf("GET after forced rescore failure: %v", err)
	}
	if fetched.ContextCurrent != true || fetched.FastRankingCurrent != false || fetched.DerivedCurrent != false {
		t.Errorf("GET recovered false current state: context=%v fast=%v derived=%v errors=%q/%q",
			fetched.ContextCurrent, fetched.FastRankingCurrent, fetched.DerivedCurrent,
			fetched.ContextError, fetched.FastRankingError)
	}
}
