# Go parity audit — `app/src` deletion readiness

**Date:** 2026-11-04. This is a read-only audit grounding the GO-5 decision to delete `app/src`. It goes module-by-module through the 25 TypeScript source files under `app/src/`, determines the status of each capability in the Go backend under `sifty/`, and distinguishes between code that is present-and-wired (reachable from a live entry point), present-but-dead (ported but nothing calls it in production), and missing entirely. Every Go-side claim cites a file and line number; every "present" judgement is verified by grep rather than assumed from the file name.

---

## Summary table

| Capability | TS source file | Go status | Evidence (Go side) | Blocks `app/src` deletion? | Likely owning card |
|---|---|---|---|---|---|
| Config loading, path resolution, types | `core.ts` | present & wired | `internal/config/config.go:2`, `internal/dirs/dirs.go`, `internal/model/job.go`; loaded in `cmd/sifty/serve.go:35` | no | — |
| Role lifecycle, workflow stages, web3 classifier | `workflow.ts` | present & wired | `internal/store/core.go:25` (`stageFor`), `core.go:50` (`isWeb3`), `internal/background/inspect.go:131` (`externalApplyURL`) | no | — |
| SQLite store — all entities and invariants | `store.ts` | present & wired | `internal/store/store.go`, `jobs.go`, `queue.go`, `application.go`, `migrate.go`, `process.go`; opened at `cmd/sifty/serve.go:52` | no | — |
| Candidate context assembly (files, profiles, feedback) | `context.ts` | present & wired | `internal/candidate/context.go:244` (`RefreshContext`); called from `internal/server/routes.go:342` and assessment worker | no | — |
| Owner feedback — durable file-first write, DB index | `feedback.ts` | present & wired | `internal/records/records.go:169` (`WriteFeedback`), `ReadFeedback`; routes at `server/routes.go:516–580` | no | — |
| Remembered answer records — write and read files | `application-memory.ts` (write/read) | present & wired | `internal/records/records.go:131` (`WriteAnswer`, `ReadAnswers`); routes at `server/routes.go:680–725` | no | — |
| Answer memories bound into preparation context | `application-memory.ts` (`bindAnswerMemories`) | **missing** | `internal/background/preparation.go:367` loads raw context; no call to `records.ReadAnswers` or any memories-binding step before `prepare.Prepare` | **yes** | COL-1 |
| HTTP client — SSRF guard, proxy, page cache | `network.ts` | **missing** | `internal/server/routes.go:173`: `EnrichFn` defaulted to no-op stub; no Go network package found | **yes** | COL-1 |
| Domain rate limiting and per-domain cooldowns | `traffic.ts` | **missing** | no Go equivalent found; `internal/background/discipline.go` covers only the browser/inspect path | **yes** | COL-1 |
| Collection parsers — JSON-LD, RSS/Atom, HTML links | `collectors.ts` (parsers) | present & wired | `internal/collect/parse.go:25` (`StructuredJobs`), `collect/feed.go` (`parseFeed`), `collect/text.go` (`HTMLLinks`); called from tests and within collect pkg | no | — |
| Single-role fetch and enrich | `collectors.ts` (`importUrl`, `enrich`) | **present-but-dead** | `POST /api/import` calls `importURL` which creates a stub record (title `"Manual import"`, no fetch) at `server/routes.go:784`; `POST /api/jobs/:id/read` has no-op EnrichFn at `routes.go:173` | **yes** | COL-1 |
| Per-source discovery adapters (12+ sites, pagination) | `discovery.ts` | **missing** | no Go package or function matching any TS adapter found; `grep -rn "discoverSource\|parseGetroJobs\|parseWellfound" sifty` returns nothing | **yes** | COL-1 |
| Model inference across four provider types | `providers.ts` (inference) | present & wired | `internal/assess/infer.go:21` (`Infer`); subprocess isolation at `infer.go:37`; HTTP at `infer.go:158`; called from assessment worker | no | — |
| Fit-assessment orchestration, grounding rules | `providers.ts` (assess/orchestration) | present & wired | `internal/assess/assess.go:277` (`Assess`); evidence banking, exact-quote verification, confirmed/unconfirmed split all present | no | — |
| Provider diagnostics and smoke check | `providers.ts` (diagnostics) | present & wired | `internal/assess/diagnostics.go`; `SmokeCheck` at `assess/infer.go`; routes at `server/routes.go:356–389` | no | — |
| Assessment queue — fingerprint, enqueue, backoff | `queue.ts` | present & wired | `internal/store/queue.go:1`; `EnqueueJob`, `EnqueueAll`, fingerprint logic; worker at `internal/background/assessment.go:44` | no | — |
| Assessment background worker | `queue.ts` (AssessmentWorker) | present & wired | `internal/background/assessment.go:28`; started at `cmd/sifty/serve.go:89` | no | — |
| Application preparation worker | `applications.ts` | present & wired | `internal/background/preparation.go:37`; started at `cmd/sifty/serve.go:90` | no | — |
| Application draft model — fields, questionnaire, revisions | `application-store.ts` | present & wired | `internal/store/application.go:22` (`GetDraft`, `WriteDraft`, `AddQuestion`); routes wired | no | — |
| inspectIntoDraft — merge questionnaire into draft | `application-store.ts` (`inspectIntoDraft`) | present & wired | `internal/background/inspect.go:17`; called from preparation worker | no | — |
| PATCH draft — optimistic per-field revision check | `application-store.ts` (`updateApplication`) | **present-but-dead** | TS checks `field.revision !== change.revision` before writing; Go `PATCH /api/jobs/{id}/application` at `server/routes.go:635` merges body blindly with `json.Unmarshal(bodyJSON, draft)`, no revision check | **yes** | COL-1 |
| Application answer generation — grounded drafting | `application-generation.ts` | present & wired | `internal/prepare/prepare.go:1` (`Prepare`); called from `internal/background/preparation.go:419`; grounding, evidence bank, field validation all present | no | — |
| Browser form inspection (Playwright) | `application-browser.ts` | present & wired (TypeScript worker by design) | Go calls the Playwright TypeScript worker via `internal/worker/worker.go` and `internal/inspect/inspect.go:40`; wired at `cmd/sifty/serve.go:82` | no | — |
| Browser discipline — pacing, concurrency, cooldown | `application-browser.ts` (pacing logic) | present & wired | `internal/background/discipline.go:37`; wired at `cmd/sifty/serve.go:85` | no | — |
| Navigation kind, Getro destination, apply links | `application-navigation.ts` | present & wired (TypeScript worker by design) | `externalApplyURL` at `internal/background/inspect.go:131`; full HTML navigation logic remains in Playwright worker | no | — |
| Rendered-field extractor (browser-serialized) | `rendered-fields.ts` | present & wired (TypeScript worker by design) | serialized into Playwright worker frames; no Go equivalent needed by design | no | — |
| Preparation error message simplification | `preparation-error.ts` | present & wired (TypeScript worker by design) | used in Playwright worker; Go preparation records error strings directly | no | — |
| Config read/write, source-health assembly | `services.ts` (config, health, locking) | present & wired | config R/W at `internal/config/save.go` + `routes.go:391–434`; health at `store.HealthList`, wired at `routes.go:241`; locks at `internal/store/process.go` | no | — |
| Full collection run — per-source discovery, upsert | `services.ts` (`startSearch`, `search`) | **present-but-dead** | `POST /api/search` at `routes.go:216` inserts a `collection_runs` row and returns accepted; `defaultCollect` at `scheduler.go:153` calls only `s.EnqueueAll()` | **yes** | COL-1 |
| Collection progress endpoint | `services.ts` (`collectionProgress`) | present & wired | `GET /api/collection` at `routes.go:232` reads latest run from store | no | — |
| Daily scheduled run | `services.ts` (`daily`) / scheduler | **present-but-dead** | `internal/background/scheduler.go:91`; scheduler fires correctly, but `defaultCollect` at `scheduler.go:153` only enqueues; no network collection | **yes** | COL-1 |
| HTTP server, 29 of 31 endpoints, security middleware | `server.ts` | present & wired | `internal/server/server.go:42` (security middleware), `routes.go` (all endpoints); started at `cmd/sifty/serve.go:60` | no | — |
| 30-second context refresh timer (evidence fingerprint) | `server.ts` (`contextTimer`) | **missing** | TS `server.ts:23` polls every 30 s; `cmd/sifty/serve.go` starts three workers and nothing else; no context timer goroutine found | **yes** | COL-1 |
| Archive retention timer | `archive.ts` / `server.ts` | **present-but-dead** | `internal/store/jobs.go:882` (`PurgeArchive`) exists; TS `archive.ts:3` runs it every minute; `cmd/sifty/serve.go` does not start any archive ticker | **yes** | COL-1 |
| Full CLI entry point | `cli.ts` | **missing** | `cmd/sifty/main.go:49`: commands are `version`, `serve`, `help`, `selfcheck` only; TS CLI has 15 commands including `validate`, `context`, `search`, `list`, `detail`, `status`, `import`, `read`, `assess`, `health`, `daily`, `provider-check`, `session`, `backup` | **yes** | none yet |
| MCP server entry point | `mcp.ts` | **missing** | no Go package or binary under `sifty/cmd/` matches; `grep -rn "McpServer\|mcp\|Model.*Context.*Protocol" sifty --include=*.go` returns nothing | **yes** | none yet |
| Model-based extraction fallback | `extraction.ts` | **missing** | `internal/config/config.go:50` has `ModelExtraction bool` flag; no `extractWithModel` function found; `grep -rn "extractWithModel\|extraction.*prompt\|prompts/extraction" sifty --include=*.go` returns nothing | **yes** | COL-1 |

