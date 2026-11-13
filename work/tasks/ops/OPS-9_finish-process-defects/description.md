# OPS-9 — Work process defects found at the ASSESSMENT-3 finish

**Epic:** ops
**Created:** 2026-11-11
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Про карточку OPS ответа пока нет. В неё вошли бы:

* Заводи. 

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Scope agreed with the owner (2026-11-11)

The owner answered «Заводи.» to this list (coordinator's wording, from the ASSESSMENT-3 finish report):

1. `work-finish` looks for the card in the main checkout and at the same time requires the main checkout to be clean; a card that lives only on its branch cannot be finished without a workaround commit (ASSESSMENT-3 needed f64a972).
2. A dead PID in `.local/deploy/prod/server.pid` after production was stopped blocks the next release: the release treats it as a foreign process, the stop path as stale.
3. Card readiness (acceptance criteria heading, open steps, unresolved attention items) is checked only inside `work-finish`; it must be checkable before the owner is asked to approve a release.
4. There is no path to record as done a card that was released outside `work-finish` (ASM-6, released 2026-11-10 with `sift-promote`); `work-finish` would squash and release again, and nothing else may write `done.md`.
5. Four tests fail on main independently of any card: `TestMigrationV14`, `TestMigrationV16ToV17PreservesData`, `TestSchema_FreshDatabase` (stale user_version constants), `TestRouteDrift` (profile, avatar and new skill routes missing from fixtures).

Also found: `bin/work-finish --reject` run outside the owner's machine fails on the absolute worktree path (it reads `/Volumes/...`); the coordinator's bridge cannot run it.

## Estimate (preliminary)

Complexity 5, AI time 2–3 h (1–4 in the work tooling with one targeted test each; 5 fixture and constant updates). Touches `bin/` and `backend` tests only; no product behaviour, no release needed except for the tooling itself.

## Added 2026-11-12

6. `bin/work-note --check` records "Exit: 0" for a command line with a pipe even when the tests failed (the pipe's last command decides); executors have to correct it by hand. Use `set -o pipefail` or record the first failing status.
7. Executors used `git stash` on the shared stack of the main repository (SETTINGS-7); the owner keeps two entries there. Work tooling should not need the shared stash; say so in the executor rules or give `work-*` a private checkpoint.
8. A build from a worktree (`make build` / `sift-deploy --env prod`) writes the main checkout's production artifacts (`bin/sifty`, `frontend/dist`); this happened from INBOX-2 on 2026-11-11/12 (INBOX-2 attention.md). A worktree must not be able to build into production paths.
9. Production started by hand with `make run` is not recorded in `.local/deploy/prod/server.pid`; the release adapter then sees a foreign process on 8080 (related to item 2).

## Found place (2026-11-12, from the SETTINGS-7 release)

Items 2 and 9 are one defect in `bin/_deploy_lib.py`. `probe_port_owner` (line 647) reads `server.pid` and, when the recorded process is not alive, raises "recorded prod server pid N does not match ...; refusing to kill it" (lines 655-658), although nothing is running and the port is free. `stop_server` (line 667) handles the same state correctly: a dead pid makes it clear the record (line 678) and continue. The release path goes through `probe_port_owner`, so a stale record blocks the release while the stop path would have accepted it. Item 9 is how the record goes stale: production started by hand with `make run` (here pid 49599 from 11:50) is never written to `server.pid`, which keeps the pid of the previous deploy (21230); after the owner stops that process, the record is dead and the next release is refused. Observed twice (ASSESSMENT-3 finish, SETTINGS-7 finish).

Workaround used on 2026-11-12: after checking that the pid was dead and port 8080 was free, `.local/deploy/prod/server.pid` was renamed to `server.pid.stale-21230` (the same naming as the older `server.pid.stale-71548`), and the release was run again. Suggested fix: in `probe_port_owner`, treat a dead pid like `stop_server` does (clear the record, then require the port to be free) and keep the refusal only for a live process that does not match.
