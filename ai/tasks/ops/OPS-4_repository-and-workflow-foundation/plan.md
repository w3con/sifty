---
estimate: 23
---
# OPS-4 — plan

**Author:** coordinator
**Created:** 2026-11-06
**Approval:** approved by the owner on 2026-11-06: «план норм, но не забудь почистить потом старые фолдеры. исполняй.» Includes cleanup of obsolete directories after verified relocation.

## Statement of work

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
- [ ] 2.2 — Keep secret checks active everywhere, including scanner sources, messages and renamed files; redact detected values in reports.
- [ ] 2.3 — Repair project hook adapters and applicability checks, preserve shared implementations, and verify installation from a fresh clone.

### Phase 3 — Relocate code and project records

*Gate: frontend/backend/worker checks pass at the new paths; no unfinished task or user data is lost.*

- [ ] 3.1 — Move frontend, backend and legacy implementation; repoint build, serve, worker and legacy-relative paths together.
- [ ] 3.2 — Move planning to work, documentation to docs and shared tooling to tools; repair submodule and symlink wiring.
- [ ] 3.3 — Add the mcp placeholder, update ignore rules and untrack incident/build artifacts while preserving local copies.
- [ ] 3.4 — Move GitHub workflows to root, correct their paths and cover backend, frontend, workers and harness checks.
- [ ] 3.5 — Audit old paths and local contents; remove obsolete directories after relocation checks, preserving retained data and work.

### Phase 4 — Simplify task management without losing work

*Gate: every existing task is accounted for; recording a new report never starts an executor or changes staging.*

- [ ] 4.1 — Implement four task states, blocker reasons and cancellation outcomes; preserve old identifiers and closed-task evidence.
- [ ] 4.2 — Add lightweight bug/idea capture and backlog/bug views derived from canonical cards; preserve owner wording and duplicates.
- [ ] 4.3 — Add task status/resume navigation; keep roadmap sequencing and epic outcomes explicit without duplicating task descriptions.

### Phase 5 — Connect requirements, code and planned changes

*Gate: a change can be traced from a user scenario through affected code/checks to decisions and open tasks.*

- [ ] 5.1 — Create the product/architecture map for current capabilities, distinguishing implemented, planned and known-broken behavior.
- [ ] 5.2 — Link epics/tasks to requirements, frontend/backend/worker/storage areas, checks and dependencies; validate local links.
- [ ] 5.3 — Document runtime data provenance and create a separate data-layout migration task with live-reference and backup requirements.

### Phase 6 — Implement the staging and release loop

*Gate: uncommitted development changes appear on 8081; 8080 changes only after owner acceptance of that task.*

- [ ] 6.1 — Watch the active task's sources: refresh frontend and rebuild/restart backend/workers automatically with visible failure status.
- [ ] 6.2 — Isolate production binary, bundle, workers and data; refresh staging data explicitly rather than on every code update.
- [ ] 6.3 — Wire approved completion to checked merge and production deployment; retain recovery state until deployment succeeds.

### Phase 7 — Verify and hand back

*Gate: the complete workflow passes in a fresh clone and scratch environments, and staging is ready for owner review.*

- [ ] 7.1 — Run application and workflow checks, including hook setup, continuous preview, failed-release recovery and clean-clone setup.
- [ ] 7.2 — Update project instructions and this card's new location; hand back staging for approval before merging/releasing.

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
- GitHub discovers root workflows; acceptance covers changes to build/deploy scripts as well as application source. Functional tests exercise lifecycle failures rather than only syntax and command help.
- Staging on 8081 follows the active task's working sources automatically, including uncommitted changes. Failed rebuilds are visible; staging identifies the revision/working state it actually serves. Only one task owns the shared preview at a time.
- Production on 8080 uses isolated artifacts, including workers. Code refreshes do not overwrite staging edits or user inputs. An explicitly requested staging-data refresh uses a consistent snapshot.
- Owner approval is the single release decision. Pre-approval review uses staging; the approved revision is checked and merged to main, production is verified, and only then is the task marked done. Release failure preserves a usable production version and task recovery state; closing this foundational task follows the same approval rule.
- The release path is exercised against scratch data/ports before production use, including restoration of binary, bundle, worker artifacts and a consistent database when rollback is necessary.
- The proposed user-provided/derived/vacancy/application data separation is documented, but actual live-data migration is a separately prepared backlog task.
- `work/incidents.md`, personal runtime data, credentials, local logs and generated binaries/bundles are not tracked in the resulting tree. Existing Git history is not rewritten.