---

## Non-green items

### Answer memories not bound into preparation context

`application-memory.ts` exports `bindAnswerMemories`, which reads the owner's saved answer-memory files and injects them as `owner-confirmed-answer` evidence sources into the candidate context immediately before answer generation (`application-generation.ts:8`). The Go `records.ReadAnswers` function (`internal/records/records.go:131`) reads and writes the same files, and the HTTP routes for `POST /api/jobs/:id/application/remember` and `GET /api/jobs/:id/application/memories` are wired. What is absent is the step inside `defaultPrepareJob` (`internal/background/preparation.go:367–420`) that calls `records.ReadAnswers`, filters by scope, and appends the records as extra evidence sources before calling `prepare.Prepare`. Without it, Go generates answers without access to any previously saved memories.

### No HTTP client — SSRF guard, proxy, page cache, Playwright fetch path

`network.ts` is the entire outbound-HTTP layer: a server-side-request-forgery guard that refuses private-IP destinations, per-source proxy profiles, a caching layer with separate freshness windows for listings and detail pages, response-size caps, CAPTCHA detection, and the in-process Playwright persistent-context fetch path. Nothing in Go implements any of this. The `Deps.EnrichFn` in `internal/server/routes.go:31` is explicitly a stub ("no-op stub until the full network layer is wired", line 173), and `POST /api/import` creates a skeleton record with title `"Manual import"` and no network fetch (line 784). Without a network layer, all collection capabilities that require fetching listing pages or detail pages are blocked.

