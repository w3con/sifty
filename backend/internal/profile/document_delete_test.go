package profile_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/config"
	"github.com/w3con/sifty/internal/dirs"
	"github.com/w3con/sifty/internal/profile"
	"github.com/w3con/sifty/internal/store"
)

// TestDeleteDocumentKeepsAcceptedFacts catches the hidden failures of deleting a profile file: the
// facts the owner already accepted from it must not be lost with it (they stay, listed as coming
// from a deleted file, while the file, its text and its open proposals are gone), and a file the
// owner configured in the data directory must be forgotten without being deleted or imported
// again on the next listing.
func TestDeleteDocumentKeepsAcceptedFacts(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "profile.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer s.Close()
	root := t.TempDir()
	cfg := &config.Config{Candidate: config.Candidate{Files: []string{"configured.md"}}}
	svc := profile.NewService(s, config.NewHolder(cfg), profile.WithDirs(dirs.Dirs{Root: root}))
	configuredPath := filepath.Join(root, "configured.md")
	if err := os.WriteFile(configuredPath, []byte("Owner's own notes"), 0o600); err != nil {
		t.Fatalf("write configured file: %v", err)
	}

	doc, err := svc.CreateDocument("cv.md", []byte("Jane Doe\nGo developer"))
	if err != nil {
		t.Fatalf("create document: %v", err)
	}
	// Analysis publishes two proposals; the owner accepts one of them through the real action.
	docID := doc.ID
	if err := s.StartProfileDocumentAnalysis(docID, `{"runId":"run-1"}`); err != nil {
		t.Fatalf("start analysis: %v", err)
	}
	proposal := func(id, field, value string) *store.ProfileFactRecord {
		return &store.ProfileFactRecord{
			ID: id, ProfileID: profile.LocalProfileID, DocumentID: &docID, FieldKey: field,
			ValueJSON: value, Quote: "Jane Doe", Origin: "document", State: profile.FactProposed,
		}
	}
	if published, err := s.CompleteProfileDocumentAnalysis(docID, "run-1", profile.DocumentAnalyzed,
		`{"runId":"run-1"}`, "", []*store.ProfileFactRecord{
			proposal("accepted-fact", "identity.displayName", `"Jane Doe"`),
			proposal("open-fact", "identity.headline", `"Go developer"`),
		}); err != nil || !published {
		t.Fatalf("publish proposals: published=%v err=%v", published, err)
	}
	current, err := svc.Get()
	if err != nil {
		t.Fatalf("get profile: %v", err)
	}
	if _, err := svc.AcceptFact("accepted-fact", current.Revision); err != nil {
		t.Fatalf("accept fact: %v", err)
	}
	original := filepath.Join(root, "evidence", "profile-documents", doc.ID, "original.md")
	if _, err := os.Stat(original); err != nil {
		t.Fatalf("original not stored: %v", err)
	}

	result, err := svc.DeleteDocument(doc.ID)
	if err != nil {
		t.Fatalf("delete document: %v", err)
	}
	if !result.FileRemoved || !result.TextRemoved || result.ProposalsRemoved != 1 || result.FactsKept != 1 {
		t.Errorf("deletion report = %+v, want file and text removed, 1 proposal removed, 1 fact kept", result)
	}
	if _, err := os.Stat(filepath.Dir(original)); !os.IsNotExist(err) {
		t.Errorf("stored file directory still exists (err=%v)", err)
	}
	docs, err := svc.ListDocuments()
	if err != nil || len(docs) != 1 || docs[0].OriginalName != "configured.md" {
		t.Fatalf("documents after delete = %+v (err=%v), want only the configured file", docs, err)
	}
	if _, err := svc.DocumentText(doc.ID); err == nil {
		t.Error("document text is still readable after delete")
	}
	facts, err := svc.ListFacts("")
	if err != nil {
		t.Fatalf("list facts: %v", err)
	}
	if len(facts) != 1 || facts[0].ID != "accepted-fact" || facts[0].State != profile.FactAccepted {
		t.Fatalf("facts after delete = %+v, want only the accepted fact", facts)
	}
	if !facts[0].DocumentDeleted || facts[0].DocumentName != "cv.md" {
		t.Errorf("accepted fact source = %q deleted=%v, want cv.md marked deleted",
			facts[0].DocumentName, facts[0].DocumentDeleted)
	}

	// Deleting the configured file forgets it; the owner's file stays on disk and does not return.
	// An analysis run replaces the document's metadata, which is how the import identity is lost.
	if err := s.UpdateProfileDocumentState(docs[0].ID, profile.DocumentAnalyzed, `{"runId":"run-2"}`, ""); err != nil {
		t.Fatalf("simulate analysis metadata: %v", err)
	}
	configured, err := svc.DeleteDocument(docs[0].ID)
	if err != nil {
		t.Fatalf("delete configured document: %v", err)
	}
	if configured.FileRemoved {
		t.Error("deleting a configured file reported its removal from disk")
	}
	if _, err := os.Stat(configuredPath); err != nil {
		t.Errorf("configured file was deleted from disk: %v", err)
	}
	if docs, err := svc.ListDocuments(); err != nil || len(docs) != 0 {
		t.Errorf("documents after deleting the configured file = %+v (err=%v), want none", docs, err)
	}
}
