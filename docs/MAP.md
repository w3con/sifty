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

**Jump to:** [code areas](#code-areas) · [requirements](#requirements) · [epics](#epics) · [canonical task index](#task-index) · [legacy cards](#legacy-cards) · [validation](#validation)

## Persistent behavior specifications

These documents describe enduring product behavior, not task execution plans. Proposed policy, agreed behavior, implementation and release are distinguished inside each specification. New changes start by reading and updating the affected specification after agreement; task cards retain steps, costs and evidence.

| Specification | Requirement | Current state |
|---|---|---|
| [Quick Match](specs/quick-match.md) | R-ASM, R-SEARCH, R-SET | Implemented in main (`58af432`) with v1 uncalibrated defaults: contract, extractor, pure evaluator, persistence/API and catalogue/Overview rendering. Normal persisted actual-profile adoption is proven on an isolated snapshot: 2271 visible jobs with numeric results at profile revision20 (1539 evaluated zero, 732 nonzero, 0 stale). The temporary 8083 review server is stopped. Released to production on 2026-11-10 from pinned main `157fe9f` with the confirmed staging profile selectively transferred and 2777 persisted numeric results; technical acceptance stands, card review and owner ranking-quality review remain. The dual-fit and targeted-extraction direction is in specification section 13; its coordinator-written starting policy (section 13.7) is owner-approved for implementation but not implemented, v1 remains current, and no paid call has run. Section 14 (2026-11-11, two fit scores Fits you / You fit with model-extracted requirements; card [ASSESSMENT-3](../work/tasks/assessment/ASSESSMENT-3_two-fit-scores/description.md)) supersedes section 13.7 after the owner rejected ASSESSMENT-2; implemented in main and released to production on 2026-11-11 (steps Q.1–Q.7 and Q.9): Fits you and a requirement list from model extraction, the worker and its order toggle, ✓ / ✕ on skill lines, "Skills you do not have". The model's matching pass is off by default and You fit is not shown until a matching method is chosen (section 14.14); matching quality moves to ASSESSMENT-5, the shortlist re-check to ASSESSMENT-6, vacancy sections to INBOX-2. |
| [Listing view](specs/listing-view.md) | R-COL, R-SEARCH | Implemented and released in INBOX-2 (main `00aafb5`, production 8080, 2026-11-13). `description_html` (sanitised HTML subset), `job_originals` (raw body, never rendered), `job_sections` (heading, class, content) stored with every vacancy; backfill on startup for existing vacancies. Single-job `GET /api/jobs/:id` returns `descriptionHtml` and `sections`; list endpoint omits sections. Frontend: Original mode renders structured HTML; By section mode shows collapsible `<details>` blocks; toggle disabled when fewer than two headed sections. Plain-text fallback when `description_html` absent. |
| [Profiles](specs/profiles.md) | R-SET, R-SEARCH, R-TRI | Agreed with the owner on 2026-11-12, not implemented: My data shared, profiles with their own goals, search, Fits you, Inbox/Shortlist and decisions; search moves from Settings to the profile; SETTINGS-9 absorbed into S.1 (owner, 2026-11-13); card [SETTINGS-5](../work/tasks/settings/SETTINGS-5_user-data-and-role-profiles/plan.md), starts after INBOX-2. Spec updated OPS-11. |
| [Vacancy view](specs/vacancy-view.md) | R-SEARCH, R-COL | Implemented and released in INBOX-2 (main `00aafb5`, production 8080, 2026-11-13). Fits you band (Money/Location/Work mode/Contract tiles with verdicts from merged facts), What they ask (model requirements grouped Required/Nice to have), What they offer (tiles from Compensation/Benefits sections). Full listing open with no By section / Original switch. |
| [Collection](specs/collection.md) | R-COL | Owner decisions of 2026-11-13 applied. 51 rules: 36 implemented, 10 not implemented (INBOX-2, COL-2, COL-4, SETTINGS-5, 4 no-card items), 5 deviate. Cards: COL-2, COL-4, COLLECTION-1, COLLECTION-2, COLLECTION-3, SETTINGS-6. |
| [Catalogue and search](specs/catalogue.md) | R-COL, R-SEARCH | Owner decisions of 2026-11-13 applied. 24 rules: 15 implemented, 3 implemented with known limits, 2 not implemented (INBOX-2), 1 not implemented (SETTINGS-5), 3 deviate. Cards: SETTINGS-5, INBOX-2, INBOX-1. |
| [Triage and decisions](specs/triage.md) | R-TRI | Owner decisions of 2026-11-13 applied. 38 rules: 30 implemented, 5 deviate, 3 not implemented (no card), 0 open questions. Cards: GO-7, TRIAGE-1, SETTINGS-5, ASSESSMENT-6, APP-8 follow-up. |
| [Application preparation](specs/application.md) | R-APP | NOT DESIGNED. Owner decisions of 2026-11-13 applied. 21 rules: 7 implemented, 2 deviate, 10 not implemented, 2 owner decision needed (ASSESSMENT-6, re-run model). Design questions deferred to a design session. Cards: APP-3, APP-4, APP-5, APP-6, APP-7, APP-8, SET-1, ASM-1, ASSESSMENT-6. |
| [Local data](specs/data.md) | R-DATA | Owner decisions of 2026-11-13 applied. 27 rules: 20 implemented, 4 not implemented (OPS-7, OPS-8, GO-6, documented procedure), 2 not implemented (new owner decisions: server-side snapshot coverage, move procedure), 1 pointer to triage. Cards: OPS-7, OPS-8, GO-6. |

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

**Tasks and gaps.** `SET-2`, `SET-3`, `SET-4`, `NAVIGATION-2`, `NAVIGATION-3`, `OPS-7`. SETTINGS-2 workflow and SETTINGS-3 fields are technically accepted; their source and DESIGN-3 layout/shared blocks/uploads/persistent avatar are in main (`564f334`/`7f80bbf`/`c827cee`/`5bedae3`, final `58af432`), approximately28–30min for the DESIGN-3 technical stage against corrected25–45min. Cards remain review; functional owner acceptance and release remain separate, and production is not updated. Continent-level preferred locations captured but not implemented. SETTINGS-4 URL inputs and ASM-5 shared-term matching remain unapproved. Real profile.pdf/cv.pdf analyses produced4/6facts and owner fact acceptance works, but repeated website timeouts and discarded malformed/ungrounded proposals remain; new evidence is in SETTINGS-2. The first-time knowledge-base answer setup (`SET-1`) is blocked on owner questions; `APP-3` depends on it.

<a id="req-col"></a>
### R-COL — Collect vacancies from configured sources

**Status:** implemented, with a known observability gap. Behaviour is specified in [Collection](specs/collection.md).

**Backend and workers.** [collect.go](../backend/internal/background/collect.go), [trigger.go](../backend/internal/background/trigger.go) and [scheduler.go](../backend/internal/background/scheduler.go); adapters in [backend/internal/source](../backend/internal/source/source.go); fetch safety and pacing in [backend/internal/fetch](../backend/internal/fetch/client.go); jobspy runs as a [worker](../backend/workers/jobspy/index.mjs).

**Frontend.** [Collection.tsx](../frontend/Collection.tsx) and [Sources.tsx](../frontend/Sources.tsx).

**Storage.** Collection runs, source runs, events and health are tables in the store; the current reader is [store/queue.go](../backend/internal/store/queue.go).

**Checks.** [collection tests](../backend/internal/background/collect_test.go), [trigger tests](../backend/internal/background/trigger_test.go), [scheduler tests](../backend/internal/background/scheduler_test.go), [source tests](../backend/internal/source/source_test.go) and [fetch tests](../backend/internal/fetch/client_test.go).

**Tasks and gaps.** `COL-1`, `COL-3` and `COL-6` are done. `COLLECTION-1` is technically accepted through d92112c (records2b8a1d5), review8083: source publication basis, date/completed age, Newest known/legacy/missing tiers, existing-job date fill/preservation, related-item/date semantics; schema18. Legacy dates without origin remain explicitly unverified rather than retrospectively certified. Owner review/integration/release pending. Automatic age scoring/cutoff and detailed assessment changes deferred. `COL-4` is the known gap: the dispatch does not write per-source `source_runs`/`collection_events`, so `GET /api/collection` reports an empty detail even when a run collected roles. `COL-2` (learned template cache) is now the home of per-source parsing rules and their repair (spec quick-match §15.3); `COL-5` was cancelled as a duplicate of INBOX-2.

Latest owner scope is the minimal manual review loop (INBOX-1), with negative Apply outcomes (APP-8) and prior-application duplicate recognition (TRIAGE-1). Owner confirmed no incoming stage: all collected jobs go to Triage after filters; retain collection on App load with cooldown. APP-8 product outcomes/button-only modal are implemented in main and remain review. TRIAGE-1 examples expose divergent production/staging application histories for the same canonical Coinbag/Uniswap listings; authority/reconciliation is not decided, no data merged. Freshness supports this loop; no new scoring engine approved.

<a id="req-search"></a>
### R-SEARCH — Search, filter and rank roles

**Status:** implemented. Catalogue and search behaviour is specified in [Catalogue and search](specs/catalogue.md).

**Frontend.** [Collection.tsx](../frontend/Collection.tsx) (search editor), [main.tsx](../frontend/main.tsx) and [CatalogueFilters.tsx](../frontend/CatalogueFilters.tsx).

**Backend and storage.** [fts_query.go](../backend/internal/store/fts_query.go), [rescore.go](../backend/internal/store/rescore.go), [stopwords.go](../backend/internal/store/stopwords.go), [blocks.go](../backend/internal/store/blocks.go), [search_yaml.go](../backend/internal/server/search_yaml.go).

**Checks.** [fts query tests](../backend/internal/store/fts_query_test.go), [rescore tests](../backend/internal/store/rescore_test.go), [stop-word tests](../backend/internal/store/stopwords_test.go), [block tests](../backend/internal/store/blocks_test.go), [search endpoint tests](../backend/internal/server/search_endpoints_test.go).

**Tasks.** `NAVIGATION-2`, `NAVIGATION-3`, `NAVIGATION-4`, `ASM-4`, `COL-6`, `TRI-1`, `TRI-2`. `COL-4` is the separate collection-detail gap, not a search gap.

<a id="req-tri"></a>
### R-TRI — Triage, dismiss and archive

**Status:** implemented, with one timer gap. Behaviour is specified in [Triage and decisions](specs/triage.md). The canonical screen names are Inbox, Shortlist and Archive (owner, 2026-11-13).

**Frontend and API.** [main.tsx](../frontend/main.tsx), [RoleContent.tsx](../frontend/RoleContent.tsx), [ApplicationConfirmation.tsx](../frontend/ApplicationConfirmation.tsx); workflow guards and archive queries in [store/core.go](../backend/internal/store/core.go) and [store/jobs.go](../backend/internal/store/jobs.go).

**Checks.** [role/store tests](../backend/internal/store/jobs_test.go), [preselection tests](../backend/internal/server/preselect_test.go), [block tests](../backend/internal/store/blocks_test.go).

**Tasks and gaps.** `NAVIGATION-1`, `NAVIGATION-4`, `TRI-1`, `TRI-2`, `TRIAGE-2`. The archive-retention method exists but no ticker calls it; `GO-7` owns that wiring.

<a id="req-asm"></a>
### R-ASM — Assess fit

**Status:** implemented, with deferred model-retry work.

**User outcome.** Every role has an instant match score; moving a role to Preparation or pressing "Assess now" runs the grounded LLM assessment against versioned candidate evidence.

**Backend and storage.** [assess](../backend/internal/assess/assess.go), [assessment worker](../backend/internal/background/assessment.go), [queue](../backend/internal/store/queue.go), [rescore](../backend/internal/store/rescore.go).

**Frontend.** [RoleContent.tsx](../frontend/RoleContent.tsx) and [Settings.tsx](../frontend/Settings.tsx).

**Checks.** [assessment tests](../backend/internal/assess/assess_test.go), [queue tests](../backend/internal/store/queue_test.go), [worker tests](../backend/internal/background/assessment_test.go), [ranking tests](../backend/internal/store/rescore_test.go).

**Tasks and gaps.** `ASM-2`, `ASM-3` and `ASM-4` are done. `ASM-1` (re-run stubborn failures on a chosen larger model) is deferred; automatic retry on the same model already exists. [Publication date and vacancy age](../work/tasks/collection/COLLECTION-1_publication-date-and-age/description.md) records the owner's request to show actual publication separately from discovery and account for age in evaluation; rule/source coverage remain to be prepared.

**Current behavior:** [Quick Match specification](specs/quick-match.md) separates normalized vacancy facts, replaceable extraction and pure scoring modules. ASM-6 is implemented in main (`58af432`) and released to production from pinned `157fe9f` on 2026-11-10, replacing corpus-relative BM25 fit ordering with a deterministic no-LLM score; FTS search remains separate. Card review and owner ranking-quality review remain. Missing salary does not become a zero, partial scores carry evidence coverage, and zero is distinct from unscored. The v1 numerical defaults were approved on 2026-11-10 and remain uncalibrated; user calibration is a separate deferred card, ASSESSMENT-1, with no weight-editing UI in v1. Implemented v1 limitations include narrow explicit headings, no inferred years or semantic matching, no currency/tax inference, and unsupported regions staying unknown. The existing model queue is not selected-only and is not changed by this specification. The follow-up dual-fit and targeted-extraction direction is in section 13; its starting policy (section 13.7) is owner-approved for implementation but not implemented, v1 remains current, and no paid call has run. Section 14 (two fit scores, ASSESSMENT-3) is implemented in main and released to production on 2026-11-11; while the model's matching pass is off (the default, section 14.14) only Fits you and the requirement list are shown.

<a id="req-app"></a>
### R-APP — Prepare application answers and apply manually

**Status:** limited; the feature is not designed and the form-inspection worker path is known broken. Behaviour is specified in [Application preparation](specs/application.md).

**Implemented.** Inspection wiring and honest states (`APP-1`, `APP-2`, `APP-4`, `APP-5`), draft editing and remembering, and the manual confirmation dialog. Code: [preparation.go](../backend/internal/background/preparation.go), [inspect.go](../backend/internal/background/inspect.go), [discipline.go](../backend/internal/background/discipline.go), [prepare](../backend/internal/prepare/prepare.go), [application store](../backend/internal/store/application.go), [ApplicationDraft.tsx](../frontend/ApplicationDraft.tsx), [ApplicationConfirmation.tsx](../frontend/ApplicationConfirmation.tsx), [RoleContent.tsx](../frontend/RoleContent.tsx).

**Known broken.** The roadmap records the Playwright form-inspection worker crashing on startup with a Node module-loader error, surfaced as `worker inspectForm: exit status 1`. Treat preparation as not working end to end until that is repaired and verified. The worker code and its tests live in [backend/workers/playwright](../backend/workers/playwright/index.mjs).

**Known partial parity.** Remembered answers are not injected into the preparation context (`APP-6`), and `PATCH /api/jobs/{id}/application` merges without an optimistic revision check (`APP-7`). `APP-3` (knowledge-base answers with a think modal) is blocked on `SET-1`; `APP-8` (negative confirmation outcomes) is in main and remains review, not backlog.

**Checks.** [preparation tests](../backend/internal/background/preparation_test.go), [inspection tests](../backend/internal/background/preparation_inspect_test.go), [inspect tests](../backend/internal/inspect/inspect_test.go), [prepare tests](../backend/internal/prepare/prepare_test.go), [application store tests](../backend/internal/store/application_test.go), [worker tests](../backend/workers/playwright/automation.test.mjs).

**Decisions.** [Application preparation decisions](../work/decisions/application-preparation.md).

**Tasks.** `APP-1`–`APP-8`, `SET-1`.

<a id="req-data"></a>
### R-DATA — Own, back up and migrate local data

**Status:** limited; provenance and recovery are documented, separation is proposed only. Behaviour is specified in [Local data](specs/data.md).

**Current implementation.** One data directory resolved by [dirs.go](../backend/internal/dirs/dirs.go); SQLite store in [store](../backend/internal/store/store.go); file-first answer and feedback records in [records](../backend/internal/records/records.go); config saves keep backups in [config/save.go](../backend/internal/config/save.go). Deployment backups and rollback are in [sift-deploy](../bin/sift-deploy).

**Documentation.** [Data directory and provenance](architecture/data-directory.md). The four-class separation is **PROPOSED**, not implemented.

**Checks.** [records tests](../backend/internal/records/records_test.go), [store schema tests](../backend/internal/store/store_test.go), [config save tests](../backend/internal/config/save_test.go).

**Tasks.** `OPS-4` step 5.3 documents the current state. [`OPS-8`](../work/tasks/ops/OPS-8_data-layout-migration/description.md) owns the live-reference audit, consistency, backup, restore/rollback and owner-approved migration. `OPS-7` owns centralized path configuration.

<a id="req-rel"></a>
### R-REL — Develop, preview, accept and release

**Status:** implemented through `work-finish`; the live production process still needs the explicit legacy adoption step.

**User outcome.** Work is checked in an isolated worktree, previewed on staging, and reaches production only on the owner's explicit word.

**Current implementation.** [sift-preview](../bin/sift-preview) continuously follows one active card worktree (including uncommitted and untracked sources) on the staging port, keeps the last healthy generation on build failure, exposes requested/served revision status and a browser banner, and enforces one owner per main repository. [sift-deploy](../bin/sift-deploy) builds candidates and publishes isolated production/staging binary, bundle, worker tree and data, verifies the served build identity, removes the old npm-install fallback and restores all artifacts plus consistent SQLite on failure. [sift-data-refresh](../bin/sift-data-refresh) is the explicit staging snapshot operation; normal deploys do not copy data. [sift-promote](../bin/sift-promote) is the only production command: it requires `--source` and `--approve` to resolve to the same pinned revision, then runs [sift-deploy](../bin/sift-deploy) `--source` (built with `git archive`, not a moving checkout). [sift-accept](../bin/sift-accept) runs acceptance checks. [work-finish](../bin/work-finish) prepares one squash commit (checked tree plus `done.md`), runs the configured `WORK_RELEASE_CMD` against that source, and publishes the commit only after release health succeeds; a failure writes no `done.md` and preserves main, the branch, worktree and recovery. `--adopt-legacy` is the explicit first-transition operation that verifies the old process and captures the assets it actually serves. The commands, migration path and boundaries are described in [the preview and environment runtime](architecture/preview-and-environments.md) and [AGENTS.md](../AGENTS.md).

**Known limitation.** The live production process still predates the isolated pid/path scheme. The first real release must use `bin/sift-promote --adopt-legacy` explicitly; no live migration was performed by implementing 6.3, and no `work-*` command starts a preview automatically.

**Checks.** [preview tests](../bin/tests/test_preview.py), [deploy/accept lifecycle tests](../bin/tests/test_lifecycle.py), [accept tests](../bin/tests/test_sift_accept.py), [CI](../.github/workflows/ci.yml).

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

**User outcome.** Presentation components and styles can be edited without reading application requests, persistence and workflow logic. Colours, spacing, radii and typography are separately planned as role-named tokens.

**Implementation.** [frontend CSS](../frontend/style.css) and mixed presentation/behavior in [main.tsx](../frontend/main.tsx), Collection, Settings, RoleContent and application components; presentation/logic separation is planned, not implemented. No token layer yet.

**Tasks.** `DESIGN-1` owns the token layer and depends on the frontend move (`GO-11`, done). `DESIGN-2` owns the separately requested readability and presentation/logic separation across all current frontend TSX; its complete four-stage plan and estimate were approved on 2026-11-07. Preserve the independent token worktree and reconcile future overlapping work deliberately.

<a id="epics"></a>
## Epics and outcomes

| Epic | Product outcome | Requirements |
|---|---|---|
| application-preparation | Inspect an application form, draft editable answers, and let the owner apply and confirm manually. | `R-APP` |
| assessment | Give every role an instant ranking and a grounded model assessment. | `R-ASM` |
| collection | Fetch configured public sources safely and store deduplicated roles. | `R-COL` |
| design | Make the visual layer editable from one token set. | `R-DESIGN` |
| go-migration | Maintain the Go application, package it, expose MCP and retire the legacy backend. | `R-PLATFORM`, `R-PKG`, `R-MCP`, `R-LEGACY` |
| navigation | Browse Collection, Triage, Preparation and Archive, with search and exclusions. | `R-SEARCH`, `R-TRI` |
| ops | Keep the repository, work harness, preview/release loop and local data safe. | `R-HARNESS`, `R-REL`, `R-DATA` |
| settings | Configure sources, models, context and the assessment queue from the UI. | `R-SET` |
| triage | Dismiss, exclude and archive roles without losing recoverable state. | `R-TRI` |

<a id="task-index"></a>
## Canonical task index

Every canonical card ID appears exactly once. State links point at the outcome evidence where one exists; dependencies are the explicit gate for starting or finishing the card.

| Card | Requirement | State | Depends on |
|---|---|---|---|
| [SETTINGS-1](../work/tasks/settings/SETTINGS-1_user-profile/description.md) | R-SET, R-SEARCH, R-ASM | review; technically accepted; source in main `564f334` (final `58af432`); owner usability findings continued in SETTINGS-2; spec [Profiles](specs/profiles.md) | owner acceptance/release separate; production not updated |
| [SETTINGS-2](../work/tasks/settings/SETTINGS-2_profile-workflow-and-fields/description.md) | R-SET, R-SEARCH, R-ASM | review; technically accepted; source in main `7f80bbf` (final `58af432`); owner review pending ([workflow plan](../work/tasks/settings/SETTINGS-2_profile-workflow-and-fields/plan.md)); spec [Profiles](specs/profiles.md) | real-document yield and release remain separate; production not updated |
| [SETTINGS-3](../work/tasks/settings/SETTINGS-3_profile-fields/description.md) | R-SET, R-ASM | review; original fields technically accepted; source in main `c827cee` (final `58af432`); continent-level preference captured, unimplemented (+30–50min provisional); spec [Profiles](specs/profiles.md) | owner acceptance/release separate; production not updated |
| [SETTINGS-5](../work/tasks/settings/SETTINGS-5_user-data-and-role-profiles/description.md) | R-SET, R-TRI, R-ASM, R-APP | approved 2026-11-12 (plan §10), spec [Profiles](specs/profiles.md); starts after INBOX-2; was: full User/role-profile design saved, explicitly deferred; separate profile pipelines/shared actual submissions agreed | no implementation/auth now; re-estimate and split before approval; canonical design in plan.md |
| [SETTINGS-4](../work/tasks/settings/SETTINGS-4_profile-url-sources/description.md) | R-SET | backlog ([URL plan](../work/tasks/settings/SETTINGS-4_profile-url-sources/plan.md); not approved); spec [Profiles](specs/profiles.md) §6 | SETTINGS-2; existing guarded fetch/provider |
| [ASM-5](../work/tasks/assessment/ASM-5_shared-term-matching/description.md) | R-SET, R-SEARCH, R-ASM | backlog ([shared matching plan](../work/tasks/assessment/ASM-5_shared-term-matching/plan.md); not approved) | SETTINGS-3; preserved SETTINGS-1 ranking; refines INBOX-1 |
| [INBOX-1](../work/tasks/inbox/INBOX-1_search-catalogue-overview-quick-match/description.md) | R-COL, R-SEARCH, R-ASM, R-SET, R-TRI | backlog ([proposed sequence](../work/tasks/inbox/INBOX-1_search-catalogue-overview-quick-match/plan.md); not approved) | SETTINGS-1 first; search → scoring → Overview |
| [APP-1](../work/tasks/application-preparation/APP-1_inspect-chain-at-selection/description.md) | R-APP | [done](../work/tasks/application-preparation/APP-1_inspect-chain-at-selection/done.md) | GO-2 |
| [APP-2](../work/tasks/application-preparation/APP-2_application-tab-states/description.md) | R-APP | [done](../work/tasks/application-preparation/APP-2_application-tab-states/done.md) | APP-1 |
| [APP-3](../work/tasks/application-preparation/APP-3_answers-from-kb-and-think-modal/description.md) | R-APP | backlog (blocked: SET-1); spec [Application preparation](specs/application.md) | SET-1, APP-1, APP-2 |
| [APP-4](../work/tasks/application-preparation/APP-4_robust-form-inspection/description.md) | R-APP | [done](../work/tasks/application-preparation/APP-4_robust-form-inspection/done.md) | APP-1 |
| [APP-5](../work/tasks/application-preparation/APP-5_respectful-browser-automation/description.md) | R-APP | [done](../work/tasks/application-preparation/APP-5_respectful-browser-automation/done.md) | APP-4 |
| [APP-6](../work/tasks/application-preparation/APP-6_answer-memories-in-prepare/description.md) | R-APP | backlog; spec [Application preparation](specs/application.md) | APP-1, APP-4 |
| [APP-7](../work/tasks/application-preparation/APP-7_draft-revision-check/description.md) | R-APP | backlog; spec [Application preparation](specs/application.md) | APP-1 |
| [APP-8](../work/tasks/application-preparation/APP-8_expired-application-outcome/description.md) | R-APP, R-TRI | review; owner/technical acceptance; source in main `a3ead8b` (final `58af432`); five outcomes/button-only modal; free-text explanation pending; spec [Triage](specs/triage.md), [Application preparation](specs/application.md) | release separate; production not updated; v16 migration gate corrected |
| [APP-9](../work/tasks/application-preparation/APP-9_apply-on-employer-site/description.md) | R-APP | backlog (captured 2026-11-13): stage 1 of the application path — form link at parse time, employer page lookup (ATS boards, then web search), search provider chain, Apply opens the direct link in a separate window; design agreed 2026-11-13 in `docs/specs/application.md` §3.0, §3.6 | plan written 2026-11-13 (A.1–A.6, 5–6.5 h), waiting for the owner's approval; runs in parallel with SETTINGS-5 |
| [ASM-1](../work/tasks/assessment/ASM-1_rerun-failed-with-model/description.md) | R-ASM | backlog (deferred); spec [Application preparation](specs/application.md) §3.1 | ASM-4 |
| [ASM-2](../work/tasks/assessment/ASM-2_ollama-num-ctx/description.md) | R-ASM | [done](../work/tasks/assessment/ASM-2_ollama-num-ctx/done.md) | GO-2 |
| [ASM-3](../work/tasks/assessment/ASM-3_skip-stopped-in-enqueue/description.md) | R-ASM | [done](../work/tasks/assessment/ASM-3_skip-stopped-in-enqueue/done.md) | COL-6, ASM-4 |
| [ASM-4](../work/tasks/assessment/ASM-4_fast-weighted-ranking/description.md) | R-ASM | [done](../work/tasks/assessment/ASM-4_fast-weighted-ranking/done.md) | COL-1 |
| [ASM-6](../work/tasks/assessment/ASM-6_deterministic-quick-match/description.md) | R-ASM, R-SEARCH, R-SET, R-APP | review; Q.1–Q.5 technically accepted, source in main `58af432`, released to production `157fe9f` with actual profile and 2777 numeric results; owner ranking-quality review separate | replaceable extraction/contract/evaluator; SETTINGS-5, calibration and LLM queue changes excluded; release separate |
| [ASSESSMENT-1](../work/tasks/assessment/ASSESSMENT-1_user-calibration-of-quick-match-weights/description.md) | R-ASM, R-SET | backlog; user weight calibration with method reference, provisional55–100min, not approved | working Quick Match and explicit scales; no v1 calibration UI |
| [ASSESSMENT-2](../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/description.md) | R-ASM, R-SEARCH, R-SET, R-COL | backlog; owner approved the complete plan on 2026-11-10 (exact wording and approval in the [card plan](../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/plan.md)); Q.1 next; whole-task estimate unchanged at165–290min; starting policy approved but not implemented, no paid call or release | released v1 in ASM-6 remains current; [approved plan](../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/plan.md); ASSESSMENT-1 calibration and ASM-6 release records are separate |
| [ASSESSMENT-3](../work/tasks/assessment/ASSESSMENT-3_two-fit-scores/description.md) | R-ASM, R-SEARCH | [done](../work/tasks/assessment/ASSESSMENT-3_two-fit-scores/done.md); in main `0bc8af6`, released to production 2026-11-11; the model's matching pass is off by default and You fit is hidden; spec [quick-match §14](specs/quick-match.md) | supersedes ASSESSMENT-2; follow-ups ASSESSMENT-4/5/6 |
| [ASSESSMENT-4](../work/tasks/assessment/ASSESSMENT-4_dismiss-requirement/description.md) | R-ASM | backlog (agreed behaviour and estimate in the card); spec [Profiles](specs/profiles.md) §3.5 | after ASSESSMENT-3 |
| [ASSESSMENT-5](../work/tasks/assessment/ASSESSMENT-5_matching-quality/description.md) | R-ASM | backlog ([plan](../work/tasks/assessment/ASSESSMENT-5_matching-quality/plan.md); not approved) | after ASSESSMENT-3; old Q.8 |
| [ASSESSMENT-6](../work/tasks/assessment/ASSESSMENT-6_shortlist-recheck/description.md) | R-ASM | backlog (draft; open questions) | after ASSESSMENT-3 |
| [ASSESSMENT-7](../work/tasks/assessment/ASSESSMENT-7_pair-judge-trial/description.md) | R-ASM | backlog; plan in the card; for a separate agent in parallel with INBOX-2 | feeds ASSESSMENT-5; spec [quick-match §15](specs/quick-match.md) |
| [ASSESSMENT-8](../work/tasks/assessment/ASSESSMENT-8_restricted-remote/description.md) | R-ASM | done; in main `eddc94a`, released 2026-11-13 | follow-up ASSESSMENT-9 |
| [ASSESSMENT-9](../work/tasks/assessment/ASSESSMENT-9_model-location-restriction/description.md) | R-ASM | done; in main `66ae60d`, released 2026-11-13 | follow-up ASSESSMENT-10 |
| [ASSESSMENT-10](../work/tasks/assessment/ASSESSMENT-10_restriction-labels-like-remote-us-only-and-us-cana/description.md) | R-ASM | done; in main `105e82e`, released 2026-11-13 | compound restriction labels resolved; cities and time zones stay unknown |
| [ASSESSMENT-11](../work/tasks/assessment/ASSESSMENT-11_remote-restriction-stated-in-text/description.md) | R-ASM | backlog (captured 2026-11-13): remote restriction stated in the vacancy text or in model-read phrases ("anywhere in the world", "US only" in the body); spec [Quick Match](specs/quick-match.md) §4.1 | not planned |
| [INBOX-2](../work/tasks/inbox/INBOX-2_listing-markup/description.md) | R-COL, R-SEARCH | backlog; owner approved the [plan](../work/tasks/inbox/INBOX-2_listing-markup/plan.md) with By section on 2026-11-11; spec [Collection](specs/collection.md) | after ASSESSMENT-3; includes INBOX-3 as L.4b |
| [INBOX-3](../work/tasks/inbox/INBOX-3_sticky-fetch-error/description.md) | R-SEARCH | backlog; fixed within INBOX-2 (L.4b); spec [Catalogue and search](specs/catalogue.md) | INBOX-2 |
| [OPS-9](../work/tasks/ops/OPS-9_finish-process-defects/description.md) | R-HARNESS, R-REL | backlog (scope agreed, estimate in the card) | found at the ASSESSMENT-3 finish |
| [OPS-10](../work/tasks/ops/OPS-10_production-db-hang/description.md) | R-REL, R-ASM | done; in main `f7190bf`, released to production 2026-11-12 | follow-up: INBOX-2 L.4d (jobs list polling) |
| [OPS-11](../work/tasks/ops/OPS-11_spec-coverage/description.md) | R-COL, R-SEARCH, R-TRI, R-APP, R-DATA, R-SET | done; in main `62c2f4b`, released 2026-11-13 | follow-up: move vacancy-view.md into docs/specs after INBOX-2 |
| [COLLECTION-4](../work/tasks/collection/COLLECTION-4_collection-run-feedback-failed-sources-on-off-swit/description.md) | R-COL | backlog (captured 2026-11-13 from OPS-11); spec [Collection](specs/collection.md) | not planned |
| [TRIAGE-3](../work/tasks/triage/TRIAGE-3_applied-tab-promises-30-day-deletion-owner-chose-n/description.md) | R-TRI | backlog (captured 2026-11-13 from OPS-11); spec [Triage](specs/triage.md) | not planned |
| [TRIAGE-4](../work/tasks/triage/TRIAGE-4_small-interface-fixes-n-and-a-keys-on-any-layout-n/description.md) | R-TRI | backlog (captured 2026-11-13 from OPS-11); spec [Triage](specs/triage.md) | not planned |
| [INBOX-4](../work/tasks/inbox/INBOX-4_location-tile-shows-the-geography-verdict/description.md) | R-SEARCH, R-ASM | done; in main `b5f8941`, released 2026-11-13: unrestricted remote ("Anywhere", bare "Remote", TELECOMMUTE) counts as a geography match; the tile step moved to INBOX-5; spec [Quick Match](specs/quick-match.md) §4.1, §14.1 | — |
| [INBOX-5](../work/tasks/inbox/INBOX-5_location-tile-verdict/description.md) | R-SEARCH | backlog (captured 2026-11-13): Location tile shows the geography verdict, moved from INBOX-4 T.1 | after INBOX-2 |
| [SETTINGS-6](../work/tasks/settings/SETTINGS-6_edit-source-url/description.md) | R-SET, R-COL | backlog (proposed behaviour and estimate in the card); spec [Collection](specs/collection.md) | small; after INBOX-2 |
| [SETTINGS-7](../work/tasks/settings/SETTINGS-7_profile-file-delete/description.md) | R-SET | [done](../work/tasks/settings/SETTINGS-7_profile-file-delete/done.md); in main `320dbf9`, released to production 2026-11-12 | — |
| [SETTINGS-8](../work/tasks/settings/SETTINGS-8_cv-fact-extraction/description.md) | R-SET | done; in main `b2cf419`, released to production 2026-11-12 | known limits in its attention.md |
| [SETTINGS-9](../work/tasks/settings/SETTINGS-9_geography-editor-location/description.md) | R-SET | backlog bug, low priority: geography editor shows location not set; spec [Profiles](specs/profiles.md) §3 rule 16 | with SETTINGS-5 S.1 |
| [SETTINGS-10](../work/tasks/settings/SETTINGS-10_skills-and-confirmed-facts/description.md) | R-SET | backlog (captured 2026-11-13, moved out of INBOX-2): requirement routing by the skill vocabulary, Move to skills / Move to facts, distinct keys for C, C++, C#; spec [Profiles](specs/profiles.md) rules 4a, 40, 42 | after INBOX-2 |
| [COL-1](../work/tasks/collection/COL-1_source-abstraction/description.md) | R-COL | [done](../work/tasks/collection/COL-1_source-abstraction/done.md) | GO-2 |
| [COL-2](../work/tasks/collection/COL-2_learned-template-cache/description.md) | R-COL | backlog; per-source rules and repair (owner, 2026-11-11; spec quick-match §15.3); spec [Collection](specs/collection.md) | COL-1; after INBOX-2 |
| [COL-3](../work/tasks/collection/COL-3_auto-and-manual-collect-trigger/description.md) | R-COL | [done](../work/tasks/collection/COL-3_auto-and-manual-collect-trigger/done.md) | COL-1 |
| [COL-4](../work/tasks/collection/COL-4_collection-progress-events/description.md) | R-COL | backlog; spec [Collection](specs/collection.md) | COL-1, COL-3 |
| [COL-5](../work/tasks/collection/COL-5_description-structure/description.md) | R-COL | cancelled (duplicate of INBOX-2) | — |
| [COL-6](../work/tasks/collection/COL-6_stop-word-collection-filter/description.md) | R-COL | [done](../work/tasks/collection/COL-6_stop-word-collection-filter/done.md) | COL-1 |
| [COL-7](../work/tasks/collection/COL-7_production-refresh-and-assess/description.md) | R-COL, R-ASM | backlog (owner-requested existing production operation) | preserve old pending queue; no Profile execution or deploy |
| [COLLECTION-1](../work/tasks/collection/COLLECTION-1_publication-date-and-age/description.md) | R-COL, R-TRI, R-ASM | review; technically accepted; source in main `3ee6260` (final `58af432`); approx35min vs20–35; source dates/basis and Newest, automatic scoring deferred; spec [Collection](specs/collection.md) | legacy origin unverified; release separate; production not updated |
| [COLLECTION-2](../work/tasks/collection/COLLECTION-2_remotive-category-pages/description.md) | R-COL | backlog; 73 Remotive category pages stored as vacancies (plan in the card); spec [Collection](specs/collection.md) | after INBOX-2 |
| [COLLECTION-3](../work/tasks/collection/COLLECTION-3_sites-behind-login/description.md) | R-COL, R-SET | backlog idea; to discuss with the owner later (LinkedIn profile import stored a sign-in page); spec [Collection](specs/collection.md) | — |
| [TRIAGE-1](../work/tasks/triage/TRIAGE-1_recognize-previously-applied-vacancy-duplicates/description.md) | R-COL, R-TRI, R-APP | backlog bug; Coinbag/Uniswap examples expose same-ID divergent application histories across environments;25–45min provisional; spec [Triage](specs/triage.md), [Collection](specs/collection.md) | choose authoritative real history before reconciliation; no automatic merge or execution approved |
| [TRIAGE-2](../work/tasks/triage/TRIAGE-2_layout-independent-select-shortcut/description.md) | R-TRI | in progress; K.2 integrated the accepted physical `KeyY` Select fix and reviewed records in main `e295b21`; K.3 push awaits coordinator review; spec [Triage](specs/triage.md) | guards/other shortcuts preserved; no release; production not updated |
| [DESIGN-1](../work/tasks/design/DESIGN-1_design-token-layer/description.md) | R-DESIGN | backlog | GO-11 |
| [DESIGN-2](../work/tasks/design/DESIGN-2_separate-presentation-and-logic/description.md) | R-DESIGN, R-SEARCH, R-TRI, R-SET, R-APP | backlog (approved; preparing worktree) | GO-11 |
| [DESIGN-3](../work/tasks/design/DESIGN-3_reusable-design-blocks/description.md) | R-DESIGN, R-SET | review; owner confirmed design finished; technical acceptance stands; source in main `5bedae3` (final `58af432`) | corrected25–45min, approximately28–30min technical stage; functional owner acceptance/release separate; production not updated |
| [GO-1](../work/tasks/go-migration/GO-1_backend-and-public-repo/description.md) | R-PLATFORM | [done](../work/tasks/go-migration/GO-1_backend-and-public-repo/done.md) | — |
| [GO-2](../work/tasks/go-migration/GO-2_runnable-backend-parity/description.md) | R-PLATFORM | [done](../work/tasks/go-migration/GO-2_runnable-backend-parity/done.md) | GO-1 |
| [GO-3](../work/tasks/go-migration/GO-3_public-rules-hooks/description.md) | R-PLATFORM | [done](../work/tasks/go-migration/GO-3_public-rules-hooks/done.md) | GO-1 |
| [GO-4](../work/tasks/go-migration/GO-4_race-free-tests/description.md) | R-PLATFORM | [done](../work/tasks/go-migration/GO-4_race-free-tests/done.md) | GO-2 |
| [GO-5](../work/tasks/go-migration/GO-5_consolidate-under-sifty/description.md) | R-LEGACY | backlog | COL-1, APP-6, APP-7, GO-7, GO-9, GO-11 |
| [GO-6](../work/tasks/go-migration/GO-6_self-host-packaging/description.md) | R-PKG | backlog | GO-5, GO-10 |
| [GO-7](../work/tasks/go-migration/GO-7_serve-background-timers/description.md) | R-TRI | backlog; spec [Triage](specs/triage.md) | GO-2 |
| [GO-8](../work/tasks/go-migration/GO-8_cli-command-parity/description.md) | R-LEGACY | [cancelled](../work/tasks/go-migration/GO-8_cli-command-parity/cancelled.md) | — |
| [GO-9](../work/tasks/go-migration/GO-9_mcp-server/description.md) | R-MCP | backlog | COL-1, GO-11 |
| [GO-10](../work/tasks/go-migration/GO-10_license-elastic-2/description.md) | R-PLATFORM | [done](../work/tasks/go-migration/GO-10_license-elastic-2/done.md) | GO-1 |
| [GO-11](../work/tasks/go-migration/GO-11_move-frontend-into-sifty/description.md) | R-PLATFORM | [done](../work/tasks/go-migration/GO-11_move-frontend-into-sifty/done.md) | GO-2 |
| [NAVIGATION-1](../work/tasks/navigation/NAVIGATION-1_navigation-shell/description.md) | R-TRI | [done](../work/tasks/navigation/NAVIGATION-1_navigation-shell/done.md) | ASM-4 |
| [NAVIGATION-2](../work/tasks/navigation/NAVIGATION-2_search-semantics-backend/description.md) | R-SEARCH | [done](../work/tasks/navigation/NAVIGATION-2_search-semantics-backend/done.md) | COL-6, TRI-1 |
| [NAVIGATION-3](../work/tasks/navigation/NAVIGATION-3_search-editor-frontend/description.md) | R-SEARCH | [done](../work/tasks/navigation/NAVIGATION-3_search-editor-frontend/done.md) | NAVIGATION-2 |
| [NAVIGATION-4](../work/tasks/navigation/NAVIGATION-4_search-ui-refinements/description.md) | R-SEARCH | [done](../work/tasks/navigation/NAVIGATION-4_search-ui-refinements/done.md) | NAVIGATION-3 |
| [OPS-1](../work/tasks/ops/OPS-1_deploy-on-acceptance/description.md) | R-REL | [done](../work/tasks/ops/OPS-1_deploy-on-acceptance/done.md) | OPS-2, ASM-4 |
| [OPS-2](../work/tasks/ops/OPS-2_relocate-sifty-home/description.md) | R-REL | [done](../work/tasks/ops/OPS-2_relocate-sifty-home/done.md) | ASM-4 |
| [OPS-3](../work/tasks/ops/OPS-3_staging-preview-flow/description.md) | R-REL | [done](../work/tasks/ops/OPS-3_staging-preview-flow/done.md) | OPS-1 |
| [OPS-4](../work/tasks/ops/OPS-4_repository-and-workflow-foundation/description.md) | R-REL, R-HARNESS | in progress | OPS-1, OPS-3 |
| [OPS-5](../work/tasks/ops/OPS-5_staging-idle-cpu/description.md) | R-REL | backlog (bug) | OPS-3 |
| [OPS-6](../work/tasks/ops/OPS-6_git-history-cleanup/description.md) | R-HARNESS | backlog (idea) | — |
| [OPS-7](../work/tasks/ops/OPS-7_configurable-paths/description.md) | R-DATA | backlog (idea); spec [Local data](specs/data.md) | OPS-4 |
| [OPS-8](../work/tasks/ops/OPS-8_data-layout-migration/description.md) | R-DATA | backlog; spec [Local data](specs/data.md) | OPS-4, OPS-7 |
| [SET-1](../work/tasks/settings/SET-1_first-time-answer-preparation/description.md) | R-SET | backlog (blocked: owner questions); spec [Profiles](specs/profiles.md) §2 (Answers for application forms in My data) | GO-2 |
| [SET-2](../work/tasks/settings/SET-2_config-live-reload/description.md) | R-SET | [done](../work/tasks/settings/SET-2_config-live-reload/done.md) | GO-2 |
| [SET-3](../work/tasks/settings/SET-3_stage-model-selection/description.md) | R-SET | [done](../work/tasks/settings/SET-3_stage-model-selection/done.md) | SET-2 |
| [SET-4](../work/tasks/settings/SET-4_status-bar-inline-models/description.md) | R-SET | backlog | SET-3 |
| [TRI-1](../work/tasks/triage/TRI-1_dismiss-and-block-rules/description.md) | R-TRI | [done](../work/tasks/triage/TRI-1_dismiss-and-block-rules/done.md) | COL-1, COL-3 |
| [TRI-2](../work/tasks/triage/TRI-2_source-block-in-settings/description.md) | R-TRI | [done](../work/tasks/triage/TRI-2_source-block-in-settings/done.md) | TRI-1 |

<a id="legacy-cards"></a>
## Legacy single-file cards

These predate the folder convention. They are retained as historical records and are not current instructions.

| Card | Role | Status |
|---|---|---|
| [job-search-assistant.md](../work/tasks/job-search-assistant.md) | Historical Stage 1–2 implementation, acceptance and transfer record. | historical |
| [sifty-dev-setup.md](../work/tasks/sifty-dev-setup.md) | Bootstrap plan and progress log for the work-management harness. | historical |

<a id="validation"></a>
## What is validated where

- `bin/map-check` validates local links and fragments in the active navigation documents, requirement references and the canonical task index.
- The canonical task index below must contain every current card ID exactly once; legacy single-file cards are listed separately.
- Archived historical prose is outside the validation scope. Old paths in `docs/product/`, `docs/architecture/go-parity-audit.md`, `legacy/app/` and historical card logs are retained for attribution and are not rewritten.
- `bin/work-test` runs the map checks and the isolated map tests as part of the harness entry used by CI.
