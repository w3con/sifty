# OPS-8 — Separate and migrate the user data layout

**Epic:** ops
**Created:** 2026-11-06
**Status:** backlog
**Kind:** task
**Short card:** steps and acceptance criteria live here.

## Why this work exists

The current data directory mixes four classes of state in one place: user-provided evidence, user-derived context and memory, vacancies and assessments, and application drafts. That makes provenance, backup, restore and environment isolation harder to reason about than they should be. [The current provenance map](../../../../docs/architecture/data-directory.md) records the evidence; it deliberately marks the four-class separation as **PROPOSED**, not implemented.

This card is the separate migration task. It does not move live data and does not split a database. It prepares an audited, reversible migration and stops for owner approval before any live change.

## Steps

- [ ] 1.1 — Audit every live reference to the data root, store path, config, answers, feedback, evidence, logs, backups and deployment artifacts across Go code, workers, scripts, docs, tests and runtime configuration. Record each source and consumer, including staging and production.
- [ ] 1.2 — Define the proposed ownership boundary for the four classes and prove SQLite plus external-file consistency: file-first answers/feedback against their index rows, context snapshots, FTS/index rebuildability, orphan detection and cross-store references. Use read-only schemas and scratch copies only.
- [ ] 1.3 — Design consistent backups for SQLite (including WAL) and external files, plus a tested restore and rollback path on scratch data. The plan must refuse a non-consistent snapshot and must preserve the previous usable state on failure.
- [ ] 1.4 — Prove staging/production isolation: separate data roots, no accidental cross-copy, explicit staging refresh only, and the same verified path for both environments. Record the exact commands and evidence.
- [ ] 1.5 — Present the migration, rollback and owner-approval gate. Do not move, split or delete live data until the owner approves that plan.

## Acceptance criteria

- A complete live-reference audit exists for the current data directory and every consumer.
- The migration plan covers SQLite-plus-file consistency, consistent backups, restore/rollback and staging/production isolation, with scratch-copy evidence.
- No live data is moved, split or deleted before explicit owner approval of the migration plan.
- The final plan preserves the file-first answer and feedback records and leaves the current production and staging data usable throughout.

## Must not touch

- Live production or staging data until the owner approves the migration plan; this backlog card authorizes preparation only.
- The current SQLite schema or file layout during preparation. Read-only schemas and path metadata are enough.
- Candidate evidence contents, remembered answers, feedback text, provider credentials or secrets. Do not print or copy private contents into the repository.
- The work-management and release boundaries in `AGENTS.md`; this card is not a release or a data move.

## Dependencies and links

- [Current data provenance and configured locations](../../../../docs/architecture/data-directory.md)
- [Requirement R-DATA in the current map](../../../../docs/MAP.md#req-data)
- [OPS-7 — centralized path configuration](../OPS-7_configurable-paths/description.md) is the related path-configuration task; the two must not silently duplicate each other.
- [OPS-4 — repository and workflow foundation](../OPS-4_repository-and-workflow-foundation/description.md) step 5.3 created this card.
