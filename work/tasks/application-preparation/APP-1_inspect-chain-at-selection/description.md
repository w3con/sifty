# APP-1 — Wire form inspection into the live chain and run it at selection

**Epic:** application-preparation
**Created:** 2026-11-03
**Status:** founded

## Why this work exists

When a job is pre-selected, opening its Application tab should read the job's application form and show the questions it contains. In the Go server this does not happen: `serve` starts the preparation worker without the Playwright inspection worker wired, so no form is ever read and the tab fails with "no unanswered readable questions". This card makes form inspection run in the live server, triggers it the moment a job is pre-selected rather than only when the tab is opened, keeps it on a model-free path so reading a form never waits behind model work, and folds in the queue-status fix that currently crashes the whole catalogue. It stops at reading the questions: drafting answers is a later card, because answers come from the knowledge base, not from per-question model generation.

## Must not touch

- The SQLite schema and migrations (open the owner's existing database in place; works today).
- The GO-1 Go cores (collection, assessment, prepare) except where inspection must be wired into the preparation path.
- Answer generation: no model-generated answers are introduced here. The generate step stays deferred to the knowledge-base answer card.
- The assessment background worker's own inference wiring (its own concern, tracked separately).

## Notes

- Inspection is the Playwright worker `node workers/playwright`, invoked by Go over the worker protocol (`internal/worker`, `internal/inspect`). Both the worker launcher and `InspectForm` already exist; `serve` just never wires them into the preparation worker (`NewPreparationWorker(s, d, cfg, nil, nil)` in `cmd/sifty/serve.go`).
- Apply-destination resolution already happens at collection and is stored as `apply_url` + `apply_status` (≈11 classes). Reading the form's questions is a separate, heavier step reserved for pre-selected jobs.
- Test jobs on the owner's real database: 458 (Catena, reachable Ashby ATS) and 758 (Aave, unresolved LinkedIn intermediary). They exercise both the "reachable form" and "no direct employer form" cases.
- The queue-status `runtime` crash fix is currently a hot patch on `main`'s working tree (uncommitted). This card commits it properly with a regression test.
