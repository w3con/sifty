package store_test

import (
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/w3con/sifty/internal/jobextract"
	"github.com/w3con/sifty/internal/matchcontract"
	"github.com/w3con/sifty/internal/model"
	"github.com/w3con/sifty/internal/store"
)

func TestMatchFactsGuardAgainstChangedSource(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "match.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()

	id, err := s.Upsert(model.JobInput{
		Title: "Rust Engineer", Company: "Acme",
		OriginalURL: "https://example.com/jobs/1",
		Description: "Requirements:\n- Rust\n", DescriptionStatus: "available structured description",
		Source: "Wellfound", Location: "Berlin, Germany",
	}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	src, err := s.CaptureMatchInput(id)
	if err != nil {
		t.Fatalf("CaptureMatchInput: %v", err)
	}
	policy := jobextract.Policy{Source: matchcontract.SourcePolicy{
		Version: "source-policy/1",
		Sources: map[string]matchcontract.SourcePolicyRule{
			"Wellfound": {Location: matchcontract.ReliabilityNeedsCorroboration},
		},
	}}
	f, err := jobextract.Extract(src, policy)
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if err := s.SaveMatchFacts(f); err != nil {
		t.Fatalf("SaveMatchFacts: %v", err)
	}
	got, err := s.GetMatchFacts(id)
	if err != nil {
		t.Fatalf("GetMatchFacts: %v", err)
	}
	if got == nil || got.FactsHash != f.FactsHash {
		t.Fatalf("stored facts = %+v, want hash %s", got, f.FactsHash)
	}

	tampered := f
	tampered.FactsHash = strings.Repeat("0", 64)
	if err := s.SaveMatchFacts(tampered); err == nil {
		t.Error("tampered facts hash was accepted")
	}
	incompatible := f
	incompatible.SchemaVersion = "2"
	if err := incompatible.SetFactsHash(); err == nil {
		t.Error("incompatible facts schema was accepted")
	}

	if _, err := s.DB.Exec("UPDATE jobs SET description=? WHERE id=?", "Requirements:\n- Go\n", id); err != nil {
		t.Fatalf("update source: %v", err)
	}
	err = s.SaveMatchFacts(f)
	if !errors.Is(err, store.ErrMatchSourceChanged) {
		t.Fatalf("stale fact write error = %v, want ErrMatchSourceChanged", err)
	}
	preserved, err := s.GetMatchFacts(id)
	if err != nil {
		t.Fatalf("GetMatchFacts after refused write: %v", err)
	}
	if preserved == nil || preserved.FactsHash != f.FactsHash {
		t.Errorf("refused write changed stored facts: %+v", preserved)
	}
}

func TestMatchFactsConcurrentWriterPreservesFacts(t *testing.T) {
	path := filepath.Join(t.TempDir(), "race.sqlite")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	id, err := s.Upsert(model.JobInput{
		Title: "Rust Engineer", Company: "Acme", OriginalURL: "https://example.com/jobs/3",
		Description: "Requirements:\n- Rust\n", DescriptionStatus: "available structured description",
		Source: "Wellfound",
	}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	src, err := s.CaptureMatchInput(id)
	if err != nil {
		t.Fatalf("CaptureMatchInput: %v", err)
	}
	f, err := jobextract.Extract(src, jobextract.Policy{})
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if err := s.SaveMatchFacts(f); err != nil {
		t.Fatalf("SaveMatchFacts: %v", err)
	}

	other, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open concurrent connection: %v", err)
	}
	defer other.Close()
	other.SetMaxOpenConns(1)
	if _, err := other.Exec("PRAGMA busy_timeout=0"); err != nil {
		t.Fatal(err)
	}
	if _, err := other.Exec("BEGIN IMMEDIATE"); err != nil {
		t.Fatalf("begin concurrent writer: %v", err)
	}
	if _, err := other.Exec("UPDATE jobs SET description=? WHERE id=?", "Requirements:\n- Go\n", id); err != nil {
		t.Fatalf("concurrent update: %v", err)
	}
	if _, err := s.DB.Exec("PRAGMA busy_timeout=1"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMatchFacts(f); err == nil {
		t.Fatal("save succeeded while another writer held an uncommitted source change")
	}
	if _, err := other.Exec("COMMIT"); err != nil {
		t.Fatalf("commit concurrent writer: %v", err)
	}
	if _, err := s.DB.Exec("PRAGMA busy_timeout=5000"); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveMatchFacts(f); !errors.Is(err, store.ErrMatchSourceChanged) {
		t.Fatalf("post-race save error = %v, want ErrMatchSourceChanged", err)
	}
	preserved, err := s.GetMatchFacts(id)
	if err != nil {
		t.Fatalf("GetMatchFacts after concurrent refusal: %v", err)
	}
	if preserved == nil || preserved.FactsHash != f.FactsHash {
		t.Errorf("concurrent writer changed stored facts: %+v", preserved)
	}
}

func TestMigrationV19PreservesV18Data(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v18.sqlite")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	id, err := s.Upsert(model.JobInput{
		Title: "Platform Engineer", Company: "Acme",
		OriginalURL:       "https://example.com/jobs/2",
		Description:       strings.Repeat("Retained vacancy text. ", 8),
		DescriptionStatus: "available structured description",
		Source:            "Wellfound",
	}, false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := s.DB.Exec("UPDATE jobs SET published_basis='datePosted', outcome_reason='salary' WHERE id=?", id); err != nil {
		t.Fatalf("seed job metadata: %v", err)
	}
	statements := []string{
		"INSERT INTO user_profiles(id,revision,data,created_at,updated_at) VALUES('local',3,'{\"targetRoles\":[\"Engineer\"]}','2026-01-01','2026-01-02')",
		"INSERT INTO profile_revisions(profile_id,revision,data,reason,origin,created_at) VALUES('local',3,'{\"targetRoles\":[\"Engineer\"]}','edit','manual','2026-01-02')",
		"INSERT INTO workflow_history(job_id,created,from_stage,to_stage,confirmed,reason) VALUES(1,'2026-01-03','prospect','saved',1,'owner choice')",
		"INSERT INTO application_drafts(job_id,data) VALUES(1,'{\"letter\":\"draft\"}')",
		"INSERT INTO application_revisions(job_id,created,reason,data) VALUES(1,'2026-01-04','edit','{\"letter\":\"draft\"}')",
	}
	for _, stmt := range statements {
		if _, err := s.DB.Exec(stmt); err != nil {
			t.Fatalf("seed v18 data %q: %v", stmt, err)
		}
	}
	s.Close()

	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw: %v", err)
	}
	for _, stmt := range []string{
		"DROP TABLE match_job_facts", "DROP TABLE match_job_inputs", "PRAGMA user_version=18",
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("prepare v18 %q: %v", stmt, err)
		}
	}
	raw.Close()

	s, err = store.Open(path)
	if err != nil {
		t.Fatalf("reopen v18: %v", err)
	}
	defer s.Close()
	var version int
	if err := s.DB.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatal(err)
	}
	if version != 22 {
		t.Errorf("user_version = %d, want 22", version)
	}
	var title, basis, reason string
	if err := s.DB.QueryRow("SELECT title, published_basis, outcome_reason FROM jobs WHERE id=?", id).
		Scan(&title, &basis, &reason); err != nil {
		t.Fatalf("read preserved job: %v", err)
	}
	if title != "Platform Engineer" || basis != "datePosted" || reason != "salary" {
		t.Errorf("job changed: title=%q basis=%q reason=%q", title, basis, reason)
	}
	var revision int
	if err := s.DB.QueryRow("SELECT revision FROM user_profiles WHERE id='local'").Scan(&revision); err != nil || revision != 3 {
		t.Errorf("profile revision = %d, %v; want 3", revision, err)
	}
	var historyReason, draft string
	if err := s.DB.QueryRow("SELECT reason FROM workflow_history WHERE job_id=?", id).Scan(&historyReason); err != nil || historyReason != "owner choice" {
		t.Errorf("workflow reason = %q, %v", historyReason, err)
	}
	if err := s.DB.QueryRow("SELECT data FROM application_drafts WHERE job_id=?", id).Scan(&draft); err != nil || draft != `{"letter":"draft"}` {
		t.Errorf("application draft = %q, %v", draft, err)
	}
	var fts int
	if err := s.DB.QueryRow("SELECT count(*) FROM jobs_fts WHERE rowid=?", id).Scan(&fts); err != nil || fts != 1 {
		t.Errorf("FTS row count = %d, %v; want 1", fts, err)
	}
	var tables int
	if err := s.DB.QueryRow("SELECT count(*) FROM sqlite_master WHERE type='table' AND name IN ('match_job_inputs','match_job_facts')").Scan(&tables); err != nil || tables != 2 {
		t.Errorf("quick match tables = %d, %v; want 2", tables, err)
	}
}
