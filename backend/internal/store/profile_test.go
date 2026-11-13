package store_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/store"
)

func openProfileStore(t *testing.T) *store.Store {
	t.Helper()
	s, err := store.Open(filepath.Join(t.TempDir(), "profile.sqlite"))
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

// TestProfileSaveRevisionRejectsStale catches a stale compare-and-swap save overwriting a newer
// profile. The owner would not see this in the UI until previously saved values silently vanish.
func TestProfileSaveRevisionRejectsStale(t *testing.T) {
	s := openProfileStore(t)

	created, err := s.CreateProfile("local", `{"identity":{"displayName":"first"}}`)
	if err != nil {
		t.Fatalf("create profile: %v", err)
	}
	if created.Revision != 1 {
		t.Fatalf("created revision = %d, want 1", created.Revision)
	}

	saved, err := s.SaveProfile("local", 1, `{"identity":{"displayName":"second"}}`, "update", "manual")
	if err != nil {
		t.Fatalf("save expected 1: %v", err)
	}
	if saved.Revision != 2 {
		t.Fatalf("saved revision = %d, want 2", saved.Revision)
	}

	_, err = s.SaveProfile("local", 1, `{"identity":{"displayName":"stale"}}`, "update", "manual")
	var conflict *store.RevisionConflictError
	if !errors.As(err, &conflict) {
		t.Fatalf("stale save error = %v, want *store.RevisionConflictError", err)
	}
	if conflict.Current == nil || conflict.Current.Revision != 2 {
		t.Fatalf("conflict current = %+v, want revision 2", conflict.Current)
	}

	current, err := s.CurrentProfile()
	if err != nil {
		t.Fatalf("current profile: %v", err)
	}
	if current.Revision != 2 {
		t.Errorf("stored revision = %d, want 2 after stale save", current.Revision)
	}
	if current.Data != `{"identity":{"displayName":"second"}}` {
		t.Errorf("stored data = %s, want the newer saved value", current.Data)
	}
}

// TestProfileSaveRevisionRollsBackOnHistoryFailure catches a partial save that updates the
// current profile but loses the immutable revision snapshot, which would make the current value
// impossible to explain or restore.
func TestProfileSaveRevisionRollsBackOnHistoryFailure(t *testing.T) {
	s := openProfileStore(t)
	const original = `{"identity":{"displayName":"original"}}`
	if _, err := s.CreateProfile("local", original); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	if _, err := s.DB.Exec(`
		CREATE TRIGGER fail_profile_revision
		BEFORE INSERT ON profile_revisions
		BEGIN
			SELECT RAISE(ABORT, 'forced history failure');
		END`); err != nil {
		t.Fatalf("create failure trigger: %v", err)
	}

	if _, err := s.SaveProfile("local", 1, `{"identity":{"displayName":"changed"}}`, "update", "manual"); err == nil {
		t.Fatal("save succeeded despite the forced history-insert failure")
	}

	current, err := s.CurrentProfile()
	if err != nil {
		t.Fatalf("current profile: %v", err)
	}
	if current.Revision != 1 {
		t.Errorf("stored revision = %d, want 1 after rollback", current.Revision)
	}
	if current.Data != original {
		t.Errorf("stored data = %s, want original value after rollback", current.Data)
	}

	var revisions int
	if err := s.DB.QueryRow(
		`SELECT COUNT(*) FROM profile_revisions WHERE profile_id='local'`).Scan(&revisions); err != nil {
		t.Fatalf("count revisions: %v", err)
	}
	if revisions != 1 {
		t.Errorf("revision rows = %d, want only the bootstrap revision after rollback", revisions)
	}
}
