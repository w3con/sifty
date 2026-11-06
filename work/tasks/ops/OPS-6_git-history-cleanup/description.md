# OPS-6 — Review and simplify the Git history

**Epic:** ops
**Created:** 2026-11-06
**Status:** backlog; preparation not started
**Kind:** idea

## Why this work exists

Owner request: «отдельная задача - надо посмотреть на git log и видимо как-то compact его - там жопа какая-то».

Review the actual history and propose a more readable commit/release history. Determine which commits have already been published and which are still local, what other branches/worktrees depend on them, and how task checkpoint evidence will remain recoverable. The request records a separate task; it does not select or authorize a history rewrite.

## Acceptance criteria

- A concrete cleanup proposal is grounded in the actual branch/publication state and explains the resulting history.
- The owner approves the method and affected revisions before any history rewrite or remote update.
- Existing work and recovery/checkpoint evidence remain recoverable.
