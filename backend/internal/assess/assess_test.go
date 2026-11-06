package assess

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type fixtureSource struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Category string `json:"category"`
	Scope    string `json:"scope"`
	Status   string `json:"status"`
}

type fixtureJob struct {
	ID                int    `json:"id"`
	Title             string `json:"title"`
	Company           string `json:"company"`
	Description       string `json:"description"`
	DescriptionStatus string `json:"descriptionStatus"`
	Location          string `json:"location"`
}

type fixture struct {
	Job            fixtureJob      `json:"job"`
	Sources        []fixtureSource `json:"sources"`
	Preferences    any             `json:"preferences"`
	ReviewNotes    []string        `json:"reviewNotes"`
	RawModelOutput RawModel        `json:"rawModelOutput"`
}

func readJSON[T any](t *testing.T, name string) T {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("parse %s: %v", name, err)
	}
	return out
}

// TestAssessParity runs the Go assessment pipeline with the same injected model output the TS
// reference used, and checks the grounded result matches the TypeScript golden within tolerance.
func TestAssessParity(t *testing.T) {
	fx := readJSON[fixture](t, "assess-fixture.json")
	golden := readJSON[Result](t, "assess-golden.json")

	var sources []EvidenceSource
	for _, s := range fx.Sources {
		sources = append(sources, EvidenceSource{
			ID: s.ID, Status: s.Status, Category: s.Category, Scope: s.Scope, Text: s.Text,
		})
	}
	c := CandidateContext{Preferences: fx.Preferences, ReviewNotes: fx.ReviewNotes, Sources: sources}
	job := Job{
		ID: fx.Job.ID, Title: fx.Job.Title, Company: fx.Job.Company,
		Description: fx.Job.Description, DescriptionStatus: fx.Job.DescriptionStatus, Location: fx.Job.Location,
	}
	p := Provider{Type: "compatible", Model: "test-model", Endpoint: "http://127.0.0.1:1234/v1",
		TimeoutMS: 120000, ContextChars: 200000}

	stub := func(_ Provider, _ string, _ any) (RawModel, error) { return fx.RawModelOutput, nil }
	got, err := Assess(c, job, p, "ASSESS", stub)
	if err != nil {
		t.Fatalf("assess: %v", err)
	}

	// Tolerance: the fit score is allowed to differ by at most 1; every grounded list must match.
	if math.Abs(got.Score-golden.Score) > 1 {
		t.Fatalf("score out of tolerance: got %v want %v", got.Score, golden.Score)
	}
	got.Score, golden.Score = 0, 0
	if !reflect.DeepEqual(got, golden) {
		gj, _ := json.MarshalIndent(got, "", "  ")
		wj, _ := json.MarshalIndent(golden, "", "  ")
		t.Fatalf("assessment differs from the TypeScript golden\n--- got ---\n%s\n--- want ---\n%s", gj, wj)
	}
}
