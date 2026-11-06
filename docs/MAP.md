# Sifty — current product and architecture map

<a id="overview"></a>
## Overview

This is the current entry point for product requirements, implementation areas, checks, decisions and task cards. It is written from the repository as it exists; it is not a build plan and it does not replace the canonical cards under `../work/tasks/`.

`product/MAP.md` is a **historical** snapshot from 2026-11-01. Its private-repository, MIT-licence, `app/` and `ai/` assumptions are not current. The current licence is [Elastic License 2.0](../LICENSE); current records are under `../work/`, and current source is `frontend/`, `backend/`, `mcp/` and the retained `legacy/app/`. See [docs/README.md](README.md).

Status vocabulary:

- **implemented** — code, tests and the current entry point exist.
- **limited** — usable, with the named gap recorded in a card.
- **known broken** — a concrete failure is recorded and must not be presented as working.
- **planned** — a card exists; implementation has not started.
- **archived** — cancelled or descoped, kept as evidence.

<a id="code-areas"></a>
## Where the current implementation lives

| Area | Current location | Notes |
|---|---|---|
| Frontend | [frontend/](../frontend/main.tsx), [package.json](../frontend/package.json) | React/Vite single page; built to `frontend/dist`. |
| Go backend and API | [backend/cmd/sifty](../backend/cmd/sifty/main.go), [routes.go](../backend/internal/server/routes.go) | One `sifty` binary, loopback HTTP API, static frontend serving. |
| Background work | [backend/internal/background](../backend/internal/background/collect.go) | Collection trigger, scheduler, assessment worker, preparation worker. |
| Storage | [backend/internal/store](../backend/internal/store/store.go), [data directory](architecture/data-directory.md) | SQLite store plus durable files under one configured data directory. |
| Workers | [playwright worker](../backend/workers/playwright/index.mjs), [jobspy worker](../backend/workers/jobspy/index.mjs), [protocol](architecture/worker-protocol.md) | TypeScript sidecars invoked by the Go core. |
| Checks | [CI](../.github/workflows/ci.yml), [backend test entry](../backend/scripts/test), [harness test entry](../bin/work-test) | Go vet/build/race tests, frontend typecheck/build, worker tests, harness and leak checks. |
| Decisions | [Go port](../work/decisions/sifty-go-port.md), [application preparation](../work/decisions/application-preparation.md) | Append-only "why" records. |
| Work records | [roadmap](../work/ROADMAP.md), [workflow](../work/WORKFLOW.md), [task cards](../work/tasks/) | Canonical statuses, dependencies and evidence. |
| Future MCP adapter | [mcp/README.md](../mcp/README.md) | Placeholder; implementation belongs to `GO-9`. |
| Retained legacy implementation | [legacy/app/README.md](../legacy/app/README.md) | Not current product code; removal is gated by `GO-5`. |

<a id="requirements"></a>
## Requirements and behavior

Each requirement names its user outcome, the current implementation, the checks that exercise it, and the tasks that own the remaining work. A task may appear under more than one requirement when it crosses a boundary.

<a id="req-set"></a>
### R-SET — Configure the local instance

**Status:** implemented, with one blocked setup area.

**User outcome.** The owner chooses sources, search terms, exclusions, models and candidate evidence from the application instead of editing files by hand.

**Frontend and API.** [Collection.tsx](../frontend/Collection.tsx) owns search and source editing; [Settings.tsx](../frontend/Settings.tsx) owns context, configuration and queue; routes are in [routes.go](../backend/internal/server/routes.go) and config loading is in [backend/internal/config](../backend/internal/config/config.go).

**Checks.** [config save tests](../backend/internal/config/save_test.go), [live reload tests](../backend/internal/server/config_reload_test.go), [search/site endpoint tests](../backend/internal/server/search_endpoints_test.go), and the frontend build in [package.json](../frontend/package.json).

**Tasks and gaps.** `SET-2`, `SET-3`, `SET-4`, `NAVIGATION-2`, `NAVIGATION-3`, `OPS-7`. The first-time knowledge-base answer setup (`SET-1`) is blocked on owner questions; `APP-3` depends on it.

<a id="req-col"></a>
### R-COL — Collect vacancies from configured sources

**Status:** implemented, with a known observability gap.

**User outcome.** Opening the application starts a throttled collection; manual and per-source buttons force a run; sources are fetched through a guarded HTTP layer and deterministic parsers.

