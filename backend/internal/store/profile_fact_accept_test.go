package store_test

import (
	"errors"
	"testing"

	"github.com/w3con/sifty/internal/store"
)

// TestProfileFactAcceptIsAtomic catches three hidden failures that are invisible in the UI:
// a stale confirmation overwriting a newer profile, a failed history insert leaving the fact
// marked accepted, and a partial update of the current profile without its revision row.
func TestProfileFactAcceptIsAtomic(t *testing.T) {
	s := openProfileStore(t)
	const original = `{"identity":{"displayName":"Manual"}}`
	if _, err := s.CreateProfile("local", original); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	docID := "doc-1"
	if err := s.CreateProfileDocument(&store.ProfileDocumentRecord{
		ID: docID, ProfileID: "local", Kind: "md", OriginalName: "cv.md",
		ProcessingState: "analyzing", ExtractedText: "Go",
	}); err != nil {
		t.Fatalf("create document: %v", err)
	}
	if err := s.StartProfileDocumentAnalysis(docID, `{"runId":"run-1"}`); err != nil {
		t.Fatalf("start analysis: %v", err)
	}
	fact := &store.ProfileFactRecord{
		ID: "fact-1", ProfileID: "local", DocumentID: &docID,
		FieldKey: "skills", ValueJSON: `{"key":"go","name":"Go","level":"Production"}`,
		Quote: "Go", Origin: "document", State: "proposed",
	}
	published, err := s.CompleteProfileDocumentAnalysis(
		docID, "run-1", "analyzed", `{"runId":"run-1"}`, "", []*store.ProfileFactRecord{fact})
	if err != nil || !published {
		t.Fatalf("complete analysis: published=%v err=%v", published, err)
	}
	facts, err := s.ListProfileFacts("local", "")
	if err != nil || len(facts) != 1 {
		t.Fatalf("facts = %d, err=%v", len(facts), err)
	}
	factID := facts[0].ID

	// A no-profile-change decision is still revision-CAS guarded.
	if _, err := s.SetProfileFactStateCAS(factID, 0, []string{"proposed"}, "accepted"); !errors.As(err, new(*store.RevisionConflictError)) {
		t.Fatalf("stale no-change accept error = %v, want *RevisionConflictError", err)
	}
	assertFactAcceptUnchanged(t, s, original, factID)

	// Stale expected revision: neither profile, history nor fact state may change.
	_, _, err = s.AcceptProfileFact(factID, 0, `{"identity":{"displayName":"Stale"}}`, "x", "fact", "accepted")
	var conflict *store.RevisionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("stale accept error = %v, want *RevisionConflictError", err)
	}
	assertFactAcceptUnchanged(t, s, original, factID)

	// A forced history failure must roll back the fact state with the profile update.
	if _, err := s.DB.Exec(`
		CREATE TRIGGER fail_fact_revision
		BEFORE INSERT ON profile_revisions
		BEGIN
			SELECT RAISE(ABORT, 'forced history failure');
		END`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}
	if _, _, err := s.AcceptProfileFact(factID, 1, `{"identity":{"displayName":"Changed"}}`, "x", "fact", "accepted"); err == nil {
		t.Fatal("accept succeeded despite the forced history failure")
	}
	assertFactAcceptUnchanged(t, s, original, factID)
	if _, err := s.DB.Exec(`DROP TRIGGER fail_fact_revision`); err != nil {
		t.Fatalf("drop failure trigger: %v", err)
	}

	// A valid confirmation writes the profile, its history row and the fact state together.
	rec, accepted, err := s.AcceptProfileFact(factID, 1, `{"identity":{"displayName":"Changed"}}`, "x", "fact", "accepted")
	if err != nil {
		t.Fatalf("valid accept: %v", err)
	}
	if rec.Revision != 2 || accepted.State != "accepted" {
		t.Fatalf("revision=%d state=%q, want 2/accepted", rec.Revision, accepted.State)
	}
	var revisions int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM profile_revisions WHERE profile_id='local'`).Scan(&revisions); err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if revisions != 2 {
		t.Errorf("revision rows = %d, want 2", revisions)
	}
}

func assertFactAcceptUnchanged(t *testing.T, s *store.Store, original, factID string) {
	t.Helper()
	current, err := s.CurrentProfile()
	if err != nil {
		t.Fatalf("current profile: %v", err)
	}
	if current.Revision != 1 || current.Data != original {
		t.Errorf("profile after failed accept = rev %d data %s, want rev 1 %s",
			current.Revision, current.Data, original)
	}
	facts, err := s.ListProfileFacts("local", "")
	if err != nil {
		t.Fatalf("list facts: %v", err)
	}
	if len(facts) != 1 || facts[0].ID != factID || facts[0].State != "proposed" {
		t.Errorf("fact state after failed accept = %+v, want proposed", facts)
	}
	var revisions int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM profile_revisions WHERE profile_id='local'`).Scan(&revisions); err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if revisions != 1 {
		t.Errorf("revision rows = %d, want 1 after failed accept", revisions)
	}
}
