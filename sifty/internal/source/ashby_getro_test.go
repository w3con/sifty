package source

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/fetch"
)

func TestAshby(t *testing.T) {
	body, err := os.ReadFile("testdata/ashby_jobs.json")
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
	adapter := &AshbyAdapter{client: c}

	refs, err := adapter.Discover(context.Background(), "https://jobs.ashbyhq.com/gammacorp")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("got %d refs, want 2", len(refs))
	}

	r0 := refs[0]
	if r0.Partial.Title != "Engineering Manager, Infrastructure" {
		t.Errorf("Title = %q", r0.Partial.Title)
	}
	if r0.Partial.Company != "Gamma Corp" {
		t.Errorf("Company = %q (expected from organization.name in fixture)", r0.Partial.Company)
	}
	if r0.Partial.Location != "London, UK" {
		t.Errorf("Location = %q", r0.Partial.Location)
	}
	if r0.Partial.OriginalURL != "https://jobs.ashbyhq.com/gammacorp/b1234567-abcd-0001-efgh-000000000001" {
		t.Errorf("OriginalURL = %q", r0.Partial.OriginalURL)
	}
	if !strings.Contains(r0.Partial.Description, "developer experience") {
		t.Errorf("Description missing expected phrase; got: %q", r0.Partial.Description)
	}
	if r0.Partial.DescriptionStatus != "available structured description" {
		t.Errorf("DescriptionStatus = %q", r0.Partial.DescriptionStatus)
	}

	r1 := refs[1]
	if r1.Partial.Title != "Senior Data Engineer" {
		t.Errorf("Title = %q", r1.Partial.Title)
	}
	if !strings.Contains(r1.Partial.Description, "analytics and machine-learning") {
		t.Errorf("Description missing expected phrase; got: %q", r1.Partial.Description)
	}

	// Registry routing: Ashby adapter must be reachable from the registry.
	fc, _ := fetch.New(fetch.Options{AllowPrivate: true})
	generic := NewGenericSource(fc)
	reg := NewRegistry(generic)
	reg.Register(`jobs\.ashbyhq\.com`, adapter)
	if reg.For("jobs.ashbyhq.com") != Source(adapter) {
		t.Error("registry did not route jobs.ashbyhq.com to AshbyAdapter")
	}
}

func TestGetro(t *testing.T) {
	body, err := os.ReadFile("testdata/getro_listing.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	c, err := fetch.New(fetch.Options{
		AllowPrivate: true,
		Transport:    &fixedBodyTransport{body: string(body), ct: "text/html"},
	})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}
	adapter := &GetroAdapter{client: c}

	refs, err := adapter.Discover(context.Background(), "https://jobs.deltavc.example/")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("got %d refs, want 2", len(refs))
	}

	r0 := refs[0]
	if r0.Partial.Title != "CTO" {
		t.Errorf("Title = %q", r0.Partial.Title)
	}
	if r0.Partial.Company != "Delta Ventures Portfolio Co" {
		t.Errorf("Company = %q", r0.Partial.Company)
	}
	// Getro postings include skills and location from the board, not a full description.
	if !strings.Contains(r0.Partial.Description, "Go") {
		t.Errorf("Description missing skills; got: %q", r0.Partial.Description)
	}
	if r0.Partial.Location == "" {
		t.Errorf("Location is empty")
	}
	if r0.Partial.OriginalURL == "" {
		t.Errorf("OriginalURL is empty")
	}

	r1 := refs[1]
	if r1.Partial.Title != "VP Product" {
		t.Errorf("Title = %q", r1.Partial.Title)
	}
}