**Backend and workers.** [collect.go](../backend/internal/background/collect.go), [trigger.go](../backend/internal/background/trigger.go) and [scheduler.go](../backend/internal/background/scheduler.go); adapters in [backend/internal/source](../backend/internal/source/source.go); fetch safety and pacing in [backend/internal/fetch](../backend/internal/fetch/client.go); jobspy runs as a [worker](../backend/workers/jobspy/index.mjs).

**Frontend.** [Collection.tsx](../frontend/Collection.tsx) and [Sources.tsx](../frontend/Sources.tsx).

**Storage.** Collection runs, source runs, events and health are tables in the store; the current reader is [store/queue.go](../backend/internal/store/queue.go).

**Checks.** [collection tests](../backend/internal/background/collect_test.go), [trigger tests](../backend/internal/background/trigger_test.go), [scheduler tests](../backend/internal/background/scheduler_test.go), [source tests](../backend/internal/source/source_test.go) and [fetch tests](../backend/internal/fetch/client_test.go).

**Tasks and gaps.** `COL-1`, `COL-3` and `COL-6` are done. `COL-4` is the known gap: the dispatch does not write per-source `source_runs`/`collection_events`, so `GET /api/collection` reports an empty detail even when a run collected roles. `COL-2` (learned template cache) and `COL-5` (description structure) are planned.

<a id="req-search"></a>
### R-SEARCH — Search, filter and rank roles

**Status:** implemented.

**User outcome.** Roles are filtered and ranked immediately: OR terms, AND terms and exclusions combine in the FTS query, and a BM25 match score orders the catalogue.

**Frontend.** [Collection.tsx](../frontend/Collection.tsx) (search editor), [main.tsx](../frontend/main.tsx) and [CatalogueFilters.tsx](../frontend/CatalogueFilters.tsx).

**Backend and storage.** [fts_query.go](../backend/internal/store/fts_query.go), [rescore.go](../backend/internal/store/rescore.go), [stopwords.go](../backend/internal/store/stopwords.go), [blocks.go](../backend/internal/store/blocks.go), [search_yaml.go](../backend/internal/server/search_yaml.go).

**Checks.** [fts query tests](../backend/internal/store/fts_query_test.go), [rescore tests](../backend/internal/store/rescore_test.go), [stop-word tests](../backend/internal/store/stopwords_test.go), [block tests](../backend/internal/store/blocks_test.go), [search endpoint tests](../backend/internal/server/search_endpoints_test.go).

**Tasks.** `NAVIGATION-2`, `NAVIGATION-3`, `NAVIGATION-4`, `ASM-4`, `COL-6`, `TRI-1`, `TRI-2`. `COL-4` is the separate collection-detail gap, not a search gap.

<a id="req-tri"></a>
### R-TRI — Triage, dismiss and archive

**Status:** implemented, with one timer gap.

**User outcome.** The owner dismisses a role, blocks a company or exclusion term, and later finds dismissed or applied roles in the archive.

**Frontend and API.** [main.tsx](../frontend/main.tsx), [RoleContent.tsx](../frontend/RoleContent.tsx), [ApplicationConfirmation.tsx](../frontend/ApplicationConfirmation.tsx); workflow guards and archive queries in [store/core.go](../backend/internal/store/core.go) and [store/jobs.go](../backend/internal/store/jobs.go).

**Checks.** [role/store tests](../backend/internal/store/jobs_test.go), [preselection tests](../backend/internal/server/preselect_test.go), [block tests](../backend/internal/store/blocks_test.go).

**Tasks and gaps.** `NAVIGATION-1`, `NAVIGATION-4`, `TRI-1`, `TRI-2`. The archive-retention method exists but no ticker calls it; `GO-7` owns that wiring.

<a id="req-asm"></a>
### R-ASM — Assess fit

**Status:** implemented, with deferred model-retry work.

**User outcome.** Every role has an instant match score; moving a role to Preparation or pressing "Assess now" runs the grounded LLM assessment against versioned candidate evidence.

**Backend and storage.** [assess](../backend/internal/assess/assess.go), [assessment worker](../backend/internal/background/assessment.go), [queue](../backend/internal/store/queue.go), [rescore](../backend/internal/store/rescore.go).

**Frontend.** [RoleContent.tsx](../frontend/RoleContent.tsx) and [Settings.tsx](../frontend/Settings.tsx).

**Checks.** [assessment tests](../backend/internal/assess/assess_test.go), [queue tests](../backend/internal/store/queue_test.go), [worker tests](../backend/internal/background/assessment_test.go), [ranking tests](../backend/internal/store/rescore_test.go).

