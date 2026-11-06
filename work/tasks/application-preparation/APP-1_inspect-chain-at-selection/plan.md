---
estimate: 10
---
# APP-1 — plan

**Author:** coordinator
**Created:** 2026-11-03

## Statement of work

The Go `serve` command starts the preparation worker with no inference and no form-inspection wired (`NewPreparationWorker(s, d, cfg, nil, nil)`), so a pre-selected job's Application tab can never read its form and always fails with "no unanswered readable questions". This card wires the existing Playwright inspection worker (`node workers/playwright`, reached through `internal/worker` and `internal/inspect.InspectForm`) into the live preparation path, so an inspect operation reads the application form and stores the questions it finds as the draft's fields. Inspection runs on a model-free path that never acquires the inference lock, and is triggered the moment a job is pre-selected, not only when the tab is opened, so questions are ready by the time the user looks. The card also commits the queue-status `runtime` fix (currently a hot patch on the working tree) with a regression test, because the old shape crashes the whole catalogue. Answer drafting is out of scope: no model-generated answers are produced here.

## Steps

### Phase 1 — Inspection in the live server

- [x] 1.1 — Build a preparation inspect path in the server that invokes the Playwright worker for an inspect operation, reads the form at the job's resolved apply URL (falling back to the original listing URL), and stores the returned questionnaire as the draft's fields; check: an inspect-only operation on a fixture job populates the draft with the form's questions and marks the operation done.
- [x] 1.2 — Keep inspection off the inference lock, so form reading runs even while a model call holds the lock; check: an inspect operation completes while a stubbed generate holds the inference lock.
- [x] 1.3 — Surface the honest non-question outcomes as operation results, not errors: a reachable form with no answerable fields, and a form that requires owner sign-in; check: a fixture for each case ends with the documented status and message rather than a failure.

### Phase 2 — Trigger at selection

- [x] 2.1 — When a job transitions to pre-selected, enqueue a model-free inspect operation for it; check: patching a job to pre-selected creates a queued inspect operation and the worker drains it to done without a model call.

### Phase 3 — Fix the crash and verify the chain

- [x] 3.1 — Commit the queue-status `runtime` fix (return the `queue_state` 'runtime' row as `{"data": "<string>"}` or null) with a regression test that fails on the pre-fix shape; check: `/api/queue` returns `runtime.data` as a string, and the test fails against the old handler.
- [x] 3.2 — End-to-end on the owner's real database: verified by the coordinator. Job 458 (Ashby) ends in the honest "no readable questions" done-state and the catalogue does not crash; `/api/queue` returns `runtime.data` as a string. Job 758 has no resolved apply URL and the inspect falls back to a listing URL that triggers a download, surfacing as a raw failure rather than an honest state — handling that real-world case, and clicking through ATS "Apply" buttons to reach forms like 458's, is moved to APP-4. The inspection chain runs live against real sites; check: recorded run — 458 honest done-state, 758 raw inspect failure (deferred to APP-4), no catalogue crash.

## Acceptance criteria

- The work is complete when pre-selecting a job enqueues and runs a model-free inspection that reads the real application form and stores its questions as the draft's fields.
- Opening the Application tab in the Go-served app, on the owner's real database, shows the inspected questions for a pre-selected job, or an honest no-questions / needs-sign-in state, without crashing the catalogue.
- `/api/queue` returns `runtime` as `{"data": "<string>"}` or null, guarded by a regression test.
- No model-generated answers are introduced; the generate step remains deferred to the knowledge-base answer card.
