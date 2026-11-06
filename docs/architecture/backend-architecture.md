# Sifty backend — architecture

> **Relocation note (2026-11-06).** The source paths named in this reference predate the structural move: the retired TypeScript implementation is now under `legacy/app/` and the Go backend under `backend/`. Behavioural descriptions and the schema appendix remain the reference; path references are historical.

**Status:** reference, current as of 2026-11-02. Describes the backend behaviour that the Go rewrite must reproduce. Grounded in the current TypeScript implementation under `app/src`; the HTTP contract and the data model are the stable interface, the module boundaries are the target shape, and the final sections map what is already in Go against what the port still owes.

**Authority of this document.** This is the coordinator's summary of `app/src`, approved but not verified line by line against every invariant. Where this document and the code disagree, the code is authoritative: this is a map, not the territory. Each module in section 5 names its source file, and that file is the authority for the exact rule the prose only summarises; before relying on a load-bearing invariant (an identity match, a fingerprint composition, a mismatch-confirmation rule, a lock protocol), read it in the named file. The parts that must not be allowed to drift from reality — the endpoint contract and the data model — are the ones section 11 marks for capture as checked fixtures rather than left as prose.

**Companion documents.** The exact request and response shape of every endpoint is in `backend-api-contract.md` beside this file. The decisions that bind the port (data compatibility, the SQLite driver, the worker model, provider invocation) are in `work/decisions/sifty-go-port.md`. The authoritative target database schema is the appendix (section 13) of this document.

## 1. What this is and who it serves

Sifty is a local-first job-triage backend for one candidate's search. A single owner runs it on their own machine; the browser interface, a React single-page application, talks to it over loopback HTTP, and nothing is exposed off the machine. The backend collects public job postings from configured sources, assesses each posting against the candidate's own evidence (CV, profile exports, owner-confirmed feedback), and helps draft answers to a shortlisted role's application form. A hosted multi-user version is expected later, so no part of the design may bake in a permanent single-machine assumption even though today's deployment is single-owner and loopback-only.

The reader of this document is the coordinator planning the Go port, an executor building it, and, once the public repository is populated, anyone reading the product's own architecture reference. Roles, not people, are named throughout.

## 2. What the backend does

The work splits into two stages the owner drives from the interface.

Stage 1 — discovery and fit. The backend fetches each configured public source, parses postings out of HTML, feeds, and public JSON APIs, deduplicates them into a single role record per real vacancy, and hydrates missing descriptions by fetching the posting's own page. It then assesses each role against the candidate's assembled evidence with a configurable local model, producing a 0–100 fit score, a grounded summary, cited evidence, gaps, and confirmed mismatches. Assessment runs continuously in the background as a durable queue, not per request.

Stage 2 — application preparation. For a role the owner pre-selects, the backend inspects the external application form (reading its questions without filling or submitting anything), then drafts answers grounded in the candidate's evidence and in owner-confirmed remembered answers. The owner edits the draft; the backend never submits.

Everything the model emits is grounded: evidence must be an exact quote from a candidate source, a claimed vacancy mismatch must be an exact quote from the posting, and an unconfirmed absence is recorded as unknown rather than asserted. A missing fact stays missing; the backend never invents one.

## 3. Architecture at a glance

The backend is one process with layered responsibilities. From the bottom up:

- **Data layer** — a single embedded SQLite database plus a set of durable markdown files for owner-authored evidence. This is the spine; every other layer reads and writes through it, and it is what survives a restart. Section 4 treats it on its own.
- **Domain layer** — the role record and its lifecycle (the workflow stages), candidate-evidence assembly, and the pure parsers that turn a fetched page into postings. No network, no model, no process boundary.
- **Collection layer** — per-source discovery adapters, the HTTP client with its network guards, proxy profiles, rate limiting and page cache, and the Playwright browser path for sources that need a real browser.
- **Inference layer** — provider configuration and the model call itself (subprocess CLIs or a loopback HTTP endpoint), the fit-assessment orchestration that banks evidence and enforces the context budget, and provider availability and diagnostics.
- **Orchestration layer** — the glue that runs a collection, the two background workers (assessment and preparation), cross-process locking, the daily scheduled run, and configuration read/write with validation.
- **Interface layer** — the HTTP server and its 31 REST endpoints serving the React interface, plus two alternate entry points (a command-line interface and a Model Context Protocol server) over the same orchestration.

Browser automation and the jobspy scraper are not in the core. In the Go target they are thin TypeScript worker processes the core invokes over a defined protocol, because the official Playwright has no Go binding; in the current TypeScript backend they are in-process, which is the one structural difference the port deliberately changes.

## 4. The data model — the SQLite store as the spine