**Tasks and gaps.** `ASM-2`, `ASM-3` and `ASM-4` are done. `ASM-1` (re-run stubborn failures on a chosen larger model) is deferred; automatic retry on the same model already exists.

<a id="req-app"></a>
### R-APP — Prepare application answers and apply manually

**Status:** limited; the form-inspection worker path is known broken.

**User outcome.** For a role in Preparation, Sift reads the application form, shows its questions, drafts editable answers from candidate evidence, and opens the employer page for the owner. Sift never fills or submits the form. The owner confirms manually when they have submitted; only that confirmation moves the role to Applied.

**Implemented.** Inspection wiring and honest states (`APP-1`, `APP-2`, `APP-4`, `APP-5`), draft editing and remembering, and the manual confirmation dialog. Code: [preparation.go](../backend/internal/background/preparation.go), [inspect.go](../backend/internal/background/inspect.go), [discipline.go](../backend/internal/background/discipline.go), [prepare](../backend/internal/prepare/prepare.go), [application store](../backend/internal/store/application.go), [ApplicationDraft.tsx](../frontend/ApplicationDraft.tsx), [ApplicationConfirmation.tsx](../frontend/ApplicationConfirmation.tsx), [RoleContent.tsx](../frontend/RoleContent.tsx).

**Known broken.** The roadmap records the Playwright form-inspection worker crashing on startup with a Node module-loader error, surfaced as `worker inspectForm: exit status 1`. Treat preparation as not working end to end until that is repaired and verified. The worker code and its tests live in [backend/workers/playwright](../backend/workers/playwright/index.mjs).

**Known partial parity.** Remembered answers are not injected into the preparation context (`APP-6`), and `PATCH /api/jobs/{id}/application` merges without an optimistic revision check (`APP-7`). `APP-3` (knowledge-base answers with a think modal) is blocked on `SET-1`; `APP-8` (negative confirmation outcomes) is backlog.

**Checks.** [preparation tests](../backend/internal/background/preparation_test.go), [inspection tests](../backend/internal/background/preparation_inspect_test.go), [inspect tests](../backend/internal/inspect/inspect_test.go), [prepare tests](../backend/internal/prepare/prepare_test.go), [application store tests](../backend/internal/store/application_test.go), [worker tests](../backend/workers/playwright/automation.test.mjs).

**Decisions.** [Application preparation decisions](../work/decisions/application-preparation.md).

**Tasks.** `APP-1`–`APP-8`, `SET-1`.

<a id="req-data"></a>
### R-DATA — Own, back up and migrate local data

**Status:** limited; provenance and recovery are documented, separation is proposed only.

**User outcome.** The owner can see which data is user-provided, user-derived, vacancy/assessment state or application state, and can back up and restore it without losing evidence.

**Current implementation.** One data directory resolved by [dirs.go](../backend/internal/dirs/dirs.go); SQLite store in [store](../backend/internal/store/store.go); file-first answer and feedback records in [records](../backend/internal/records/records.go); config saves keep backups in [config/save.go](../backend/internal/config/save.go). Deployment backups and rollback are in [sift-deploy](../bin/sift-deploy).

**Documentation.** [Data directory and provenance](architecture/data-directory.md). The four-class separation is **PROPOSED**, not implemented.

**Checks.** [records tests](../backend/internal/records/records_test.go), [store schema tests](../backend/internal/store/store_test.go), [config save tests](../backend/internal/config/save_test.go).

**Tasks.** `OPS-4` step 5.3 documents the current state. `OPS-8` owns the live-reference audit, consistency, backup, restore/rollback and owner-approved migration. `OPS-7` owns centralized path configuration.

<a id="req-rel"></a>
### R-REL — Develop, preview, accept and release

**Status:** limited; the release loop is still manual.

**User outcome.** Work is checked in an isolated worktree, previewed on staging, and reaches production only on the owner's explicit word.

**Current implementation.** [sift-deploy](../bin/sift-deploy) builds and deploys staging or production with per-environment binary, bundle, data and rollback; [sift-promote](../bin/sift-promote) is the only production command; [sift-accept](../bin/sift-accept) runs acceptance checks; [work-finish](../bin/work-finish) merges without deploying. The current loop is documented in [AGENTS.md](../AGENTS.md).

**Known limitation.** Continuous staging preview, source watching and the owner-approved release automation remain `OPS-4` phase 6. Staging still requires an explicit `bin/sift-deploy --env staging`; workers are shared.

