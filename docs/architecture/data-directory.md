# The data directory: current provenance and configured locations

Sifty is self-contained: everything it reads and writes at runtime lives under one local data directory. The directory is configured by the `--data-dir` flag, then the `SIFTY_DATA_DIR` environment variable, then a default of `./data`. Paths are made absolute and never resolved from a git root; the resolver is `backend/internal/dirs/dirs.go`.

This page records the **current** layout and provenance from the code, the configuration schema and path metadata. It does not contain candidate evidence, credentials or runtime record contents. The four-class separation at the end is **PROPOSED**, not implemented.

## Current layout

```
<data-dir>/
  config/
    search.yaml       sources, OR/AND search terms, exclusions
    runtime.yaml      collection and assessment bounds, port, daily schedule
    providers.yaml    inference providers and the provider chosen per stage
    candidate.yaml    candidate evidence file references and profiles
    network.yaml      per-source network profiles (optional)
  config-backups/     timestamped copies written on a successful config save
  evidence/           created by Sifty; not the implicit base for candidate.files
  answers/            remembered application answers, one markdown file per answer
  feedback/           owner-confirmed feedback, one markdown file per statement
  store.sqlite        the operational SQLite store
  store.sqlite-wal    SQLite WAL sidecar while the store is open
  store.sqlite-shm    SQLite shared-memory sidecar while the store is open
```

`<data-dir>` also receives environment-specific operational files through the deploy scripts: `prod-server.log`, `staging-server.log`, `deploy.log` and `store.sqlite.backup.<timestamp>` snapshots. The binaries and their `.prev` rollback copies live under `bin/`, not the data directory.

**Candidate file paths.** Relative paths in `candidate.yaml` `files` are resolved under the **data directory root**, not under `evidence/`. The same applies to a profile's optional relative `export` path. In the observed local configuration this is why the CV, profile and knowledge-base files are referenced as root-relative paths (`cv.pdf`, `profile.pdf`, `kb/...`) while `evidence/` exists but is empty. `evidence/` is a conventional place for owner-supplied files, not a resolver rule.

**Configuration saves.** A successful save validates the new YAML, writes a timestamped copy under `config-backups/`, then writes the new file with private permissions. The backup directory is created on demand (`backend/internal/config/save.go`).

## Current provenance

The table separates the origin of the data from where Sifty currently keeps it. "User-provided" means the owner authored or supplied it; "derived" means Sifty assembled it from that input.

| Class | What it is | Current location | Written by |
|---|---|---|---|
| User-provided evidence | CV and profile PDFs, knowledge-base/FAQ text and other files referenced by `candidate.yaml`; remote profile exports | Files under the data directory, at the paths named in `candidate.yaml`; `evidence/` is available but not implicit | Owner; Sifty only reads them |
| User-provided configuration | Source list, search terms, exclusions, provider definitions and stage choices, candidate evidence references | `config/*.yaml`; previous versions in `config-backups/` | Owner or `PUT /api/config/*` / stage-provider routes |
| User-derived context | Assembled, versioned evidence snapshot, including remote profile text and feedback sources | SQLite `contexts` and `context_refreshes` | Context refresh in `backend/internal/candidate` |
| User-derived memory (answers) | Remembered application answers, scoped job/company/general | File-first under `answers/`, with a base64 JSON record in a `SIFT_ANSWER` comment marker | `POST /api/jobs/{id}/application/remember` |
| User-derived memory (feedback) | Owner-confirmed candidate feedback and extracted facts/conflicts | File-first under `feedback/`, with a `SIFT_FEEDBACK` marker; indexed in SQLite `feedback` | `POST /api/jobs/{id}/feedback` |
| Vacancies | Roles, sightings, raw captures, dedup aliases, tombstones, workflow history | SQLite `jobs`, `sightings`, `snapshots`, `job_aliases`, `deleted_listings`, `workflow_history` | Collection upsert and workflow routes |
| Assessments | Assessment history, current result, queue, queue state, retry metadata | SQLite `assessments`, `assessment_queue`, `queue_state`, `jobs.assessment` | Assessment worker and queue routes |
| Applications | Drafts, revisions, questionnaire snapshots, preparation operations | SQLite `application_drafts`, `application_revisions`, `questionnaire_snapshots`, `preparation_operations` | Preparation worker and application routes |
| Collection/operations | Collection runs, source runs, events, health, discovery cache, locks | SQLite `collection_runs`, `source_runs`, `collection_events`, `runs`, `health`, `discovery_cache`, `locks` | Collection trigger, scheduler and workers |
| Derived/rebuildable | Full-text index and match score; in-memory page cache | SQLite `jobs_fts` and `jobs.match_score`; the page cache is process memory only | Store refresh/rescore and fetch client |
| Runtime artifacts | Logs, database backups, config backups, deployment rollback state | Data directory and `bin/*.prev` | Deploy scripts and config save |

