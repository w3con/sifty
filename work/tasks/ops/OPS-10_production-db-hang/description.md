# OPS-10 — Production hangs: SQLite driver mutex leak under status polling

**Epic:** ops
**Created:** 2026-11-12
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Profile
Candidate profile
Close profile
Loading profile…

не работает профиль

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Diagnosis (executor, 2026-11-12; dump in `.local/diag-profile-hang/goroutine-dump-20261112.txt`)

- Every request that reads the database hung from about 12:55; requests that do not read it answered. The store has one connection (`store.go`, `SetMaxOpenConns(1)`).
- The holder: the HTTP handler of `GET /api/quick-match/analysis` (`server/routes.go:468` → `cmd/sifty/serve.go:138` → `background/fitworker.go:193` `FitWorker.Status` → `store/match_fit.go:119` `ModelWorkRows`, a `rows.Next` loop over every vacancy), stuck on a mutex in the driver: `modernc.org/libc@v1.79.0/libc.go:2763` `AtomicLoadNUint16`, called from `_walFindFrame` of `modernc.org/sqlite@v1.61.1`. 22 other goroutines (profile, jobs, queue, the fit worker) waited for the connection.
- Root cause: libc v1.79.0 takes the global mutex `atomicLoadStore16` and releases it without `defer`; a page fault while reading the WAL index (`-shm`) becomes a panic that `_modernc_seh_try` recovers into error 8714, leaving the mutex locked forever. Production logged the same 8714 on 2026-11-10 16:05 and was restarted at 16:11 (probably the same hang; not proven). In libc v1.81.0 that mutex no longer exists.
- Amplifier (ASSESSMENT-3): `Status` scans all vacancies with descriptions under the single connection on every UI poll (every 2.5 s). Unverified, possibly bigger: the UI may also call `refresh` → `/api/jobs` (34.9 MB, 2.7 s) on each poll.
- SETTINGS-7 is not involved. Staging (older build) did not hang.
- Production was restarted with the same release at 13:10 (`sift-promote`, pid record correct); it can hang again until this card ships.

## Plan (coordinator, 2026-11-12)

- [ ] **H.1 — Driver** (10–15 min; owner questioned 20–40): `modernc.org/sqlite` v1.62.0 and an explicit `modernc.org/libc` v1.81.0 in `go.mod` (the executor checked the 1.81.0 source); build; existing store/background tests. No automated test for the leak (it needs fault injection in the driver).
- [ ] **H.2 — Status without the database** (45–75 min): the worker caches the counts it already computes in `Step` (PlanWork stats, including early exits: empty queue, paused, no profile) under its mutex; `Status` returns the cache and never touches the store. One targeted test: after a `Step`, close the store and `Status` still answers.
- [ ] **H.2b — Several readers, one writer** (45–75 min; owner «ok», 2026-11-12): one stuck or slow read must not block every other request. Keep a single write connection and add a small pool of read-only connections (WAL allows concurrent readers), or equivalent. Every connection must get the same settings: `busy_timeout`, `foreign_keys` and WAL are today applied once with `Exec` on the single connection, so they must move into the connection string (per-connection pragmas) or a connection hook. Write transactions start as IMMEDIATE so two writers never deadlock on upgrade. Route plain reads (lists, status, profile) to readers; everything in a transaction and every write stays on the writer. One targeted test: while a read transaction is held open on one connection, another read and a write complete. Note in the report that a lock inside the driver (the libc case of H.1) would still stop all connections; H.2b is against ordinary slow or stuck queries.
- [ ] **H.3 — Health that cannot hang** (20–30 min, coordinator's decision on the executor's question): `GET /api/health` checks the database with a short timeout (e.g. 2 s) on a context and answers 503 "database not responding" instead of waiting; the UI's existing server-down banner shows it. No automatic exit or restart: there is no supervisor, so exiting would leave production down until a manual start.
- [ ] **H.4 — Measure the polling** (15–20 min, measure only): what the UI requests every 2.5 s on production (jobs list included or not, sizes, times). If `/api/jobs` is refetched on every poll, record it for a follow-up in INBOX-2 or a new card — do not change `frontend/main.tsx` here (INBOX-2 is changing the same poll loop).
- [ ] **H.5 — Release** (owner's approval): this card ships on its own, before INBOX-2. Before the release, clear the stale pid record the documented way (OPS-9 items 2 and 9 — hit twice on 2026-11-12).

## Estimate

Complexity 4, AI time about 2.25–3.75 h (H.2b added). No paid services. Must not touch: INBOX-2 and its worktree, staging, `frontend/main.tsx`.