The store is a first-class part of the backend, not an implementation detail. It holds the entire mutable state of the search, enforces the identity and concurrency invariants, and is the only thing that persists. The Go port has no equivalent yet: the Go side currently persists only answer and feedback files, so the whole store described here is new work.

The store opens one SQLite database (`jobs.sqlite`) in WAL mode with a busy timeout and foreign keys on, and runs an idempotent migration sequence on open (versions 1–10). Conceptually it holds these entities:

| Entity (tables) | What it holds and why |
|---|---|
| **Role** (`jobs`) | One row per real vacancy, keyed by a stable integer id and an `identity` of normalised company+title. The canonical record: title, company, URLs, description and its status, workflow status, timestamps, the current assessment (denormalised JSON) and last assessment error. |
| **Sightings** (`sightings`) | Each source+URL at which a role was observed, with a unique URL constraint. One role can be seen at many sources; this is how the same vacancy across boards collapses to one role. |
| **Identity plumbing** (`job_aliases`, `id_sequence`, `deleted_listings`) | Alias chains that keep an old id resolving to the merged role after two listings are found to be the same; a monotonic id sequence so ids are never reused; tombstones (hashed listing keys) so a permanently deleted role is not resurrected by the next collection. |
| **Raw captures** (`snapshots`) | The raw parsed posting as seen on each upsert, retained for provenance. |
| **Candidate context** (`contexts`, `context_refreshes`) | Versioned snapshots of the assembled candidate evidence; assessment pins the exact context version it used. |
| **Assessment results** (`assessments`) | The history of fit results and failures per role, alongside the denormalised current result on `jobs`. |
| **Assessment queue** (`assessment_queue`, `queue_state`) | The durable work queue for background assessment: per-role fingerprint, chosen provider, status, priority, attempts, backoff time, owner PID; plus queue-wide enable flag and runtime pause state. |
| **Collection telemetry** (`collection_runs`, `source_runs`, `collection_events`, `runs`, `health`) | Per-run and per-source coverage accounting and an event log, plus daily-run records and per-source health. The interface's source-health and collection-progress views read these. |
| **Discovery cache** (`discovery_cache`) | Cached public inventories per source so a repeat run re-filters locally instead of re-fetching, within a freshness window. |
| **Workflow history** (`workflow_history`) | Every stage transition, with whether it was explicitly confirmed. |
| **Application drafts** (`application_drafts`, `application_revisions`, `questionnaire_snapshots`, `preparation_operations`) | The Stage 2 state: the current draft per role, its full revision history, the inspected questionnaire snapshots, and the preparation worker's operation records. |
| **Feedback index** (`feedback`) | A pointer index to owner-confirmed feedback; the durable form of that feedback is the markdown file (section 8), written first so it survives a database loss. |
| **Locks** (`locks`) | Named cross-process mutexes (`daily`, `inference`) carrying the holder PID, so collection, assessment and deletion serialise across processes and a dead holder is recovered. |

Store invariants, all enforced here rather than in callers:

- **Identity and dedup.** `upsert` matches an incoming posting to an existing role by sighting URL, apply URL, or normalised identity (guarding against the `Unknown` company and same-host false matches), merges duplicates when a detail fetch reveals two board records are one role, and carries the strongest workflow status forward. A description only replaces an existing one when it is genuinely richer, never silently downgraded.
- **Stable ids and tombstones.** Ids come from a sequence that never reuses a value even across merges and deletes; a permanent delete writes tombstone hashes so the role cannot reappear, and also prunes it from the discovery cache.
- **Transactional writes.** Every multi-statement mutation runs inside `BEGIN IMMEDIATE`, committing or rolling back as a unit.
- **Workflow guards.** Moving a role to `applied` requires it to be pre-selected and explicitly confirmed; deletion is refused while collection, assessment or preparation is active, and is refused for a role still in triage (dismiss it there instead).

## 5. Modules and responsibilities

Each module below is a current `app/src` file; the Go port reorganises them into packages but owes the same responsibilities.