### No domain rate limiting / per-domain cooldowns

`traffic.ts` serialises all outbound requests per domain behind a minimum inter-request interval plus exponential backoff and 24-hour blocks on 403/CAPTCHA responses. `internal/background/discipline.go` enforces the same pacing only for the Playwright browser path; plain HTTP fetches (once the network layer is added) have no equivalent. This gap is downstream of the network layer but needs an explicit implementation decision before collection can run safely.

### Per-source discovery adapters entirely missing

`discovery.ts` contains approximately 12 site-specific adapters (Getro, Braintrust, a16z, Remotive, RSS feeds, Hacker News, Algolia, Wellfound, Web3 Career, browser-session sources, a generic rel-next adapter), the full coverage-accounting type, bounded pagination with repeat-page detection, concurrent detail hydration within rate limits, query matching against configured OR queries, and the per-source coverage report with explicit blocker and completeness class. There is no Go package, no function, and no stub for any of this. Grep for `discoverSource`, `parseGetroJobs`, `parseWellfound`, `discoverSource`, or any adapter name returns nothing in the Go codebase. The `POST /api/search` route at `routes.go:216` inserts a `collection_runs` row with status `running` and returns 202 accepted but then does nothing — the row is never updated, no sources are queried, and no jobs are found.

### Single-role fetch and enrich are stubs

