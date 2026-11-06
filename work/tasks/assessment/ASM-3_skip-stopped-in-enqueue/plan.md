---
estimate: 3
---
# ASM-3 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

`EnqueueAll` in `sifty/internal/store/queue.go` runs after each collection to queue every stored role for assessment. COL-6 added stop-words that hide a role whose title contains an active stop-word from every view and skip such postings at collection, but `EnqueueAll` still queues the already-stored hidden roles, so a configured provider spends tokens assessing roles the owner will never see. This work makes `EnqueueAll` skip a role whose computed `Stopped` flag is true. The flag is already populated by the `s.Get(id)` call inside the loop, so the change reuses it with no new query. Removing a stop-word later lets a subsequent `EnqueueAll` pick the role up again, so the skip stays reversible.

## Decisions settled with the owner (2026-11-05)

- Stop-word-hidden roles are not queued for assessment. Block-rule-hidden roles are out of scope and stay assessable.

## Out of scope — do not touch

The assessment logic, the queue model and `EnqueueJob`, the fingerprint/dedup check in `EnqueueAll`, and the COL-6 stop-word matching (reused, not changed). Block rules.

## Acceptance criteria

- After a collection run, no role whose title matches an active stop-word is present in the assessment queue as a newly enqueued entry.
- A role not hidden by any stop-word is still queued exactly as before.
- Removing a stop-word and running `EnqueueAll` again queues the previously hidden role.
- `go test -race ./...` and `bin/work-test` pass.

## Checkpoints

**1 — Skip stopped roles in EnqueueAll**

*Gate: `cd sifty && go test -race ./internal/store/...`*

- [x] 1.1 — In `EnqueueAll`, skip a role whose `job.Stopped` is true (the flag the loop's `s.Get(id)` already populates) before enqueuing it.
- [x] 1.2 — A unit test adds a stop-word matching one of two stored roles, runs `EnqueueAll`, and asserts the matching role is absent from the queue while the other is present; removing the stop-word and re-running queues it.

**2 — Verify**

*Gate: `bin/work-test`*

- [x] 2.1 — `bin/work-test` passes.

## Gate evidence

Дословный вывод каждой прогнанной проверки записывается сюда, без сокращений и пересказа.