- **core** — configuration schemas and loading for the five config files (`search`, `runtime`, `providers`, `candidate`, `network`), path resolution, hashing, canonical-URL normalisation, query matching, the `Job`/`JobInput` types and the assessment result schema. Path resolution is today coupled to the FCTO git root (`ROOT`, `rootPath`, `localPath`, the retired FCTO state variable); the port replaces this with a configurable data directory, which is the self-containment requirement.
- **workflow** — the role lifecycle: four stages (`triage`, `preselected`, `no`, `applied`) with their stored-status aliases, the archive-retention window, the web3 classifier, and external-apply-URL resolution.
- **store** — the Store class of section 4.
- **context** — assembles candidate evidence into a versioned context: reads candidate files (PDF via `pdftotext`, text directly) and profile exports or public profile pages, folds in owner feedback, and selects evidence within a model's character budget by keyword relevance with no silent truncation.
- **feedback** — owner-confirmed candidate feedback: writes each record to a durable markdown file first and indexes it in the database second, extracts conservative explicit facts, flags unresolved conflicts rather than overwriting, binds feedback into the candidate context, and scopes job-specific feedback to its role.
- **application-memory** — owner-confirmed remembered answers, same file-first durable pattern, scoped general/company/role, bound into the context for drafting.
- **network** — the HTTP client: a server-side-request-forgery guard that refuses private and non-public addresses and credentialed URLs, per-source proxy profiles, global traffic rate limiting, a page cache with separate listing and detail freshness windows, output size caps, CAPTCHA/challenge detection, and a Playwright persistent-context browser path. Only an allowlisted set of read-only search POST endpoints is permitted; every other POST is refused.
- **collectors** — pure parsers with no network: structured JSON-LD `JobPosting`, a detail-page extractor that also resolves the real apply destination without ever submitting a form, RSS/Atom feeds, and a strict role-link fallback; plus the per-source capability description and the `importUrl`/`enrich` services that fetch and upsert a single role.
- **discovery** — the per-source engine: around a dozen site-specific adapters (Getro portfolio APIs, Braintrust, a16z, Remotive, RSS feeds, Hacker News hiring threads, Algolia-backed search, Wellfound/Apollo, Web3 Career, browser-session sources, a generic rel-next adapter), full coverage accounting, bounded pagination with repeat-page detection, concurrent detail hydration within request limits, query matching, and a structured per-run coverage report with an explicit blocker and completeness class.
- **providers** — the inference boundary: the model call across four provider types (subscription CLIs `opencode` and `claude` run as isolated subprocesses, and `ollama`/OpenAI-compatible runtimes over a loopback HTTP endpoint), the fit-assessment orchestration that banks candidate evidence into referenceable snippets, enforces the context budget, validates that the model only cited supplied evidence and only quoted exact vacancy text, and separates confirmed mismatches from unconfirmed absences; plus availability probing, a connectivity smoke check, and per-provider diagnostics. Subprocess inference runs with a stripped environment that carries no API credentials and no inherited tool or Model Context Protocol configuration.
- **queue** — assessment scheduling: a fingerprint over the role, context version, preferences, prompt and provider that detects when a result is stale, `enqueueJob`/`enqueueAll`, and the `AssessmentWorker` — a persistent background loop that claims one eligible role at a time under the inference lock, gates on provider availability (pausing the queue rather than falling back to a paid path), retries with exponential backoff, and recovers rows abandoned by a dead process.
- **applications** — the `PreparationWorker`: a background loop that, for one queued preparation at a time, optionally opens an owner sign-in window, inspects the application form through the browser path (reading questions only), then drafts answers under the shared inference lock; cancellation and interruption preserve saved answers.
- **application-store** — the draft model: the questionnaire and answer-field types, folding an inspection into a draft while preserving owner edits, manual questions, owner edits under optimistic concurrency (per-field revision and signature checks), and merging model-generated answers without clobbering concurrent edits.
- **services** — orchestration: config validation, source-health assembly, a full collection run with its telemetry, `startSearch`, collection-progress with dead-process recovery, the `acquireLock`/`exclusive` cross-process mutex, the daily scheduled run, and config read/save with schema validation and secret rejection.
- **server** — the HTTP interface of section 7: Express, the 31 endpoints, the loopback/Host/Origin/content-type security middleware, static serving of the built interface, and the lifecycle of the two workers plus the archive-retention timer and the candidate-evidence refresh timer.
- **cli**, **mcp** — a command-line entry point and a Model Context Protocol server over the same services.

## 6. How it works — the three flows

**Collection.** `startSearch` acquires the `daily` lock and opens a `collection_runs` row. For each enabled source, `discoverSource` picks the site's adapter, pages through its public listing under the configured bounds, matches postings against the configured OR queries, hydrates missing descriptions by fetching each posting's own page concurrently, and streams every matched posting into `store.upsert`, which deduplicates and enqueues it for assessment. Coverage — reviewed, matched, added, updated, duplicates, per-query counts, blockers — is written to `source_runs` and surfaced as source health. A source that is blocked, needs login, or whose adapter no longer matches the page is reported as such with an owner action, never as an empty result.

**Assessment.** The `AssessmentWorker` ticks continuously. It re-fingerprints eligible roles, claims the highest-priority pending role under the `inference` lock, checks provider availability, assembles the candidate context for that role (feedback and answer memories bound and scoped), banks the selected evidence into referenceable snippets within the model's budget, calls the model for a structured result, verifies every cited evidence id and every quoted mismatch against the actual source text, splits confirmed mismatches from unconfirmed absences, and stores the graded result. If the provider is unavailable the queue pauses rather than paying for a fallback; failures back off exponentially.

