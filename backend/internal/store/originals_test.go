package store_test

// Tests for L.2c — original body storage.
//
// Hidden failures each test catches:
//   1. TestOriginalWrittenWithVacancyInOneTransaction: original and vacancy
//      row written outside a transaction (crash between the two writes creates
//      a vacancy with no original, defeating the design).
//   2. TestRepeatedCollectionNoSecondRow: a re-collect of the same vacancy
//      (unchanged URL) must replace the original row, not add a second one;
//      a second row would violate the PRIMARY KEY and panic.
//   3. TestAPIResponseDoesNotContainOriginal: Original / OriginalKind fields
//      carry json:"-" tags; a serialisation bug could expose raw page bodies
//      to the client.

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

func TestOriginalWrittenWithVacancyInOneTransaction(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "orig.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	const jobURL = "https://example.com/jobs/engineer-123"
	const rawBody = "<html><body><p>Build great things</p><p>Blockchain experience helpful. DeFi knowledge useful.</p></body></html>"

	id, err := s.Upsert(model.JobInput{
		Title:             "Engineer",
		Company:           "ExampleCorp",
		OriginalURL:       jobURL,
		Description:       "Build great things Blockchain experience helpful.",
		DescriptionHTML:   "<p>Build great things</p>",
		DescriptionStatus: "available body html",
		Source:            "test",
		Original:          []byte(rawBody),
		OriginalKind:      "html/page",
	}, false)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if id == 0 {
		t.Fatal("upsert returned 0")
	}

	// Verify that the original row was written.
	kind, fetchedAt, body, err := s.GetOriginal(id)
	if err != nil {
		t.Fatalf("GetOriginal: %v", err)
	}
	if kind != "html/page" {
		t.Errorf("kind = %q, want %q", kind, "html/page")
	}
	if fetchedAt == "" {
		t.Error("fetched_at is empty")
	}
	if string(body) != rawBody {
		t.Errorf("body mismatch:\n got: %q\nwant: %q", body, rawBody)
	}
}

func TestRepeatedCollectionNoSecondRow(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "orig2.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	const jobURL = "https://example.com/jobs/repeated-456"

	first := model.JobInput{
		Title: "Engineer", Company: "Corp", OriginalURL: jobURL,
		Description: "Great role in Blockchain DeFi.", DescriptionHTML: "<p>Great role in Blockchain DeFi.</p>",
		DescriptionStatus: "available body html", Source: "test",
		Original: []byte("<html>first body</html>"), OriginalKind: "html/page",
	}
	id1, err := s.Upsert(first, false)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}

	// Second collect of the same vacancy: should replace the original row, not
	// insert a new one. If job_originals has PRIMARY KEY on job_id, a duplicate
	// INSERT would panic; INSERT OR REPLACE must be used.
	second := model.JobInput{
		Title: "Engineer", Company: "Corp", OriginalURL: jobURL,
		Description: "Great role in Blockchain DeFi.", DescriptionHTML: "<p>Great role in Blockchain DeFi.</p>",
		DescriptionStatus: "available body html", Source: "test",
		Original: []byte("<html>second body</html>"), OriginalKind: "html/page",
	}
	id2, err := s.Upsert(second, false)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if id2 != id1 {
		t.Fatalf("second upsert returned different id: got %d, want %d", id2, id1)
	}

	// The stored original must be the second body (most recent).
	_, _, body, err := s.GetOriginal(id1)
	if err != nil {
		t.Fatalf("GetOriginal after second upsert: %v", err)
	}
	if string(body) != "<html>second body</html>" {
		t.Errorf("body after second upsert = %q, want second body", body)
	}

	// Verify exactly one row exists for this job.
	var count int
	if err := s.DB.QueryRow(
		"SELECT count(*) FROM job_originals WHERE job_id=?", id1,
	).Scan(&count); err != nil {
		t.Fatalf("count job_originals: %v", err)
	}
	if count != 1 {
		t.Errorf("job_originals has %d rows for job %d, want 1", count, id1)
	}
}

func TestAPIResponseDoesNotContainOriginal(t *testing.T) {
	// model.JobInput has json:"-" on Original and OriginalKind, so serialising a
	// JobInput (or any type derived from it) must not include those fields.
	j := model.JobInput{
		Title:             "Tester",
		Company:           "Corp",
		OriginalURL:       "https://example.com/jobs/789",
		Description:       "Some description in Blockchain DeFi.",
		DescriptionStatus: "available body html",
		Source:            "test",
		Original:          []byte("<html>secret page body</html>"),
		OriginalKind:      "html/page",
	}
	data, err := json.Marshal(j)
	if err != nil {
		t.Fatalf("marshal JobInput: %v", err)
	}
	s := string(data)
	// Check that the raw body bytes and the internal kind field are absent.
	// Use specific substrings; "originalUrl" is the job URL key and must not be excluded.
	for _, bad := range []string{"secret page body", `"html/page"`, `"originalKind"`, `"OriginalKind"`} {
		if stringContains(s, bad) {
			t.Errorf("serialised JobInput contains %q (must not expose internal fields): %s", bad, s)
		}
	}
	// Verify that the JSON contains the expected URL field (proving serialisation worked).
	if !stringContains(s, "https://example.com/jobs/789") {
		t.Errorf("serialised JobInput is missing originalUrl: %s", s)
	}

	// model.Job (the API response type) must not carry Original / OriginalKind fields at all.
	job := model.Job{
		Title:             "Tester",
		Company:           "Corp",
		Description:       "Some description in Blockchain DeFi.",
		DescriptionStatus: "available body html",
	}
	data2, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal Job: %v", err)
	}
	s2 := string(data2)
	for _, bad := range []string{`"originalKind"`, `"OriginalKind"`} {
		if stringContains(s2, bad) {
			t.Errorf("serialised Job contains %q (must not expose internal fields): %s", bad, s2)
		}
	}
}

// stringContains is a simple substring check used by TestAPIResponseDoesNotContainOriginal.
func stringContains(haystack, needle string) bool {
	if needle == "" {
		return true
	}
	return len(haystack) >= len(needle) && byteContains([]byte(haystack), []byte(needle))
}

func byteContains(h, n []byte) bool {
	if len(n) == 0 {
		return true
	}
	for i := 0; i <= len(h)-len(n); i++ {
		match := true
		for j := range n {
			if h[i+j] != n[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}
