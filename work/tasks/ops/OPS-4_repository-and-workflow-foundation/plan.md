---
estimate: 23
---
# OPS-4 — plan

**Author:** coordinator
**Created:** 2026-11-06
**Approval:** approved by the owner on 2026-11-06: «план норм, но не забудь почистить потом старые фолдеры. исполняй.» Includes cleanup of obsolete directories after verified relocation.

## Statement of work

**Final owner acceptance/release approval — 2026-11-06 23:13 CET:** owner said «принимай и закрывай» after 6.2/6.3 delivery and staging handback. This authorizes final prototype-mode acceptance, the explicit first legacy-to-isolated production transition, and normal checked card completion. Derive the exact approved source from the already accepted implementation plus final records; no new feature/code scope is included. Coordinator accepts; the executor performs all builds, preservation, Git and deployment mechanics. Final stage complexity is 3; AI preparation/preservation 2–4 minutes, acceptance preparation 3–6, release 3–6, verification 1–3, overall approximately 10–20 minutes, with stop boundary 23:35 CET. No new paid service; monetary usage cost unknown. Source checkpoint history and owner pending work must remain recoverable.

**Latest acceptance/renewal:** 6.2 is technically accepted through `a89625a`; twenty original steps are accepted. Owner renewed «доделывай 6.2, 6.3 (через субагентов)» after the pause. Remaining estimate communicated at 22:26 CET is 25–45 minutes AI, with stop boundary 23:11 CET. Assign only 6.3 next, preserving the existing prototype/no-suites policy and explicit owner approval for any live production release. The implementation must support the actual old-layout production process, not only a newly managed scratch server.

**Approved continuation — 2026-11-06 21:00 CET:** owner requested «ну давай уж тогда и 6.2 и 6.3 - это вроде закроет большой кусок функциональности», approved adding one-card squash integration («ок, можешь сделать, как дойдут руки») and acknowledged continuation. Deliver independent environment artifacts with deliberate staging data refresh, then checked owner-approved release and final completion. Demonstrate the normal command path and one controlled failed release/restore on temporary data/ports. Apply the accepted command implementation to this project and leave the currently running staging available for owner review; no live production release is authorized by implementation approval. General path centralization, existing-history rewriting and the cancelled enforcement plugin are excluded.

### Continuation estimate and steps

**Complexity:** 8: process ownership, filesystem artifacts, SQLite recovery and Git integration are coupled. Keep them in this existing card as one release outcome, but assign/review 6.2 and 6.3 independently. **AI preparation:** 5–10 minutes. **AI implementation:** 25–40 minutes, including executor context and expected corrections. **AI acceptance:** 10–20 minutes. **Total:** 40–70 minutes, excluding owner waiting; stop at the upper bound. No new paid service is required; monetary usage cost is unknown.

| Step | Result | Why it is needed and how it will be used | AI time |
|---|---|---|---|
| Review saved implementation/current runtime | One coherent continuation and preserved owner edits | The executor builds on current main/staging rather than an obsolete partial snapshot | 5–10 min |
| Complete 6.2 | Separate environment artifacts, explicit data refresh and recovery | Staging updates preserve production and current staging input; use the actual deploy command on isolated data | 15–25 min |
| Complete 6.3 | Checked card-granularity squash/release/completion | The owner approves staging once; the normal finish path deploys that revision before successful closure | 10–15 min |
| Accept and hand back | Observe success and one failure/restore; current instructions/site | Establish working use of the commands and preserve recovery before owner release | 10–20 min |

These estimates partition the same total; do not add phase estimates again. Checks are builds, code review and a bounded scratch command demonstration, not preserved automated suites or new test infrastructure. Work started at 21:00 CET; stop boundary is 22:10 CET unless the owner revises scope/cost.

**Latest scope/result:** the owner authorized 6.1 only after the relocation and separately requested moving root prompts into config. Continuous preview is now running on 8081 from the main checkout with observed automatic uncommitted frontend rebuild and live HTML update; 6.1 is technically accepted under the prototype build/code/site-review policy. Nineteen original steps are accepted. The new preview is handed back for owner review; remaining environment/release automation and full verification are deferred. Do not treat the original 23-step card as fully complete.

**Current delivery status (supersedes old execution directions, not the original checklist):** the owner authorized applying the prepared relocation to the main checkout, which is now done. Eighteen steps through 5.3 are technically accepted. Step 6.1 is checked by its executor but remains unaccepted and has not been activated on live staging. Step 6.2 (complete environment isolation/explicit data refresh) and 6.3 (automatic checked merge/release/completion) remain incomplete and deferred. Step 7.1's comprehensive workflow/recovery verification was not performed; automated suites were later disabled by owner instruction. Step 7.2 is partly delivered: the main checkout now has the new structure and current records, but the new preview has not been handed back for owner review. No `done.md` is written and the original 23-step plan is not represented as fully complete.