**Preparation.** For a pre-selected role the owner queues a preparation. The `PreparationWorker` inspects the external form through the browser path (questions only, nothing filled or submitted), folds the questionnaire into the draft while preserving existing answers, then — under the inference lock — drafts answers grounded in the candidate context and remembered answers. The owner reviews and edits; edits use optimistic concurrency so a background draft never overwrites a concurrent manual change. Nothing is ever submitted.

## 7. Interfaces

**HTTP REST — the React interface contract.** All under `/api`, loopback only, served alongside the static interface build. This is the exact surface the Go port must reproduce for the existing frontend to work unchanged.

| Method & path | Purpose |
|---|---|
| `GET /api/jobs` | List roles filtered by status and query. |
| `GET /api/jobs/:id` | One role with its full assessment and sources. |
| `PATCH /api/jobs/:id` | Move a role's workflow stage (confirmed flag for applied). |
| `DELETE /api/jobs/:id` | Permanent delete (requires explicit confirmation). |
| `POST /api/jobs/:id/seen` | Mark a role seen. |
| `GET /api/health` | Per-source collection health. |
| `GET /api/context` | Current candidate-context snapshot. |
| `GET /api/filter-options` | The search configuration for filters. |
| `GET /api/assessment-options` | Selected and available assessment providers. |
| `GET /api/queue` | Assessment queue status and counts. |
| `GET /api/jobs/:id/application` | The role's application draft and recent operations. |
| `POST /api/jobs/:id/application/prepare` | Queue a preparation (inspect/generate/login). |
| `POST /api/jobs/:id/application/cancel` | Cancel an active preparation. |
| `PATCH /api/jobs/:id/application` | Owner edit of a draft field or selected file. |
| `POST /api/jobs/:id/application/questions` | Add a manual question. |
| `POST /api/jobs/:id/application/remember` | Remember an answer for future applications. |
| `GET /api/jobs/:id/application/memories` | Remembered answers applicable to the role. |
| `GET /api/application-files/:name` | Serve `cv.pdf` or `profile.pdf` inline. |
| `PUT /api/queue/assessment` | Enable or disable background assessment. |
| `GET /api/jobs/:id/feedback` | Feedback records for the role. |
| `POST /api/jobs/:id/feedback` | Save feedback and trigger reassessment. |
| `GET /api/providers` | Provider diagnostics. |
| `POST /api/providers/:name/check` | Provider connectivity smoke check. |
| `GET /api/context` / `POST /api/context` | Read / refresh candidate context. |
| `GET /api/collection` | Current collection run progress. |
| `POST /api/search` | Start a collection run (optionally one source, forced). |
| `POST /api/import` | Import a single role by URL. |
| `POST /api/jobs/:id/read` | Re-fetch and enrich one role's detail. |
| `POST /api/jobs/:id/assess` | Enqueue one role for assessment now. |
| `GET /api/config/:name` / `PUT /api/config/:name` | Read / write a config file (validated). |

A write must be `application/json`; an unknown `/api` path returns a 404 that tells the interface to refresh.

**Command-line interface.** `cli.ts` exposes the same services (validate, context, search, list, detail, status, import, read, assess, health, provider-check, daily, serve, session, backup) for owner operation and scripting.

**Model Context Protocol server.** `mcp.ts` exposes the services as MCP tools.

**Worker protocol.** The core-to-worker boundary for browser automation and the jobspy scraper: a request/response contract over a child process, already defined on the Go side in the worker package, with the Playwright and jobspy workers as its two implementations.

**Provider inference.** Either a subprocess CLI (`claude`, `opencode`) run with an isolated environment, or a loopback HTTP call to an `ollama` or OpenAI-compatible runtime; no remote, credentialed, or paid fallback path.

## 8. Storage systems, data ownership, and write paths

The backend keeps data in three systems, plus one external store it is currently coupled to and the Go product drops.

- **The SQLite database** — all operational state: roles and their sightings, assessments and the assessment queue, workflow history, application drafts, collection telemetry, the feedback and context indexes, and the cross-process locks. Section 4 details it. Today it is one file, `jobs.sqlite`, inside the local state directory.
- **The local state directory** — everything transient or machine-local that is not in the database: the database file itself, the page cache, per-source browser session directories and proxy state, provider inference working directories, timestamped config backups, and preparation-error dumps. Today this is a `.local` directory resolved from the repository root.
- **Durable owner files** — two kinds of plain files that are the authoritative form of their data, written so they survive a database loss. First, owner inputs the backend only reads: the candidate CV and profile documents (`cv.pdf`, `profile.pdf`), the five configuration YAML files, and the assessment prompt. Second, owner-authored records the backend writes file-first and only indexes in the database: confirmed candidate feedback and remembered application answers.
- **The knowledge base (Obsidian vault, `kb/`)** — the FCTO private vault, not a Sifty storage system of its own. In the current FCTO coupling it is where the owner-authored records default: feedback lands in `kb/Applications/candidate-feedback/` and remembered answers in `kb/Applications/answers/`, so they are visible and editable in Obsidian. This is exactly the coupling the Go rewrite removes: the self-contained product has no knowledge base, and the owner inputs, owner records, database and caches all live under one configurable data directory with file templates, with no resolution from a git root and no `FCTO`-specific naming or `kb/` writes.

