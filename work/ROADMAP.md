# Sift — build roadmap

Single source for the card sequence and current status. Read this first on resume after a context reset. Each card's own folder under `work/tasks/<epic>/<ID>_<slug>/` holds its description, plan and checkpoints; this file only holds the order, dependencies and status across cards.

Updated: 2026-11-06.

## Current planning work

**Resume point:** steps 1.1–3.3 are implemented and committed on `work/OPS-4`; ten of 23 steps are complete and 3.4–3.5 remain. Records now live under `work/`, product/architecture material under `docs/product/` and `docs/architecture/`, and the optional shared tooling at `tools/` with its pinned gitlink and relative links preserved. `mcp/README.md` is a placeholder only. `work/incidents.md` and `sifty/dist-staging` are ignored and untracked while their local bytes remain; the main checkout must preserve its copies before merge per the card inventory's retention section. Checks passed: live `work-resume` at the new card path, `bin/work-test`, scanner staged/tree sweeps, and an isolated public clone without the private submodule (`work-setup`, `work-test`, `work-plan`, `work-new`). Next: coordinator reviews 3.2–3.3, then assigns 3.4. Main remains at `c0181a1`; no merge/deploy has run. CPU, Git-history cleanup and expired-vacancy reports are captured in separate backlog cards.

**Repository and workflow foundation — in progress; execution and acceptance stages verified.** The owner approved the coordinator's complete plan: top-level `frontend/`, `backend/`, `mcp/`, `legacy/app/`, `docs/`, `work/` and `tools/`, safe acceptance, root GitHub workflows, automatic staging, task capture, a requirement/dependency map and a bounded executor trial. Cleanup of obsolete directories is explicit in step 3.5 and the acceptance criteria. The plan is in `work/tasks/ops/OPS-4_repository-and-workflow-foundation/plan.md`. Existing feature cards remain pending; no new bug report or feature idea automatically interrupts the active work. The previously planned design-token work must use the new frontend location once the foundation is accepted.

## Locked decisions (the direction behind the cards)

- Go is the single backend; the old TypeScript backend in `app/src` is retired (gated — see GO-5).
- The approved structural target separates `frontend/`, `backend/` (including schema and workers), `mcp/` and `docs/` at the repository root; this supersedes consolidation under `sifty/`. Relocation is implemented through 3.3; 3.4–3.5 remain.
- The project is source-available and self-hostable; license is Elastic License 2.0 (not "open source"; self-host and modify allowed, offering it to others as a managed service is not).
- Install story: Docker Compose as the one-command path; a documented manual developer setup alongside.
- Extraction is deterministic by default (standards and ATS adapters); a model is a last-resort fallback and stays off; a learned per-host template cache with validate-on-use/invalidate-on-failure is a later step (COL-2), only its interface is written in COL-1.
- Extraction unit is the platform, not the page; the model, when used, learns a reusable selector once rather than parsing every page.

## Card sequence (recommended order)

### Incoming bug reports (not scheduled)

- [High staging server CPU while idle](tasks/ops/OPS-5_staging-idle-cpu/description.md): owner reports approximately 45% CPU with no user activity; Activity Monitor identifies `sifty-staging`, not a browser renderer. Cause unverified; captured without starting investigation or interrupting the foundation task.

### Incoming ideas and separate work (not scheduled)

- [Centralize all paths in configuration](tasks/ops/OPS-7_configurable-paths/description.md): the owner requested a separate improvement plan after structural relocation, covering every path rather than deployment settings alone. Planning requested; implementation not approved.
- [Review and simplify Git history](tasks/ops/OPS-6_git-history-cleanup/description.md): inspect the log and propose cleanup, with the method and any rewrite to be approved separately.
- [Expired vacancy response in application confirmation](tasks/application-preparation/APP-8_expired-application-outcome/description.md): add the missing manual closed/expired outcome; persistence/visibility details remain to be prepared.

**Goal (owner, 2026-11-04): the owner needs to be finding work, not spending another week-plus building this app.** The sequence is cut to the minimum that makes the tool usable for the owner's own daily search. The core loop already works today — collection flows (COL-1/COL-3), assessment runs, descriptions are readable, the owner decides and applies on the listing site by hand. Productization (retire the old TypeScript backend, packaging, MCP, CLI parity) and the large, currently-broken application-preparation build are deferred until the owner asks, or until shipping the app to others becomes the goal.

The 2026-11-05 session shipped TRI-1, TRI-2, COL-6, ASM-3, ASM-4, OPS-2, OPS-1, and OPS-3. The application runs on port 8080 against `./data`; staging is on port 8081 against `./data-staging`. To build: `make build`. To preview an in-progress branch on staging: create a worktree for it and run `bin/sift-deploy --env staging` from that worktree. To promote to production on the owner's word: `bin/sift-promote`. Finishing a card merges to main only; no staging deploy runs automatically. See `AGENTS.md` for the full two-environment deploy loop. After the deploy loop correction comes the navigation restructure.