Restructure the repository around `frontend/`, `backend/`, `mcp/`, `legacy/app/`, `docs/`, `work/` and `tools/`; preserve unfinished work and runtime data. Repair acceptance and publication checks, simplify task tracking, establish requirement-to-code navigation and implement the agreed automatic staging / owner-approved production loop. The coordinator plans and reviews; coding runs in visible subagents on `opencode-go/deepseek-v4.1-flash` through the owner's subscription. Verify actual executor identity and results; do not silently substitute another provider/model. New bug reports and ideas are recorded without changing the active task. The exclusions and source conversation are in `description.md`.

## Target structure

- `frontend/`: current `sifty/web`, with its own dependencies, configuration and separate `dist/` and `dist-staging/` outputs.
- `backend/`: current Go module, `cmd/`, `internal/`, workers and backend-specific development scripts; preserve its Go module import path.
- `mcp/`: a tracked description of the future standalone TypeScript HTTP adapter; implementation remains a separate existing task.
- `legacy/app/`: current retired TypeScript implementation, tests and operations tools, retained until parity permits removal.
- `docs/`: current product and architecture documentation; `docs/MAP.md` links requirements, architecture, checks, decisions and tasks.
- `work/`: task cards, epics, decisions, roadmap and backlog/bug views. `work/incidents.md` and verbose local run logs are ignored; useful task outcomes and check summaries are versioned.
- `tools/`: relocate the shared tooling submodule and its existing links without changing their upstream contents; keep personal shared-tool integration optional for a fresh public clone.
- Root `bin/`, `config/`, `prompts/`, `Makefile`, license and agent entry points retain project-level purposes. Runtime `data/` and `data-staging/`, dependencies and build outputs remain local.

## Execution boundaries

After full-plan approval, assign only a bounded, independently checkable stage to each visible OpenCode executor session. The coordinator supplies exact step IDs, checks and a stopping point; all remaining open steps are context only. Review the stage and send corrections to the same executor while useful; start the next independent stage with a fresh context in the same worktree after the prior session stops. Persist checks, decisions, remaining findings and the next action before any context-size handoff. No numeric context threshold is yet selected or enforced. OpenCode is the primary tool; further Claude Code adapter/compatibility work is excluded. Reusable cross-project rules are a separate shared-repository task, not an implicit global-tooling rewrite here.

## Steps

### Phase 1 — Establish a safe execution and acceptance path

*Gate: focused command tests prove that incomplete or dirty work cannot be discarded, merged or marked done.*

- [x] 1.1 — Inventory active worktrees, unfinished tasks and local artifacts; record preservation requirements and baseline checks.
- [x] 1.2 — Define a project-local executor profile with an explicit model and visible session dispatch; verify actual model identity.
- [x] 1.3 — Harden acceptance: require main, clean source/target copies, closed steps, resolved blockers and unchanged tested revision.
- [x] 1.4 — Test failed checks, moved branches and dirty/rejected work; preserve branches and working copies on failure.

### Phase 2 — Make publication checks compatible with this repository

*Gate: representative public task/command references pass; representative synthetic secrets fail; hook setup is verifiable.*

- [x] 2.1 — Separate secret detection from configurable formatting rules; permit task identifiers and work commands in management/tooling files.
- [x] 2.2 — Keep secret checks active everywhere, including scanner sources, messages and renamed files; redact detected values in reports.
- [x] 2.3 — Repair project hook adapters and applicability checks, preserve shared implementations, and verify installation from a fresh clone.

### Phase 3 — Relocate code and project records

*Gate: frontend/backend/worker checks pass at the new paths; no unfinished task or user data is lost.*

- [x] 3.1 — Move frontend, backend and legacy implementation; repoint build, serve, worker and legacy-relative paths together.
- [x] 3.2 — Move planning to work, documentation to docs and shared tooling to tools; repair submodule and symlink wiring.
- [x] 3.3 — Add the mcp placeholder, update ignore rules and untrack incident/build artifacts while preserving local copies.
- [x] 3.4 — Move GitHub workflows to root, correct their paths and cover backend, frontend, workers and harness checks.
- [x] 3.5 — Audit old paths and local contents; remove obsolete directories after relocation checks, preserving retained data and work.

### Phase 4 — Simplify task management without losing work

*Gate: every existing task is accounted for; recording a new report never starts an executor or changes staging.*

- [x] 4.1 — Implement four task states, blocker reasons and cancellation outcomes; preserve old identifiers and closed-task evidence.
- [x] 4.2 — Add lightweight bug/idea capture and backlog/bug views derived from canonical cards; preserve owner wording and duplicates.
- [x] 4.3 — Add task status/resume navigation; keep roadmap sequencing and epic outcomes explicit without duplicating task descriptions.

### Phase 5 — Connect requirements, code and planned changes

*Gate: a change can be traced from a user scenario through affected code/checks to decisions and open tasks.*

