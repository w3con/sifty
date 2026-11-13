package store_test

import (
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/store"
)

// TestAssessmentStaleOnProfileAndContextRevision catches the hidden failure where a detailed
// assessment produced from an old profile/context (including one finishing late) is shown as
// current because only the vacancy fingerprint was compared.
func TestAssessmentStaleOnProfileAndContextRevision(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "stale.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	jobID, err := s.Upsert(fixtureJobInput("fixture"), false)
	if err != nil {
		t.Fatalf("upsert job: %v", err)
	}
	job, err := s.Get(jobID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}

	if _, err := s.CreateProfile("local", `{}`); err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if err := s.SaveContext(map[string]any{
		"version": "ctx-1", "profileId": "local", "profileRevision": 1,
	}); err != nil {
		t.Fatalf("save context 1: %v", err)
	}
	if _, err := s.SaveProfile("local", 1, `{}`, "update", "manual"); err != nil {
		t.Fatalf("save profile revision 2: %v", err)
	}

	assessment := func(contextVersion string, profileRevision int) map[string]any {
		return map[string]any{
			"score": 80.0, "summary": "fixture",
			"jobVersion":     store.JobVersionHash(job.Title, job.Company, job.Description, job.DescriptionStatus, job.Location),
			"contextVersion": contextVersion, "profileId": "local", "profileRevision": profileRevision,
		}
	}
	// Profile is at revision 2 while the context and assessment are still revision 1.
	if err := s.Assessment(jobID, assessment("ctx-1", 1)); err != nil {
		t.Fatalf("record old assessment: %v", err)
	}
	stale, err := s.Get(jobID)
	if err != nil {
		t.Fatalf("get after old assessment: %v", err)
	}
	if m := stale.Assessment.(map[string]any); m["stale"] != true {
		t.Errorf("assessment from older profile revision is current: %v", m["stale"])
	}

	// Refresh the context for revision 2; now a matching assessment is current.
	if err := s.SaveContext(map[string]any{
		"version": "ctx-2", "profileId": "local", "profileRevision": 2,
	}); err != nil {
		t.Fatalf("save context 2: %v", err)
	}
	if err := s.Assessment(jobID, assessment("ctx-2", 2)); err != nil {
		t.Fatalf("record current assessment: %v", err)
	}
	current, err := s.Get(jobID)
	if err != nil {
		t.Fatalf("get after current assessment: %v", err)
	}
	if m := current.Assessment.(map[string]any); m["stale"] != false {
		t.Errorf("matching assessment marked stale: %v", m["stale"])
	}

	// A late old result must not become current merely by being written last.
	if err := s.Assessment(jobID, assessment("ctx-1", 1)); err != nil {
		t.Fatalf("record late old assessment: %v", err)
	}
	late, err := s.Get(jobID)
	if err != nil {
		t.Fatalf("get after late assessment: %v", err)
	}
	if m := late.Assessment.(map[string]any); m["stale"] != true {
		t.Errorf("late old assessment appears current: %v", m["stale"])
	}

	// The list path must agree with Get.
	jobs, err := s.List("", "")
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(jobs) != 1 {
		t.Fatalf("list length = %d, want 1", len(jobs))
	}
	if m := jobs[0].Assessment.(map[string]any); m["stale"] != true {
		t.Errorf("list hid stale assessment: %v", m["stale"])
	}
}
