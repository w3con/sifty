# COL-4 — Record per-source collection progress for the run indicator

**Epic:** collection
**Created:** 2026-11-04
**Status:** founded (not scheduled; plan.md before execution)

## Why this work exists

The collection-progress indicator in the UI (the `RunProgress` component in `app/web/Sources.tsx`, fed by `GET /api/collection`) shows no per-source detail, because every run reports `sources: []` and `events: []`. The reading side already exists: `store.LatestCollectionRun` queries the `source_runs` and `collection_events` tables and returns their rows, and both tables exist in the schema (`internal/store/migrate.go`, created ~line 111 and ~line 119). What is missing is the writing side: the collection dispatch built in COL-1 (`internal/background/collect.go` — `defaultCollect` / `collectSources`) runs sources and imports jobs but never inserts a `source_runs` row per source or emits a `collection_events` row as it progresses. So the tables stay empty and the indicator stays blank even though jobs are being collected.

This card adds that writing side so a running or finished collection shows which sources ran, their status and coverage, and a timeline of events, without changing what the reader or the UI already expect.

## What this card should deliver

- The collection dispatch records one `source_runs` row per source it attempts, keyed to the current collection run id, with the source name, a status that moves from running to complete/failed, and a coverage summary (e.g. counts found/imported) in the `coverage` JSON column the reader already decodes.
- The dispatch emits `collection_events` rows as collection progresses (source started, source finished with a count, source failed with the error), keyed to the run id, so the event timeline the reader returns is non-empty.
- `GET /api/collection` then surfaces real per-source progress for a manual, auto or daily-triggered run, and the existing `RunProgress` indicator renders it with no frontend change (or a minimal one if the shapes differ).
- Daily-scheduler runs: decide and implement whether they also get per-source detail, or remain summary-only (the reader currently hard-codes empty slices for `origin == "daily"`; see the Notes).

## Must not touch

- The COL-1 fetch and source internals (SSRF guard, cache, per-domain traffic discipline, adapters, extraction waterfall) — this adds progress recording around the dispatch, it does not change how a source is fetched or parsed.
- The cooldown/lock/trigger logic from COL-3.
- The `source_runs` / `collection_events` reader in `store.LatestCollectionRun` and the schema for those tables — the shapes are fixed by the reader; match them.
- The public-subtree rules for `sifty/`.

## Notes

- Reader to match: `internal/store/queue.go` `LatestCollectionRun` — `source_runs(source, status, coverage)` and `collection_events(created, source, message)`, both filtered by `run_id`.
- Writer location: `internal/background/collect.go` (`defaultCollect` / `collectSources`), the COL-1 dispatch now invoked by the COL-3 trigger and the daily scheduler.
- New store writers are likely needed (e.g. `StartSourceRun` / `FinishSourceRun` / `AppendCollectionEvent`); none exist yet.
- The daily scheduler currently records into the separate `runs` table, and `LatestCollectionRun` returns empty detail for `origin == "daily"`. A clean option is to have the daily scheduler also drive the same dispatch that writes `source_runs` / `collection_events`; decide in plan.md.
- Verified live on 2026-11-04: a forced run (`runId 4`) collected jobs (catalogue rose past 1300) while `GET /api/collection` still returned empty `sources`/`events`, confirming the gap is the absent writer, not the reader.

## Steps

- [ ] To be defined in plan.md.

## Acceptance criteria

- To be defined in plan.md.
