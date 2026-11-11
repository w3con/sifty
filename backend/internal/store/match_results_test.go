package store_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/quickmatch"
	"github.com/w3con/sifty/internal/store"
)

func matchRules() matchcontract.Rules {
	return matchcontract.Rules{
		SchemaVersion: matchcontract.SchemaVersion, RulesVersion: matchcontract.RulesVersion,
		Weights: map[matchcontract.Group]float64{
			matchcontract.GroupRole:         0.25,
			matchcontract.GroupCompensation: 0.20, matchcontract.GroupGeography: 0.15,
			matchcontract.GroupConditions: 0.10,
		},
		RequiredWeight: 2, PreferredWeight: 1,
	}
}

func matchPolicy() matchcontract.SourcePolicy {
	return matchcontract.SourcePolicy{
		Version: "source-policy/1",
		Sources: map[string]matchcontract.SourcePolicyRule{
			"Wellfound": {Location: matchcontract.ReliabilityNeedsCorroboration},
		},
	}
}

// TestMatchResultCASPreservesStaleData catches the hidden failure where an in-flight profile,
// source or rule change is overwritten by an unconditional result write, and where the legacy
// BM25 rescore overwrites a deterministic Quick Match score.
func TestMatchResultCASPreservesStaleData(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "results.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	if _, err := s.CreateProfile("local", `{"targetRoles":["Product Designer"],"skills":[]}`); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	id, err := s.Upsert(model.JobInput{
		Title: "Rust Engineer", Company: "Acme", OriginalURL: "https://example.com/rust",
		Description: "Requirements:\n- Rust\n", DescriptionStatus: "available structured description",
		Source: "Wellfound", Location: "Berlin, Germany",
	}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	src, err := s.CaptureMatchInput(id)
	if err != nil {
		t.Fatalf("CaptureMatchInput: %v", err)
	}
	policy := matchPolicy()
	facts, err := jobextract.Extract(src, jobextract.Policy{Source: policy})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if err := s.SaveMatchFacts(facts); err != nil {
		t.Fatalf("SaveMatchFacts: %v", err)
	}

	rules := matchRules()
	rulesID, _ := matchcontract.RulesIdentity(rules)
	policyID, _ := matchcontract.SourcePolicyIdentity(policy)
	snapshot := matchcontract.ProfileSnapshot{ProfileID: "local", ProfileRevision: 1}
	result, err := quickmatch.Evaluate(snapshot, facts, rules)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	result.Provenance.RulesIdentity = rulesID
	result.Provenance.SourcePolicyIdentity = policyID
	if err := s.SaveMatchResult(id, result); err != nil {
		t.Fatalf("SaveMatchResult: %v", err)
	}

	current, reasons, err := s.MatchResultCurrent(id, result, rulesID, facts.SourcePolicyVersion, policyID)
	if err != nil || !current || len(reasons) != 0 {
		t.Fatalf("fresh result current = %v reasons=%v err=%v", current, reasons, err)
	}

	// The legacy BM25 path must not overwrite a persisted Quick Match score.
	if err := s.Rescore("Rust"); err != nil {
		t.Fatalf("Rescore: %v", err)
	}
	job, err := s.Get(id)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if job.MatchScore != 0 {
		t.Fatalf("legacy BM25 overwrote Quick Match score = %d, want 0", job.MatchScore)
	}

	// A profile change invalidates the saved result; the old row remains preserved.
	if _, err := s.SaveProfile("local", 1, `{"targetRoles":["Rust Engineer"],"skills":[]}`, "update", "manual"); err != nil {
		t.Fatalf("SaveProfile: %v", err)
	}
	current, reasons, err = s.MatchResultCurrent(id, result, rulesID, facts.SourcePolicyVersion, policyID)
	if err != nil || current || len(reasons) == 0 {
		t.Fatalf("old profile result current = %v reasons=%v err=%v", current, reasons, err)
	}
	if err := s.SaveMatchResult(id, result); !errors.Is(err, store.ErrMatchResultStale) {
		t.Fatalf("stale profile write error = %v, want ErrMatchResultStale", err)
	}
	preserved, err := s.GetMatchResult(id)
	if err != nil || preserved == nil || preserved.Result.Provenance.ProfileRevision != 1 {
		t.Fatalf("stale write changed stored result: %+v err=%v", preserved, err)
	}

	// A source change is refused even when the profile revision is no longer authoritative.
	if _, err := s.DB.Exec("UPDATE jobs SET description=? WHERE id=?", "Requirements:\n- Go\n", id); err != nil {
		t.Fatalf("change source: %v", err)
	}
	current, reasons, err = s.MatchResultCurrent(id, result, rulesID, facts.SourcePolicyVersion, policyID)
	if err != nil || current || len(reasons) == 0 {
		t.Fatalf("changed source current = %v reasons=%v err=%v", current, reasons, err)
	}
	if err := s.SaveMatchResult(id, result); !errors.Is(err, store.ErrMatchResultStale) {
		t.Fatalf("stale source write error = %v, want ErrMatchResultStale", err)
	}

	// An active rules identity that differs by content invalidates the old result.
	changed := rules
	changed.Weights[matchcontract.GroupRole] = 0.20
	changed.Weights[matchcontract.GroupCompensation] = 0.35
	changedID, _ := matchcontract.RulesIdentity(changed)
	if changedID == rulesID {
		t.Fatal("rules identity did not change with content")
	}
	current, reasons, err = s.MatchResultCurrent(id, result, changedID, facts.SourcePolicyVersion, policyID)
	if err != nil || current || len(reasons) == 0 {
		t.Fatalf("changed rules current = %v reasons=%v err=%v", current, reasons, err)
	}
}

// TestMigrationV20PreservesQ1Data catches loss of profile, jobs or retained facts when the
// additive result table is introduced over an existing v19 database.
func TestMigrationV20PreservesQ1Data(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v19.sqlite")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, err := s.CreateProfile("local", `{"targetRoles":["Rust"]}`); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	id, err := s.Upsert(model.JobInput{
		Title: "Rust Engineer", Company: "Acme", OriginalURL: "https://example.com/m",
		Description: "Requirements:\n- Rust\n", DescriptionStatus: "available structured description",
		Source: "Wellfound",
	}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	src, err := s.CaptureMatchInput(id)
	if err != nil {
		t.Fatalf("Capture: %v", err)
	}
	f, err := jobextract.Extract(src, jobextract.Policy{Source: matchPolicy()})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if err := s.SaveMatchFacts(f); err != nil {
		t.Fatalf("SaveMatchFacts: %v", err)
	}
	// Simulate the pre-v20 state: v19 schema without the new result table.
	if _, err := s.DB.Exec("DROP TABLE match_job_results"); err != nil {
		t.Fatalf("drop result table: %v", err)
	}
	if _, err := s.DB.Exec("PRAGMA user_version=19"); err != nil {
		t.Fatalf("set version: %v", err)
	}
	if err := s.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	s, err = store.Open(path)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer s.Close()
	var version int
	if err := s.DB.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("version: %v", err)
	}
	if version != 22 {
		t.Fatalf("user_version = %d, want 22", version)
	}
	if p, err := s.CurrentProfile(); err != nil || p == nil || p.Revision != 1 {
		t.Fatalf("profile lost: %+v err=%v", p, err)
	}
	if got, err := s.GetMatchFacts(id); err != nil || got == nil || got.FactsHash != f.FactsHash {
		t.Fatalf("facts lost: %+v err=%v", got, err)
	}
	if got, err := s.GetMatchResult(id); err != nil || got != nil {
		t.Fatalf("new result table state = %+v err=%v, want empty", got, err)
	}
}
