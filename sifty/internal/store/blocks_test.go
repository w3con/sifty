package store_test

import (
	"testing"

	"github.com/w3con/sifty/internal/model"
)

// TestBlockRuleCRUD covers AddBlockRule (including idempotency), ListBlockRules, and RemoveBlockRule.
func TestBlockRuleCRUD(t *testing.T) {
	s := openFresh(t)

	// Fresh store: no rules.
	list, err := s.ListBlockRules()
	if err != nil {
		t.Fatalf("ListBlockRules on empty store: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("empty store: got %d rules, want 0", len(list))
	}

	// AddBlockRule: insert a new rule.
	r, err := s.AddBlockRule("domain", "example.com")
	if err != nil {
		t.Fatalf("AddBlockRule: %v", err)
	}
	if r.Kind != "domain" || r.Value != "example.com" {
		t.Errorf("rule fields: got {%q %q}, want {domain example.com}", r.Kind, r.Value)
	}
	if r.ID == 0 {
		t.Error("rule.ID must not be zero")
	}

	// AddBlockRule: idempotent — same (kind, value) returns the same id.
	r2, err := s.AddBlockRule("domain", "example.com")
	if err != nil {
		t.Fatalf("AddBlockRule (idempotent): %v", err)
	}
	if r2.ID != r.ID {
		t.Errorf("idempotent insert: got id %d, want %d", r2.ID, r.ID)
	}

	// A different kind produces a different rule even for the same value.
	r3, err := s.AddBlockRule("company", "example.com")
	if err != nil {
		t.Fatalf("AddBlockRule company/example.com: %v", err)
	}
	if r3.ID == r.ID {
		t.Error("different kind must produce a different rule id")
	}

	// ListBlockRules: returns all inserted rules.
	list2, err := s.ListBlockRules()
	if err != nil {
		t.Fatalf("ListBlockRules: %v", err)
	}
	if len(list2) != 2 {
		t.Fatalf("ListBlockRules: got %d rules, want 2", len(list2))
	}

	// RemoveBlockRule: removes the first rule.
	if err := s.RemoveBlockRule(r.ID); err != nil {
		t.Fatalf("RemoveBlockRule: %v", err)
	}
	list3, _ := s.ListBlockRules()
	if len(list3) != 1 {
		t.Errorf("after remove: got %d rules, want 1", len(list3))
	}

	// RemoveBlockRule: returns an error for a non-existent id.
	if err := s.RemoveBlockRule(r.ID); err == nil {
		t.Error("removing non-existent rule: want error")
	}
}

// TestBlockMatchDomain verifies that a domain rule hides a job whose original_url host matches.
func TestBlockMatchDomain(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src") // OriginalURL = "https://jobs.example.com/postings/123"
	id, _ := s.Upsert(j, false)

	// No rules: not blocked.
	job, _ := s.Get(id)
	if job.Blocked {
		t.Error("no rules: job should not be blocked")
	}
	if job.BlockedBy != "" {
		t.Errorf("no rules: BlockedBy should be empty, got %q", job.BlockedBy)
	}

	// Domain rule matching the job's URL host.
	if _, err := s.AddBlockRule("domain", "jobs.example.com"); err != nil {
		t.Fatalf("AddBlockRule: %v", err)
	}
	job, _ = s.Get(id)
	if !job.Blocked {
		t.Error("domain rule: job should be blocked")
	}
	if job.BlockedBy != "domain:jobs.example.com" {
		t.Errorf("BlockedBy: got %q, want domain:jobs.example.com", job.BlockedBy)
	}

	// A rule for a different domain leaves the job unblocked.
	s2 := openFresh(t)
	s2.Upsert(j, false)
	s2.AddBlockRule("domain", "other.example.com")
	job2, _ := s2.Get(id)
	if job2.Blocked {
		t.Error("non-matching domain rule: job should not be blocked")
	}
}

// TestBlockMatchCompany verifies that a company rule hides a job by normalizeText(company).
func TestBlockMatchCompany(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src") // Company = "Acme Corp"
	id, _ := s.Upsert(j, false)

	// Company rule matching normalizeText("Acme Corp") = "acme corp".
	if _, err := s.AddBlockRule("company", "acme corp"); err != nil {
		t.Fatalf("AddBlockRule: %v", err)
	}
	job, _ := s.Get(id)
	if !job.Blocked {
		t.Error("company rule: job should be blocked")
	}
	if job.BlockedBy != "company:acme corp" {
		t.Errorf("BlockedBy: got %q, want company:acme corp", job.BlockedBy)
	}

	// A company rule for a different company leaves the job unblocked.
	s2 := openFresh(t)
	s2.Upsert(j, false)
	s2.AddBlockRule("company", "widgetco")
	job2, _ := s2.Get(id)
	if job2.Blocked {
		t.Error("non-matching company rule: job should not be blocked")
	}
}

// TestBlockMatchSource verifies the all-sources-blocked semantics.
func TestBlockMatchSource(t *testing.T) {
	s := openFresh(t)

	// Single-source job from LinkedIn.
	j1 := fixtureJobInput("LinkedIn")
	id1, _ := s.Upsert(j1, false)

	// Block LinkedIn: single-source job is blocked.
	s.AddBlockRule("source", "LinkedIn")
	job1, _ := s.Get(id1)
	if !job1.Blocked {
		t.Error("single-source job (LinkedIn blocked): should be blocked")
	}

	// Two-source job: LinkedIn + Indeed via same apply_url (triggers dedup).
	apply := "https://apply.megacorp.com/job/42"
	j2a := model.JobInput{
		Title: "Platform Engineer", Company: "MegaCorp",
		OriginalURL:       "https://linkedin.com/jobs/42",
		ApplyURL:          &apply,
		Description:       "Build and operate the platform infrastructure for a rapidly scaling product team.",
		DescriptionStatus: "full",
		Source:            "LinkedIn",
	}
	j2b := model.JobInput{
		Title: "Platform Engineer", Company: "MegaCorp",
		OriginalURL:       "https://indeed.com/job/42",
		ApplyURL:          &apply,
		Description:       "Build and operate the platform infrastructure for a rapidly scaling product team.",
		DescriptionStatus: "full",
		Source:            "Indeed",
	}
	id2, _ := s.Upsert(j2a, false)
	id2b, _ := s.Upsert(j2b, false)
	if id2 != id2b {
		t.Fatalf("dedup: same apply_url must produce same id, got %d and %d", id2, id2b)
	}

	// Only LinkedIn is blocked: Indeed not blocked → job not fully blocked.
	job2, _ := s.Get(id2)
	if job2.Blocked {
		t.Error("two-source job (LinkedIn blocked, Indeed not): should NOT be blocked")
	}

	// Block Indeed as well: all sources now blocked → job is blocked.
	s.AddBlockRule("source", "Indeed")
	job2b, _ := s.Get(id2)
	if !job2b.Blocked {
		t.Error("two-source job (LinkedIn + Indeed both blocked): should be blocked")
	}
}

// TestBlockJobDerivation verifies BlockJob derives the correct (kind, value) pair.
func TestBlockJobDerivation(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("LinkedIn") // OriginalURL = "https://jobs.example.com/postings/123"
	id, _ := s.Upsert(j, false)

	// Company dimension: always uses normalizeText(company), regardless of URL host.
	r, err := s.BlockJob(id, "company")
	if err != nil {
		t.Fatalf("BlockJob company: %v", err)
	}
	if r.Kind != "company" || r.Value != "acme corp" {
		t.Errorf("BlockJob company: got {%q %q}, want {company acme corp}", r.Kind, r.Value)
	}

	// Source dimension: creates a source rule for the first source.
	s2 := openFresh(t)
	id2, _ := s2.Upsert(j, false)
	r2, err := s2.BlockJob(id2, "source")
	if err != nil {
		t.Fatalf("BlockJob source: %v", err)
	}
	if r2.Kind != "source" || r2.Value != "LinkedIn" {
		t.Errorf("BlockJob source: got {%q %q}, want {source LinkedIn}", r2.Kind, r2.Value)
	}

	// Invalid dimension returns an error.
	if _, err := s.BlockJob(id, "unknown"); err == nil {
		t.Error("BlockJob unknown dimension: want error")
	}
}
