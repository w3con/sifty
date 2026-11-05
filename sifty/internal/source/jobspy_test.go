package source

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"testing"

	"github.com/w3con/sifty/internal/collect"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/worker"
)

// stubWorkerScript writes a small shell script that acts as a jobspy worker stub:
// it reads one JSON line from stdin, ignores it, and writes a canned response.
func stubWorkerScript(t *testing.T, postings []model.JobInput) worker.Worker {
	t.Helper()

	resultJSON, err := json.Marshal(postings)
	if err != nil {
		t.Fatalf("marshal stub postings: %v", err)
	}
	response := map[string]any{"ok": true, "result": json.RawMessage(resultJSON)}
	responseJSON, _ := json.Marshal(response)

	// Write a small shell stub that ignores stdin and echoes the canned response.
	scriptPath := t.TempDir() + "/stub-jobspy.sh"
	script := "#!/bin/sh\nread -r _line\nprintf '%s\\n' '" + string(responseJSON) + "'\n"
	if err := os.WriteFile(scriptPath, []byte(script), 0700); err != nil {
		t.Fatalf("write stub script: %v", err)
	}

	if _, err := exec.LookPath("sh"); err != nil {
		t.Skip("sh not available")
	}

	return worker.Worker{
		Command: "sh",
		Args:    []string{scriptPath},
		Env:     []string{"PATH=/usr/bin:/bin"},
	}
}

func TestJobspySource(t *testing.T) {
	wantPostings := []model.JobInput{
		{
			Title: "CTO", Company: "Acme", OriginalURL: "https://jobs.acme.com/cto",
			Description: "Lead the technology organization.", DescriptionStatus: "available",
			Location: "Remote", Source: "linkedin",
		},
		{
			Title: "VP Engineering", Company: "Beta Inc", OriginalURL: "https://beta.example/vpe",
			Description: "Build and grow the engineering team.", DescriptionStatus: "available",
			Location: "Paris", Source: "linkedin",
		},
	}

	w := stubWorkerScript(t, wantPostings)
	params := collect.JobspyParams{Sites: []string{"linkedin"}, Query: "CTO"}
	src := NewJobspySource(w, params)

	refs, err := src.Discover(context.Background(), "https://ignored.example/")
	if err != nil {
		t.Fatalf("Discover: %v", err)
	}
	if len(refs) != len(wantPostings) {
		t.Fatalf("got %d refs, want %d", len(refs), len(wantPostings))
	}
	for i, ref := range refs {
		want := wantPostings[i]
		if ref.Partial.Title != want.Title {
			t.Errorf("ref[%d].Title = %q, want %q", i, ref.Partial.Title, want.Title)
		}
		if ref.Partial.Company != want.Company {
			t.Errorf("ref[%d].Company = %q, want %q", i, ref.Partial.Company, want.Company)
		}
		if ref.URL != want.OriginalURL {
			t.Errorf("ref[%d].URL = %q, want %q", i, ref.URL, want.OriginalURL)
		}

		// Resolve must return the partial unchanged (no network fetch for jobspy).
		j, err := src.Resolve(context.Background(), ref)
		if err != nil {
			t.Fatalf("Resolve: %v", err)
		}
		if j.Title != want.Title || j.Description != want.Description {
			t.Errorf("Resolve[%d]: got {Title:%q Desc:%q}, want {Title:%q Desc:%q}",
				i, j.Title, j.Description, want.Title, want.Description)
		}
	}
}
