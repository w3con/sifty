package store

import (
	"database/sql"
	"fmt"
)

const targetVersion = 27

// baseSchema contains every CREATE TABLE IF NOT EXISTS and CREATE INDEX IF NOT EXISTS statement
// needed to produce the complete Sifty schema through user_version 16 (later versions add
// separate tables/columns below). Every statement is idempotent,
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

// profileSchema creates the additive v15 profile tables. user_profiles holds the current
// confirmed typed JSON and its revision; profile_documents records input metadata and extracted
// text while the original file stays under the data directory; profile_facts carries attributed
// proposals, manual assertions and their state; profile_revisions is the immutable snapshot
// history. All statements are idempotent and touch no existing table or row.
const profileSchema = `
CREATE TABLE IF NOT EXISTS user_profiles(
  id         TEXT    PRIMARY KEY,
  revision   INTEGER NOT NULL,
  data       TEXT    NOT NULL,
  created_at TEXT    NOT NULL,
  updated_at TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS profile_documents(
  id                TEXT PRIMARY KEY,
  profile_id        TEXT NOT NULL REFERENCES user_profiles(id),
  kind              TEXT NOT NULL DEFAULT '',
  original_name     TEXT NOT NULL DEFAULT '',
  storage_path      TEXT,
  source_url        TEXT,
  content_hash      TEXT,
  processing_state  TEXT NOT NULL DEFAULT 'pending',
  extracted_text    TEXT,
  analysis_metadata TEXT,
  error             TEXT,
  created_at        TEXT NOT NULL,
  updated_at        TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS profile_documents_profile ON profile_documents(profile_id);

CREATE TABLE IF NOT EXISTS profile_facts(
  id               TEXT PRIMARY KEY,
  profile_id       TEXT NOT NULL REFERENCES user_profiles(id),
  document_id      TEXT REFERENCES profile_documents(id),
  field_key        TEXT NOT NULL,
  value_json       TEXT NOT NULL,
  quote            TEXT,
  source_reference TEXT,
  origin           TEXT NOT NULL,
  state            TEXT NOT NULL,
  supersedes_id    TEXT REFERENCES profile_facts(id),
  created_at       TEXT NOT NULL,
  updated_at       TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS profile_facts_profile_field ON profile_facts(profile_id, field_key);

CREATE TABLE IF NOT EXISTS profile_revisions(
  id         INTEGER PRIMARY KEY,
  profile_id TEXT    NOT NULL REFERENCES user_profiles(id),
  revision   INTEGER NOT NULL,
  data       TEXT    NOT NULL,
  reason     TEXT    NOT NULL,
  origin     TEXT    NOT NULL,
  created_at TEXT    NOT NULL,
  UNIQUE(profile_id, revision)
);
CREATE INDEX IF NOT EXISTS profile_revisions_profile ON profile_revisions(profile_id);
`

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

	// Phase 3 — v14: unify exclusions by migrating company block rules into stop_words and
	// retiring source block rules (source control moves to the search.yaml enabled flag).
	if version < 14 {
		// Copy every company block rule into stop_words (INSERT OR IGNORE skips duplicates).
		if _, err := db.Exec(`
			INSERT OR IGNORE INTO stop_words(word, created_at)
			SELECT value, created_at FROM block_rules WHERE kind='company'
		`); err != nil {
			return fmt.Errorf("migrate company block rules to stop_words: %w", err)
		}
		// Remove all source block rules; source enable/disable is now the search.yaml enabled flag.
		if _, err := db.Exec(`DELETE FROM block_rules WHERE kind='source'`); err != nil {
			return fmt.Errorf("remove source block rules: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version=14"); err != nil {
			return fmt.Errorf("set user_version=14: %w", err)
		}
	}

	// Phase 4 — v15: additive profile tables. No existing table, row, config or feedback entry
	// is modified; the bootstrap import runs later through the profile service.
	if version < 15 {
		if _, err := db.Exec(profileSchema); err != nil {
			return fmt.Errorf("apply profile schema: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version=15"); err != nil {
			return fmt.Errorf("set user_version=15: %w", err)
		}
	}

	// Phase 5 — v16: record which profile revision produced each fast match_score. A NULL/0
	// value means no confirmed profile was used, so reads can report an old score honestly
	// rather than presenting it as current.
	if version < 16 {
		if err := addColumnIfMissing(db, "jobs", "match_profile_revision", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		if _, err := db.Exec("PRAGMA user_version=16"); err != nil {
			return fmt.Errorf("set user_version=16: %w", err)
		}
	}

	// Phase 6 — v17: durable manual outcome reason. Jobs keep the current reason so the
	// archive can render it after reload; workflow_history keeps the reason recorded with
	// each transition for audit. Scraped vacancy metadata is untouched.
	if version < 17 {
		if err := addColumnIfMissing(db, "jobs", "outcome_reason", "TEXT"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "workflow_history", "reason", "TEXT"); err != nil {
			return err
		}
		if _, err := db.Exec("PRAGMA user_version=17"); err != nil {
			return fmt.Errorf("set user_version=17: %w", err)
		}
	}

	// Phase 7 — v18: optional per-row publication basis. Existing rows keep
	// NULL (unverified legacy source date); only future genuine source
	// publication fields set a basis. No historical origin is invented.
	if version < 18 {
		if err := addColumnIfMissing(db, "jobs", "published_basis", "TEXT"); err != nil {
			return err
		}
		if _, err := db.Exec("PRAGMA user_version=18"); err != nil {
			return fmt.Errorf("set user_version=18: %w", err)
		}
	}

	// Phase 8 — v19: additive Quick Match input/fact storage. The retained snapshot and its
	// facts are separate rows so the source hash and the production versions stay inspectable
	// independently. No existing table, column, index or row is changed.
	if version < 19 {
		if _, err := db.Exec(quickMatchSchema); err != nil {
			return fmt.Errorf("apply quick match schema: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version=19"); err != nil {
			return fmt.Errorf("set user_version=19: %w", err)
		}
	}

	// Phase 9 — v20: additive Quick Match result storage. Existing tables, columns and rows are
	// unchanged; a result row is written only after the same-transaction input checks in
	// SaveMatchResult pass.
	if version < 20 {
		if _, err := db.Exec(quickMatchResultSchema); err != nil {
			return fmt.Errorf("apply quick match result schema: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version=20"); err != nil {
			return fmt.Errorf("set user_version=20: %w", err)
		}
	}

	// Phase 10 — v21: additive two-score model storage (docs/specs/quick-match.md section 14).
	// Two tables: the requirements a model extracted from one retained vacancy source, and the
	// verdicts a second pass gave against one profile revision. Existing tables, columns and rows
	// are unchanged.
	if version < 21 {
		if _, err := db.Exec(modelPassSchema); err != nil {
			return fmt.Errorf("apply model pass schema: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version=21"); err != nil {
			return fmt.Errorf("set user_version=21: %w", err)
		}
	}

	// Phase 11 — v22: additive storage of the two-score result per vacancy (the evaluator's output,
	// recomputed after every model pass, every profile or rules change). Existing tables, columns
	// and rows are unchanged.
	if version < 22 {
		if _, err := db.Exec(fitResultSchema); err != nil {
			return fmt.Errorf("apply fit result schema: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version=22"); err != nil {
			return fmt.Errorf("set user_version=22: %w", err)
		}
	}

	// Phase 12 — v23: additive structured HTML column for newly collected vacancies.
	// NULL for all existing rows; L.3 will fill them from stored text without network.
	// description (plain text) stays the input of search, Quick Match and model passes.
	if version < 23 {
		if err := addColumnIfMissing(db, "jobs", "description_html", "TEXT"); err != nil {
			return err
		}
		if _, err := db.Exec("PRAGMA user_version=23"); err != nil {
			return fmt.Errorf("set user_version=23: %w", err)
		}
	}

	// Phase 13 — v24: additive table job_originals stores the raw page body or feed
	// item of each vacancy as received, gzip-compressed. Never returned by the API,
	// never rendered; used to re-parse without a network fetch (spec §15.2.1).
	if version < 24 {
		if _, err := db.Exec(jobOriginalsSchema); err != nil {
			return fmt.Errorf("apply job_originals schema: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version=24"); err != nil {
			return fmt.Errorf("set user_version=24: %w", err)
		}
	}

	// Phase 14 — v25: additive needs_refetch column on jobs. Set by the local
	// backfill for glued rows (single continuous text without structure) that
	// require a page re-fetch to recover the full HTML. The refetch path clears it.
	if version < 25 {
		if err := addColumnIfMissing(db, "jobs", "needs_refetch", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		if _, err := db.Exec("PRAGMA user_version=25"); err != nil {
			return fmt.Errorf("set user_version=25: %w", err)
		}
	}

	// Phase 15 — v26: additive job_sections table stores the description split into
	// labelled blocks by heading. Derived from description_html via the section
	// splitter; rebuilt by BuildJobSections at startup and written with each new
	// vacancy in Upsert. Existing rows gain sections through the startup step.
	if version < 26 {
		if _, err := db.Exec(jobSectionsSchema); err != nil {
			return fmt.Errorf("apply job_sections schema: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version=26"); err != nil {
			return fmt.Errorf("set user_version=26: %w", err)
		}
	}

	// Phase 16 — v27: seed the catalogue_seq counter. The initial value is set to the
	// current maximum job id so the counter is already past every existing vacancy.
	// INSERT OR IGNORE ensures an existing row (from a previous partial run) is kept.
	if version < 27 {
		if _, err := db.Exec(`
			INSERT OR IGNORE INTO id_sequence(name, value)
			SELECT 'catalogue_seq', COALESCE(MAX(id), 0) FROM jobs`); err != nil {
			return fmt.Errorf("seed catalogue_seq: %w", err)
		}
		if _, err := db.Exec("PRAGMA user_version=27"); err != nil {
			return fmt.Errorf("set user_version=27: %w", err)
		}
	}

	return nil
}

const jobSectionsSchema = `
CREATE TABLE IF NOT EXISTS job_sections(
  job_id   INTEGER PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
  sections TEXT    NOT NULL
);
`

// fitResultSchema holds one two-score result per vacancy. The scalar columns repeat numbers that
// are also inside the record so the catalogue can order and the worker can prioritise without
// decoding every record; the record is the authority.
const fitResultSchema = `
CREATE TABLE IF NOT EXISTS match_fit_results(
  job_id           INTEGER PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
  profile_revision INTEGER NOT NULL,
  rules_identity   TEXT    NOT NULL,
  description_hash TEXT    NOT NULL,
  fits_raw         REAL,
  you_raw          REAL,
  schema_version   TEXT    NOT NULL,
  record           TEXT    NOT NULL,
  computed_at      TEXT    NOT NULL
);
`

// modelPassSchema holds the persisted output of the two model passes. An extraction is keyed by
// the retained source it read and the extractor that read it; matches are keyed by the exact
// requirement list they judged, the profile revision they saw and the matcher that judged them,
// so a source change, a profile change or a version change each re-runs only the pass it affects.
const modelPassSchema = `
CREATE TABLE IF NOT EXISTS match_vacancy_extractions(
  job_id            INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  source_hash       TEXT    NOT NULL,
  extractor_version TEXT    NOT NULL,
  requirements_hash TEXT    NOT NULL,
  requirement_count INTEGER NOT NULL,
  schema_version    TEXT    NOT NULL,
  record            TEXT    NOT NULL,
  created_at        TEXT    NOT NULL,
  PRIMARY KEY(job_id, source_hash, extractor_version)
);
CREATE INDEX IF NOT EXISTS match_vacancy_extractions_requirements
  ON match_vacancy_extractions(job_id, requirements_hash);
CREATE TABLE IF NOT EXISTS match_requirement_matches(
  job_id            INTEGER NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
  requirements_hash TEXT    NOT NULL,
  profile_revision  INTEGER NOT NULL,
  matcher_version   TEXT    NOT NULL,
  schema_version    TEXT    NOT NULL,
  record            TEXT    NOT NULL,
  created_at        TEXT    NOT NULL,
  PRIMARY KEY(job_id, requirements_hash, profile_revision, matcher_version)
);
`

const quickMatchSchema = `
CREATE TABLE IF NOT EXISTS match_job_inputs(
  job_id          INTEGER PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
  raw_source_hash TEXT    NOT NULL,
  snapshot        TEXT    NOT NULL,
  captured_at     TEXT    NOT NULL
);
CREATE TABLE IF NOT EXISTS match_job_facts(
  job_id                INTEGER PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
  facts_hash            TEXT    NOT NULL,
  raw_source_hash       TEXT    NOT NULL,
  schema_version        TEXT    NOT NULL,
  extractor_version     TEXT    NOT NULL,
  source_policy_version TEXT    NOT NULL,
  facts                 TEXT    NOT NULL,
  updated_at            TEXT    NOT NULL
);
`

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

// jobOriginalsSchema creates the job_originals table, which stores the raw page
// body or feed item of each vacancy, gzip-compressed, one row per vacancy.
// The row is replaced on refetch. It is never selected by list/detail API endpoints
// and is never indexed by FTS.
const jobOriginalsSchema = `
CREATE TABLE IF NOT EXISTS job_originals(
  job_id     INTEGER PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
  kind       TEXT    NOT NULL,
  fetched_at TEXT    NOT NULL,
  body       BLOB    NOT NULL
);
`

const quickMatchResultSchema = `
CREATE TABLE IF NOT EXISTS match_job_results(
  job_id                INTEGER PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE,
  profile_id            TEXT    NOT NULL,
  profile_revision      INTEGER NOT NULL,
  facts_hash            TEXT    NOT NULL,
  raw_source_hash       TEXT    NOT NULL,
  schema_version        TEXT    NOT NULL,
  extractor_version     TEXT    NOT NULL,
  source_policy_version TEXT    NOT NULL,
  source_policy_identity TEXT   NOT NULL,
  rules_version         TEXT    NOT NULL,
  rules_identity        TEXT    NOT NULL,
  status                TEXT    NOT NULL,
  score                 INTEGER,
  raw_score             REAL,
  result                TEXT    NOT NULL,
  computed_at           TEXT    NOT NULL
);
`
