package store

import (
	"database/sql"
	"fmt"
)

const targetVersion = 13

// baseSchema contains every CREATE TABLE IF NOT EXISTS and CREATE INDEX IF NOT EXISTS statement
// needed to produce the complete Sifty schema at user_version 10. Every statement is idempotent,
// so migrate executes the whole block unconditionally. Columns that the TypeScript store added
// after the initial CREATE TABLE (apply_status, tags, seen_at, applied_at on jobs;
// override_provider and routing on assessment_queue) are handled separately by addColumnIfMissing,
// so both a fresh database and an existing one already carrying those columns reach the same shape.
const baseSchema = `
CREATE TABLE IF NOT EXISTS migrations(version INTEGER PRIMARY KEY);

CREATE TABLE IF NOT EXISTS jobs(
  id               INTEGER PRIMARY KEY,
  identity         TEXT    NOT NULL,
  title            TEXT    NOT NULL,
  company          TEXT    NOT NULL,
  original_url     TEXT    NOT NULL,
  apply_url        TEXT,
  description      TEXT    NOT NULL,
  description_status TEXT  NOT NULL,
  location         TEXT,
  published_at     TEXT,
  expires_at       TEXT,
  status           TEXT    NOT NULL DEFAULT 'new',
  first_seen       TEXT    NOT NULL,
  last_seen        TEXT    NOT NULL,
  assessment       TEXT,
  assessment_error TEXT
);
CREATE INDEX IF NOT EXISTS jobs_identity ON jobs(identity);
CREATE INDEX IF NOT EXISTS jobs_apply    ON jobs(apply_url);

CREATE TABLE IF NOT EXISTS sightings(
  job_id INTEGER NOT NULL REFERENCES jobs(id),
  source TEXT    NOT NULL,
  url    TEXT    NOT NULL UNIQUE,
  seen   TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS health(
  source  TEXT PRIMARY KEY,
  status  TEXT NOT NULL,
  message TEXT NOT NULL,
  checked TEXT NOT NULL,
  count   INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS snapshots(
  id       INTEGER PRIMARY KEY,
  job_id   INTEGER REFERENCES jobs(id),
  captured TEXT NOT NULL,
  data     TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS contexts(
  version TEXT PRIMARY KEY,
  created TEXT NOT NULL,
  data    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS context_refreshes(
  id      INTEGER PRIMARY KEY,
  version TEXT NOT NULL,
  created TEXT NOT NULL,
  data    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS assessments(
  id      INTEGER PRIMARY KEY,
  job_id  INTEGER REFERENCES jobs(id),
  created TEXT NOT NULL,
  data    TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS runs(
  id      INTEGER PRIMARY KEY,
  started TEXT    NOT NULL,
  finished TEXT,
  status  TEXT    NOT NULL,
  detail  TEXT
);

CREATE TABLE IF NOT EXISTS locks(
  name     TEXT PRIMARY KEY,
  pid      INTEGER NOT NULL,
  acquired TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS job_aliases(
  old_id INTEGER PRIMARY KEY,
  new_id INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS collection_runs(
  id          INTEGER PRIMARY KEY,
  started     TEXT    NOT NULL,
  finished    TEXT,
  status      TEXT    NOT NULL,
  pid         INTEGER NOT NULL,
  mode        TEXT    NOT NULL,
  config_hash TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS source_runs(
  run_id   INTEGER NOT NULL,
  source   TEXT    NOT NULL,
  status   TEXT    NOT NULL,
  coverage TEXT    NOT NULL,
  PRIMARY KEY(run_id, source)
);

CREATE TABLE IF NOT EXISTS collection_events(
  id      INTEGER PRIMARY KEY,
  run_id  INTEGER NOT NULL,
  created TEXT    NOT NULL,
  source  TEXT,
  message TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS discovery_cache(
  key     TEXT    PRIMARY KEY,
  created INTEGER NOT NULL,
  data    TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS assessment_queue(
  job_id      INTEGER PRIMARY KEY,
  fingerprint TEXT    NOT NULL,
  provider    TEXT    NOT NULL,
  status      TEXT    NOT NULL,
  priority    INTEGER NOT NULL DEFAULT 0,
  attempts    INTEGER NOT NULL DEFAULT 0,
  next_at     INTEGER NOT NULL DEFAULT 0,
  started     INTEGER,
  finished    INTEGER,
  error       TEXT,
  owner       INTEGER
);

CREATE TABLE IF NOT EXISTS feedback(
  id      TEXT PRIMARY KEY,
  job_id  INTEGER NOT NULL,
  created TEXT    NOT NULL,
  path    TEXT    NOT NULL,
  data    TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS queue_state(
  key  TEXT PRIMARY KEY,
  data TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS workflow_history(
  id         INTEGER PRIMARY KEY,
  job_id     INTEGER NOT NULL,
  created    TEXT    NOT NULL,
  from_stage TEXT    NOT NULL,
  to_stage   TEXT    NOT NULL,
  confirmed  INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS application_drafts(
  job_id INTEGER PRIMARY KEY,
  data   TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS application_revisions(
  id      INTEGER PRIMARY KEY,
  job_id  INTEGER NOT NULL,
  created TEXT    NOT NULL,
  reason  TEXT    NOT NULL,
  data    TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS questionnaire_snapshots(
  id      INTEGER PRIMARY KEY,
  job_id  INTEGER NOT NULL,
  created TEXT    NOT NULL,
  data    TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS preparation_operations(
  id       INTEGER PRIMARY KEY,
  job_id   INTEGER NOT NULL,
  status   TEXT    NOT NULL,
  created  TEXT    NOT NULL,
  updated  TEXT    NOT NULL,
  pid      INTEGER,
  request  TEXT    NOT NULL,
  error    TEXT,
  progress TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS deleted_listings(hash TEXT PRIMARY KEY);

CREATE TABLE IF NOT EXISTS block_rules(
  id         INTEGER PRIMARY KEY,
  kind       TEXT    NOT NULL,
  value      TEXT    NOT NULL,
  created_at TEXT    NOT NULL,
  UNIQUE(kind, value)
);

CREATE TABLE IF NOT EXISTS stop_words(
  id         INTEGER PRIMARY KEY,
  word       TEXT    NOT NULL,
  created_at TEXT    NOT NULL,
  UNIQUE(word)
);

CREATE TABLE IF NOT EXISTS id_sequence(name TEXT PRIMARY KEY, value INTEGER NOT NULL);
`

