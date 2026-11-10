package server

import (
	"testing"

	"github.com/w3con/sifty/internal/model"
)

func TestMatchEnrichJobKeepsQueryIdentityAndDoesNotGuess(t *testing.T) {
	jobs := []model.JobInput{
		{Title: "A", Company: "Acme", OriginalURL: "https://x.test/job?id=1", PublishedAt: "2025-01-01"},
		{Title: "B", Company: "Acme", OriginalURL: "https://x.test/job?id=2", PublishedAt: "2025-02-02"},
	}
	job := &model.Job{Title: "B", Company: "Acme", OriginalURL: "https://x.test/job?id=2"}
	got, ok := matchEnrichJob(jobs, job)
	if !ok || got.PublishedAt != "2025-02-02" {
		t.Fatalf("matched %q ok=%v; id=2 date must not take id=1's date", got.PublishedAt, ok)
	}
	unmatched := &model.Job{Title: "Other", Company: "Acme", OriginalURL: "https://x.test/job?id=3"}
	if _, ok := matchEnrichJob(jobs, unmatched); ok {
		t.Error("ambiguous multi-posting page was guessed instead of left partial")
	}
}