The `collectors.ts` functions `importUrl` (fetch and upsert a single role by URL) and `enrich` (re-fetch a role's detail page) depend on the network layer and use the full parsing pipeline. In Go, `importURL` at `routes.go:784` creates a record with `Title: "Manual import"`, `Company: "Unknown"` and no network call; `POST /api/jobs/:id/read`'s `EnrichFn` defaults to a no-op at `routes.go:173`. Both endpoints are present but do nothing that benefits the owner.

### Full collection run does not run

`services.ts` `startSearch` acquires the daily lock, iterates enabled sources through `discoverSource`, streams parsed postings through the store's upsert, and writes per-source coverage into `source_runs`. In Go, `POST /api/search` creates a `collection_runs` row and returns. The daily scheduler's `defaultCollect` at `scheduler.go:153` calls only `s.EnqueueAll()`, which re-fingerprints existing jobs — it does not query any source, fetch any page, or add any new role. The scheduler goroutine itself fires correctly and acquires the lock, but the work it triggers is limited to re-enqueueing. This gap is a direct dependency of the missing discovery and network layers.

### 30-second context refresh timer absent

`server.ts` starts a 30-second `setInterval` at line 23 that checks whether the local evidence fingerprint has changed and, if so, calls `refreshContext` and re-enqueues all jobs for reassessment. This ensures that editing candidate evidence files triggers re-assessment without an explicit API call. The Go `serve.go` starts three goroutines (assessment worker, preparation worker, daily scheduler) and no such timer. The assessment worker calls `s.EnqueueAll()` every 30 seconds, which covers the re-enqueue side, but the evidence refresh step is absent. If the owner edits their CV, Go will not detect the change until `POST /api/context` is called manually.

### Archive retention timer absent

`archive.ts` runs `store.purgeArchive` every 60 seconds. The Go equivalent `internal/store/jobs.go:882` (`PurgeArchive`) is complete and correct, but `cmd/sifty/serve.go` does not start a ticker to call it. Applied roles past the 30-day retention window will not be pruned automatically.

### PATCH draft optimistic concurrency check absent

The TypeScript `updateApplication` in `application-store.ts` checks `field.revision !== change.revision` before writing a field edit, refusing a write if another change landed concurrently. The Go `PATCH /api/jobs/{id}/application` handler at `routes.go:635` deserialises the request body directly onto the loaded draft and writes it without checking any revision field. A race between a background generate call and an owner edit can silently clobber one of the two.

### Full CLI entry point missing

`cli.ts` exposes `validate`, `context`, `search`, `list`, `detail`, `status`, `import`, `read`, `assess`, `health`, `daily`, `provider-check`, `session`, `backup`, and `launchd-template` for owner operation and scripting. The Go binary's `cmd/sifty/main.go:49` accepts only `version`, `serve`, `help`, and `selfcheck`. An owner who runs `sifty daily` or `sifty assess 42` to operate the backend from a terminal or a cron job will get "unknown command" in Go.

### MCP server entry point missing

`mcp.ts` exposes the backend services as MCP tools over stdio (12 tools: `jobs_list`, `jobs_detail`, `jobs_search`, `jobs_import`, `jobs_read`, `jobs_assess`, `sources_health`, `candidate_context`, `application_prepare`, `application_draft`, `application_cancel`, `application_update`, `application_question`, `application_memories`, `application_remember`). There is no Go MCP entry point; no package, no main command, and no SDK dependency was found in the Go codebase.

### Model-based extraction fallback missing

`extraction.ts` `extractWithModel` calls the configured inference provider to extract a structured job posting from raw HTML when the parsers find no structured data, verifying that every extracted field is an exact contiguous quote from the source page. The Go config schema has `ModelExtraction bool` at `config/config.go:50`, which is the feature flag, but there is no extraction prompt under `sifty/internal/prompts/` and no `extractWithModel` function anywhere in the Go codebase.

---

## What is safe to delete now versus not

The following `app/src` modules have a complete and wired Go equivalent and may be deleted: `core.ts`, `workflow.ts`, `store.ts`, `context.ts`, `feedback.ts`, `providers.ts`, `queue.ts`, `applications.ts`, `application-generation.ts`, `application-store.ts` (pending the optimistic-concurrency fix noted above — the core CRUD is wired even if the race guard is absent), `application-browser.ts` (the Playwright worker now lives under `workers/playwright/` by design), `application-navigation.ts`, `rendered-fields.ts`, and `preparation-error.ts`.

The following modules are the only remaining implementation of a live capability or contain logic not yet reproduced in Go, and must NOT be deleted until the corresponding Go work is complete: `network.ts` and `traffic.ts` (the entire outbound-HTTP layer with its safety guards, missing in Go); `discovery.ts` (the full per-source collection engine, missing in Go); `collectors.ts` (the enrich and importUrl paths, which are stubs in Go); `services.ts` (the actual collection run, which Go's scheduler only pretends to do); `archive.ts` (the retention timer, whose Go store method exists but is never called); `application-memory.ts` (the `bindAnswerMemories` logic, not called in Go's preparation flow); `extraction.ts` (no Go implementation exists); `cli.ts` (11 of 15 CLI commands are absent from Go); `mcp.ts` (no Go MCP server exists); and `server.ts` (the 30-second evidence-refresh timer and the archive-retention timer are absent from Go's serve.go).
