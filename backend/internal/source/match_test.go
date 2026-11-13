package source

import (
	"testing"

	"github.com/w3con/sifty/internal/model"
)

func TestMatchStructuredJobKeepsQueryIdentity(t *testing.T) {
	jobs := []model.JobInput{
		{Title: "A", Company: "Acme", OriginalURL: "https://x.test/job?id=1", PublishedAt: "2025-01-01"},
		{Title: "B", Company: "Acme", OriginalURL: "https://x.test/job?id=2", PublishedAt: "2025-02-02"},
	}
	got, ok := matchStructuredJob(jobs, Ref{URL: "https://x.test/job?id=2"})
	if !ok || got.PublishedAt != "2025-02-02" {
		t.Fatalf("matched %v ok=%v; id=2 date must not take id=1's date", got.PublishedAt, ok)
	}
	if samePostingURL("https://x.test/job?id=1", "https://x.test/job?id=2") {
		t.Error("identity query parameters were dropped")
	}
}

func TestMatchStructuredJobNoGuessForUnmatchedMultipost(t *testing.T) {
	jobs := []model.JobInput{
		{Title: "A", Company: "Acme", OriginalURL: "https://x.test/job?id=1", PublishedAt: "2025-01-01"},
		{Title: "B", Company: "Acme", OriginalURL: "https://x.test/job?id=2", PublishedAt: "2025-02-02"},
	}
	got, ok := matchStructuredJob(jobs, Ref{
		URL:     "https://x.test/job?id=3",
		Partial: model.JobInput{Title: "Other", Company: "Acme"},
	})
	if ok {
		t.Fatalf("guessed unrelated posting %q", got.OriginalURL)
	}
	if got.PublishedAt != "" {
		t.Fatalf("unmatched page leaked a date: %q", got.PublishedAt)
	}
}

func TestSamePostingURLStripsOnlyTracking(t *testing.T) {
	if !samePostingURL("https://x.test/job?id=1&utm_source=feed", "https://x.test/job?id=1") {
		t.Error("known tracking parameter was not stripped")
	}
	if samePostingURL("https://x.test/job?id=1", "https://x.test/job?id=2") {
		t.Error("identity parameter was stripped")
	}
}
