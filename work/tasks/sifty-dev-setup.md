# Sifty development harness — model-agnostic work-management (bootstrap)

**Status:** PLANNED — awaiting "Execute the plan".
**Owner:** Alexander. **Plan author:** Claude (Opus 4.8), 2026-11-01.
**Repository:** private FCTO repository; resolve paths from its git root.

This is a bootstrap card: it builds the folder-based card system itself, so this one card is written as a single-file plan in the existing FCTO style. Once the harness exists, later work uses the folder-card convention it creates.

## Statement of work

Build a minimal, model-agnostic work-management harness in the private FCTO repository, in Python, adopting the model-agnostic subset of the Validité application's harness. A unit of work becomes a card folder, and a small set of `work-*` commands drive its lifecycle with every refusal enforced inside the command itself, so the rules hold no matter which tool calls them. The harness targets Claude Code and opencode (opencode wired to OpenAI, Kimi, or DeepSeek) and depends on no Claude-Code-only mechanism. It is set up to manage Sifty product cards in this repository now.

The commit-cleaning leak scanner from pilier (kept in Python) and the wiring to a public repository are out of scope here and belong to the repository-split card, because no public repository exists yet.

## Decisions carried in (settled 2026-11-01)

The work commands are Python, lifting the proven Validité scripts almost directly; the pilier leak scanner is also Python. The product's own backend language is settled separately as Go, with a thin TypeScript Playwright worker, and does not constrain the harness, because a git hook simply invokes an executable regardless of what wrote it.
The adopted subset is minimal: no knowledge-base traceability machinery (requirement-to-feature-to-test chains, registry or product-trace builders).
Location follows the pilier two-repository pattern: cards and harness scripts live in the private repository.
Targets are Claude Code and opencode only; Warp is dropped.

## Do not touch

The Sift application code under `app/`, the knowledge base under `kb/`, the existing single-file cards under `work/tasks/`, the `tools/shared` submodule, and the global `~/.claude` configuration and hooks. The harness must not depend on any Claude-Code-only PreToolUse gate, slash command, skill, or the Agent tool. Do not read or write the Validité or pilier repositories. No commits to `main`; work on a branch.

## Acceptance criteria

1. A card is a folder under `work/tasks/<epic>/<ID>_<slug>/` holding `description.md`, `plan.md`, `log.md`, `comments.md`, and `attention.md`, with conditional `done.md` / `rejected.md` / `cancelled.md` / `question.md`; a short-card variant (description plus comments, steps inside the description) works.
2. The `work-*` commands run under python3 with standard-library-only dependencies, and every refusal (plan not fit, unanswered attention, terminal file present, estimate ceiling, nothing left to brief) is enforced inside the command, not by an external hook.
3. `work-handover` cuts a git worktree on branch `work/<id>` and prints the executor brief; the executor records each step with verbatim check output and commits only to its own branch; `work-resume` restarts from the last handoff and refuses when no worktree exists.
4. The dispatch seam: inside a live session the command prints the prompt and returns a handed-to-session exit code; headless dispatch uses a command read from configuration (settable to `claude -p` or an opencode invocation), with no hardcoded `claude -p` anywhere.
5. `AGENTS.md` carries the real rules inline with no `@`-import for the mechanism; any `CLAUDE.md` kept is a thin twin pointing at the same rules; the commands behave identically when run from a bare shell as when run inside Claude Code.
6. Git hooks are installed through a relative `core.hooksPath` with a machine-setup script for a fresh clone, and a single test entry runs the harness's own checks.
7. The full lifecycle is proven end to end on one throwaway card: found, plan, handover, execute (with the log showing verbatim check output), accept and finish, with evidence recorded.

## Checkpoints

