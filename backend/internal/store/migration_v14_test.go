package store_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/w3con/sifty/internal/store"
)

// TestMigrationV14 verifies that the v13 → v14 migration copies company block rules into
// stop_words and removes source block rules, leaving title block rules untouched. Opening the
// database also applies the additive v15 profile schema, so the reported version is 16.
//
// The setup opens a raw SQLite file, creates only the tables that the migration touches, sets
// user_version=13, inserts representative v13 data, and then opens it through store.Open so the
// migration runs in the same code path that production uses.
func TestMigrationV14(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "v13.sqlite")

	// --- build a minimal v13 database ---
	raw, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatalf("open raw db: %v", err)
	}

	// block_rules and stop_words are the only tables the migration reads or writes.
	for _, stmt := range []string{
		`PRAGMA journal_mode=WAL`,
		`CREATE TABLE block_rules(
			id         INTEGER PRIMARY KEY,
			kind       TEXT    NOT NULL,
			value      TEXT    NOT NULL,
			created_at TEXT    NOT NULL,
			UNIQUE(kind, value)
		)`,
		`CREATE TABLE stop_words(
			id         INTEGER PRIMARY KEY,
			word       TEXT    NOT NULL,
			created_at TEXT    NOT NULL,
			UNIQUE(word)
		)`,
		`PRAGMA user_version=13`,
	} {
		if _, err := raw.Exec(stmt); err != nil {
			t.Fatalf("setup %q: %v", stmt, err)
		}
	}

	// Insert data that exercises every migration branch:
	//   - company block rule → must be copied to stop_words
	//   - source block rule  → must be deleted (source control moves to search.yaml)
	//   - title block rule   → must remain in block_rules, not copied to stop_words
	//   - company duplicate  → stop_words already holds a matching word; INSERT OR IGNORE must skip
	for _, row := range []struct{ kind, value, createdAt string }{
		{"company", "acme", "2026-01-01T00:00:00Z"},
		{"company", "megacorp", "2026-01-02T00:00:00Z"},
		{"source", "LinkedIn", "2026-01-03T00:00:00Z"},
		{"title", "senior", "2026-01-04T00:00:00Z"},
	} {
		if _, err := raw.Exec(
			`INSERT INTO block_rules(kind, value, created_at) VALUES (?, ?, ?)`,
			row.kind, row.value, row.createdAt,
		); err != nil {
			t.Fatalf("insert block_rule %v: %v", row, err)
		}
	}
	// Pre-seed stop_words with "acme" to verify that INSERT OR IGNORE does not duplicate it.
	if _, err := raw.Exec(
		`INSERT INTO stop_words(word, created_at) VALUES (?, ?)`, "acme", "2025-12-01T00:00:00Z",
	); err != nil {
		t.Fatalf("insert stop_word: %v", err)
	}

	if err := raw.Close(); err != nil {
		t.Fatalf("close raw db: %v", err)
	}

	// --- run the migration via store.Open ---
	s, err := store.Open(path)
	if err != nil {
		t.Fatalf("store.Open after seeding v13: %v", err)
	}
	defer s.Close()

	// user_version must now be 16 (v14 data migration plus the additive v15 profile schema).
	var version int
	if err := s.DB.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		t.Fatalf("read user_version: %v", err)
	}
	if version != 18 {
		t.Errorf("user_version = %d, want 18", version)
	}

	// stop_words must contain exactly "acme" and "megacorp" (no duplicates for acme, no source).
	rows, err := s.DB.Query(`SELECT word FROM stop_words ORDER BY word`)
	if err != nil {
		t.Fatalf("query stop_words: %v", err)
	}
	var words []string
	for rows.Next() {
		var w string
		if err := rows.Scan(&w); err != nil {
			t.Fatalf("scan stop_word: %v", err)
		}
		words = append(words, w)
	}
	rows.Close()

	want := []string{"acme", "megacorp"}
	if len(words) != len(want) {
		t.Errorf("stop_words: got %v, want %v", words, want)
	} else {
		for i, w := range words {
			if w != want[i] {
				t.Errorf("stop_words[%d]: got %q, want %q", i, w, want[i])
			}
		}
	}

	// block_rules: company rules remain (they still serve the job-blocking dimension); only source
	// rules are deleted. So block_rules must have company/acme, company/megacorp, title/senior —
	// but no source rule.
	var remaining []struct{ kind, value string }
	rows2, err := s.DB.Query(`SELECT kind, value FROM block_rules ORDER BY kind, value`)
	if err != nil {
		t.Fatalf("query block_rules: %v", err)
	}
	for rows2.Next() {
		var r struct{ kind, value string }
		if err := rows2.Scan(&r.kind, &r.value); err != nil {
			t.Fatalf("scan block_rule: %v", err)
		}
		remaining = append(remaining, r)
	}
	rows2.Close()

	wantRules := []struct{ kind, value string }{
		{"company", "acme"},
		{"company", "megacorp"},
		{"title", "senior"},
	}
	if len(remaining) != len(wantRules) {
		t.Errorf("block_rules after migration: got %v (len %d), want %v", remaining, len(remaining), wantRules)
	} else {
		for i, got := range remaining {
			if got.kind != wantRules[i].kind || got.value != wantRules[i].value {
				t.Errorf("block_rules[%d] = {%q, %q}, want {%q, %q}", i, got.kind, got.value, wantRules[i].kind, wantRules[i].value)
			}
		}
	}

	// Source rule must be gone from block_rules.
	var srcCount int
	if err := s.DB.QueryRow(`SELECT COUNT(*) FROM block_rules WHERE kind='source'`).Scan(&srcCount); err != nil {
		t.Fatalf("count source rules: %v", err)
	}
	if srcCount != 0 {
		t.Errorf("source block rules remaining after migration: %d, want 0", srcCount)
	}
}
