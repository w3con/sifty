# Decisions — Sifty Go backend port

Append-only record of the decisions that bind the Go rewrite of the Sifty backend. The "what" lives in `docs/architecture/backend-architecture.md` and `docs/architecture/backend-api-contract.md`; this file holds the "why" so it does not rot inside those documents. Each entry is dated and is the coordinator's decision unless noted; approval of a document that cites a decision is not authorship of the decision.

## 2026-11-02 — Data compatibility: Go opens the owner's existing database in place

**Decision.** The Go backend opens the same SQLite database file (`jobs.sqlite`) with the same schema and the same `user_version`, in place. It does not start from a fresh database and does not require an export/import. A new installation creates the schema; an existing installation is opened and used as-is.

**Why.** The target is a drop-in replacement: "works as the old backend" includes the owner's accumulated state — collected roles, assessments, workflow history, application drafts, remembered answers and feedback. The SQLite file format is stable and language-independent, so a Go driver reads a database written by the Node implementation without conversion. Starting fresh would silently discard the owner's search; a bespoke migration would be a second thing to maintain for no benefit over reading the format directly.

**Consequence.** The Go store must reproduce the final schema exactly (the architecture document's schema appendix is the authority) and must honour `user_version` and the idempotent `CREATE TABLE IF NOT EXISTS` / `ALTER` migration approach, so opening either a fresh or an existing database converges on the same shape.

## 2026-11-02 — SQLite driver: pure-Go, cgo-free

**Decision.** Use a pure-Go, cgo-free SQLite driver (`modernc.org/sqlite`) rather than a cgo binding (`mattn/go-sqlite3`).

**Why.** The card that created the Go backend made a single self-contained binary a goal and cross-compilation easy; cgo reintroduces a C toolchain dependency and complicates both. The workload is one owner's catalogue — thousands of rows, not millions — so the cgo driver's speed advantage is irrelevant. Pure-Go keeps the single-binary, no-external-toolchain property.

**Consequence.** WAL mode, `busy_timeout`, foreign keys and `BEGIN IMMEDIATE` transactions are all supported by the pure-Go driver and must be set exactly as the Node store sets them.

## 2026-11-02 — Background workers: goroutines over the same database locks

**Decision.** The assessment worker and the preparation worker become goroutines, each a ticker loop mirroring the current one-second interval, coordinated through the same `locks` table and `BEGIN IMMEDIATE` transactions, with `context.Context` for cancellation in place of the abort controllers.

**Why.** The coordination that matters is already in the database (named locks, queue-row ownership, transactional claims), not in the Node event loop; reproducing the lock protocol preserves the cross-process serialisation the design depends on. Goroutines map cleanly onto the existing per-second tick and keep the workers inside the one backend process, as today.

**Consequence.** The process-id liveness checks (`process.kill(pid,0)`) used to recover abandoned locks and queue rows are the one piece that assumes a single machine and one process family. The port keeps equivalent liveness recovery for the self-hosted single-owner target, but marks it as the boundary to revisit for a hosted multi-process version; it is not to be generalised silently.

## 2026-11-02 — Documentation-parity convention: discrepancies update the documents

**Decision.** When a behaviour difference is found between the Go backend and the documented API contract or the fixture files, the Go code is the authority and the documents are updated to match. The fixture file is corrected (or replaced with a captured fixture from the running Go server), the allow-list in `backend/internal/server/replay_test.go` is annotated with the reason, and `docs/architecture/backend-api-contract.md` is updated when the endpoint shape changes. A difference that is intentional but unexplained is not allowed: it must either be absent from the allow-list (which will then fail CI) or present in the allow-list with a one-sentence explanation.

**Why.** The plan statement of work says "the code is authoritative where it and the prose disagree." The documents derive from the TypeScript implementation; the Go port supersedes it. Updating the document closes the loop so the next reader has an accurate description of what the server actually does, and an allow-listed divergence with an explanation is honest about what was decided and why.

**How to act on a found discrepancy.** Three files may need updating:

1. `backend/internal/server/testdata/fixtures/<n>_<method>_<name>.json` — update `responseBodyShape`, `requiredFields`, and `responseBody` (for captured fixtures) to reflect the actual Go response.
2. `backend/internal/server/replay_test.go` — add an entry to `allowedDivergences` with the fixture filename as key and a one-sentence explanation as value, if the Go response legitimately differs from a captured TypeScript response.
3. `docs/architecture/backend-api-contract.md` — update the endpoint description when the response shape, status code, or field names change.

The route drift check (`TestRouteDrift` in `drift_test.go`) must also be kept consistent: if an endpoint is added or removed, add or remove its fixture and update the contract document.

**Consequence.** The fixture set and the contract document are living, writable documents, not frozen snapshots. They are updated in the same commit that changes server behaviour, so CI always passes on the current branch rather than accumulating a backlog of allow-listed divergences. See `docs/architecture/backend-api-contract.md` for the current endpoint shapes and `backend/internal/server/testdata/fixtures/` for the fixture set.

## 2026-11-02 — Provider inference: same external tools, Go process and HTTP calls

**Decision.** The Go core invokes the same provider backends: the subscription CLIs (`claude`, `opencode`) as child processes with the same isolated environment and the same arguments, and the `ollama` / OpenAI-compatible runtimes over loopback HTTP. No provider behaviour changes in the port.

**Why.** The provider contract is external and owner-configured; changing it would change results and break parity. Reproducing the exact invocation (stripped environment carrying no credentials or inherited tool configuration, loopback-only endpoints, no paid fallback) preserves both the behaviour and the safety guarantees.

**Consequence.** The isolated-environment construction and the output caps are load-bearing security invariants, not conveniences, and the port reproduces them exactly.
