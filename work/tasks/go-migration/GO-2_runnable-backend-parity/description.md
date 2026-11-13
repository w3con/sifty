# GO-2 — Runnable Go backend at parity with the TypeScript app

**Epic:** go-migration
**Created:** 2026-11-02
**Status:** founded

## Why this work exists

The previous card ported the backend's computational cores to Go (collection, fit assessment, application-answer preparation) and proved they produce equivalent output on a shared fixture. What it did not build is the application around those cores: there is no HTTP server, no data store, no background workers, no served interface. The Go binary today only prints its version and runs a self-check. The existing React frontend therefore cannot run against it at all, so nothing is usable in a browser.

This card makes the Go backend a drop-in replacement for the TypeScript one. The same React frontend, unchanged, must work against the Go backend exactly as it worked against the TypeScript backend: the same HTTP interface on the same configured port, the same data, the same behaviour. The outcome the owner sees is this: open the browser at the configured port, and the application works as before, now served by the Go backend. This is the step that turns the port from a set of passing unit tests into a running product.

## Must not touch

The TypeScript application under `app/` stays working until the owner chooses cutover; this card does not delete it, break it, or change its behaviour. It is the reference the Go backend is measured against and the fallback while the Go backend is built.

The React frontend source under `app/web` is the contract to satisfy, not to rewrite: the whole point is that the existing frontend runs unchanged against the Go backend.

The owner's existing SQLite database (`jobs.sqlite`) holds the accumulated search: collected roles, assessments, workflow history, application drafts, remembered answers and feedback. The Go backend opens it in place; this card must not corrupt, discard, or require re-importing that state.

The knowledge base under `kb/`, the work-management harness under `bin/` and the other cards under `work/tasks/`, and the `tools/shared` submodule.

No public push to `github.com/w3con/sifty`: that is the previous card's owner-gated tail and is out of scope here. No commits to `main`; work on a branch.

## Notes

The specification is already written and is the authority for this card: `docs/architecture/backend-architecture.md` (what the backend is, the data model, the modules, the flows, the storage map, the target schema), `docs/architecture/backend-api-contract.md` (the per-endpoint request and response contract the frontend depends on), and `work/decisions/sifty-go-port.md` (the binding port decisions: open the existing database in place, pure-Go cgo-free driver, goroutine workers over the database locks, provider invocation unchanged).

This card builds on the previous card's Go cores and the thin TypeScript worker boundary it defined; the coordinator ensures the base the executor works from carries that code before handover. Browser automation stays in the TypeScript Playwright worker invoked by the Go core, as before; there is no browser-automation code in Go.

Feedback and remembered answers are written to the configurable local data directory rather than into `kb/`, following the previous card's self-contained decision. This is a deliberate divergence from the TypeScript app's `kb/Applications` location and is recorded in the architecture document; it does not change the HTTP contract the frontend sees.

The parity mechanism is recorded request and response fixtures captured from the running TypeScript app and replayed against the Go backend, so the contract is enforced by a failing test rather than by trust in the prose. The same card adds a drift check and wires the fixtures into the test entry so the documentation cannot silently rot.