// fts5Stmt creates the FTS5 virtual table. No SQL triggers are created for FTS maintenance.
//
// All three trigger variants (AI, AU, AD) were tested and proved unreliable under
// modernc.org/sqlite 3.54.1: a token inserted by one trigger-fired transaction accumulates in
// a committed level-0 FTS5 segment; subsequent delete+insert pairs (whether in another trigger
// or in Go code) produce additional segments whose negative entries do not reliably cancel the
// original positive entries, leaving stale tokens queryable. Direct Go-side delete+insert that
// does NOT follow a trigger-inserted row works correctly (confirmed by TestFTSStandardNoTriggers).
// All FTS maintenance is therefore done explicitly in Store.refreshFTS (called by Upsert after
// its transaction commits) and in Store.Delete (called before the row is removed from jobs).
const fts5Stmt = `CREATE VIRTUAL TABLE IF NOT EXISTS jobs_fts USING fts5(title, description, company, tags)`

// migrate checks the current user_version and, when below targetVersion, idempotently applies
// every schema change needed to reach it.
func migrate(db *sql.DB) error {
	var version int
	if err := db.QueryRow("PRAGMA user_version").Scan(&version); err != nil {
		return fmt.Errorf("read user_version: %w", err)
	}
	if version >= targetVersion {
		return nil
	}

	// Phase 1 — bring to v12 (original migration, idempotent).
	if version < 12 {
		// Create all tables idempotently. On an existing database, every CREATE TABLE IF NOT EXISTS
		// is a no-op; on a fresh one the full schema is built in one step.
		if _, err := db.Exec(baseSchema); err != nil {
			return fmt.Errorf("apply base schema: %w", err)
		}

		// Columns added by incremental TypeScript migrations that a fresh jobs / assessment_queue
		// table does not yet carry. addColumnIfMissing is a no-op when the column is present.

		// v2: apply_status
		if err := addColumnIfMissing(db, "jobs", "apply_status", "TEXT NOT NULL DEFAULT 'not resolved'"); err != nil {
			return err
		}
		// v3: tags
		if err := addColumnIfMissing(db, "jobs", "tags", "TEXT NOT NULL DEFAULT '[]'"); err != nil {
			return err
		}
		// v7: override_provider, routing
		if err := addColumnIfMissing(db, "assessment_queue", "override_provider", "TEXT"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "assessment_queue", "routing", "TEXT"); err != nil {
			return err
		}
		// v8: seen_at, applied_at
		if err := addColumnIfMissing(db, "jobs", "seen_at", "TEXT"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "jobs", "applied_at", "TEXT"); err != nil {
			return err
		}

		// v10: initialise id_sequence from the highest existing id, keeping any pre-existing value
		// when the row already exists (ON CONFLICT DO UPDATE with max).
		// The WHERE TRUE clause is required: without it, SQLite's parser reads the ON after the
		// subquery as the start of a JOIN condition and rejects the statement.
		_, err := db.Exec(`
			INSERT INTO id_sequence(name, value)
			SELECT 'jobs', max(n)
			FROM (
				SELECT COALESCE(max(id),     0) n FROM jobs
				UNION ALL
				SELECT COALESCE(max(old_id), 0) n FROM job_aliases
				UNION ALL
				SELECT COALESCE(max(new_id), 0) n FROM job_aliases
			)
			WHERE TRUE
			ON CONFLICT(name) DO UPDATE SET value = max(value, excluded.value)`)
		if err != nil {
			return fmt.Errorf("seed id_sequence: %w", err)
		}

		if _, err := db.Exec("PRAGMA user_version=12"); err != nil {
			return fmt.Errorf("set user_version=12: %w", err)
		}
	}

	// Phase 2 — v13: FTS5 index and match_score column.
	if version < 13 {
		// Add the BM25 relevance score column; default 0 until Rescore runs.
		if err := addColumnIfMissing(db, "jobs", "match_score", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		// Create the FTS5 virtual table (no triggers; see fts5Stmt comment for the rationale).
		if _, err := db.Exec(fts5Stmt); err != nil {
			return fmt.Errorf("apply FTS5 schema: %w", err)
		}
		// Backfill the FTS index from existing jobs (a no-op on a fresh database).
		if _, err := db.Exec(`
			INSERT INTO jobs_fts(rowid, title, description, company, tags)
			SELECT id, title, description, company, COALESCE(tags, '[]') FROM jobs`); err != nil {
			return fmt.Errorf("backfill jobs_fts: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version=13"); err != nil {
			return fmt.Errorf("set user_version=13: %w", err)
		}
	}

	return nil
}

// addColumnIfMissing adds column with definition to table only when it is not already present.
// It is the idempotent equivalent of ALTER TABLE … ADD COLUMN, which SQLite does not support
// with IF NOT EXISTS.
func addColumnIfMissing(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return fmt.Errorf("table_info(%s): %w", table, err)
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notnull, pk int
		var name, typ string
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &typ, &notnull, &dflt, &pk); err != nil {
			return fmt.Errorf("scan column in %s: %w", table, err)
		}
		if name == column {
			return nil // already present
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate columns of %s: %w", table, err)
	}
	if _, err := db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition); err != nil {
		return fmt.Errorf("add %s.%s: %w", table, column, err)
	}
	return nil
}