The table below is the per-datum map: what each kind of data is, where and how it is stored, which module writes it, and when. "Owner" means the data is authored by the person and only read by the backend; every other row is written by the backend.

| Data | What it is / who owns it | Where and how stored | Written by (module → trigger) | When |
|---|---|---|---|---|
| **CV and profile documents** | `cv.pdf`, `profile.pdf` — the candidate's own documents. Owner-provided; backend reads only. | Plain files, today at the repository root; in Go, in the data directory. PDF text extracted with `pdftotext`. | Owner places the files; backend never writes them. | Set up once, updated by the owner as needed. |
| **User profile configuration** | `candidate.yaml`: evidence file list, online profiles (e.g. LinkedIn URL and optional export), cache window, source roles, review notes. Owner-owned. | A configuration YAML file (`config/candidate.yaml`). | `services.saveConfig` on `PUT /api/config/candidate`, or the owner editing the file directly. | When the owner changes their profile setup. |
| **Search, runtime, providers, network configuration** | The other four YAML files: sources and queries, collection bounds and schedule, provider selection, proxy profiles. Owner-owned. | Configuration YAML files under `config/`, schema-validated, with a timestamped backup on each save. | `services.saveConfig` on `PUT /api/config/:name`. | When the owner changes settings. |
| **Assembled candidate context** | A versioned snapshot of all candidate evidence (files + profiles + bound feedback and answer memories), with budgeted selection for the model. Derived, backend-written. | SQLite (`contexts`, `context_refreshes`) as versioned JSON. | `context.refreshContext` → `store.saveContext`. | On server start, on a 30-second timer when the evidence fingerprint changes, on `POST /api/context`, and lazily before an assessment whose preferences drifted. |
| **Vacancies (roles)** | The deduplicated role records collected from sources, with their per-source sightings and raw captures. Backend-written. | SQLite (`jobs`, `sightings`, `snapshots`, and the identity tables `job_aliases`/`id_sequence`/`deleted_listings`). | `store.upsert`, called from `discovery.discoverSource` via `services.search`, and from `collectors.importUrl`/`enrich`. | During a collection run, on manual import, and on a single-role re-fetch. |
| **Assessment results** | The fit score, grounded summary, evidence, gaps and confirmed mismatches per role, plus failures. Backend-written. | SQLite (`assessments` history; current result denormalised on `jobs.assessment`). | `store.assessment`/`assessmentFailure` via `providers.assess`, driven by the `queue.AssessmentWorker`. | Continuously in the background whenever a role's inputs are stale and a provider is available. |
| **Assessment queue** | The durable per-role work queue and its enable/pause state. Backend-written. | SQLite (`assessment_queue`, `queue_state`). | `queue.enqueueJob`/`enqueueAll` and the worker's own transitions; `setAssessmentEnabled` on `PUT /api/queue/assessment`. | On each upsert, on context or configuration change, on an explicit assess request, and every worker tick. |
| **Workflow transitions** | The role's stage and the history of its moves. Backend-written, owner-driven. | SQLite (`jobs.status`, `workflow_history`). | `store.status` on `PATCH /api/jobs/:id`. | When the owner moves a role between triage, pre-selected, rejected and applied. |
| **Confirmed candidate feedback** | The owner's confirmed statements about the candidate (corrections, absences, preferences), the ground truth for reassessment. Owner-authored, backend-persisted. | Durable markdown file first (today `kb/Applications/candidate-feedback/`, in Go the data directory), indexed in SQLite (`feedback`). | `feedback.saveFeedback` on `POST /api/jobs/:id/feedback`; triggers reassessment. | When the owner saves feedback. This is the vacancy/candidate feedback feature being built out next; the storage pattern is already the one above. |
| **Remembered application answers** | Owner-confirmed answers to reuse across applications, scoped general/company/role. Owner-authored, backend-persisted. | Durable markdown file first (today `kb/Applications/answers/`, in Go the data directory). | `application-memory.rememberAnswer` on `POST /api/jobs/:id/application/remember`. | When the owner clicks Remember on a draft answer. |
| **Application drafts** | The Stage 2 questionnaire, answer fields and their full revision history per role. Backend-written, owner-edited. | SQLite (`application_drafts`, `application_revisions`, `questionnaire_snapshots`, `preparation_operations`). | `application-store` (inspect-into-draft, manual question, owner edit, merge-generated) and the `applications.PreparationWorker`. | During a preparation run and on each owner edit. |
| **Collection telemetry and source health** | Per-run and per-source coverage, an event log, and health. Backend-written. | SQLite (`collection_runs`, `source_runs`, `collection_events`, `runs`, `health`). | `services.search` and `store.health`. | During and after a collection run. |
| **Discovery cache** | Cached public inventories per source, to re-filter locally instead of re-fetching. Backend-written. | SQLite (`discovery_cache`). | `services.search` save-inventory callback. | During a collection run, within a freshness window. |
| **Page cache, browser sessions, inference and error dumps** | Fetched-page cache, per-source persistent browser profiles and proxy state, provider working directories, config backups, preparation-error files. Backend-written, machine-local. | The local state directory (today `.local/`, in Go the data directory). | `network` (page cache, browser session), `providers.infer`, `services.saveConfig`, the preparation worker. | As fetches, inferences, saves and preparations occur. |

