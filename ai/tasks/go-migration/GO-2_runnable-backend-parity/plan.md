---
estimate: 22
---
# GO-2 — plan

**Author:** coordinator
**Created:** 2026-11-02

## Statement of work

Build the application layer around the Go cores so the existing React frontend runs unchanged against the Go backend exactly as it ran against the TypeScript backend. The backend opens the owner's existing SQLite database in place and reproduces its schema; it loads configuration and candidate context from the local data directory; it exposes the thirty-one documented HTTP endpoints at contract parity on the configured loopback port, behind the same security middleware, and serves the existing built React frontend as static assets. Assessment, application-answer preparation and the daily collection run become background goroutines coordinated through the same database locks, with the provider cores invoked unchanged. Parity is proven by recorded request and response fixtures captured from the running TypeScript app and replayed against the Go backend, and the documentation is kept alive by a route-drift check and by running those fixtures in the test entry. The TypeScript application stays working throughout; cutover and the public push are out of scope. The specification is `ai/sifty/backend-architecture.md`, `ai/sifty/backend-api-contract.md` and `ai/decisions/sifty-go-port.md`, and the code is authoritative where it and the prose disagree.

## Steps

### Phase 1 — SQLite store on the owner's existing database

- [x] 1.1 — Open the existing database in place with the pure-Go driver, set WAL, busy_timeout, foreign keys and immediate transactions, and reproduce the schema idempotently honouring user_version; check: a fresh and the owner's existing database both converge on the documented schema.
- [x] 1.2 — Job persistence: upsert, get, list by status and query sorted by score, stage, status and seen transitions, and guarded delete; check: a fixture job round-trips and listings match the documented shape.
- [x] 1.3 — Assessment, workflow and queue persistence (stage and status, archive retention, web3 flag, queue rows); check: workflow and queue state for a fixture match the documented shape.
- [x] 1.4 — Application drafts, questions and operations tables, plus feedback and remembered-answer records in the configurable data directory; check: a draft and a remembered answer round-trip.

### Phase 2 — Configuration, candidate context, provider inference

- [x] 2.1 — Load and schema-validate the five configuration files and save them with secret rejection; check: each config reads back and an invalid or secret-bearing save is refused.
- [x] 2.2 — Candidate context assembly: refresh, per-job context and source snapshots with hashes; check: context for a fixture matches the documented shape.
- [x] 2.3 — Provider inference wired to the Go cores, subprocess command-line tools with an isolated environment and loopback runtimes, plus diagnostics and smoke check; check: provider diagnostics and a smoke check return the documented shapes.

### Phase 3 — HTTP server, security, static frontend

- [x] 3.1 — HTTP server on loopback at the configured port with the Host and Origin allowlist, the one-megabyte JSON cap, the fixed response headers, the uniform error shape and the unknown-endpoint response; check: a bad Host returns 403, a wrong content type 415 and an unknown path the documented 404.
- [x] 3.2 — Serve the existing built React frontend as static assets with single-page fallback; check: the root serves the application and a deep link falls back to the index.

### Phase 4 — The HTTP endpoints at contract parity

- [x] 4.1 — Roles endpoints: list, get, patch, delete, seen, read and assess; check: the recorded fixtures for each replay identically.
- [x] 4.2 — Collection and sources: search, collection progress, health and import, with the discovery orchestration behind them; check: the recorded fixtures replay and a fixture source run yields the documented progress.
- [x] 4.3 — Assessment queue and options: queue status, enable and disable, assessment options; check: the recorded fixtures replay.
- [x] 4.4 — Candidate context, providers and configuration endpoints; check: the recorded fixtures replay.
- [x] 4.5 — Feedback endpoints, triggering reassessment; check: the recorded fixtures replay.
- [x] 4.6 — Application-preparation endpoints and the inline PDF file serving; check: the recorded fixtures replay.

### Phase 5 — Background workers and the daily scheduler

- [x] 5.1 — Assessment worker as a goroutine ticker over the queue with the lock protocol, liveness recovery and context cancellation; check: enqueued jobs drain to assessed on a fixture.
- [x] 5.2 — Preparation worker as a goroutine for inspect, generate and cancel with provider routing; check: a queued preparation produces a draft on a fixture.
- [x] 5.3 — Daily collection scheduler as a goroutine at the configured hour and minute; check: the scheduler fires a run at a simulated time.

### Phase 6 — Parity and end-to-end

- [x] 6.1 — Capture recorded request and response fixtures from the running TypeScript app across every documented endpoint; check: the fixture set covers all thirty-one endpoints.
- [x] 6.2 — Replay harness diffing Go responses against the fixtures, with explained divergences allow-listed; check: every fixture passes.
- [x] 6.3 — End-to-end browser run: the existing React frontend served by the Go binary drives Stage 1 and Stage 2 against a seeded data directory; check: a recorded run shows collection, assessment and preparation working.

### Phase 7 — Keep the documentation alive

- [x] 7.1 — Drift check comparing the documented endpoint list against the server's registered routes; check: it fails on a seeded divergence and passes on the real routes.
- [x] 7.2 — Wire the API fixtures as a snapshot test into the test entry and continuous integration; check: the test entry runs them.
- [x] 7.3 — Record the convention that a found behaviour discrepancy updates the documents, and link the decisions log; check: the convention text and the link are present.

## Acceptance criteria

- The existing React frontend, unchanged and served by the Go binary, drives Stage 1 (collection and fit) and Stage 2 (application-answer preparation) in a browser against the owner's data, behaving as it did against the TypeScript application.
- All thirty-one documented endpoints are implemented at contract parity; recorded request and response fixtures captured from the TypeScript application replay identically against the Go backend, with any divergence explained and allow-listed.
- The Go backend opens the owner's existing database in place, reproducing the documented schema and honouring user_version, without discarding accumulated state.
- Assessment, preparation and the daily collection run as background goroutines coordinated through the database locks, with liveness recovery kept for the single-owner target.
- Provider inference invokes the same external tools with the isolated environment and output caps unchanged.
- A route-drift check and the API fixtures run in the test entry, so the documentation cannot silently rot; the TypeScript application under `app/` stayed working throughout.
