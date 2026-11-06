# NAVIGATION-4 executor handback

Steps 1.1 through 4.2 are complete on branch `work/NAVIGATION-4`. Step 4.3 (staging check) is left for the coordinator.

## What was built

**Backend (steps 1.1–1.2)**

`store.BuildFTSQueryExprs` in `sifty/internal/store/fts_query.go` now combines groups with AND at the top level instead of OR. Each OR group still OR-joins its terms internally; each AND group still AND-joins internally. Empty groups are skipped (vacuous true). A single group returns its expression without an extra AND wrapper, so the backward-compatible flat `titles`/`keywords` path (one OR group) is unchanged. `go test -race ./internal/store/... ./internal/background/... ./internal/server/...` passes.

The test file `fts_query_test.go` was updated: the "multiple groups are ORed at top level" test was renamed and its expected value flipped to AND; three new tests were added (AND-across-groups, single-group no-AND guard). `rescore_test.go` gained `TestRescoreWithExprs` which inserts roles that match only the OR group, only both groups, and neither, then calls `Rescore` via `BuildFTSQueryExprs` to prove AND narrowing and that fit scores are still produced.

**Search UI (steps 2.1–2.2)**

`app/web/Collection.tsx` was rewritten. The search section is now three columns — OR terms, AND terms, Exclusions — each rendered as tag chips (term + ×) with an add input at the top. Every add and remove persists immediately via `PUT /api/search/expressions` (non-empty groups only) or the existing stop-word add/remove. The draft state, the Save button and the "Unsaved changes" indicator are gone. The help text describes the AND-between-columns semantics.

CSS for the three-column layout and chips was added to `app/web/polish.css` (`.search-columns`, `.search-col`, `.col-input`, `.term-chips`, `.term-chip`).

**Sources UI (steps 3.1–3.2)**

The sites list was replaced with an HTML table (`.sites-table` class): one row per source with name, URL, enabled checkbox and Remove button. The add-row form below uses a `.sites-add-form` grid layout. The `Sources` coverage component and its `api('health')` poll were removed entirely. The Collect button remains, with a single "Collection in progress…" line shown when `run?.status === 'running'`. CSS for table and add-form was added to `polish.css`.

**Archive (step 4.1)**

In `app/web/main.tsx`, when `view === 'applied'` (`isArchive`): the Unread button and Sort control are hidden; the applied list is sorted by `appliedAt` descending (null dates sort last) instead of going through `sortCatalogue`. Triage and Preparation are unchanged.

**Verification (step 4.2)**

`go test -race ./...` — all packages pass. `tsc --noEmit` — no errors. `vite build` — 54 modules, no warnings.

## Files changed

- `sifty/internal/store/fts_query.go` — AND combination, updated doc comment
- `sifty/internal/store/fts_query_test.go` — updated and new combinator tests
- `sifty/internal/store/rescore_test.go` — `TestRescoreWithExprs`
- `app/web/Collection.tsx` — full rewrite (three-column search, table sources)
- `app/web/polish.css` — chip and table CSS classes
- `app/web/main.tsx` — Archive sort and toolbar changes
