package prepare

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type fixtureSource struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	Category string `json:"category"`
	Scope    string `json:"scope"`
	Text     string `json:"text"`
}

type fixtureProvider struct {
	Model        string `json:"model"`
	ContextChars int    `json:"contextChars"`
}

type fixtureJob struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Company     string `json:"company"`
	Description string `json:"description"`
	Location    string `json:"location"`
}

type fixtureContext struct {
	Preferences any             `json:"preferences"`
	ReviewNotes []string        `json:"reviewNotes"`
	Sources     []fixtureSource `json:"sources"`
}

type fixture struct {
	Prompt                string          `json:"prompt"`
	QuestionnaireLanguage string          `json:"questionnaireLanguage"`
	Provider              fixtureProvider `json:"provider"`
	Job                   fixtureJob      `json:"job"`
	Context               fixtureContext  `json:"context"`
	Fields                []Field         `json:"fields"`
	ModelOutput           ModelOutput     `json:"modelOutput"`
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

// answerSet keys each answer by field id and renders it as canonical JSON, so the comparison is
// order-independent and insensitive to nil-versus-empty slice representation.
func answerSet(t *testing.T, answers []Answer) map[string]string {
	t.Helper()
	m := map[string]string{}
	for _, a := range answers {
		b, err := json.Marshal(a)
		if err != nil {
			t.Fatalf("marshal answer %s: %v", a.ID, err)
		}
		m[a.ID] = string(b)
	}
	return m
}

// TestPrepareParity runs the Go application-answer preparation with the same injected model output
// the TypeScript reference used, and checks the prepared answers match the TypeScript golden.
func TestPrepareParity(t *testing.T) {
	fx := readJSON[fixture](t, "prepare-fixture.json")
	golden := readJSON[Result](t, "prepare-golden.json")

	var sources []EvidenceSource
	for _, s := range fx.Context.Sources {
		sources = append(sources, EvidenceSource{
			ID: s.ID, Status: s.Status, Category: s.Category, Scope: s.Scope, Text: s.Text,
		})
	}
	c := CandidateContext{Preferences: fx.Context.Preferences, ReviewNotes: fx.Context.ReviewNotes, Sources: sources}
	job := Job{ID: fx.Job.ID, Title: fx.Job.Title, Company: fx.Job.Company,
		Description: fx.Job.Description, Location: fx.Job.Location}
	p := Provider{Model: fx.Provider.Model, ContextChars: fx.Provider.ContextChars}

	infer := func(_ Provider, _ string, _ any) (ModelOutput, error) { return fx.ModelOutput, nil }
	got, err := Prepare(c, job, p, fx.Prompt, fx.QuestionnaireLanguage, fx.Fields, "", infer)
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}

	if got.Language != golden.Language {
		t.Fatalf("language: got %q want %q", got.Language, golden.Language)
	}
	if got.SelectedFile != golden.SelectedFile {
		t.Fatalf("selectedFile: got %q want %q", got.SelectedFile, golden.SelectedFile)
	}
	gotSet, wantSet := answerSet(t, got.Answers), answerSet(t, golden.Answers)
	if !reflect.DeepEqual(gotSet, wantSet) {
		gj, _ := json.MarshalIndent(got.Answers, "", "  ")
		wj, _ := json.MarshalIndent(golden.Answers, "", "  ")
		t.Fatalf("prepared answers differ from the TypeScript golden\n--- got ---\n%s\n--- want ---\n%s", gj, wj)
	}
}

func scalar(s string) AnswerValue  { return AnswerValue{Str: s} }
func list(v ...string) AnswerValue { return AnswerValue{IsList: true, List: v} }
func intp(n int) *int              { return &n }

// TestPrepareGuards checks the deterministic validation guards reject the same conditions the
// TypeScript generateApplication throws on.
func TestPrepareGuards(t *testing.T) {
	c := CandidateContext{Preferences: map[string]any{}, ReviewNotes: []string{}, Sources: []EvidenceSource{
		{ID: "cv", Status: "available", Category: "experience", Scope: "CV", Text: "Built Go services.\n\nShipped Rust tooling."},
	}}
	job := Job{ID: 1, Title: "CTO", Company: "Acme", Description: "We need a CTO with Go.", Location: "Remote"}
	p := Provider{Model: "m", ContextChars: 50000}

	field := func(f Field) []Field { return []Field{f} }
	base := Field{ID: "q", Label: "Q", Type: "textarea", Options: []string{}, SourceURL: "u", Origin: "site", Language: "und"}
	withType := func(typ string, opts []string, max *int) Field {
		f := base
		f.Type = typ
		f.Options = opts
		f.MaxLength = max
		return f
	}
	model := func(a ModelAnswer) ModelOutput {
		return ModelOutput{Language: "en", SelectedFile: "cv.pdf", Answers: []ModelAnswer{a}}
	}
	run := func(fields []Field, out ModelOutput) error {
		_, err := Prepare(c, job, p, "PROMPT", "en", fields, "", func(Provider, string, any) (ModelOutput, error) { return out, nil })
		return err
	}

	cases := []struct {
		name   string
		fields []Field
		out    ModelOutput
		fail   bool
	}{
		{"no unanswered fields", field(Field{ID: "q", Type: "textarea", Value: scalar("done")}), model(ModelAnswer{ID: "q", Value: scalar("x")}), true},
		{"unknown question", field(base), model(ModelAnswer{ID: "other", Value: scalar("x")}), true},
		{"duplicate question", field(base), ModelOutput{Answers: []ModelAnswer{{ID: "q", Value: scalar("a")}, {ID: "q", Value: scalar("b")}}}, true},
		{"unsupported evidence", field(base), model(ModelAnswer{ID: "q", Value: scalar("x"), EvidenceIDs: []string{"E99"}}), true},
		{"type mismatch list for text", field(base), model(ModelAnswer{ID: "q", Value: list("a")}), true},
		{"type mismatch scalar for checkbox", field(withType("checkbox", []string{"A"}, nil)), model(ModelAnswer{ID: "q", Value: scalar("A")}), true},
		{"invented option", field(withType("radio", []string{"Yes", "No"}, nil)), model(ModelAnswer{ID: "q", Value: scalar("Maybe"), EvidenceIDs: []string{"E1"}}), true},
		{"exceeded maxLength", field(withType("text", []string{}, intp(3))), model(ModelAnswer{ID: "q", Value: scalar("toolong"), EvidenceIDs: []string{"E1"}}), true},
		{"valid answer", field(base), model(ModelAnswer{ID: "q", Value: scalar("I built Go services."), EvidenceIDs: []string{"E1"}}), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := run(tc.fields, tc.out)
			if tc.fail && err == nil {
				t.Fatalf("expected an error, got none")
			}
			if !tc.fail && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}