### Scheduled (minimal, for the owner's own use)

1. **OPS-1 — done.** The two-environment deploy loop: `bin/sift-deploy --env staging|prod`, `bin/sift-promote`, `bin/sift-accept`. See `AGENTS.md` for the full loop documentation.
2. **OPS-3 — done.** Corrected the loop so staging previews in-progress work: each env owns its own binary and bundle; staging deploy never touches prod artifacts; `bin/work-finish` no longer auto-deploys staging; production advances only via `bin/sift-promote`.
2. **Navigation restructure — done 2026-11-06, all three cards merged to main and live on staging (not yet promoted to prod).** The owner's design (settled 2026-11-05) grew past a pure reshuffle and was split into three logical cards:
   - **NAVIGATION-1 (frontend, independent, first) — done 2026-11-05, merged to main.** Top tabs Collection · Triage · Preparation · Archive (Pre-selected renamed to **Preparation**, not "Selected"); Collection tab with Sources + Search sections; Archive with an Applied/Dismissed toggle over existing data; Settings trimmed to Context/Configuration/Queue. Pure frontend on existing endpoints. The Sources section carries the existing read-only coverage view for now; its editable list is built in -2/-3 (below).
   - **NAVIGATION-2 (backend, before -3) — done 2026-11-06, merged to main.** OR/AND search-term model in `search.yaml` honoured by collection and ASM-4 ranking; a single exclusion mechanism matching title or company that folds in TRI-1 company blocks and COL-6 stop-words; source-block (TRI-2) retired in favour of the Sources list; migration of existing block rules; endpoints for the search expressions, the exclusion terms and the source sites list (add/remove/enable a site).
   - **NAVIGATION-3 (frontend, depends on -2) — done 2026-11-06, merged to main.** The OR/AND builder and the unified exclusion list in Collection→Search; an editable Sources list in Collection→Sources (add/remove/enable a site) over the -2 endpoint; removal of the old block-rule, source-block and stop-word UI; no source control left in Settings. Decided 2026-11-05: remove the always-on bottom status footer and move its queue counts and "Assess vacancies" toggle into Settings→Queue, so the queue lives only in Settings.

   All three done and merged to main; staging serves the result at http://localhost:8081. Production promotion (`bin/sift-promote`) is still pending the owner's word. Industry/title-category selection is deliberately out of scope (per-source taxonomies differ).

   - **NAVIGATION-4 (backend + frontend) — done 2026-11-06, merged to main.** First-day-of-use refinements: Search becomes three columns of tag chips (OR, AND, Exclusions) with immediate save and no Save button; OR and AND groups now combine with AND in the FTS query (any OR term AND all AND terms, minus exclusions); Sources becomes a one-row-per-source table and the "Source coverage" diagnostic panel is removed; Archive loses the Sort control and Unread button and is ordered by application date.
3. **COL-5** — preserve heading/list structure in the job description view (our CSS, new `description_html` field, plain `job.Description` untouched). Cheap readability win for daily scanning; optional, skippable. Needs `plan.md` before execution.

After these: stop and reassess. Do not continue into the deferred pile without a fresh owner decision.

### Deferred until the owner asks (not scheduled)

- **Application preparation + auto-submit** — the biggest, least-working piece. The form-inspection worker currently crashes on startup (Node module-loader error, surfaced in the UI as `worker inspectForm: exit status 1`), so preparation is broken, not merely weak. A multi-week build; do it only as a focused push when wanted. The crash is parked here, not chased now. **APP-6** (inject remembered answers into drafting — premature until questionnaire extraction yields any questions) and **APP-7** (draft revision check) belong to this deferred block.
- **SET-4** — taller status bar with inline model selection. Polish.
- **GO-11** (move the frontend into `sifty/web` with its own build; repoint Makefile/sift-deploy/sift-accept; leave `app/src` in place) — **done 2026-11-06, merged to main, live on staging.** The owner chose to move the frontend first, before the rest of the consolidation and before the design pass, so design lands on the final location.
- **GO-7** (two missing serve.go background tickers: evidence-fingerprint context refresh and archive-retention purge), **GO-9** (MCP entry point — **decision 2026-11-06: stays TypeScript in `sifty/mcp` as an HTTP client to the Go backend, not a Go rewrite**; any backend op it needs is added to the Go HTTP API), **GO-5** (retire `app/src`), **GO-6** (packaging + Elastic License 2.0 swap) — productization/cleanup, no bearing on the owner's own search. **GO-8 (CLI parity) is descoped** — the owner does not use a CLI or cron; only what MCP needs is carried, under GO-9. GO-5 is gated by the parity audit (needs APP-6/APP-7/GO-7/GO-9 and the frontend move GO-11 first; CLI no longer a blocker). GO-5/6/9 need `plan.md` before execution.
- **DESIGN-1** (design-token layer and style cleanup) — planned 2026-11-06, runs after GO-11 is merged. A pure refactor before the design pass: define role-named CSS tokens (colour/spacing/radii/typography) on `:root`, replace the ~44 hardcoded colours and the component inline styles with tokens, keep the app looking identical, and leave the structure ready for a later light/dark theme. The actual redesign (with the design tool) comes after this.
- **COL-2** (learned per-host extraction template cache), **COL-4** (per-source collection progress indicator), **ASM-1** (on-demand retry of failed assessments on a chosen, larger model) — reserved; founded, plans not written.

