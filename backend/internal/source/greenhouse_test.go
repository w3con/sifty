package source

import (
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/fetch"
)

// fixedBodyTransport is an http.RoundTripper that returns a fixed status, body
// and Content-Type regardless of the request URL.
type fixedBodyTransport struct {
	status int
	body   string
	ct     string
}

func (f *fixedBodyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	code := f.status
	if code == 0 {
		code = 200
	}
	return &http.Response{
		StatusCode: code,
		Header:     http.Header{"Content-Type": []string{f.ct}},
		Body:       io.NopCloser(strings.NewReader(f.body)),
		Request:    req,
	}, nil
}

// newGreenhouseFixture returns a GreenhouseAdapter backed by the given fixture file.
func newGreenhouseFixture(t *testing.T, fixturePath string) *GreenhouseAdapter {
	t.Helper()
	body, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read fixture %s: %v", fixturePath, err)
	}
	c, err := fetch.New(fetch.Options{
		AllowPrivate: true,
		Transport:    &fixedBodyTransport{body: string(body), ct: "application/json"},
	})
	if err != nil {
		t.Fatalf("fetch.New: %v", err)
	}
	return &GreenhouseAdapter{client: c}
}

func TestGreenhouse(t *testing.T) {
	adapter := newGreenhouseFixture(t, "testdata/greenhouse_jobs.json")

	// Use a fake boards.greenhouse.io URL; the transport ignores the actual host.
	refs, err := adapter.Discover(context.Background(), "https://boards.greenhouse.io/acmecorp")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != 2 {
		t.Fatalf("got %d refs, want 2", len(refs))
	}

	// First posting — verify required fields are exact quotes from the fixture.
	r0 := refs[0]
	if r0.Partial.Title != "Staff Software Engineer, Platform" {
		t.Errorf("Title = %q", r0.Partial.Title)
	}
	if r0.Partial.Location != "San Francisco, CA or Remote" {
		t.Errorf("Location = %q", r0.Partial.Location)
	}
	if r0.Partial.OriginalURL != "https://boards.greenhouse.io/acmecorp/jobs/4567890" {
		t.Errorf("OriginalURL = %q", r0.Partial.OriginalURL)
	}
	// Description must contain exact text from the fixture — no model paraphrase.
	if !strings.Contains(r0.Partial.Description, "billions of events per day") {
		t.Errorf("Description missing expected exact phrase; got: %q", r0.Partial.Description)
	}
	if r0.Partial.DescriptionStatus != "available structured description" {
		t.Errorf("DescriptionStatus = %q", r0.Partial.DescriptionStatus)
	}
	// The fixture supplies only updated_at; a modification time must never
	// become a claimed publication date.
	if r0.Partial.PublishedAt != "" {
		t.Errorf("updated_at leaked into PublishedAt: %q", r0.Partial.PublishedAt)
	}

	// Second posting.
	r1 := refs[1]
	if r1.Partial.Title != "Product Manager, Growth" {
		t.Errorf("Title = %q", r1.Partial.Title)
	}
	if !strings.Contains(r1.Partial.Description, "consumer and enterprise product lines") {
		t.Errorf("Description missing expected phrase; got: %q", r1.Partial.Description)
	}

	// Resolve returns the partial without a network fetch when description is complete.
	j, err := adapter.Resolve(context.Background(), r0)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if j.Title != r0.Partial.Title {
		t.Errorf("Resolve title mismatch: got %q", j.Title)
	}
}
