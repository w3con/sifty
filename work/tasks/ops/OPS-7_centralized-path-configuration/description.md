# OPS-7 — Centralize configured paths

**Epic:** ops
**Created:** 2026-11-06
**Status:** backlog; separate plan requested, implementation not authorized

## Why this work exists

Owner's words: «потом отдельно составь план по улучшению - это не гоже столько hardcoded хуйни в коде иметь»; «все должно задаваться где-то в конфигах»; «все пути».

After the repository foundation, inventory path definitions across build, runtime, workers, deployment and work-management commands. Prepare one improvement plan identifying the configuration source, defaults, overrides, consumers and compatibility checks. This is separate preparation, not a new refactor during the foundation task.

## Acceptance criteria

- The separate plan identifies duplicated path settings and the configuration source each consumer will use.
- Defaults and overrides support a public clone, linked worktrees and both deployment environments.
- The owner reviews the complete plan before implementation begins.