The SQLite store is the operational spine. The two durable owner-record kinds — remembered answers and feedback — are deliberately **file-first**: the markdown file is the authoritative record and the database index is secondary, so a database loss does not destroy the owner's statements. There is no cross-file transaction between those files and SQLite; consistency is maintained by the application's write order and must be checked explicitly in any migration.

The current store schema target is `user_version` 14 (`backend/internal/store/migrate.go`). It opens in WAL mode with a busy timeout and foreign keys enabled (`backend/internal/store/store.go`). The store may be open while the server is running, so a plain file copy is not a safe backup; use SQLite's backup mechanism, as `bin/sift-deploy` does.

## Observed local state (metadata only)

At the time of this inspection the ignored production and staging data directories were present in the main checkout. Read-only path listing showed both contain `config/` with the five YAML files, root-level CV/profile PDFs, empty `answers/`, `feedback/` and `evidence/` directories, `store.sqlite` with WAL sidecars, timestamped SQLite backups and server/deploy logs. The production directory also carried `kb/`; staging also carried `config-backups/` from UI saves. No file contents were read for this map.

These names are evidence of the current deployment shape, not a committed schema. Treat the data directories as personal runtime state: ignored by Git, never copied into tracked files, and never inspected by content for documentation work.

## PROPOSED separation

> **Status: PROPOSED — not implemented.** Nothing has been moved, split or deleted. The live migration and its approval belong to [OPS-8](../../work/tasks/ops/OPS-8_data-layout-migration/description.md); [OPS-7](../../work/tasks/ops/OPS-7_configurable-paths/description.md) owns the separate path-configuration work.

The proposed target is a logical separation of four data classes, with explicit ownership and backup rules:

1. **User-provided evidence** — CV/profile documents, knowledge-base/FAQ text, profile exports and the configuration that references them. Read-only inputs; a migration must never rewrite or silently normalize them. Prefer a dedicated evidence root instead of root-relative paths.
2. **User-derived context and memory** — assembled context snapshots, remembered answers and confirmed feedback. Preserve the file-first records and their markers; if SQLite is split, the file records remain authoritative and the index must be rebuildable.
3. **Vacancies and assessments** — roles, sightings, captures, aliases, tombstones, workflow history, assessments and the assessment queue. These are backend-produced operational state and can be rebuilt from collected postings plus assessments, but the current history has value and must be backed up.
4. **Applications** — drafts, revisions, questionnaire snapshots and preparation operations. These contain owner edits and must be preserved with the vacancy they belong to.

The exact physical shape — separate directories, separate SQLite files, or one database with an ownership boundary — is deliberately **not decided here**. A migration must first audit live references, prove SQLite plus external-file consistency, take a consistent backup, verify restore/rollback on a scratch copy, keep staging and production isolated, and obtain owner approval before touching live data.

## Migration boundary

[OPS-8](../../work/tasks/ops/OPS-8_data-layout-migration/description.md) is the separate backlog card for the audit and migration. It requires:

- a live-reference audit across code, configs, scripts, workers, docs and runtime paths;
- a consistency check covering SQLite tables and the file-first answer/feedback records;
- consistent backups and a tested restore/rollback;
- staging/production isolation, with no cross-copy except an explicit staging refresh;
- owner approval before any live migration or database split.

No data is moved and no database is split as part of the structural repository work.

## Answer and feedback files

Each remembered answer and each feedback statement is a markdown file that is both human-readable and machine-recoverable: the prose is the record, and a base64-encoded JSON copy of the same record sits in an HTML comment marker at the end (`SIFT_ANSWER` for answers, `SIFT_FEEDBACK` for feedback). The file is the first durable write and survives a missing model, a store failure or a restart.

The exact shapes are the templates `backend/internal/records/templates/answer.md.tmpl` and `backend/internal/records/templates/feedback.md.tmpl`, embedded in the binary. A first write produces a file that follows the matching template; this is proven by the tests in `backend/internal/records`.
