package matchsvc_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/matchsvc"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

// TestRecomputeReusesFactsAcrossProfileAndRuleChanges catches the hidden failure where a
// profile or weight change re-fetches/re-extracts a source, or silently leaves an old result
// current after a configuration content change without a manual version bump.
func TestRecomputeReusesFactsAcrossProfileAndRuleChanges(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "svc.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	if _, err := s.CreateProfile("local", `{"targetRoles":["Rust Engineer"],"skills":[{"key":"rust","name":"Rust"}]}`); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	id, err := s.Upsert(model.JobInput{
		Title: "Rust Engineer", Company: "Acme", OriginalURL: "https://example.com/a",
		Description: "Requirements:\n- Rust\n", DescriptionStatus: "available structured description",
		Source: "Wellfound", Location: "Berlin, Germany",
	}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	holder := config.NewHolder(&config.Config{})
	svc := matchsvc.New(s, holder)
	first, err := svc.Recompute(context.Background())
	if err != nil {
		t.Fatalf("Recompute: %v", err)
	}
	if first.Processed != 1 || first.Scored != 1 {
		t.Fatalf("summary = %+v, want one scored job", first)
	}
	before, err := s.GetMatchResult(id)
	if err != nil || before == nil {
		t.Fatalf("result after first pass: %+v err=%v", before, err)
	}

	// Sentinel values are direct evidence that profile/rule-only changes did not re-extract.
	if _, err := s.DB.Exec("UPDATE match_job_facts SET updated_at='sentinel-facts' WHERE job_id=?", id); err != nil {
		t.Fatalf("sentinel facts: %v", err)
	}
	if _, err := s.DB.Exec("UPDATE match_job_inputs SET captured_at='sentinel-input' WHERE job_id=?", id); err != nil {
		t.Fatalf("sentinel input: %v", err)
	}

	// Profile-only change: reuse the persisted facts and input.
	if _, err := s.SaveProfile("local", 1, `{"targetRoles":["Product Designer"],"skills":[{"key":"rust","name":"Rust"}]}`, "update", "manual"); err != nil {
		t.Fatalf("save profile: %v", err)
	}
	if _, err := svc.Recompute(context.Background()); err != nil {
		t.Fatalf("recompute after profile: %v", err)
	}
	assertSentinels(t, s, id, "profile")
	afterProfile, err := s.GetMatchResult(id)
	if err != nil || afterProfile == nil || afterProfile.Result.Provenance.ProfileRevision != 2 {
		t.Fatalf("profile revision not refreshed: %+v err=%v", afterProfile, err)
	}

	// Rules content change without a manual rules_version bump: identity changes, facts reused.
	q := config.DefaultQuickMatch()
	q.Weights.Role = 0.20
	q.Weights.Professional = 0.35
	holder.Set(&config.Config{Runtime: config.Runtime{QuickMatch: q}})
	if _, err := svc.Recompute(context.Background()); err != nil {
		t.Fatalf("recompute after rules: %v", err)
	}
	assertSentinels(t, s, id, "rules")
	afterRules, err := s.GetMatchResult(id)
	if err != nil || afterRules == nil {
		t.Fatalf("result after rules: %+v err=%v", afterRules, err)
	}
	if afterRules.Result.Provenance.RulesIdentity == before.Result.Provenance.RulesIdentity {
		t.Fatal("rules identity did not change with configuration content")
	}

	// A result computed under the old rules must be refused after the active rules changed;
	// the newer row must remain untouched.
	err = svc.SaveComputedResult(id, before.Result)
	if !errors.Is(err, matchsvc.ErrConfigChanged) {
		t.Fatalf("old rules write error = %v, want ErrConfigChanged", err)
	}
	preserved, err := s.GetMatchResult(id)
	if err != nil || preserved == nil || preserved.Result.Provenance.RulesIdentity != afterRules.Result.Provenance.RulesIdentity {
		t.Fatalf("old rules write replaced newer row: %+v err=%v", preserved, err)
	}

	// Source-policy CONTENT change without a version bump must actually re-extract facts and
	// change their hash; the active view must then treat the new facts as current.
	q.SourceFields["Wellfound"] = config.QuickMatchSourceField{Location: "usable"}
	holder.Set(&config.Config{Runtime: config.Runtime{QuickMatch: q}})
	if _, err := svc.Recompute(context.Background()); err != nil {
		t.Fatalf("recompute after source policy: %v", err)
	}
	var factsAt, inputAt string
	if err := s.DB.QueryRow("SELECT updated_at FROM match_job_facts WHERE job_id=?", id).Scan(&factsAt); err != nil {
		t.Fatalf("facts timestamp: %v", err)
	}
	if err := s.DB.QueryRow("SELECT captured_at FROM match_job_inputs WHERE job_id=?", id).Scan(&inputAt); err != nil {
		t.Fatalf("input timestamp: %v", err)
	}
	if factsAt == "sentinel-facts" || inputAt == "sentinel-input" {
		t.Fatalf("source-policy content change reused old facts/input: %q/%q", factsAt, inputAt)
	}
	afterPolicy, err := s.GetMatchResult(id)
	if err != nil || afterPolicy == nil {
		t.Fatalf("result after source policy: %+v err=%v", afterPolicy, err)
	}
	if afterPolicy.Result.Provenance.FactsHash == afterRules.Result.Provenance.FactsHash {
		t.Fatal("source-policy content change did not change the facts hash")
	}
	facts, err := s.GetMatchFacts(id)
	if err != nil || facts == nil {
		t.Fatalf("facts after source policy: %+v err=%v", facts, err)
	}
	foundUsable := false
	for _, loc := range facts.Geography.HiringLocations {
		if loc.Evidence.Reliability == matchcontract.ReliabilityUsable {
			foundUsable = true
		}
	}
	if !foundUsable {
		t.Fatalf("source-policy change did not re-extract location reliability: %+v", facts.Geography.HiringLocations)
	}

	// A capture that succeeded after a source change but whose extraction/fact save failed must
	// not leave newer input paired with older facts forever. The next recompute must recover.
	if _, err := s.DB.Exec("UPDATE jobs SET description=? WHERE id=?", "Requirements:\n- Go\n", id); err != nil {
		t.Fatalf("change source after facts: %v", err)
	}
	input, err := s.CaptureMatchInput(id)
	if err != nil {
		t.Fatalf("capture only: %v", err)
	}
	inputHash, err := matchcontract.HashRetainedSource(input)
	if err != nil {
		t.Fatalf("hash input: %v", err)
	}
	if _, err := svc.Recompute(context.Background()); err != nil {
		t.Fatalf("recompute after capture-only: %v", err)
	}
	recovered, err := s.GetMatchFacts(id)
	if err != nil || recovered == nil {
		t.Fatalf("recovered facts: %+v err=%v", recovered, err)
	}
	if recovered.RawSourceHash != inputHash {
		t.Fatalf("facts source hash = %s, want retained input %s", recovered.RawSourceHash, inputHash)
	}
	view, err := svc.View(id)
	if err != nil || view == nil || view.Stale {
		t.Fatalf("recovered result not current: %+v err=%v", view, err)
	}
}

func assertSentinels(t *testing.T, s *store.Store, id int64, stage string) {
	t.Helper()
	var factsAt, inputAt string
	if err := s.DB.QueryRow("SELECT updated_at FROM match_job_facts WHERE job_id=?", id).Scan(&factsAt); err != nil {
		t.Fatalf("%s facts timestamp: %v", stage, err)
	}
	if err := s.DB.QueryRow("SELECT captured_at FROM match_job_inputs WHERE job_id=?", id).Scan(&inputAt); err != nil {
		t.Fatalf("%s input timestamp: %v", stage, err)
	}
	if factsAt != "sentinel-facts" || inputAt != "sentinel-input" {
		t.Fatalf("%s change re-extracted or re-captured facts/input: %q/%q", stage, factsAt, inputAt)
	}
}
