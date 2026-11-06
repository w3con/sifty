# APP-7 — Optimistic revision check on application draft edits

**Epic:** application-preparation
**Created:** 2026-11-04
**Status:** founded
**Short card:** steps and acceptance criteria live here.

## Why this work exists

The `PATCH /api/jobs/{id}/application` handler loads the draft, merges the request body onto it, and writes, without checking whether the draft changed since the client read it (`internal/server/routes.go`, around the PATCH handler). The preparation worker edits the same draft in the background while the owner is editing it in the interface. With no concurrency check, whichever write lands second silently overwrites the other: a background generation can erase an owner's edit, or an owner's edit can erase freshly generated fields. The retired TypeScript `updateApplication` refused a write whose base revision no longer matched, so the two could not clobber each other.

The Go draft already carries a draft-level `Revision` that `WriteDraft` increments on every save (`internal/store/application.go`). This card uses it to reject a stale edit instead of applying it blindly.

## Steps

- [ ] 1.1 — Make `PATCH /api/jobs/{id}/application` carry the base revision the edit was made against, and reject the write with a conflict status when it does not match the stored draft revision; on a matching revision apply the edit as today (which already increments the revision via `WriteDraft`); check: a Go test proves a PATCH carrying a stale revision is refused and leaves the stored draft unchanged, and a PATCH carrying the current revision succeeds and advances the revision.

## Acceptance criteria

- A concurrent background generation and owner edit can no longer silently overwrite each other: a draft edit made against an out-of-date revision is refused with a conflict rather than applied, and an edit against the current revision applies and advances the revision.

## Must not touch

- The draft data model beyond using the existing `Revision` field, the other application endpoints, and the preparation worker's own write path.
- The assessment and collection layers.