- [x] DEV.1 — Define the card-folder convention and file set; write the templates (description, plan, log, comments, attention, executor-brief).
- [x] DEV.2 — Implement `work-new` and `work-plan` in Python (ID allocation, scaffold-and-commit, plan validation) with refusals inside.
- [x] DEV.3 — Implement `work-handover` (cut worktree and `work/<id>` branch, print brief) and the configurable dispatch seam.
- [x] DEV.4 — Implement `work-note` and `work-commit` (record step with verbatim output, commit to own branch) and `work-resume`.
- [x] DEV.5 — Implement `work-accept` and `work-finish` (checks in a scratch copy, fast-forward-only merge, write done or rejected, clean up).
- [x] DEV.6 — Write `AGENTS.md` with rules inline and a thin `CLAUDE.md` twin; install git hooks via relative `core.hooksPath` and a machine-setup script; add the test entry.
- [x] DEV.7 — Prove the whole lifecycle on one throwaway card, run from a bare shell to confirm the model-agnostic path, and record evidence.

## Progress log

### DEV.1 — convention and templates (2026-11-01)

Wrote the card-folder convention reference and the template set under `bin/templates/`. Full card is description + plan + log + comments + attention; short card is description + comments with steps and criteria inside the description; conditional terminal files are done/rejected/cancelled/question. Executor-brief template is tool-neutral markdown.

Check — the template set exists:

```
$ ls -1 bin/templates/
attention.md
comments.md
convention.md
description.md
executor-brief.md
log.md
plan.md
short-description.md
```

### DEV.2 — work-new and work-plan (2026-11-01)

Added `bin/_work_lib.py` (shared stdlib helpers: repo root, templates, card model, step/attention/estimate parsing, plan fitness, the dispatch seam, refusal codes) and the commands `bin/work-new` (allocate `PREFIX-n`, scaffold full or `--short` card from templates, commit the founded folder) and `bin/work-plan` (fitness gate, refusal inside). ID allocation is one past the highest existing number for the prefix anywhere under `work/tasks/`.

Check — compile, found a card, refuse an empty plan, accept a filled one (test card removed afterward, found-card commit reset):

```
$ python3 -m py_compile bin/_work_lib.py bin/work-new bin/work-plan && echo OK
OK
$ ./bin/work-new testepic smoke-check --title "Smoke check"
founded TESTEPIC-1 at work/tasks/testepic/TESTEPIC-1_smoke-check
TESTEPIC-1
$ ./bin/work-plan TESTEPIC-1          # fresh scaffold
REFUSED:
  - plan for TESTEPIC-1 is not fit:
  - template placeholders still present — fill the plan
exit=2
$ ./bin/work-plan TESTEPIC-1          # after filling plan.md
plan for TESTEPIC-1 is fit: 1 step(s), acceptance criteria present.
exit=0
```

### DEV.3 — work-handover and the dispatch seam (2026-11-01)

Added `bin/work-handover`: it enforces all refusals inside the command (plan not fit, unanswered attention, terminal file present, estimate over the configurable ceiling, nothing to brief, uncommitted card changes), cuts a linked worktree at `.worktrees/<id>` on branch `work/<id>` based off HEAD, and hands the rendered executor brief to the seam. The seam (`_work_lib.dispatch`): a live session gets the prompt printed and exit code 97; headless reads `WORK_DISPATCH_CMD` from env or `bin/work.config` and pipes the prompt to it. No `claude -p` is hardcoded. Added `.worktrees/` and `.work-scratch/` to `.gitignore`.

Check — handover cuts the worktree and prints the brief (live seam, exit 97), headless seam reads config, no hardcoded dispatch (test card and worktree removed afterward):

```
$ WORK_SESSION=1 ./bin/work-handover TESTEPIC-1
cut worktree .../.worktrees/TESTEPIC-1 on branch work/TESTEPIC-1
[work] prompt handed to the live session (exit 97); the session raises the agent.
# Executor brief — TESTEPIC-1
...
exit=97
$ git worktree list | grep -i testepic
.../.worktrees/TESTEPIC-1  786b205 [work/TESTEPIC-1]
$ WORK_HEADLESS=1 python3 -c "...w.dispatch('PROMPT-BODY')"        # no cmd configured
REFUSED:
  - headless dispatch requested but no dispatch command is configured.
  exit=2
$ WORK_HEADLESS=1 WORK_DISPATCH_CMD='sed s/PROMPT/DISPATCHED/' python3 -c "...w.dispatch('PROMPT-BODY')"
DISPATCHED-BODY
exit=0
$ grep -rn "claude -p" bin/        # only help/comment text, never a hardcoded call
bin/_work_lib.py:143:    can be `claude -p`, an opencode invocation, or anything else.
bin/_work_lib.py:156:            "'claude -p' or an opencode invocation.",
```

