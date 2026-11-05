package store_test

import (
	"testing"

	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

// upsertTitle inserts a minimal job with the given title and returns its id.
func upsertTitle(t *testing.T, s *store.Store, title string) int64 {
	t.Helper()
	applyURL := "https://example.com/apply/" + title
	url := "https://jobs.example.com/" + title
	id, err := s.Upsert(model.JobInput{
		Title:             title,
		Company:           "Test Co",
		OriginalURL:       url,
		ApplyURL:          &applyURL,
		Description:       title + " role at Test Co.",
		DescriptionStatus: "full",
		Source:            "test",
	}, false)
	if err != nil {
		t.Fatalf("Upsert %q: %v", title, err)
	}
	return id
}

// matchScore reads the stored match_score for the given job id.
func matchScore(t *testing.T, s *store.Store, id int64) int {
	t.Helper()
	var score int
	if err := s.DB.QueryRow("SELECT match_score FROM jobs WHERE id=?", id).Scan(&score); err != nil {
		t.Fatalf("read match_score for id=%d: %v", id, err)
	}
	return score
}

// TestRescore verifies that Rescore computes BM25 scores, scales to [0, 100] and
// writes them to match_score, with unmatched roles set to 0.
func TestRescore(t *testing.T) {
	s := openFresh(t)

	// Insert three roles that differ in how well they match the query "blockchain engineer".
	// - "Blockchain Engineer" is a direct phrase match.
	// - "Senior Blockchain Developer" contains only "blockchain".
	// - "Frontend Designer" contains neither term.
	idHigh := upsertTitle(t, s, "Blockchain Engineer")
	idMid := upsertTitle(t, s, "Senior Blockchain Developer")
	idLow := upsertTitle(t, s, "Frontend Designer")

	// Use two separate terms (not one phrase) so partial matches can earn an intermediate score.
	// "Blockchain Engineer" matches both; "Senior Blockchain Developer" matches only "blockchain".
	query := store.BuildFTSQuery([]string{"blockchain", "engineer"})
	if err := s.Rescore(query); err != nil {
		t.Fatalf("Rescore: %v", err)
	}

	high := matchScore(t, s, idHigh)
	mid := matchScore(t, s, idMid)
	low := matchScore(t, s, idLow)

	t.Logf("match_score: Blockchain Engineer=%d, Senior Blockchain Developer=%d, Frontend Designer=%d",
		high, mid, low)

	// The best match must score 100; the unmatched role must score 0.
	// Matched roles scale to [1, 100] so the minimum matched role scores at least 1.
	if high != 100 {
		t.Errorf("best match: want 100, got %d", high)
	}
	if low != 0 {
		t.Errorf("unmatched role: want 0, got %d", low)
	}
	if mid == 0 {
		t.Errorf("partial match should score > 0, got 0")
	}
	// All scores must be in [0, 100].
	for _, sc := range []int{high, mid, low} {
		if sc < 0 || sc > 100 {
			t.Errorf("score %d out of range [0, 100]", sc)
		}
	}
	// Better match must rank higher.
	if high <= mid {
		t.Errorf("better match (%d) should outscore partial match (%d)", high, mid)
	}
}

// TestRescoreEmptyQuery resets all scores to 0 without touching the FTS index.
func TestRescoreEmptyQuery(t *testing.T) {
	s := openFresh(t)
	id := upsertTitle(t, s, "Blockchain Engineer")

	// Seed a non-zero score.
	if _, err := s.DB.Exec("UPDATE jobs SET match_score=42 WHERE id=?", id); err != nil {
		t.Fatalf("seed: %v", err)
	}

	if err := s.Rescore(""); err != nil {
		t.Fatalf("Rescore empty: %v", err)
	}
	if sc := matchScore(t, s, id); sc != 0 {
		t.Errorf("after empty Rescore: want 0, got %d", sc)
	}
}
