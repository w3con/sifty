# AGENTS.md — work-management harness

This file is the cross-tool source of truth for how work is managed in this repository. It holds the rules inline, not behind an import, so any tool reads them: Claude Code, opencode, or a plain shell. The rules below are enforced inside the `bin/work-*` commands themselves, so they hold no matter which tool runs them.

Roles: the owner states why a card exists; the coordinator plans it and accepts the result; the executor builds it.

## A card is a folder

A unit of work is a card: a folder `ai/tasks/<epic>/<ID>_<slug>/`. The file set and the short-card variant are defined in `bin/templates/convention.md`. Steps are `- [ ]` (open) or `- [x]` (closed) lines in `plan.md`, or in `description.md` for a short card. The card is markdown, so it survives any session or tool dying.

## Lifecycle commands

All commands are `python3`, standard-library only, run from anywhere in the repository.

- `bin/work-new <epic> <slug> [--prefix P] [--title T] [--short]` — allocate the next `PREFIX-n` ID, scaffold the file set from the templates, and commit the founded folder.
- `bin/work-plan <id>` — validate that the plan is fit to hand over. Refuses when there are no steps, no acceptance criteria, or template placeholders remain.
- `bin/work-handover <id>` — the gate to the executor. Refuses when the plan is not fit, attention is open, a terminal file is present, the declared estimate exceeds the ceiling, nothing is left to brief, or the card has uncommitted changes. Then cuts a worktree at `.worktrees/<id>` on branch `work/<id>` and hands off the executor brief.
- `bin/work-note <id> --step "<name>" --check "<command>"` — run the check and append the step with its verbatim output and exit code to `log.md`. `--message "<text>"` records a free note.
- `bin/work-commit <id> -m MSG` — commit the executor's work. `MSG` is required and must be a Conventional Commits subject (`type(scope): imperative summary`). Refuses any branch other than `work/<id>`, so executor work never lands on the base branch.
- `bin/work-resume <id>` — restart a cut-short run. Refuses when no worktree exists for `work/<id>`; otherwise re-renders the brief at the remaining open steps.
- `bin/work-accept <id>` — run the acceptance checks in a throwaway scratch copy and report pass or fail. Never merges.
- `bin/work-finish <id>` — run the checks, fast-forward-only merge `work/<id>` into the current branch, write `done.md`, and remove the worktree and branch. `--reject "<reason>"` writes `rejected.md` and cleans up without merging. `--force` skips the acceptance check. Never pushes unless `WORK_PUSH=1`.

## The dispatch seam

Each command that hands off a prompt uses one seam. Inside a live session it prints the prompt and exits with code 97, so the session raises the agent itself. Run headless, it pipes the prompt to the command in `WORK_DISPATCH_CMD` (environment, or `bin/work.config`). That command is never hardcoded: set it to `claude -p`, an opencode invocation, or anything else. A live session is detected by `WORK_SESSION=1`, a known session marker, or a TTY; `WORK_HEADLESS=1` forces headless.

Configuration keys (environment overrides `bin/work.config`): `WORK_DISPATCH_CMD`, `WORK_ACCEPT_CMD` (default `bin/work-test`), `WORK_ESTIMATE_CEILING` (default 25), `WORK_PUSH`.

## The executor's standing behaviour

Work only inside your worktree, on branch `work/<id>`. One step at a time: do the step, record its check with `work-note`, tick its box, then `work-commit`. Never merge, never push, never raise sub-agents. On any stop, leave the branch committed so `work-resume` can pick it up. When every step is closed, run `bin/work-test` once and stop; acceptance and merge are the coordinator's act. If a business question blocks you, write `question.md` and stop.

## Setup and tests

`bin/work-setup` wires a fresh clone: it sets `core.hooksPath` to the relative `.githooks` and makes the hooks and commands executable. `bin/work-test` runs the harness's own checks and is the single test entry used by continuous integration and by `work-accept`.

## Not relied on

The mechanism depends on no Claude-Code-only feature: no PreToolUse gate, no slash command, no skill, no Agent tool, no `@`-import, and no hardcoded `claude -p`. A `CLAUDE.md`, where kept, is a thin twin pointing back to this file.
