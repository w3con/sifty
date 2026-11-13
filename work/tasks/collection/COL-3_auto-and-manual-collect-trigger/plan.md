---
estimate: 7
---
# COL-3 — plan

**Author:** coordinator
**Created:** 2026-11-04

## Statement of work

Make collection runnable on demand and auto-triggered when the owner opens or reloads the app, throttled on the server so it runs no more often than a configurable cooldown (default 3 hours). A single backend trigger serialises through the same cross-process collection lock the daily scheduler uses, records a `collection_runs` run, executes the COL-1 source dispatch, and finishes the run; when not forced it is skipped if the last collection run (from any trigger) started within the cooldown. `POST /api/search` is wired to this trigger asynchronously, with `force` from the body; the manual "Collect jobs" and per-source buttons send `force: true`, which bypasses the cooldown but not the per-domain pacing. The frontend issues one unforced auto-collect call on app mount, so opening or reloading the app starts a throttled run. The daily scheduler stays and its runs count toward the cooldown.

## Steps

### Phase 1 — Backend: a locked, throttled, runnable collection

- [x] 1.1 — Add a backend trigger (e.g. `TriggerCollection(store, dirs, cfg, opts{Source string, Force bool})`) that acquires the same cross-process collection lock the daily scheduler uses, and when not forced returns a "skipped: cooldown" result if the most recent collection run started within the cooldown; otherwise records a `collection_runs` run, executes the COL-1 dispatch (`collectSources` / `defaultCollect` core), and finishes the run; check: unit tests prove a forced call runs the dispatch and records start+finish, an unforced call within the cooldown is skipped and records nothing, and an unforced call past the cooldown runs.
- [x] 1.2 — Make the cooldown a runtime config value (default 3 hours) read from the config, not a constant; check: config loads the interval and a test with a short interval exercises the boundary on either side.
- [x] 1.3 — Make the cooldown and the `GET /api/collection` "last run" reflect collection from any trigger (daily, manual, auto), so a recent daily run also suppresses an unforced trigger; check: a test where a recent daily run suppresses an unforced trigger and an older one does not.

### Phase 2 — Wire the HTTP endpoint

- [x] 2.1 — Wire `POST /api/search` to call the trigger asynchronously (goroutine), passing `Source` and `Force` from the body and returning the runID/status (or a skipped status within cooldown); check: an integration test proves a forced POST starts a run that reaches the store and `GET /api/collection` reports it, and an unforced POST within the cooldown returns a skipped status without a new run.

### Phase 3 — Auto-trigger on app load

- [x] 3.1 — On app mount, issue one unforced `POST /api/search` (auto-collect) so opening or reloading the app starts a throttled run, keep the manual "Collect jobs" (`app/web/Settings.tsx`) and per-source "Search source" (`app/web/Sources.tsx`) buttons sending `force: true`, and keep the collection-progress indicator reflecting running/last-run; check: the mount effect fires the call once per load rather than per render (and a within-cooldown load is a server-side no-op visible as a skipped status).

## Acceptance criteria

- Opening or reloading the app starts a collection run at most once per cooldown (default 3 hours), with no button pressed.
- The manual "Collect jobs" and per-source buttons force a run immediately, bypassing the cooldown but still serialised by the collection lock; per-domain pacing from COL-1 still applies, so forcing cannot hammer a host.
- Two triggers never run two collections at once (the lock), and an unforced trigger within the cooldown is a no-op.
- The daily scheduler still runs and its runs count toward the cooldown; collection from any trigger is visible in `GET /api/collection`.
- No change to the COL-1 fetch/source internals, the assessment/preparation pipelines, or the `sifty/` public-subtree rules.