- [x] 5.1 — Create the product/architecture map for current capabilities, distinguishing implemented, planned and known-broken behavior.
- [x] 5.2 — Link epics/tasks to requirements, frontend/backend/worker/storage areas, checks and dependencies; validate local links.
- [x] 5.3 — Document runtime data provenance and create a separate data-layout migration task with live-reference and backup requirements.

### Phase 6 — Implement the staging and release loop

*Gate: uncommitted development changes appear on 8081; 8080 changes only after owner acceptance of that task.*

- [x] 6.1 — Watch the active task's sources: refresh frontend and rebuild/restart backend/workers automatically with visible failure status.
- [x] 6.2 — Isolate production binary, bundle, workers and data; refresh staging data explicitly rather than on every code update.
- [x] 6.3 — Wire approved completion to a checked one-card squash merge and production deployment; retain recovery until success.

The owner approved card-granularity squash integration: intermediate commits remain recoverable on work branches, and each completed card contributes one meaningful main commit. Implement this with 6.3's release path while preserving readiness/revision/data safeguards. Existing-history consolidation remains the separately recorded history-review work; this approval does not rewrite it.

### Phase 7 — Verify and hand back

*Gate under the owner's prototype policy: the accepted source builds, recorded scratch recovery evidence is reviewed, staging is owner-accepted, and the actual release is healthy. Preserved automated suites stay skipped; they are not represented as passed.*

- [x] 7.1 — Complete prototype acceptance using builds, recorded workflow/recovery evidence and live review; keep automated suites skipped.
- [x] 7.2 — Finalize current instructions and records; record owner staging acceptance and prepare the pinned approved release.

## Acceptance criteria

- The agreed top-level structure builds and runs, and current application behavior is preserved. The legacy implementation and unrelated worktrees remain recoverable.
- Obsolete source/planning directories are removed after verifying relocation and updating live references. Inspect ignored/untracked contents before cleanup; preserve runtime data, retained legacy code, backups and unrelated worktrees rather than deleting them as leftovers.
- All existing unfinished tasks are inventoried and reachable; historical identifiers and decisions remain attributable. Old MIT/private-repository statements are clearly historical, not current instructions.
- A bug report or idea creates or updates a backlog card without launching implementation. New cards are minimal; preparing a plan does not require a separate state or agent.
- Backlog, in-progress, review and done are the only normal task states. A blocker records its reason and cancellation remains distinct from successful completion.
- The map links product requirements to affected code, data, checks, decisions and tasks. Impact analysis and updating affected links are part of preparation and completion.
- Model/provider selection is project-local and configurable; the owner selected OpenCode Go / DeepSeek V4.1 Flash for coding subagents. The coordinator retains planning/review/technical acceptance. A bounded first implementation stage records the actual model, available usage information, review findings and correction effort; no fallback to another model/provider occurs without an owner decision.
- Executor briefs scope one checkable stage and preserve a review/resume point. Same-stage corrections reuse useful context; independent stages use fresh sessions without concurrent writers in one worktree.
- General hooks remain shared. Project adapters skip inapplicable knowledge-base work and report genuinely missing required tools. Setup verifies effective Git hook installation instead of merely printing success.
- Building the application and running the core task commands/checks from a public clone do not require access to another private repository or the owner's global agent configuration.
- Real secret patterns are checked in every tracked area without printing secret values. Task identifiers and development commands are legal in work records, tooling, agent instructions and commit references. Formatting policy is explicit per area; `work/` may use Russian.
- Failed checks, dirty worktrees, open steps, unresolved blockers, changed tested revisions or a wrong target branch prevent finishing. No force option bypasses data-preservation checks, and rejection does not destroy recoverable work.
- GitHub discovers the root manual-only workflow; acceptance covers app/build changes with builds and the recorded bounded command/recovery demonstrations. Automated suites are paused by owner instruction and are never reported passed when skipped.
- Staging on 8081 follows the active task's working sources automatically, including uncommitted changes. Failed rebuilds are visible; staging identifies the revision/working state it actually serves. Only one task owns the shared preview at a time.
- Production on 8080 uses isolated artifacts, including workers. Code refreshes do not overwrite staging edits or user inputs. An explicitly requested staging-data refresh uses a consistent snapshot.
- Owner approval is the single release decision. Pre-approval review uses staging; the approved revision is checked and merged to main, production is verified, and only then is the task marked done. Release failure preserves a usable production version and task recovery state; closing this foundational task follows the same approval rule.
- The release path is exercised against scratch data/ports before production use, including restoration of binary, bundle, worker artifacts and a consistent database when rollback is necessary.
- The proposed user-provided/derived/vacancy/application data separation is documented, but actual live-data migration is a separately prepared backlog task.
- `work/incidents.md`, personal runtime data, credentials, local logs and generated binaries/bundles are not tracked in the resulting tree. Existing Git history is not rewritten.
