package profile_test

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
)

// TestCancelledOrLateAnalysisPublishesNoFacts catches the hidden failure where a cancelled or
// superseded provider run completes late and replaces the proposal set after the user has already
// continued editing. The manual profile must keep its confirmed value and revision.
func TestCancelledOrLateAnalysisPublishesNoFacts(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "profile.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()

	svc := profile.NewService(s, config.NewHolder(&config.Config{}))
	current, err := svc.Get()
	if err != nil {
		t.Fatalf("bootstrap profile: %v", err)
	}
	manual := current.Data
	manual.Identity.DisplayName = "Manual value"
	saved, err := svc.Update(current.Revision, manual)
	if err != nil {
		t.Fatalf("manual update: %v", err)
	}

	docID := "doc-1"
	if err := s.CreateProfileDocument(&store.ProfileDocumentRecord{
		ID: docID, ProfileID: profile.LocalProfileID, Kind: "md", OriginalName: "cv.md",
		ProcessingState: profile.DocumentAnalyzing, ExtractedText: "Manual value\nGo",
	}); err != nil {
		t.Fatalf("create document: %v", err)
	}
	if err := s.StartProfileDocumentAnalysis(docID, `{"runId":"run-1"}`); err != nil {
		t.Fatalf("start run 1: %v", err)
	}
	if _, err := svc.CancelDocumentAnalysis(docID); err != nil {
		t.Fatalf("cancel analysis: %v", err)
	}
	// A retry starts another run. The old run's late completion must not publish over it.
	if err := s.StartProfileDocumentAnalysis(docID, `{"runId":"run-2"}`); err != nil {
		t.Fatalf("start run 2: %v", err)
	}

	lateFact := &store.ProfileFactRecord{
		ID: "late-fact", ProfileID: profile.LocalProfileID, DocumentID: &docID,
		FieldKey: "identity.displayName", ValueJSON: `"Late value"`, Quote: "Late value",
		Origin: "document", State: profile.FactProposed,
	}
	published, err := s.CompleteProfileDocumentAnalysis(
		docID, "run-1", profile.DocumentAnalyzed, `{"runId":"run-1"}`, "",
		[]*store.ProfileFactRecord{lateFact})
	if err != nil {
		t.Fatalf("complete late run: %v", err)
	}
	if published {
		t.Fatal("late result was published after cancellation/supersession")
	}
	facts, err := s.ListProfileFacts(profile.LocalProfileID, "")
	if err != nil {
		t.Fatalf("list facts: %v", err)
	}
	if len(facts) != 0 {
		t.Fatalf("late run stored %d facts, want 0", len(facts))
	}
	stored, err := s.CurrentProfile()
	if err != nil {
		t.Fatalf("current profile: %v", err)
	}
	if stored.Revision != saved.Revision {
		t.Errorf("revision = %d, want manual revision %d", stored.Revision, saved.Revision)
	}
	var storedData profile.ProfileData
	if err := json.Unmarshal([]byte(stored.Data), &storedData); err != nil {
		t.Fatalf("decode stored profile: %v", err)
	}
	if storedData.Identity.DisplayName != "Manual value" {
		t.Errorf("manual display name = %q, want Manual value", storedData.Identity.DisplayName)
	}
}
