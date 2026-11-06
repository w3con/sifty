# AGENTS.md — work-management harness

This file is the source of truth for how work is managed in this repository. OpenCode is the primary agent tool. Keep the core workflow tool-neutral; do not add Claude Code compatibility work. The `bin/work-*` commands implement the current lifecycle, but not every behavioral rule below is enforced by code.

Roles: the owner states why a card exists; the coordinator plans it and accepts the result; the executor builds it.

## Start or resume a session

Read [the workflow](ai/WORKFLOW.md), then [the roadmap's current planning work](ai/ROADMAP.md), then the active card's description, plan, attention and log. Inspect `git status`, the relevant diff and `git worktree list` before editing. Preserve existing changes and unrelated working copies.

Use the card's recorded approval, completed checks and next step to resume. An open checkbox or a prepared plan is not authorization to execute. Record decisions and the next action on disk before a restart or context reset; chat history is not the checkpoint.

The roadmap records the current resume point. The foundation plan distinguishes agreed direction from proposed implementation; do not treat target directories or planned automation as already available.

## Incoming requests and preparation

Capture new bugs and ideas in the backlog without starting them or interrupting the active task. Preserve the owner's wording and add evidence to an existing matching card rather than duplicating it. An explicit instruction to change the active task is steering, not merely a backlog report.

Keep one canonical card per task. Backlog and bug lists are views; the roadmap holds sequencing and dependencies; epics describe product outcomes. During preparation, check the affected requirements, implementation, tests and dependent work. Raise contradictions or scope growth before implementation and obtain approval of the complete plan.

## Coordination and implementation

The main session prepares tasks and plans, analyzes dependencies, reviews code and performs technical acceptance. The owner approves product scope and final completion. Delegate coding to the project `executor` subagent, configured in `.opencode/agents/executor.md` with model `opencode-go/deepseek-v4.1-flash`, using the owner's OpenCode Go subscription.

Launch the executor through the live session's visible subagent tool with the approved card and its worktree brief. Do not substitute another model/provider or run a hidden agent through a shell wrapper. Verify the actual executor model from session metadata rather than its self-report. If the configured agent is unavailable or the model cannot run, report the blocker without silently falling back to the coordinator's model or a direct billed API.

The coordinator maintains plans, task records and project instructions directly. This role split does not authorize implementation before the task's plan is approved. Provider/model profiles are replaceable adapters; the core work commands remain tool-neutral.

Assign one bounded, independently checkable stage per executor session. Its brief must name the card, worktree, exact step IDs, checks and stopping point. Listing every open step does not assign every step. Review the diff and check evidence before assigning the next stage; send corrections to the same executor while its context remains useful. Start the next independent stage with a fresh executor context in the same worktree, only after the previous executor has stopped.

If context becomes too large, save completed work, check results, unresolved findings and the precise next action in the card before handing off to a fresh executor. Distinguish current context size from cumulative token spend. Do not claim a numerical context limit or automatic enforcement unless configured and verified. The owner reviews the completed result on staging, not every technical stage.

## A card is a folder

A unit of work is a card: a folder `ai/tasks/<epic>/<ID>_<slug>/`. The file set and the short-card variant are defined in `bin/templates/convention.md`. Steps are `- [ ]` (open) or `- [x]` (closed) lines in `plan.md`, or in `description.md` for a short card. The card is markdown, so it survives any session or tool dying.

## Current lifecycle commands

All commands are `python3`, standard-library only, run from anywhere in the repository.

- `bin/work-new <epic> <slug> [--prefix P] [--title T] [--short]` — allocate the next `PREFIX-n` ID, scaffold the file set from the templates, and commit the founded folder.
- `bin/work-plan <id>` — validate that the plan is fit to hand over. Refuses when there are no steps, no acceptance criteria, or template placeholders remain.
- `bin/work-handover <id>` — the gate to the executor. Refuses when the plan is not fit, attention is open, a terminal file is present, the declared estimate exceeds the ceiling, nothing is left to brief, or the card has uncommitted changes. Then cuts a worktree at `.worktrees/<id>` on branch `work/<id>` and hands off the executor brief.
- `bin/work-note <id> --step "<name>" --check "<command>"` — run the check and append the step with its verbatim output and exit code to `log.md`. `--message "<text>"` records a free note. The log always ends with exactly one newline, so recording a check does not itself fail `git diff --check`.
- `bin/work-commit <id> -m MSG` — commit the executor's work. `MSG` is required and must be a Conventional Commits subject (`type(scope): imperative summary`). Refuses any branch other than `work/<id>`, so executor work never lands on the base branch.
- `bin/work-resume <id>` — restart a cut-short run. Refuses when no worktree exists for `work/<id>`; otherwise re-renders the brief at the remaining open steps.
- `bin/work-accept <id>` — run the acceptance checks in a throwaway scratch copy and report pass or fail. Requires the base branch (`WORK_BASE_BRANCH`, default `main`), clean source and target checkouts including untracked files, every step closed, attention resolved and no terminal outcome. The scratch copy is detached at the exact captured source SHA; a pre-existing scratch path is preserved, never force-deleted. Never merges.
- `bin/work-finish <id>` — run the checks, fast-forward-only merge the tested `work/<id>` revision into the base branch, write `done.md`, and remove the worktree and branch without force. It applies the same preconditions as `work-accept` and verifies the source revision, target revision and target branch are unchanged across acceptance before merging. No staging deploy is run; finishing a card only merges to main, and production advances only via `bin/sift-promote` on the owner's word. `--reject "<reason>"` writes `rejected.md` and preserves the work branch, worktree and any uncommitted changes; nothing is merged or cleaned up. `--force` skips only the acceptance command, never readiness, cleanliness or preservation checks. Never pushes unless `WORK_PUSH=1`.

## The dispatch seam

Each command that hands off a prompt uses one seam. Inside a live session it prints the prompt and exits with code 97, so the session raises the agent itself. Run headless, it pipes the prompt to the command in `WORK_DISPATCH_CMD` (environment, or `bin/work.config`). That command is never hardcoded: set it to `claude -p`, an opencode invocation, or anything else. A live session is detected by `WORK_SESSION=1`, a known session marker, or a TTY; `WORK_HEADLESS=1` forces headless.

Configuration keys (environment overrides `bin/work.config`): `WORK_DISPATCH_CMD`, `WORK_ACCEPT_CMD` (default `bin/sift-accept` in this repo), `WORK_BASE_BRANCH` (default `main`), `WORK_ESTIMATE_CEILING` (default 25), `WORK_PUSH`.

## The executor's standing behaviour

Work only inside your worktree, on branch `work/<id>`, within the assigned stage. One step at a time: do the step, record its check with `work-note`, tick its box, then `work-commit`. Stop at the stage boundary and record the review/resume point even if other steps remain open. Never merge, never push, never raise sub-agents. On any stop, preserve incomplete work without marking it complete and leave a committed checkpoint so `work-resume` can pick it up. When every step in the card is closed, run `bin/work-test` once and stop; acceptance and merge are the coordinator's act. If a business question blocks you, write `question.md` and stop.

## Setup and tests

`bin/work-setup` wires a fresh clone: it sets `core.hooksPath` to the relative `.githooks` and makes the hooks and commands executable. `bin/work-test` runs the harness's own checks, including the isolated lifecycle tests under `bin/tests/`, and is the single test entry used by continuous integration and by `work-accept`.

## Current two-environment deploy loop

This section describes the existing commands. The target loop in the foundation plan adds automatic staging, isolated workers, explicit data refresh and a single owner-approved completion/release decision. Until those changes are implemented and verified, use the current commands deliberately; do not assume automatic preview or release exists.

This repository runs a two-environment deploy loop for the sifty application. Each environment owns its own compiled binary and its own frontend bundle; a staging deploy never touches the production binary or bundle.

**Environments:**
- Production — port 8080, data dir `<repo>/data`, binary `bin/sifty`, dist `sifty/dist`. Updated only by `bin/sift-promote` on the owner's explicit word.
- Staging — port 8081, data dir `<repo>/data-staging`, binary `bin/sifty-staging`, dist `sifty/dist-staging`. Previews an in-progress branch; not currently updated automatically. Workers are currently shared under `sifty/workers`.

**Commands:**
- `bin/sift-deploy --env staging|prod [--build-only]` — build the Go binary and React frontend for the named environment, back up the target env's data, stop the env's server, start the new binary detached, run a health and smoke check, and roll back the binary and data on any failure. Per-env config (port, data dir, binary, dist, log) lives in `bin/work.config`. Every run appends a log line to `<data_dir>/deploy.log`. When run from a linked worktree, staging artifacts are written to the main repo so they persist after the worktree is removed.
- `bin/sift-promote [--force]` — the only command that updates production. Verifies the current branch is `main`, then runs `bin/sift-deploy --env prod`. Run this on the owner's explicit confirmation after reviewing staging.
- `bin/sift-accept` — the acceptance command (`WORK_ACCEPT_CMD`). Always runs the harness self-test (`bin/work-test`). Application checks (`go test -race ./...`, `npm ci`, `npm run build` = `tsc --noEmit` + `vite build`) also run when the card changed `sifty/` or `app/` source, or build/deploy/config inputs (`bin/sift-*`, `bin/work.config`, `Makefile`, `config/`, `.github/`). A failed `npm ci` fails acceptance; there is no silent `npm install` fallback. The frontend build writes to a temporary directory so the scratch copy stays removable without force.

**Preview workflow:**
1. Create a worktree for the in-progress branch: `git worktree add .worktrees/<id> work/<id>`.
2. From that worktree, run `bin/sift-deploy --env staging`. This builds the branch's binary and bundle into the main repo's staging paths and restarts 8081.
3. The owner reviews the change at `http://localhost:8081`. Production on 8080 is unaffected.
4. `bin/work-accept <id>` runs `bin/sift-accept` in a throwaway scratch copy after the acceptance preconditions pass. It always runs the harness self-test and, for app/build-touching cards, the application suite.
5. `bin/work-finish <id>` fast-forward-merges the tested work revision into main and removes the worktree and branch without force. No deploy is run; staging and production are both untouched by finishing.
6. When satisfied with staging, the owner runs `bin/sift-promote` to advance production.

**Rollback:** `bin/sift-deploy` keeps the previous binary as `<binary>.prev` and a timestamped `store.sqlite.backup.<timestamp>` in the data dir. On any build, start, or smoke failure it restores both and restarts the old server.

**Staging data:** each staging deploy copies a consistent snapshot of the prod database (`sqlite3 <prod>/store.sqlite ".backup <staging>/store.sqlite"`) so the owner reviews new work against real-shaped data, without touching the live database.

## License

This project is source-available under the Elastic License 2.0. You may download, modify and self-host it for your own use. You may not offer it to third parties as a hosted or managed service. See `LICENSE` for the full terms.

## Not relied on

The mechanism depends on no Claude-Code-only feature: no PreToolUse gate, no slash command, no skill, no Agent tool, no `@`-import, and no hardcoded `claude -p`. A `CLAUDE.md`, where kept, is a thin twin pointing back to this file.
