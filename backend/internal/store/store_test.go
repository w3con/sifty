package store_test

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/store"
)

// expectedTables is the complete set of tables the documented schema must contain at user_version 13.
// Derived from the target schema appendix in docs/architecture/backend-architecture.md section 13.
var expectedTables = []string{
	"migrations",
	"jobs",
	"jobs_fts",
	"sightings",
	"job_aliases",
	"id_sequence",
	"deleted_listings",
	"block_rules",
	"stop_words",
	"snapshots",
	"contexts",
	"context_refreshes",
	"assessments",
	"assessment_queue",
	"queue_state",
	"health",
	"runs",
	"collection_runs",
	"source_runs",
	"collection_events",
	"discovery_cache",
	"locks",
	"workflow_history",
	"feedback",
	"application_drafts",
	"application_revisions",
	"questionnaire_snapshots",
	"preparation_operations",
}

// expectedColumns maps table names to columns (including incrementally-added ones) that must
// all be present. Only key tables are listed; others are covered by expectedTables.
var expectedColumns = map[string][]string{
	"jobs": {
		"id", "identity", "title", "company", "original_url", "apply_url",
		"description", "description_status", "location", "published_at", "expires_at",
		"status", "first_seen", "last_seen", "assessment", "assessment_error",
		// added by incremental TypeScript migrations v2, v3, v8:
		"apply_status", "tags", "seen_at", "applied_at",
		// added by v13:
		"match_score",
	},
	"assessment_queue": {
		"job_id", "fingerprint", "provider", "status", "priority", "attempts",
		"next_at", "started", "finished", "error", "owner",
		// added by migration v7:
		"override_provider", "routing",
	},
}

// TestSchema_FreshDatabase opens a brand-new database and verifies it converges on the
// documented schema at user_version 14 with all tables and columns.
func TestSchema_FreshDatabase(t *testing.T) {
	s, err := store.Open(filepath.Join(t.TempDir(), "fresh.sqlite"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer s.Close()
	checkSchema(t, s.DB, "fresh database")
}

// TestSchema_ExistingDatabase copies the owner's live database to a temporary directory
// and opens it with the Go store. The migration runs to user_version 14,
// and the schema must match the documented shape.
//
// Set SIFTY_OWNER_DB to the path of a local jobs database to run this test; it is skipped
// when the variable is not set so the test suite runs cleanly in CI and public checkouts.
func TestSchema_ExistingDatabase(t *testing.T) {
	ownerDB := os.Getenv("SIFTY_OWNER_DB")
	if ownerDB == "" {
		t.Skip("SIFTY_OWNER_DB not set; skipping existing-database check")
	}
	if _, err := os.Stat(ownerDB); err != nil {
		t.Skipf("database at SIFTY_OWNER_DB not found (%v); skipping existing-database check", err)
	}

	data, err := os.ReadFile(ownerDB)
	if err != nil {
		t.Fatalf("read owner database: %v", err)
	}
	dst := filepath.Join(t.TempDir(), "jobs.sqlite")
	if err := os.WriteFile(dst, data, 0o600); err != nil {
		t.Fatalf("write copy: %v", err)
	}

	s, err := store.Open(dst)
	if err != nil {
		t.Fatalf("Open copy of owner database: %v", err)
	}
	defer s.Close()
	checkSchema(t, s.DB, "owner's existing database")
}

// checkSchema asserts that db is at user_version 14 in WAL mode and contains all documented
// tables and columns.
func checkSchema(t *testing.T, db *sql.DB, label string) {
	t.Helper()

	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("[%s] PRAGMA user_version: %v", label, err)
	}
	if version != 14 {
		t.Errorf("[%s] user_version = %d, want 14", label, version)
	}

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("[%s] PRAGMA journal_mode: %v", label, err)
	}
	if mode != "wal" {
		t.Errorf("[%s] journal_mode = %q, want wal", label, mode)
	}

	// Collect all present tables.
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' ORDER BY name")
	if err != nil {
		t.Fatalf("[%s] query tables: %v", label, err)
	}
	present := make(map[string]bool)
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("[%s] scan table name: %v", label, err)
		}
		present[name] = true
	}
	rows.Close()

	for _, table := range expectedTables {
		if !present[table] {
			t.Errorf("[%s] table %q is missing", label, table)
		}
	}

	// Verify columns in key tables.
	for table, wantCols := range expectedColumns {
		colRows, err := db.Query("PRAGMA table_info(" + table + ")")
		if err != nil {
			t.Fatalf("[%s] table_info(%s): %v", label, table, err)
		}
		colPresent := make(map[string]bool)
		for colRows.Next() {
			var cid, notnull, pk int
			var name, typ string
			var dflt sql.NullString
			if err := colRows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
				t.Fatalf("[%s] scan table_info(%s): %v", label, table, err)
			}
			colPresent[name] = true
		}
		colRows.Close()
		for _, col := range wantCols {
			if !colPresent[col] {
				t.Errorf("[%s] table %s: column %q missing", label, table, col)
			}
		}
	}
}
