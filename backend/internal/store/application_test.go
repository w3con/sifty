package store_test

import (
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/records"
)

// TestDraftRoundTrip verifies that a draft can be written and read back.
func TestDraftRoundTrip(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src")
	id, _ := s.Upsert(j, false)

	// No draft initially → empty draft.
	draft, err := s.GetDraft(id)
	if err != nil {
		t.Fatalf("GetDraft (empty): %v", err)
	}
	if draft.Revision != 0 {
		t.Errorf("empty draft revision: got %d, want 0", draft.Revision)
	}
	if draft.SelectedFile != "profile.pdf" {
		t.Errorf("empty draft selectedFile: got %q, want profile.pdf", draft.SelectedFile)
	}

	// Write a draft.
	draft.Notes = append(draft.Notes, "Ready to apply.")
	written, err := s.WriteDraft(draft, "initial")
	if err != nil {
		t.Fatalf("WriteDraft: %v", err)
	}
	if written.Revision != 1 {
		t.Errorf("after WriteDraft: revision = %d, want 1", written.Revision)
	}

	// Read it back.
	back, err := s.GetDraft(id)
	if err != nil {
		t.Fatalf("GetDraft after write: %v", err)
	}
	if back.Revision != 1 {
		t.Errorf("read-back revision: got %d, want 1", back.Revision)
	}
	if len(back.Notes) != 1 || back.Notes[0] != "Ready to apply." {
		t.Errorf("read-back notes: got %v, want [Ready to apply.]", back.Notes)
	}
}

// TestAddQuestion verifies that a manual question is appended to the draft.
func TestAddQuestion(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src")
	id, _ := s.Upsert(j, false)

	updated, err := s.AddQuestion(id, "Why do you want to join?", "textarea", nil)
	if err != nil {
		t.Fatalf("AddQuestion: %v", err)
	}
	if len(updated.Fields) != 1 {
		t.Fatalf("fields after AddQuestion: got %d, want 1", len(updated.Fields))
	}
	q, ok := updated.Fields[0].(map[string]any)
	if !ok {
		t.Fatalf("field is not a map: %T", updated.Fields[0])
	}
	if q["label"] != "Why do you want to join?" {
		t.Errorf("label: got %v", q["label"])
	}
	if q["origin"] != "manual" {
		t.Errorf("origin: got %v, want manual", q["origin"])
	}
}

// TestRememberedAnswer verifies the file-first write and read of a remembered answer.
func TestRememberedAnswer(t *testing.T) {
	d := dirs.Dirs{Root: filepath.Join(t.TempDir(), "data")}
	if err := d.Ensure(); err != nil {
		t.Fatalf("Ensure: %v", err)
	}

	rec := &records.AnswerRecord{
		ID:            "test-answer-id",
		Created:       "2026-11-02T10:00:00.000Z",
		Scope:         "general",
		JobID:         42,
		Company:       "Acme Corp",
		URL:           "https://example.com/apply/123",
		Question:      "Describe your Go experience.",
		Answer:        "I have 5 years of Go experience.",
		Language:      "en",
		FieldID:       "field-1",
		FieldRevision: 1,
		DraftRevision: 1,
		Evidence:      nil,
		Source:        "manual",
	}

	path, err := records.WriteAnswer(d, rec)
	if err != nil {
		t.Fatalf("WriteAnswer: %v", err)
	}
	if path == "" {
		t.Fatal("WriteAnswer: returned empty path")
	}

	// Read back.
	answers, err := records.ReadAnswers(d)
	if err != nil {
		t.Fatalf("ReadAnswers: %v", err)
	}
	if len(answers) != 1 {
		t.Fatalf("ReadAnswers: got %d, want 1", len(answers))
	}
	if answers[0].Question != rec.Question {
		t.Errorf("question: got %q, want %q", answers[0].Question, rec.Question)
	}
	if answers[0].Answer != rec.Answer {
		t.Errorf("answer: got %q, want %q", answers[0].Answer, rec.Answer)
	}
}

// TestFeedbackIndex verifies the database index for a feedback record.
func TestFeedbackIndex(t *testing.T) {
	s := openFresh(t)
	j := fixtureJobInput("src")
	id, _ := s.Upsert(j, false)

	data := map[string]any{
		"text": "Candidate is strong in distributed systems.",
		"scope": "general",
		"facts": []any{},
	}
	if err := s.IndexFeedback("fb-001", id, "feedback/2026-11-02-fb-001.md", data); err != nil {
		t.Fatalf("IndexFeedback: %v", err)
	}

	rows, err := s.FeedbackRows(id)
	if err != nil {
		t.Fatalf("FeedbackRows: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("FeedbackRows: got %d, want 1", len(rows))
	}
	if rows[0]["id"] != "fb-001" {
		t.Errorf("feedback id: got %v, want fb-001", rows[0]["id"])
	}
}
