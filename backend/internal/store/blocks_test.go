package store_test

import (
	"testing"
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

// Source-block tests have been removed: source block rules are retired in migration v14.
// Source enable/disable is now the search.yaml enabled flag.

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

	// Source dimension is no longer supported — returns an error.
	if _, err := s.BlockJob(id, "source"); err == nil {
		t.Error("BlockJob source dimension: want error (source blocks retired)")
	}

	// Invalid dimension returns an error.
	if _, err := s.BlockJob(id, "unknown"); err == nil {
		t.Error("BlockJob unknown dimension: want error")
	}
}