## 9. Checks, security and integrity

- **Network exposure.** Loopback bind only; every request's Host and Origin are validated against the configured port; writes must be JSON; `nosniff` and `no-referrer` headers are set.
- **Outbound safety.** A server-side-request-forgery guard refuses private, loopback-internal and non-HTTP destinations and credentialed URLs; only an allowlisted set of read-only search POSTs is permitted; responses are size-capped; access challenges and CAPTCHAs are detected and reported, not worked around.
- **Inference isolation.** Subprocess providers run with a stripped environment carrying no API keys and no inherited tool or Model Context Protocol configuration; non-CLI providers must point at a loopback endpoint; there is no paid fallback — unavailability pauses the queue.
- **Grounding and honesty.** The model may cite only supplied evidence ids; a claimed vacancy mismatch must be an exact quote from the posting and an evidence quote must be exact source text, both verified after the call; unconfirmed absences are recorded as unknown, not asserted; the context budget is enforced with no silent truncation; a missing fact stays missing.
- **Data integrity.** Identity deduplication with alias merging; monotonic non-reused ids; tombstones against resurrection; `BEGIN IMMEDIATE` transactions; optimistic concurrency on draft edits via revision and signature; cross-process locks with dead-holder recovery.
- **Workflow and retention.** Applied requires a pre-selected role and explicit confirmation; deletion is blocked during active work and in triage; applied roles are archived after the retention window.
- **Configuration safety.** Config saves are schema-validated and reject inline secrets, keeping a timestamped backup of the previous file.

## 10. Current state and the parity target

The Go rewrite to date ported the computational cores and proved them equivalent to the TypeScript implementation on fixtures, at the package level: structured collection parsers, the fit-assessment logic, the application-answer preparation logic, form inspection through the Playwright worker, and answer/feedback file records. The Go binary today exposes only `version`, `help` and `selfcheck`; none of the stages are runnable from it and there is no store, no service layer, no background worker, no HTTP server, and no interface.

For the existing React interface to work against Go exactly as against TypeScript, the port must build the application layer this document describes: the SQLite store with all its entities and invariants (section 4); the service orchestration (collection run, daily, config read/write, locking) and the two background workers (assessment and preparation) with their queues; provider availability, diagnostics and the smoke check; the discovery engine wiring the already-ported parsers and the jobspy worker into runnable per-source collection; and the HTTP server exposing all 31 endpoints (section 7) with the same security middleware, serving the built interface. The already-ported cores are the inner logic these layers call; the missing layers are the bulk of the backend as a running application. The acceptance test is behavioural: the owner drives the existing interface against the Go backend and every view behaves as it does against TypeScript. That behavioural test is the final gate, not the parity mechanism; the mechanism is the recorded fixtures of section 11, diffed Go-against-TypeScript, so parity is judged by a diff rather than by opinion.

## 11. What the port reproduces, what it changes, and what is captured as fact

Not every current behaviour is a target. Some of it is FCTO-coupling the self-contained product drops, and porting it faithfully would carry the accidents into new code. The port reproduces one list and deliberately replaces the other.

**Reproduce** — behaviour the interface and the owner depend on:

- The HTTP contract: all 31 endpoints with their request and response shapes, status codes and error forms, plus static serving of the built interface and the security middleware.
- The data model's entities and invariants: identity deduplication and merging, stable non-reused ids, tombstones against resurrection, transactional writes, the workflow guards, and optimistic concurrency on draft edits.
- The collection adapters' output and coverage semantics: per-source parsing, bounded pagination with repeat-page detection, query matching, and description hydration.
- The fit-assessment grounding rules: evidence banking within the context budget, exact-quote verification of cited evidence and claimed mismatches, the confirmed-versus-unconfirmed split, and no silent truncation.
- The queue semantics: fingerprint staleness, availability gating with no paid fallback, exponential backoff, and cancellation that preserves saved answers.
- Provider inference across the four provider types with an isolated subprocess environment.
- Durable-file-first writing of owner feedback and remembered answers.
- The network safety guards: loopback bind, Host and Origin checks, the server-side-request-forgery guard, the read-only POST allowlist, output size caps, and challenge detection.

