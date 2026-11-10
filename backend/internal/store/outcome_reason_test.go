package store_test

import (
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/store"
)

// TestNegativeOutcomeReasonAndStaleGuard covers the two hidden failures this card
// introduces: a dismissed reason must survive a reload in the store, and a stale
// negative confirmation must not erase an application another actor just confirmed.
func TestNegativeOutcomeReasonAndStaleGuard(t *testing.T) {
	s := openFresh(t)
	id, err := s.Upsert(fixtureJobInput("outcome-source"), false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := s.Status(id, "preselected", false); err != nil {
		t.Fatalf("move to preselected: %v", err)
	}

	if _, err := s.StatusWithReason(id, "no", false, "bogus"); err == nil {
		t.Fatal("unknown outcome reason was accepted")
	}
	updated, err := s.StatusWithReason(id, "no", false, "closed")
	if err != nil {
		t.Fatalf("dismiss with reason: %v", err)
	}
	if updated.Stage != "no" || updated.OutcomeReason == nil || *updated.OutcomeReason != "closed" {
		t.Fatalf("stage/reason after dismiss = %q/%v, want no/closed", updated.Stage, updated.OutcomeReason)
	}
	reloaded, err := s.Get(id)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.OutcomeReason == nil || *reloaded.OutcomeReason != "closed" {
		t.Fatalf("reason after reload = %v, want closed", reloaded.OutcomeReason)
	}
	var historyReason string
	if err := s.DB.QueryRow(
		"SELECT reason FROM workflow_history WHERE job_id=? ORDER BY id DESC LIMIT 1", id).
		Scan(&historyReason); err != nil {
		t.Fatalf("read workflow history reason: %v", err)
	}
	if historyReason != "closed" {
		t.Fatalf("history reason = %q, want closed", historyReason)
	}

	// Another actor confirms this application; a stale negative outcome must fail
	// without erasing the applied timestamp, stage or history.
	second, err := s.Upsert(fixtureJobInput("outcome-source-2"), false)
	if err != nil {
		t.Fatalf("Upsert second: %v", err)
	}
	if _, err := s.Status(second, "preselected", false); err != nil {
		t.Fatalf("second preselected: %v", err)
	}
	applied, err := s.Status(second, "applied", true)
	if err != nil {
		t.Fatalf("applied: %v", err)
	}
	if applied.AppliedAt == nil {
		t.Fatal("applied timestamp missing")
	}
	var historyBefore int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM workflow_history WHERE job_id=?", second).Scan(&historyBefore); err != nil {
		t.Fatalf("history count: %v", err)
	}
	if _, err := s.StatusWithReason(second, "no", false, "stale"); err == nil {
		t.Fatal("stale negative outcome overwrote a confirmed application")
	}
	final, err := s.Get(second)
	if err != nil {
		t.Fatalf("final reload: %v", err)
	}
	if final.Stage != "applied" || final.AppliedAt == nil || *final.AppliedAt != *applied.AppliedAt {
		t.Fatalf("confirmed application was changed: stage=%q appliedAt=%v", final.Stage, final.AppliedAt)
	}
	if final.OutcomeReason != nil {
		t.Fatalf("negative reason leaked onto a confirmed application: %v", final.OutcomeReason)
	}
	var historyAfter int
	if err := s.DB.QueryRow("SELECT COUNT(*) FROM workflow_history WHERE job_id=?", second).Scan(&historyAfter); err != nil {
		t.Fatalf("history count after: %v", err)
	}
	if historyAfter != historyBefore {
		t.Fatalf("stale negative outcome added history: %d -> %d", historyBefore, historyAfter)
	}

	// An ordinary dismissal without a reason remains valid.
	third, err := s.Upsert(fixtureJobInput("outcome-source-3"), false)
	if err != nil {
		t.Fatalf("Upsert third: %v", err)
	}
	plain, err := s.Status(third, "no", false)
	if err != nil {
		t.Fatalf("ordinary dismissal: %v", err)
	}
	if plain.Stage != "no" || plain.OutcomeReason != nil {
		t.Fatalf("ordinary dismissal = %q/%v, want no/nil", plain.Stage, plain.OutcomeReason)
	}
}

// TestMigrationV16ToV17PreservesData catches the release blocker where an existing
// v16 database returned before the phase-17 columns were added. It builds a genuine
// v16-shaped database (new columns dropped, user_version 16), reopens it, and
// verifies both columns, the version, original job/history data and ordinary reads.
func TestMigrationV16ToV17PreservesData(t *testing.T) {
	path := filepath.Join(t.TempDir(), "v16.sqlite")
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	id, err := s.Upsert(fixtureJobInput("migration-source"), false)
	if err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	if _, err := s.Status(id, "preselected", false); err != nil {
		t.Fatalf("preselected: %v", err)
	}
	if _, err := s.StatusWithReason(id, "no", false, "closed"); err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	// Downgrade only the v17 additions, leaving the rest as a real v16 schema.
	for _, stmt := range []string{
		"ALTER TABLE jobs DROP COLUMN outcome_reason",
		"ALTER TABLE workflow_history DROP COLUMN reason",
		"PRAGMA user_version=16",
	} {
		if _, err := s.DB.Exec(stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	s.Close()

	s2, err := store.Open(path)
	if err != nil {
		t.Fatalf("reopen v16 database: %v", err)
	}
	defer s2.Close()

	var version int
	if err := s2.DB.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("user_version: %v", err)
	}
	if version != 18 {
		t.Fatalf("user_version = %d, want 18", version)
	}
	for table, column := range map[string]string{"jobs": "outcome_reason", "workflow_history": "reason"} {
		found := false
		rows, err := s2.DB.Query("PRAGMA table_info(" + table + ")")
		if err != nil {
			t.Fatalf("table_info %s: %v", table, err)
		}
		for rows.Next() {
			var cid int
			var name, typ string
			var notnull, pk int
			var dflt any
			if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
				rows.Close()
				t.Fatalf("scan table_info %s: %v", table, err)
			}
			if name == column {
				found = true
			}
		}
		rows.Close()
		if !found {
			t.Fatalf("%s.%s missing after migration", table, column)
		}
	}
	job, err := s2.Get(id)
	if err != nil {
		t.Fatalf("existing ordinary read: %v", err)
	}
	if job.Title != "Senior Go Engineer" || job.Stage != "no" || job.OutcomeReason != nil {
		t.Fatalf("existing job changed: title=%q stage=%q reason=%v", job.Title, job.Stage, job.OutcomeReason)
	}
	var fromStage, toStage string
	if err := s2.DB.QueryRow(
		"SELECT from_stage,to_stage FROM workflow_history WHERE job_id=? ORDER BY id DESC LIMIT 1", id).
		Scan(&fromStage, &toStage); err != nil {
		t.Fatalf("existing history: %v", err)
	}
	if fromStage != "preselected" || toStage != "no" {
		t.Fatalf("history changed: %s -> %s", fromStage, toStage)
	}
	// A post-migration write must still persist a new reason.
	second, err := s2.Upsert(fixtureJobInput("migration-source-2"), false)
	if err != nil {
		t.Fatalf("post-migration Upsert: %v", err)
	}
	if _, err := s2.Status(second, "preselected", false); err != nil {
		t.Fatalf("post-migration preselected: %v", err)
	}
	updated, err := s2.StatusWithReason(second, "no", false, "not_suitable")
	if err != nil || updated.OutcomeReason == nil || *updated.OutcomeReason != "not_suitable" {
		t.Fatalf("post-migration reason write: %v %v", updated.OutcomeReason, err)
	}
}