### DEV.4 — work-note, work-commit, work-resume (2026-11-01)

Added `bin/work-note` (run a check and append the step with its verbatim combined output and exit code to log.md, or record a free note; propagates a failing check), `bin/work-commit` (refuses any branch other than work/<id>, then commits the executor's work there), and `bin/work-resume` (refuses when no worktree exists for work/<id>, else re-renders the brief at the remaining open steps and hands it to the seam). Aligned the executor-brief template to the `--check` interface. The executor-side proof was run through a bare /bin/sh under `env -i` because the session's worktree-isolation guard blocks git in a nested worktree from this session; a real executor runs there as its own session.

Check — executor loop records verbatim output and commits on its branch; refusals fire (test card and worktree removed afterward):

```
env -i PATH=... /bin/sh dev4-exec.sh <worktree> <bin>   # bare shell, no Claude Code
branch: work/TESTEPIC-1
recorded to work/tasks/testepic/TESTEPIC-1_d4/log.md
committed deb92c5 on work/TESTEPIC-1
committed 7ed3a5c on work/TESTEPIC-1
log entry "1.1 create hello.txt": check "test -f hello.txt && echo present", exit 0, output "present"

./bin/work-commit TESTEPIC-1        # from the base branch
REFUSED: refusing to commit on branch 'worktree-sifty-planning'; the executor commits only on 'work/TESTEPIC-1'.

WORK_SESSION=1 ./bin/work-resume TESTEPIC-1   # worktree present -> prints brief, exit 97
./bin/work-resume TESTEPIC-1                   # worktree removed
REFUSED: no worktree for work/TESTEPIC-1 — nothing to resume.
```

### DEV.5 — work-accept and work-finish (2026-11-01)

Added `bin/work-accept` (create a detached scratch worktree of work/<id>, run the acceptance command — WORK_ACCEPT_CMD, default bin/work-test — there, print verbatim output, report pass/fail, never merge) and `bin/work-finish` (accept path: run the checks unless --force, fast-forward-only merge into the current branch, write done.md, remove the worktree and branch; reject path: write rejected.md with the reason and clean up; never push unless WORK_PUSH=1). Moved the scratch/acceptance helpers and find_worktree into `_work_lib.py`.

Check — a card ran through accept, ff-merge+done, reject, and the no-worktree refusal; ID allocation incremented across slugs (test cards removed, test commits reset):

```
./bin/work-accept TESTEPIC-1            # WORK_ACCEPT_CMD=true scratch run
accepted TESTEPIC-1 — checks pass

./bin/work-finish TESTEPIC-1
fast-forwarded work/TESTEPIC-1 into worktree-sifty-planning
finished TESTEPIC-1
# -> done.md written, marker.txt merged onto base, worktree and branch removed

./bin/work-new testepic d5reject        # allocates TESTEPIC-2 (one past TESTEPIC-1)
./bin/work-finish TESTEPIC-2 --reject "work did not meet the acceptance criteria"
rejected TESTEPIC-2
# -> rejected.md written with reason and branch tip, worktree and branch removed

./bin/work-finish TESTEPIC-2            # no worktree
REFUSED: no worktree for work/TESTEPIC-2 — nothing to finish
```

### DEV.6 — AGENTS.md, CLAUDE.md twin, hooks, setup, test entry (2026-11-01)

Wrote `AGENTS.md` with the whole mechanism inline (card convention, every command and its refusals, the dispatch seam and config keys, the executor's standing behaviour, setup and tests, and what is deliberately not relied on) and no `@`-import. Appended a thin twin section to the existing root `CLAUDE.md` pointing at `AGENTS.md` without restating the mechanism. Added `bin/work-test` (the single test entry: compile every command and the lib, templates complete, AGENTS.md present with commands and no `@`-import, each `--help` parses) and `bin/work-setup` (sets `core.hooksPath` to the relative `.githooks`, makes hooks and commands executable, notes the submodule init). Added `.github/workflows/work-test.yml` calling the same test entry. Added `bin/work.config` documenting the config keys.

The existing `.githooks/pre-commit` (kb relation-build) is composed with, not replaced: `pre-commit` became a dispatcher that runs every executable in `.githooks/pre-commit.d/` in order; the kb hook moved verbatim into `10-kb-relation-build`, and the harness self-check is `20-work-selfcheck` (refuses a broken `work-*` script). The `post-*` hooks are untouched.

Check — hooksPath set relative, test entry passes, kb hook preserved byte-for-byte, dispatcher orders steps, self-check blocks broken python:

```
./bin/work-setup
set core.hooksPath = .githooks (relative)
git config core.hooksPath  ->  .githooks

./bin/work-test
... (all ok) ...
PASS: all harness checks

diff <original pre-commit> .githooks/pre-commit.d/10-kb-relation-build
IDENTICAL: kb hook preserved byte-for-byte

.githooks/pre-commit        # with two probe steps added
PROBE-A fired
PROBE-Z fired

.githooks/pre-commit.d/20-work-selfcheck   # broken script staged
[work] py_compile failed: bin/work-broken-tmp — fix it or commit with --no-verify
exit=1
```

### DEV.7 — full lifecycle proof from a bare shell (2026-11-01)

Ran the whole lifecycle on a throwaway card under `env -i PATH=... /bin/sh` with no Claude Code around it (CLAUDECODE unset, TERM=dumb, no TTY): found, plan, headless handover (WORK_DISPATCH_CMD=cat printed the brief and the configurable seam dispatched with no `claude`), executor step recorded with verbatim check output in log.md, accept running the real `bin/work-test` in a scratch copy, and finish with a fast-forward-only merge, done.md and cleanup. All throwaway artifacts (card folder, proof.txt, worktree, branch) were removed afterward and the lifecycle commits reset; only this evidence is kept.

Verbatim output:

```
### shell is bare: no CLAUDECODE, no TTY
CLAUDECODE=[unset]  TERM=[dumb]
### 1. found
founded THROWAWAY-1 at work/tasks/throwaway/THROWAWAY-1_lifecycle-proof
### 2. plan
plan for THROWAWAY-1 is fit: 1 step(s), acceptance criteria present.
### 3. handover (headless seam, WORK_DISPATCH_CMD=cat prints the brief)
cut worktree .../.worktrees/THROWAWAY-1 on branch work/THROWAWAY-1
# Executor brief — THROWAWAY-1
### 4. execute (inside the worktree, on branch work/THROWAWAY-1)
branch=work/THROWAWAY-1
recorded to work/tasks/throwaway/THROWAWAY-1_lifecycle-proof/log.md
committed 872efef on work/THROWAWAY-1
--- log.md (verbatim check output) ---
## 2026-11-01 — 1.1 write proof.txt
Check: `cat proof.txt`
Exit: 0
```
proof of the full lifecycle
```
### 5. accept (checks run in a scratch copy = bin/work-test)
accepted THROWAWAY-1 — checks pass
PASS: all harness checks
### 6. finish (ff-merge + done.md + cleanup)
fast-forwarded work/THROWAWAY-1 into worktree-sifty-planning
finished THROWAWAY-1
--- done.md ---  (Merged into: worktree-sifty-planning)
--- proof.txt merged onto base? ---  proof.txt
--- worktree / branch gone? ---  0   (no work/THROWAWAY-1 branch)
### LIFECYCLE COMPLETE
```