**Change** — coupling and single-machine accidents to replace, not port:

- Path resolution from the git root (`ROOT`, `rootPath`, `localPath`, the retired FCTO state variable) becomes one configurable data directory with file templates.
- Writing owner records into the knowledge base (`kb/Applications/...`) becomes writing them into the data directory; no `kb/` and no Obsidian dependence.
- In-process browser automation and the jobspy scraper become thin TypeScript worker processes the Go core invokes over the worker protocol.
- Cross-process coordination by process-id liveness (the `process.kill(pid,0)` locks and worker ownership) is the design point most in tension with the expected hosted version; the port decides its coordination model explicitly, recorded in the port card and the decisions log, rather than copying the process-id approach.
- `FCTO`-specific naming in state paths and in the collection user-agent is dropped.

The decisions that bound the port are settled in `work/decisions/sifty-go-port.md`: the Go backend opens the owner's existing SQLite database in place and reproduces its schema exactly (section 13), through a pure-Go cgo-free driver; the background workers become goroutines coordinated through the same database locks; and provider invocation is reproduced unchanged. The process-id liveness recovery is kept for the single-owner target and flagged there as the boundary to revisit for a hosted version, not generalised silently.

**Capture as fact.** The two parts that must not drift from reality are the endpoint contract and the data model. The port captures them as checked fixtures — recorded request/response pairs snapshotted from the running TypeScript backend, and the schema — so that Go-versus-TypeScript parity, and this document's own correctness, are enforced by a failing test rather than by trust in prose. This is the living-document mechanism; the harness additions that keep it alive are planned with the port card.

## 12. Glossary

Terms are fixed here so they do not drift across the port. The code is older than this vocabulary and is not always consistent; this document is.

- **Role** (the code also says *job*): one real vacancy, one row in `jobs`. This document says role.
- **Posting**: a single parsed listing as seen at a source before deduplication; it becomes, or merges into, a role.
- **Sighting**: a (source, URL) pair at which a role was observed; one role can have many.
- **Stage** versus **status**: the stage is the workflow position (triage, pre-selected, rejected, applied); the status is its stored alias (new, saved, rejected, applied). They translate one to one.
- **Evidence** versus **context**: evidence is the raw owner material (CV, profile exports, confirmed feedback, remembered answers); the context is the assembled, versioned, budget-selected snapshot built from that evidence and pinned by each assessment.
- **Assessment** (the *fit* result): the model's grounded evaluation of a role against the context, scored 0–100 with cited evidence, gaps and mismatches.
- **Preparation**: the Stage 2 drafting of answers to a role's application form.
- **Provider** and **inference**: the configured model backend, and the act of calling it.
- **Background worker** versus **TypeScript worker**: a background worker is an in-backend loop (the assessment worker, the preparation worker); a TypeScript worker is a separate child process the Go core invokes over the worker protocol (the Playwright worker, the jobspy worker). The two are unrelated despite the shared word.
- **Data directory**: the single configurable location for all local state in the self-contained product, replacing the `.local` state directory and the `kb/` writes.

## 13. Appendix — target database schema

The authoritative schema the Go store must reproduce, consolidated to its final shape (migration `user_version` 10). The database opens with `PRAGMA journal_mode=WAL; PRAGMA busy_timeout=5000; PRAGMA foreign_keys=ON;` and every multi-statement mutation runs inside `BEGIN IMMEDIATE`. Tables are created idempotently (`CREATE TABLE IF NOT EXISTS`); an existing database is opened in place, a fresh one is created to this shape.

