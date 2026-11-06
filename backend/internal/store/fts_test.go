package store_test

import (
	"testing"

	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

// ftsMatchCount returns the number of rows in jobs_fts matching the given term.
func ftsMatchCount(t *testing.T, s *store.Store, term string) int {
	t.Helper()
	var count int
	if err := s.DB.QueryRow(
		"SELECT COUNT(*) FROM jobs_fts WHERE jobs_fts MATCH ?", term,
	).Scan(&count); err != nil {
		t.Fatalf("FTS MATCH %q: %v", term, err)
	}
	return count
}

// ftsJob returns a minimal job input whose indexed fields (title, description, company, tags) are
// controlled precisely. The title contains the word "cobalt" and the description does NOT, so
// 'cobalt' can be used as a canary: present when the first title is indexed, gone when it changes.
func ftsJob(source string) model.JobInput {
	applyURL := "https://example.com/apply/fts"
	return model.JobInput{
		Title:             "Cobalt Backend Developer",
		Company:           "FTS Corp",
		Tags:              []string{"backend"},
		OriginalURL:       "https://jobs.example.com/fts/1",
		ApplyURL:          &applyURL,
		ApplyStatus:       "direct",
		Description:       "We build reliable distributed systems.",
		DescriptionStatus: "full",
		Location:          "Remote",
		Source:            source,
	}
}

// TestFTSSync verifies that Go-level FTS maintenance keeps jobs_fts in sync with the jobs table.
//
// All FTS maintenance is done in Store.refreshFTS (called by Upsert after the main transaction
// commits) and in Store.Delete (called after the job row is removed). SQL triggers are not used
// because SQLite FTS5 pending-terms do not reliably propagate negative token entries across
// segment boundaries (see fts5Stmt comment in migrate.go for the full explanation).
func TestFTSSync(t *testing.T) {
	s := openFresh(t)

	// 1. Upsert a new job: 'cobalt' must appear in jobs_fts via refreshFTS.
	id, err := s.Upsert(ftsJob("src"), false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if n := ftsMatchCount(t, s, "cobalt"); n == 0 {
		t.Error("FTS after insert: 'cobalt' not found — refreshFTS did not add entry")
	}

	// 2. Re-Upsert with a changed title: refreshFTS must remove 'cobalt' and add 'rust'.
	// The description does not change and does not contain 'cobalt', so the absence of 'cobalt'
	// after the update is a reliable signal that the old title tokens were removed.
	j := ftsJob("src")
	j.Title = "Lead Rust Engineer"
	if _, err := s.Upsert(j, false); err != nil {
		t.Fatalf("re-Upsert with changed title: %v", err)
	}
	if n := ftsMatchCount(t, s, "cobalt"); n != 0 {
		t.Error("FTS after update: 'cobalt' still found — refreshFTS did not remove old entry")
	}
	if n := ftsMatchCount(t, s, "rust"); n == 0 {
		t.Error("FTS after update: 'rust' not found — refreshFTS did not add new entry")
	}

	// 3. Delete the job: Store.Delete must remove the FTS entry.
	if _, err := s.Status(id, "rejected", false); err != nil {
		t.Fatalf("Status to rejected: %v", err)
	}
	if _, _, err := s.Delete(id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if n := ftsMatchCount(t, s, "rust"); n != 0 {
		t.Error("FTS after delete: 'rust' still found — Delete did not remove FTS entry")
	}
}
