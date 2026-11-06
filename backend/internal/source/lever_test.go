package source

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/fetch"
)

func TestLever(t *testing.T) {
	body, err := os.ReadFile("testdata/lever_postings.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	c, err := fetch.New(fetch.Options{
		AllowPrivate: true,
		Transport:    &fixedBodyTransport{body: string(body), ct: "application/json"},
	})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}
	adapter := &LeverAdapter{client: c}

	refs, err := adapter.Discover(context.Background(), "https://jobs.lever.co/betacorp")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("got %d refs, want 2", len(refs))
	}

	// First posting.
	r0 := refs[0]
	if r0.Partial.Title != "Head of Engineering" {
		t.Errorf("Title = %q", r0.Partial.Title)
	}
	if r0.Partial.Company != "Betacorp" {
		t.Errorf("Company = %q (want title-cased slug)", r0.Partial.Company)
	}
	if r0.Partial.Location != "Paris, France" {
		t.Errorf("Location = %q", r0.Partial.Location)
	}
	if r0.Partial.OriginalURL != "https://jobs.lever.co/betacorp/d8a9b2c0-1234-4abc-8def-000000000001" {
		t.Errorf("OriginalURL = %q", r0.Partial.OriginalURL)
	}
	// Exact-quote validation — the description must contain text from the fixture.
	if !strings.Contains(r0.Partial.Description, "world-class team") {
		t.Errorf("Description missing expected phrase; got: %q", r0.Partial.Description)
	}
	if r0.Partial.DescriptionStatus != "available structured description" {
		t.Errorf("DescriptionStatus = %q", r0.Partial.DescriptionStatus)
	}

	// Second posting.
	r1 := refs[1]
	if r1.Partial.Title != "Staff Product Designer" {
		t.Errorf("Title = %q", r1.Partial.Title)
	}
	if !strings.Contains(r1.Partial.Description, "design language and interaction patterns") {
		t.Errorf("Description missing expected phrase; got: %q", r1.Partial.Description)
	}

	// Resolve must return the partial (description already complete).
	j, err := adapter.Resolve(context.Background(), r0)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if j.Title != r0.Partial.Title {
		t.Errorf("Resolve title mismatch")
	}
}
