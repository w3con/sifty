# GO-7 — Wire missing serve.go background tickers

**Epic:** go-migration
**Created:** 2026-11-04
**Status:** founded
**Short card:** steps and acceptance criteria live here.

## Why this work exists

The TypeScript server started two periodic tickers that the Go `cmd/sifty/serve.go` does not: an evidence-fingerprint check and an archive purge. `serve.go` starts only the assessment worker, the preparation worker and the daily scheduler (confirmed in the 2026-11-04 parity audit).

Two consequences. First, when the owner edits their evidence (CV, profile exports, feedback), the Go backend does not notice: the assessment worker re-enqueues jobs every 30 seconds, but nothing recomputes the assembled candidate context from the changed files, so reassessment runs against stale evidence until `POST /api/context` is called by hand. The TypeScript `server.ts` watched the evidence fingerprint and called `refreshContext` when it changed. Second, the archive is never pruned: `store.PurgeArchive` (`internal/store/jobs.go`) is complete and correct, but no ticker calls it, so applied roles past the retention window accumulate forever.

Both store methods already exist; this card only starts the two goroutines that call them.

## Steps

- [ ] 1.1 — Add an evidence-fingerprint refresh ticker in `serve.go`: on an interval, when the candidate-evidence fingerprint has changed, call `RefreshContext` so subsequent assessments use the updated evidence; check: a Go test with an injected clock and fingerprint proves a changed fingerprint triggers a context refresh and an unchanged fingerprint does not.
- [ ] 1.2 — Add an archive-retention ticker in `serve.go` that calls `store.PurgeArchive` on an interval; check: a Go test proves a role past the retention window is purged by the ticker and a fresher one is kept.

## Acceptance criteria

- Editing candidate evidence files causes reassessment against the updated evidence without a manual API call, and applied roles past the retention window are purged automatically. Both run as goroutines started in `serve.go` alongside the existing workers, each stopping cleanly on shutdown.

## Must not touch

- The assessment worker, preparation worker and daily scheduler loops themselves, and the existing behaviour of `RefreshContext` and `PurgeArchive`.
- The public-subtree rules for `sifty/` (no card identifiers, tool-authorship trailers, absolute local paths or Russian prose in code or in commit messages touching `sifty/`).
