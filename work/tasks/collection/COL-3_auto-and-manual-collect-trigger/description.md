# COL-3 — Auto-collect on app load (throttled) and working manual trigger

**Epic:** collection
**Created:** 2026-11-04
**Status:** founded

## Why this work exists

After COL-1 the Go backend can collect, but only the daily scheduler triggers it. The manual "Collect jobs" and per-source "Search source" buttons still hit `POST /api/search`, which only writes a `collection_runs` row and returns "running" without running anything, and there is no way to collect on demand or when the owner opens the app. The owner wants new postings to appear automatically when they open or reload the app, without pressing a button, but no more often than a safe interval (default 3 hours) so a source does not ban the client for hammering.

This card makes collection runnable on demand and auto-triggered on app load, throttled. The throttle is enforced on the server by the time since the last collection run, so reloading the page repeatedly cannot start more than one run per interval. The manual buttons force a run immediately, bypassing the throttle but still serialised by the collection lock, and the per-domain traffic discipline from COL-1 still paces the actual fetches, so even a forced run cannot hammer one host.

## What this card delivers

- A single backend trigger that serialises through the same cross-process collection lock the daily scheduler uses, records a `collection_runs` run, executes the COL-1 source dispatch, and finishes the run; when not forced, it is skipped if the last collection run started within the cooldown.
- The cooldown as a runtime config value (default 3 hours), not a hardcoded constant.
- `POST /api/search` wired to that trigger (asynchronously), with `force` from the request body; the manual and per-source buttons send `force: true`.
- The frontend issuing one unforced auto-collect call on app mount, so opening or reloading the app starts a throttled run; the existing collection-progress indicator reflects it.
- The cooldown and the `GET /api/collection` "last run" reflecting collection from any trigger (daily, manual, auto), so a recent daily run also suppresses an auto trigger.

## Must not touch

- The daily scheduler stays; its runs count toward the cooldown.
- The COL-1 fetch and source internals (reuse the dispatch; do not change the SSRF guard, cache, traffic discipline, adapters or waterfall).
- The assessment and preparation pipelines.
- The public-subtree rules for `sifty/`.

## Notes

- Reuse: the collection run core from `internal/background/collect.go` (`defaultCollect` / `collectSources`), the `collection_runs` lifecycle (`StartCollectionRun` / `FinishCollectionRun` / `LatestCollectionRun`), and the cross-process lock (`AcquireLock` / `ReleaseLock`) the scheduler uses.
- The scheduler currently records its run separately (`StartDailyRun`); make the cooldown consider collection from any trigger (unify on `collection_runs`, or take the latest of both), so the daily run suppresses an auto trigger within the window.
- `force` bypasses the run-level cooldown only; per-domain pacing (COL-1) still applies, so forcing is safe.
- The manual "Collect jobs" button is in `app/web/Settings.tsx` (`collect`), the per-source button in `app/web/Sources.tsx`.
