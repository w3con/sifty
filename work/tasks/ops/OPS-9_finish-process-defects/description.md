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
