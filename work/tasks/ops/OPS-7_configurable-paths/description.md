# OPS-7 — Centralize all paths in configuration

**Epic:** ops
**Created:** 2026-11-06
**Author:** coordinator, based on the owner's request
**Status:** backlog; prepare the plan after the repository-structure relocation
**Kind:** idea

## Owner's request

«потом отдельно составь план по улучшению - это не гоже столько hardcoded хуйни в коде иметь»

«все должно задаваться где-то в конфигах»

«все пути»

Latest clarification: «Проверю remove и устраню зависимость от старых имён. - отдельной задачей, с выкорчевывание hardcoded paths из файлов и помещением всего в конфиги».

## Preliminary estimate

- **Complexity:** 8 (provisional): path consumers span backend, legacy code, build, deployment and work commands. Preparation should split independent migrations rather than hide them in one large implementation.
- **AI preparation:** 15–25 minutes to inventory consumers and propose the split.
- **AI implementation:** 45–90 minutes provisionally across the resulting bounded tasks; not authorized by this capture.
- **AI acceptance:** 15–30 minutes for builds, code review and real configured-path use.
- **Total AI elapsed:** 75–145 minutes, highly provisional until the path inventory. No new paid services proposed; monetary usage cost is unknown. Owner waiting time is excluded.

## Deliverable

Prepare a separate, evidence-based plan for putting all filesystem paths under configuration across application code, work-management commands, build, deployment, workers, tests and automation. Inventory literals and existing configuration, define path roots and overrides, identify tool-imposed entrypoint constraints, and describe a verified migration. The intended result is that moving a component or changing an environment directory requires changing configuration rather than repairing scattered code. Preserve relative-path portability, public-clone setup and production/staging isolation. The owner has requested planning; implementation requires approval of the resulting plan.

## Sequence

Finish the structural relocation in the active foundation card first. Prepare this card's plan against those final paths. This report does not switch the active implementation task.

## Acceptance criteria

- The separate plan identifies duplicated path settings and the configuration source each consumer will use.
- Defaults and overrides support a public clone, linked worktrees and both deployment environments.
- The owner reviews the complete plan before implementation begins.
