# ASM-3 — Skip stop-word-hidden roles in the assessment queue

**Epic:** assessment
**Created:** 2026-11-05
**Status:** founded (plan.md before execution)

## Why this work exists

COL-6 added stop-words: a role whose title contains an active stop-word is hidden from every view and skipped at collection. But `EnqueueAll` (`sifty/internal/store/queue.go`), which runs after each collection to queue roles for assessment, still queues these hidden roles, so a configured provider spends tokens assessing roles the owner will never see. New junk is already kept out by the collection skip; this closes the remaining gap for roles already stored when a stop-word is added.

## Must not touch

The assessment logic itself, the queue model, the fingerprint/dedup in `EnqueueAll`, and the stop-word matching from COL-6 (reused, not changed). Block rules are out of scope: a blocked role stays assessable, only a stop-word-hidden role is skipped.

## Notes

`EnqueueAll` already calls `s.Get(id)`, which populates `job.Stopped` (COL-6). The change is to skip a role whose `Stopped` is true before enqueuing it, so it reuses the existing computed flag with no new query. Removing the stop-word later lets a subsequent `EnqueueAll` pick the role up again.
