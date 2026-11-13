package store_test

import (
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

// TestDescriptionHTMLExistingRowInvariant verifies that when a vacancy already exists
// in the store with description_html IS NULL (i.e., collected before the L.2 feature),
// a subsequent Upsert with the same URL and a *different* description produced by the
// new converter does NOT overwrite the stored description.
//
// Hidden failure: the new PlainFromStructured output silently replaces the old CleanHTML
// output, changing every description hash and triggering re-extraction of ~2800 vacancies
// in the ASSESSMENT-3 worker.
func TestDescriptionHTMLExistingRowInvariant(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "test.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	const originalURL = "https://example.com/jobs/engineer"
	const originalDesc = "Senior engineer. 5+ years experience. Blockchain knowledge required. " +
		"Competitive salary and equity offered. We are building the future of DeFi and Web3."

	// First upsert: simulates the old path (no DescriptionHTML set, as if the row
	// was written before the converter existed). The description is what CleanHTML
	// would have produced.
	first := model.JobInput{
		Title:             "Senior Engineer",
		Company:           "Acme",
		OriginalURL:       originalURL,
		Description:       originalDesc,
		DescriptionHTML:   "", // not set = legacy row (description_html stays NULL)
		DescriptionStatus: "available cleaned html",
		Source:            "test",
	}
	id1, err := s.Upsert(first, false)
	if err != nil {
		t.Fatalf("first upsert: %v", err)
	}
	if id1 == 0 {
		t.Fatal("first upsert returned 0 (deleted?)")
	}

	// Confirm description_html IS NULL after the first upsert.
	var descHTMLAfterFirst *string
	var tmp *string
	row := s.DB.QueryRow("SELECT description_html FROM jobs WHERE id=?", id1)
	if err := row.Scan(&tmp); err != nil {
		t.Fatalf("read description_html: %v", err)
	}
	descHTMLAfterFirst = tmp
	if descHTMLAfterFirst != nil {
		t.Errorf("first upsert: expected description_html IS NULL, got %q", *descHTMLAfterFirst)
	}

	// Second upsert: same URL, but now DescriptionHTML and a *different* description
	// (simulating what PlainFromStructured(StructuredHTML(...)) would produce).
	const newDesc = "Senior engineer.\n\n5+ years experience.\n\nBlockchain knowledge required.\n\n" +
		"Competitive salary and equity offered.\n\nWe are building the future of DeFi and Web3."
	const newDescHTML = "<p>Senior engineer.</p><p>5+ years experience.</p>"

	second := model.JobInput{
		Title:             "Senior Engineer",
		Company:           "Acme",
		OriginalURL:       originalURL,
		Description:       newDesc,
		DescriptionHTML:   newDescHTML,
		DescriptionStatus: "available body html",
		Source:            "test",
	}
	id2, err := s.Upsert(second, false)
	if err != nil {
		t.Fatalf("second upsert: %v", err)
	}
	if id2 != id1 {
		t.Fatalf("second upsert returned a different id: got %d, want %d", id2, id1)
	}

	// The stored description must be byte-identical to the original.
	var storedDesc string
	if err := s.DB.QueryRow("SELECT description FROM jobs WHERE id=?", id1).Scan(&storedDesc); err != nil {
		t.Fatalf("read stored description: %v", err)
	}
	if storedDesc != originalDesc {
		t.Errorf("existing-row invariant violated: stored description changed\n  was: %q\n  got: %q", originalDesc, storedDesc)
	}

	// The description_html must still be NULL (legacy row; L.3 fills it locally).
	var descHTMLAfterSecond *string
	if err := s.DB.QueryRow("SELECT description_html FROM jobs WHERE id=?", id1).Scan(&descHTMLAfterSecond); err != nil {
		t.Fatalf("read description_html after second upsert: %v", err)
	}
	if descHTMLAfterSecond != nil {
		t.Errorf("existing-row invariant violated: description_html set on legacy row: %q", *descHTMLAfterSecond)
	}
}

// TestDescriptionHTMLNewRowStoresBothForms verifies that a freshly inserted vacancy
// (no prior row) has both description and description_html stored.
func TestDescriptionHTMLNewRowStoresBothForms(t *testing.T) {
	dir := t.TempDir()
	s, err := store.Open(filepath.Join(dir, "test2.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	input := model.JobInput{
		Title:             "Go Engineer",
		Company:           "BetaCorp",
		OriginalURL:       "https://beta.example/jobs/go-engineer",
		Description:       "We build distributed systems at scale. Go, Kubernetes, and DeFi knowledge required.",
		DescriptionHTML:   "<p>We build distributed systems at scale.</p><ul><li>Go</li><li>Kubernetes</li></ul>",
		DescriptionStatus: "available body html",
		Source:            "test",
	}
	id, err := s.Upsert(input, false)
	if err != nil {
		t.Fatalf("upsert: %v", err)
	}

	var storedDesc, storedHTML string
	if err := s.DB.QueryRow("SELECT description, COALESCE(description_html,'') FROM jobs WHERE id=?", id).
		Scan(&storedDesc, &storedHTML); err != nil {
		t.Fatalf("read stored fields: %v", err)
	}
	if storedDesc != input.Description {
		t.Errorf("description mismatch: got %q, want %q", storedDesc, input.Description)
	}
	if storedHTML != input.DescriptionHTML {
		t.Errorf("description_html mismatch: got %q, want %q", storedHTML, input.DescriptionHTML)
	}
}