```sql
CREATE TABLE migrations(version INTEGER PRIMARY KEY);

CREATE TABLE jobs(
  id INTEGER PRIMARY KEY, identity TEXT NOT NULL, title TEXT NOT NULL, company TEXT NOT NULL,
  original_url TEXT NOT NULL, apply_url TEXT, description TEXT NOT NULL, description_status TEXT NOT NULL,
  location TEXT, published_at TEXT, expires_at TEXT, status TEXT NOT NULL DEFAULT 'new',
  first_seen TEXT NOT NULL, last_seen TEXT NOT NULL, assessment TEXT, assessment_error TEXT,
  apply_status TEXT NOT NULL DEFAULT 'not resolved', tags TEXT NOT NULL DEFAULT '[]',
  seen_at TEXT, applied_at TEXT);
CREATE INDEX jobs_identity ON jobs(identity);
CREATE INDEX jobs_apply ON jobs(apply_url);

CREATE TABLE sightings(job_id INTEGER NOT NULL REFERENCES jobs(id), source TEXT NOT NULL,
  url TEXT NOT NULL UNIQUE, seen TEXT NOT NULL);

CREATE TABLE job_aliases(old_id INTEGER PRIMARY KEY, new_id INTEGER NOT NULL);
CREATE TABLE id_sequence(name TEXT PRIMARY KEY, value INTEGER NOT NULL);
CREATE TABLE deleted_listings(hash TEXT PRIMARY KEY);

CREATE TABLE snapshots(id INTEGER PRIMARY KEY, job_id INTEGER REFERENCES jobs(id),
  captured TEXT NOT NULL, data TEXT NOT NULL);

CREATE TABLE contexts(version TEXT PRIMARY KEY, created TEXT NOT NULL, data TEXT NOT NULL);
CREATE TABLE context_refreshes(id INTEGER PRIMARY KEY, version TEXT NOT NULL,
  created TEXT NOT NULL, data TEXT NOT NULL);

CREATE TABLE assessments(id INTEGER PRIMARY KEY, job_id INTEGER REFERENCES jobs(id),
  created TEXT NOT NULL, data TEXT NOT NULL);

CREATE TABLE assessment_queue(
  job_id INTEGER PRIMARY KEY, fingerprint TEXT NOT NULL, provider TEXT NOT NULL, status TEXT NOT NULL,
  priority INTEGER NOT NULL DEFAULT 0, attempts INTEGER NOT NULL DEFAULT 0, next_at INTEGER NOT NULL DEFAULT 0,
  started INTEGER, finished INTEGER, error TEXT, owner INTEGER,
  override_provider TEXT, routing TEXT);
CREATE TABLE queue_state(key TEXT PRIMARY KEY, data TEXT NOT NULL);

CREATE TABLE health(source TEXT PRIMARY KEY, status TEXT NOT NULL, message TEXT NOT NULL,
  checked TEXT NOT NULL, count INTEGER NOT NULL);

CREATE TABLE runs(id INTEGER PRIMARY KEY, started TEXT NOT NULL, finished TEXT,
  status TEXT NOT NULL, detail TEXT);
CREATE TABLE collection_runs(id INTEGER PRIMARY KEY, started TEXT NOT NULL, finished TEXT,
  status TEXT NOT NULL, pid INTEGER NOT NULL, mode TEXT NOT NULL, config_hash TEXT NOT NULL);
CREATE TABLE source_runs(run_id INTEGER NOT NULL, source TEXT NOT NULL, status TEXT NOT NULL,
  coverage TEXT NOT NULL, PRIMARY KEY(run_id, source));
CREATE TABLE collection_events(id INTEGER PRIMARY KEY, run_id INTEGER NOT NULL,
  created TEXT NOT NULL, source TEXT, message TEXT NOT NULL);

CREATE TABLE discovery_cache(key TEXT PRIMARY KEY, created INTEGER NOT NULL, data TEXT NOT NULL);

CREATE TABLE locks(name TEXT PRIMARY KEY, pid INTEGER NOT NULL, acquired TEXT NOT NULL);

CREATE TABLE workflow_history(id INTEGER PRIMARY KEY, job_id INTEGER NOT NULL, created TEXT NOT NULL,
  from_stage TEXT NOT NULL, to_stage TEXT NOT NULL, confirmed INTEGER NOT NULL DEFAULT 0);

CREATE TABLE feedback(id TEXT PRIMARY KEY, job_id INTEGER NOT NULL, created TEXT NOT NULL,
  path TEXT NOT NULL, data TEXT NOT NULL);

CREATE TABLE application_drafts(job_id INTEGER PRIMARY KEY, data TEXT NOT NULL);
CREATE TABLE application_revisions(id INTEGER PRIMARY KEY, job_id INTEGER NOT NULL,
  created TEXT NOT NULL, reason TEXT NOT NULL, data TEXT NOT NULL);
CREATE TABLE questionnaire_snapshots(id INTEGER PRIMARY KEY, job_id INTEGER NOT NULL,
  created TEXT NOT NULL, data TEXT NOT NULL);
CREATE TABLE preparation_operations(id INTEGER PRIMARY KEY, job_id INTEGER NOT NULL, status TEXT NOT NULL,
  created TEXT NOT NULL, updated TEXT NOT NULL, pid INTEGER, request TEXT NOT NULL,
  error TEXT, progress TEXT NOT NULL);
```

Columns carrying `data`, `coverage`, `assessment`, `request` and `detail` hold JSON text. Timestamps are ISO-8601 strings except `discovery_cache.created`, `assessment_queue.next_at/started/finished` and the queue times, which are epoch-millisecond integers. The `jobs.assessment` column is the denormalised current result; `assessments` is its history.
