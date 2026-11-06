package collect

import (
	"context"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
	"time"

	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/worker"
)

func TestJobspySidecar(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not installed; skipping jobspy sidecar")
	}
	root, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatalf("resolve root: %v", err)
	}
	w := worker.Worker{
		Command: "node",
		Args:    []string{filepath.Join(root, "workers", "jobspy", "index.mjs")},
		Dir:     root,
		Timeout: 30 * time.Second,
	}
	params := JobspyParams{Records: []map[string]any{
		{"title": "Senior Go Engineer", "company": "Acme", "job_url": "https://acme.test/j/1",
			"description": "Build systems", "location": "Remote", "date_posted": "2026-10-01", "site": "indeed"},
		{"title": "", "job_url": "https://x.test/skip"},       // filtered: no title
		{"company": "Nope", "job_url": "https://y.test/skip"}, // filtered: no title field
		{"title": "No URL", "company": "Z"},                   // filtered: no job_url
		{"title": "Platform Lead", "company": "Beta", "job_url": "https://beta.test/j/2", "site": "linkedin"},
	}}
	got, err := CollectViaJobspy(context.Background(), w, params)
	if err != nil {
		t.Fatalf("collect via jobspy: %v", err)
	}
	want := []model.JobInput{
		{Title: "Senior Go Engineer", Company: "Acme", OriginalURL: "https://acme.test/j/1", ApplyURL: nil,
			Description: "Build systems", DescriptionStatus: "jobspy description", Location: "Remote",
			PublishedAt: "2026-10-01", Source: "indeed"},
		{Title: "Platform Lead", Company: "Beta", OriginalURL: "https://beta.test/j/2", ApplyURL: nil,
			Description: "", DescriptionStatus: "missing: no jobspy description", Source: "linkedin"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("jobspy sidecar output mismatch:\n got %+v\nwant %+v", got, want)
	}
}
