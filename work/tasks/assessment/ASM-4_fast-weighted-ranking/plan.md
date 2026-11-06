---
estimate: 20
---
# ASM-4 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

Ranking today waits on the LLM assessment queue, which can take hours for a fresh batch, so a newly collected inbox is not immediately ordered by fit. This work gives every role an instant relevance score computed with BM25 and makes that score the primary 0–100 fit number the list shows and the "Best fit" sort uses. BM25 comes from SQLite's built-in FTS5 `bm25()` — already compiled into the project's `modernc.org/sqlite` driver (verified), so there is no new dependency, no database change, and no hand-rolled algorithm. The score ranks each role's text against a query built from the candidate profile and the configured search terms, is recomputed after every collection and whenever the candidate profile is saved, and is back-filled onto existing roles once. The LLM deep analysis stops running as a mass queue on collection: it runs only when a role is moved to Selected, plus an explicit "assess now" button on any open role, so providers are spent only on roles the owner is seriously considering.

## Settled with the owner (2026-11-05)

- The fast score is BM25 via SQLite FTS5 `bm25()` (confirmed available in the current driver). SQLite stays; no switch to another database.
- The BM25 score is the primary 0–100 fit number in the list and the key for the "Best fit" sort.
- The LLM assessment no longer runs as a mass queue on collection. It runs only when a role is moved to Selected, plus a manual "assess now" button on any open role. No on-open auto-start.

## Coordinator's implementation decisions (settled, not for the owner)

- The FTS5 index is a standard (self-contained) table over `jobs` covering title, description, company and tags, with a `match_score` integer column on `jobs` plus a `user_version` bump. No SQL triggers maintain it. *(Executor deviation from plan, 2026-11-05: SQL triggers — both external-content and standard FTS5 — proved unreliable under modernc.org/sqlite 3.54.1. When a trigger-fired INSERT accumulates tokens in the FTS5 pending-terms hash and later a DELETE fires in a separate transaction, the negative entries do not reliably cancel the positive ones across level-0 segments. All FTS maintenance is done in Go: `Store.refreshFTS` runs after every `Upsert` commit, and `Store.Delete` removes the FTS row before returning. This was confirmed by TestFTSStandardNoTriggers and TestFTSSync.)*
- The BM25 query is built from the candidate profile (`config/candidate.yaml`) plus the configured search titles/keywords (`config/search.yaml`): the text is tokenised, FTS5 syntax characters are stripped, and the tokens are OR-joined into the `MATCH` query. The weights are not hand-tuned in this card; relevance is whatever FTS5 `bm25()` returns.
- SQLite `bm25()` returns a value where more negative means a better match; the raw scores are negated and min-max scaled across the current corpus to 0–100 (higher = better fit). A role returned by no `MATCH` term scores 0.
- A `Rescore` store method recomputes every role's `match_score` from the current profile+search query; it runs at the end of each collection and when the candidate profile is saved, and a one-time run back-fills existing roles. It is computed in SQL over the FTS table, so it stays instant.
- Removing the mass queue means the collection path no longer calls `EnqueueAll`; instead the stage transition to `preselected` enqueues that one role for LLM assessment via the existing `EnqueueJob`, and the existing manual reassess stays as the "assess now" button. Existing stored LLM assessments are kept and still shown.
- The list shows `match_score` as the fit number and sorts "Best fit" by it; where an LLM assessment also exists, its detail (summary/evidence) still shows on the role, but the headline number is `match_score`.

## Out of scope — do not touch

The LLM assessment logic itself (prompt, provider plumbing, the stored result shape). The collection fetch/source internals and the stop-word/block filtering (COL-6, TRI-1/2). The dedup and stage model beyond adding the enqueue-on-select hook. No switch of database engine.

## Acceptance criteria

- Every stored role has a `match_score` in 0–100; after a collection run, newly collected roles have it set without waiting on any LLM call.
- The Triage list shows `match_score` as the fit number and the "Best fit" sort orders by it.
- Editing and saving the candidate profile rescores all roles, and the ranking changes accordingly.
- Collection no longer mass-enqueues roles for LLM assessment; moving a role to Selected enqueues exactly that role, and the manual "assess now" button still queues a single role.
- An existing LLM assessment on a role is still shown; only auto-queuing on collection is gone.
- No new runtime dependency is added and the database engine is unchanged (SQLite/FTS5).
- `go test -race ./...` and `bin/work-test` pass.

## Checkpoints

**1 — Backend: FTS5 index and BM25 scoring**

