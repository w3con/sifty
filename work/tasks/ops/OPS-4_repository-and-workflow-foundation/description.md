# OPS-4 — Repository structure and development workflow

**Epic:** ops
**Created:** 2026-11-06
**Author:** coordinator, based on the owner's conversation
**Status:** in progress; steps 1.1–4.3 implemented on `work/OPS-4`; coordinator review of phase 4 pending

## Why this work exists

The repository mixes the current Go application, its React frontend, a legacy TypeScript implementation, project planning and shared agent tooling. The development workflow must preserve incoming requests without starting them, make the effect of a proposed change visible before implementation, and show ongoing work automatically on staging. The owner acts as product owner; the coordinator prepares work, challenges inconsistencies and accepts technical results; an executor implements agreed work in its own working copy. A role does not require a particular model or a separate process.

## Owner's statements

- Structure: «нужны папки /frontend - где живет фронт, /backend, где живет back. /mcp - для будущего mcp.»
- Approval of the structural direction: «структуру утвердили.»
- Incoming work: «если я вижу какой-то баг - я тебе сообщаю, но ты это записываешь в бэклог (если фича) или трекер багов, но не бросаешься его сразу делать.»
- Planning: «Что есть план, с эпиками, с roadmap, с описанием задач - и мы готовим задачи и делаем их.»
- Impact analysis: «еще нужна карта работ/требований/зависимостей» and «Мне нужно, чтобы ты работал как хороший коллега-программист, а я как продукт, по сути.»
- Environments: «идея была сделать версию, где все изменения в процессе разработки сразу отражаются, и ты даже не спрашиваешь меня - обновлять или нет. в main/prod переезжает, когда я утверждаю (закрываю) работу.»
- Local incident record: «incidents.md ... это пусть будет в gitignore - репо публичный, этого там быть не должно».
- Shared hooks: «это хуки единые на все проекты - они лежат в папке /Dev/ai - и общие. работают в других проектах через symlinks».
- Cost: «вопрос расхода токенов» and «могу подключить deepseek v4.1 для имплементации задач».
- Executor decision: «испльзуй тогда для кодинга субагентов на OpenCode GO / DeepSeek v4.1Flash - у меня есть подписка.» and «Ты остаешься для планированипя задач/приемки/проверки сделанного.»
- Acceptance fixes: «давай улучшим»; GitHub workflow placement: «почини»; present deliverable: «напишешь себе план?».
- Full-plan approval and cleanup: «план норм, но не забудь почистить потом старые фолдеры. исполняй.»

## Approved scope

Deliver the agreed top-level structure, a small tool-neutral work tracker with requirement-to-code links, safe acceptance and release commands, and continuously updated staging. Use four task states: backlog, in progress, review and done. Preparation is content in a backlog card; blocking is a reason attached to a nonterminal task; cancellation is an archived outcome distinct from successful completion. The owner selected coding subagents on OpenCode Go / DeepSeek V4.1 Flash; the coordinator retains planning, review and technical acceptance. Verify this split on a bounded real-work stage without silently switching models. Keep shared hooks shared and adapt this project's integration rather than altering global behavior. Document the proposed separation of runtime data and register its migration as a distinct task; this foundational task does not relocate live user files or split the database. The owner approved the complete coordinator-authored plan and added cleanup of obsolete directories after verified relocation. Preserve retained legacy code, runtime data, backups and unfinished work during that cleanup.

## Must not touch

- Vacancy collection, ranking, preparation and user-facing behavior, except the agreed development and release workflow.
- Production data, candidate evidence and external knowledge bases during structural relocation; a release may use only the normal verified deployment path after owner acceptance.
- The contents of the retired TypeScript implementation beyond relocation and necessary path repairs; incomplete parity still prevents deletion.
- Unrelated active worktrees, unfinished work and existing branch history.
- Shared tooling implementation, global Claude/OpenCode settings or another project's hooks; any necessary upstream change is a separately described follow-up.
- License terms: Elastic License 2.0 remains authoritative.

## Sources

- `AGENTS.md`, `CLAUDE.md`, `Makefile`, `.gitmodules`, `.gitignore`, `.githooks/`, `bin/work-*`, `bin/_work_lib.py`, `bin/sift-*`, `bin/work.config` and `bin/templates/`.
- `work/ROADMAP.md`, `docs/product/MAP.md`, `work/backlog.md`, `docs/architecture/{backend-architecture,backend-api-contract,go-parity-audit,dev-setup-notes}.md` and existing task cards.
- `docs/architecture/data-directory.md`, `backend/internal/dirs/dirs.go`, candidate/record storage code, frontend configuration and nested GitHub workflow.
- Shared hook sources and local Git hook configuration; model catalog and authentication-provider names, without copying credentials.

## Notes

- The older product map/backlog still describe a private repository, MIT licensing and a future public split. These are historical statements, not the current requirements. Preserve their history while making the current entry points unambiguous.
- `tools/shared` is a Git submodule; `tools/bin`, `tools/harness` and `tools/templates` are tracked symlinks. This project's `.githooks/*` are tracked regular files, and `core.hooksPath` is unset in the inspected checkout. Global Claude hooks and project Git hooks are different integration points.
- The full checkpoints and acceptance criteria live in `plan.md`. Execution evidence belongs in `log.md` until the new tracking convention is implemented; this card must remain resumable across its own migration.