**Checks.** [deploy/accept lifecycle tests](../bin/tests/test_lifecycle.py), [accept tests](../bin/tests/test_sift_accept.py), [CI](../.github/workflows/ci.yml).

**Tasks.** `OPS-1` and `OPS-3` are done; `OPS-4` phases 6–7 are open. `OPS-5` (staging idle CPU) and `OPS-6` (Git history review) are separate backlog items.

<a id="req-harness"></a>
### R-HARNESS — Capture and navigate work

**Status:** implemented.

**User outcome.** A bug or idea is recorded without starting work; cards have four normal states; the coordinator can see the exact next step and active worktree.

**Implementation.** [work commands](../bin/work-new), [shared library](../bin/_work_lib.py), [templates](../bin/templates/convention.md), [workflow](../work/WORKFLOW.md), [AGENTS.md](../AGENTS.md).

**Checks.** [harness entry](../bin/work-test), [lifecycle tests](../bin/tests/test_lifecycle.py), [navigation tests](../bin/tests/test_navigation.py), [capture tests](../bin/tests/test_capture.py), [state tests](../bin/tests/test_task_states.py).

**Tasks.** `OPS-4` phases 1–4; `OPS-6` is a separate history-maintenance idea.

<a id="req-platform"></a>
### R-PLATFORM — Maintain the Go application and repository foundation

**Status:** implemented through the current structural move; release automation is separate.

**User outcome.** The application is a single Go binary plus a React frontend and TypeScript workers, with one license, one repository layout and checked public-bound content.

**Implementation.** [Go binary](../backend/cmd/sifty/main.go), [frontend](../frontend/main.tsx), [workers](../backend/workers/playwright/index.mjs), [license](../LICENSE), [leak scanner](../backend/scripts/leakcheck/census.md), [CI](../.github/workflows/ci.yml).

**Checks.** Go vet/build/race tests, frontend locked build, worker tests, leak scanner self-test and tree sweep in [CI](../.github/workflows/ci.yml).

**Tasks.** `GO-1`, `GO-2`, `GO-3`, `GO-4`, `GO-10`, `GO-11` are done. `OPS-5` records a staging CPU report.

<a id="req-mcp"></a>
### R-MCP — Expose Sifty over MCP

**Status:** planned.

**User outcome.** An external assistant can drive the same backend through a standalone MCP adapter.

**Implementation.** No implementation yet; [mcp/README.md](../mcp/README.md) records the intended TypeScript HTTP adapter.

**Tasks and dependencies.** `GO-9`; it depends on the Go HTTP API and on collection/assessment/preparation being usable. `GO-8` (general CLI parity) is cancelled and is not a dependency.

<a id="req-pkg"></a>
### R-PKG — Self-host packaging and documentation

**Status:** planned; the license is already implemented.

**User outcome.** A technical user can clone and run the complete product with Docker Compose or a documented manual setup.

**Implementation.** [README](../backend/README.md), [license](../LICENSE), [deployment scripts](../bin/sift-deploy); no container/compose packaging yet.

**Tasks.** `GO-6`; `GO-10` is done.

<a id="req-legacy"></a>
### R-LEGACY — Retire the legacy TypeScript implementation safely

**Status:** planned; retained code is not current product code.

**User outcome.** The old backend is removed only when parity is proven, so no working capability disappears.

**Implementation.** [legacy/app/README.md](../legacy/app/README.md), [parity audit](architecture/go-parity-audit.md).

**Tasks.** `GO-5`; `GO-8` is cancelled (owner does not use a CLI/cron). `GO-7` and `GO-9` are part of the parity gate; `APP-6`/`APP-7` are recorded gaps.

<a id="req-design"></a>
### R-DESIGN — Make the visual layer editable

**Status:** planned.

**User outcome.** Colours, spacing, radii and typography live in role-named tokens so a later design pass edits one layer instead of scattered values.

**Implementation.** [frontend CSS](../frontend/style.css) and component styles; no token layer yet.

**Tasks.** `DESIGN-1`; it depends on the frontend move (`GO-11`, done) and now targets the current `frontend/` location.

<a id="validation"></a>
## What is validated where

- `bin/map-check` validates local links and fragments in the active navigation documents, requirement references and the canonical task index.
- The canonical task index below must contain every current card ID exactly once; legacy single-file cards are listed separately.
- Archived historical prose is outside the validation scope. Old paths in `docs/product/`, `docs/architecture/go-parity-audit.md`, `legacy/app/` and historical card logs are retained for attribution and are not rewritten.
- `bin/work-test` runs the map checks and the isolated map tests as part of the harness entry used by CI.