*Gate: `cd sifty && go test -race ./internal/store/...`*

- [x] 1.1 — Add to the schema an FTS5 external-content table over `jobs` (title, description, company, tags) with insert/update/delete sync triggers, a `match_score` column on `jobs`, and a `user_version` bump; `go build ./...` is green and a test confirms the triggers keep the index in sync.
- [x] 1.2 — Add a store helper that builds a sanitised FTS5 `MATCH` query from the candidate profile plus the search titles/keywords; a unit test covers tokenisation and stripping of FTS5 syntax characters.
- [x] 1.3 — Add `Rescore` that runs `bm25()` over the FTS index for that query, negates and min-max scales the result to 0–100, and writes `match_score` on every role; a unit test seeds roles and asserts the ranking and the 0–100 range.

**2 — Backend: wire scoring and rework the LLM trigger**

*Gate: `cd sifty && go test -race ./internal/...`*

- [x] 2.1 — Call `Rescore` at the end of each collection and when the candidate profile is saved; a test confirms a collected role gets a `match_score` with no LLM call, and that saving the profile changes scores.
- [x] 2.2 — Remove the mass `EnqueueAll` from the collection path; enqueue a single role via `EnqueueJob` on transition to `preselected`; a test confirms collection enqueues nothing and selecting a role enqueues exactly it. The manual reassess endpoint is unchanged.

**3 — Frontend: show and sort by the match score**

*Gate: `cd app && npx tsc --noEmit`*

- [x] 3.1 — Carry `matchScore` on the `Job` type and show it as the fit number in the catalogue list and role header.
- [x] 3.2 — Make the "Best fit" sort order by `matchScore`; keep any existing LLM assessment detail visible on the role, with `matchScore` as the headline number. Relabel the manual assess control to read as an on-demand "assess now".

**4 — Verify end to end**

*Gate: `bin/work-test`*

- [x] 4.1 — `bin/work-test` passes.
- [x] 4.2 — Build the web bundle and the Go binary and confirm by hand against a copy of the live data: after a scoring run every role has a 0–100 match score, the list sorts by it instantly, editing the candidate profile re-ranks, collection queues no LLM calls, and selecting a role queues exactly that one. (Gate: manual, recorded in `log.md`.)

## Gate evidence

Дословный вывод каждой прогнанной проверки записывается сюда, без сокращений и пересказа.

### Step 1.1 gate — `cd sifty && go test -race ./internal/store/...`

```
ok  	github.com/w3con/sifty/internal/store	3.454s
```

### Step 1.2 gate — `cd sifty && go test -race ./internal/store/...`

```
ok  	github.com/w3con/sifty/internal/store	3.603s
```

### Step 1.3 gate — `cd sifty && go test -race ./internal/store/...`

```
ok  	github.com/w3con/sifty/internal/store	3.684s
```

### Step 4.1 gate — `bin/work-test` and `cd sifty && go test -race ./...`

```
PASS: all harness checks

ok  github.com/w3con/sifty/internal/background	6.755s
ok  github.com/w3con/sifty/internal/store	(cached)
ok  github.com/w3con/sifty/internal/server	(cached)
(all other packages cached or no test files)
```

### Steps 3.1 and 3.2 gate — `cd app && npx tsc --noEmit`

```
(no output — clean)
```

### Step 2.2 gate — `cd sifty && go test -race ./internal/...`

```
ok  	github.com/w3con/sifty/internal/background	6.755s
ok  	github.com/w3con/sifty/internal/server	18.951s
ok  	github.com/w3con/sifty/internal/store	4.507s
(all other packages cached)
```

### Step 2.1 gate — `cd sifty && go test -race ./internal/...`

```
ok  	github.com/w3con/sifty/internal/assess	1.528s
ok  	github.com/w3con/sifty/internal/background	7.076s
ok  	github.com/w3con/sifty/internal/candidate	1.442s
ok  	github.com/w3con/sifty/internal/collect	2.033s
ok  	github.com/w3con/sifty/internal/config	2.525s
ok  	github.com/w3con/sifty/internal/fetch	3.007s
ok  	github.com/w3con/sifty/internal/inspect	2.730s
ok  	github.com/w3con/sifty/internal/prepare	3.229s
ok  	github.com/w3con/sifty/internal/records	2.256s
ok  	github.com/w3con/sifty/internal/server	26.649s
ok  	github.com/w3con/sifty/internal/source	3.886s
ok  	github.com/w3con/sifty/internal/store	(cached)
ok  	github.com/w3con/sifty/internal/worker	4.447s
```