Side task: **parity audit** — DONE 2026-11-04, verified, written to `docs/architecture/go-parity-audit.md` (14 TS modules safe to delete, 11 must stay until ported). It is the authority for what GO-5 may delete. Two groups of gaps block deletion:

- **Collection vertical — now COL-1 (scope widened 2026-11-04).** Missing in Go: the outbound-HTTP layer with SSRF guard/proxy/page-cache (`network.ts`), per-domain rate limiting (`traffic.ts`), all ~12 per-source discovery adapters (`discovery.ts`), the real collection run (`services.ts` — Go's is a stub), single-URL import/enrich (`collectors.ts` — Go stubs). COL-1 now owns all of this. The model-extraction fallback (`extraction.ts`) stays deliberately off and is COL-2's seam, not a COL-1 gap; `extraction.ts` may be deleted in GO-5 since COL-2 rebuilds the model fallback in Go from spec, not from the TS source.
- **Separate gaps — now carded (2026-11-04):** APP-6 (remembered answers not injected into drafting), APP-7 (`PATCH` draft revision check), GO-7 (evidence-fingerprint refresh timer + archive-retention timer), GO-8 (CLI 4 of ~15 commands), GO-9 (MCP server absent, a planned paid-tier feature per `ai/research/sift-public-service-research.md`).

Blocked on the owner: **SET-1** (first-time knowledge-base answer preparation) has seven open owner questions in its card; **APP-3** depends on SET-1. Not scheduled until those are answered.

## Status

### 2026-11-05 session — done, merged to main

- **TRI-1 — done, merged.** Company/source block rules (sticky dismissal already existed at the storage level). Block by the normalised company name; a role is source-blocked only when every one of its sources is blocked; blocked roles are hidden from Triage, kept in Selected/Archive, never deleted, fully reversible. Settings "Dismissed & blocked" panel. Schema → user_version 11, `block_rules` table.
- **TRI-2 — done, merged.** The Triage per-role control is "Block company" only; blocking a source moved to Settings (a sources list with role counts + a Block control). Added `GET /api/sources`, `POST /api/block-rules`.
- **COL-6 — done, merged.** Stop-words: a case-insensitive substring of the title; hides matching roles from every view and the per-stage counts, skips them at collection, reversible; live "would hide N" preview; Settings "Stop-words" panel. Schema → user_version 12, `stop_words` table; `stopped` is a computed flag, not stored.
- **ASM-3 — done, merged.** `EnqueueAll` skips stop-word-hidden roles, so no provider spend on invisible roles.
- Also on `main`: non-assessed roles open on the Job-description tab (`RoleContent.tsx`).
- **ASM-4 — done, merged.** Fast weighted ranking via SQLite FTS5 `bm25()`: an instant 0–100 match score and "Best fit" sort; the mass LLM queue removed, the LLM runs only when a role is moved to Selected plus a manual "assess now". Verified on a copy of live data (2496 roles, 1642 scored across the full range).
- **OPS-2 — done, merged.** The whole application relocated from the FCTO knowledge-base repository to this standalone home: one build command (`make build`), production on port 8080 against `./data`, data git-ignored and off `/tmp`. The CV and portfolio website stay in FCTO. OPS-1 (deploy loop) is the next card.
- **OPS-1 — done, merged to main.** Two-environment deploy loop: `bin/sift-deploy --env staging|prod`, `bin/sift-promote`, `bin/sift-accept`. See `AGENTS.md` for full documentation.
- **OPS-3 — done, merged to main.** Corrected the deploy loop: each env owns its own binary (`bin/sifty` / `bin/sifty-staging`) and bundle (`app/dist` / `app/dist-staging`); staging shows an in-progress branch from its worktree; `bin/work-finish` no longer auto-deploys staging; production advances only via `bin/sift-promote`.
- **Running app** — port 8080 against `./data` (prod, binary `bin/sifty`), port 8081 against `./data-staging` (staging, binary `bin/sifty-staging`). To preview a branch on staging: `bin/sift-deploy --env staging` from a worktree for that branch. To promote prod: `bin/sift-promote`.
- **Process:** `bin/work-finish` prints a reminder to update this roadmap after each card.

### Earlier

- **ASM-2 — done, merged to main** (`2a0102a`). `inferHTTP` now sets `options.num_ctx` on the ollama request from `ContextChars` (≈4 chars/token, capped at 128k tokens); compatible path untouched; tests added. Card closed (`done.md`).
- **COL-1 — done, merged to main** (step commits `6fba75a`…`7b71df2`, finish `09da3fc`). The whole collection layer: `internal/fetch` (SSRF-guarded client, cache, per-domain traffic), `internal/source` (interface, registry, generic + jobspy sources, Greenhouse/Lever/Ashby/Getro adapters, COL-2 seam), the real `defaultCollect` run, and `importURL`/`EnrichFn` wired through the client. `go test -race ./...` green; coordinator-verified. Card closed (`done.md`). The executor pre-ran `work-finish` against its brief and left a half-applied state; recovered by the coordinator — see `work/incidents.md`.
- **COL-3 — done, merged to main** (fast-forward finish; branch `work/COL-3`, 6 commits). `TriggerCollection`/`TriggerAsync` in `internal/background/trigger.go` serialise through the `"daily"` cross-process lock, gate on a configurable cooldown (`config.Runtime.CollectionCooldownMS`, default 3 h) read against `LatestCollectionRun` (now a UNION of `collection_runs` and daily `runs`, so any trigger counts); `POST /api/search` calls `CollectFn` (wired to `TriggerAsync` in serve.go) and returns `{runId,status}` with status `running`/`skipped`; the frontend fires one unforced `POST /api/search` on mount (`app/web/main.tsx`) and the manual/per-source buttons send `force:true` (`app/web/Settings.tsx`). `go test -race ./...` green; coordinator-verified against the code. Card closed (`done.md`).
- **Collection proven live (2026-11-04 ~18:25).** Rebuilt the Go binary from `sifty/` and the web bundle (`npx vite build` → `app/dist`), killed the old pre-merge process, replaced `/tmp/claude-501/sifty-main`, restarted with the same args (`serve --data-dir /tmp/claude-501/sift-data --dist …/app/dist`). A forced `POST /api/search {"force":true}` returned `{"runId":4,"status":"running"}` and a real run began; the catalogue count rose from ~1300 (frozen) to 1336 and kept climbing. Some sources time out (e.g. Web3 Careers), which is normal. **Known rough edge (pre-existing, not a COL-3 regression): `GET /api/collection` returns `sources:[]` and `events:[]` for both daily and triggered runs, so the UI collection-progress indicator shows no per-source detail — a later observability card could populate it.**
  - The manual `/tmp`-based recipe is retired (OPS-2 moved data to `./data`; OPS-1 replaced `make run` with the scripted deploy loop). Current recipe: `bin/sift-deploy --env staging` for staging, `bin/sift-promote` for production.
- APP-6, APP-7, GO-7, GO-8, GO-9 founded from the parity audit; not started.
- Plans still to write before execution: GO-5, GO-6, GO-8, GO-9 need `plan.md`; COL-2 and TRI-1 need `plan.md` and, for TRI-1, the owner's answers first.

## Committed on `main` from the 2026-11-04 session

- `b50c4a3` — roadmap added and the six build cards founded.
- `4d6bfc6` — `app/web/main.tsx`, `app/web/Icon.tsx`, `app/web/style.css` (status bar shows live session models with labels and target/pencil icons and a fixed running-job suffix; header settings icon is a gear; Settings stage-model block laid out correctly — the base SET-4 builds on); `config/candidate.yaml` (`ai/HANDOVER.md` removed from candidate sources, a review note de-referenced); `app/.claude/settings.json` (subagent model pinned to sonnet).
- `.worktrees/` and `store.sqlite` added to `.gitignore`; the accidentally-staged executor worktree and the runtime database are no longer tracked.
- The demo (`/tmp/claude-501/sifty-main` on port 4317) runs the merged APP-2 code with these edits and the `qwen3.5-9b`/`qwen3.5-4b` provider config; its `config/providers.yaml` and `config/candidate.yaml` live under its data dir, not in the repo.

## Done this session (merged to main)

- APP-2 (Application tab: spinner, questions-before-answers, honest empty states, provider default from stage) — accepted, merged, pushed earlier in the chain through `ce8aeff`/APP-2 finish.
