# OPS-11 — brief for feature specification writers

Owner, 2026-11-13: «да, переделывай. Еще момент - спеки мы пишем для фичей, не для задач. Т.е. например для "оценка fit" или "сбор вакансий" или "отображение информации о вакансии", или "подготовка отклика". Т.е. по сути скорее для эпиков. Не для кнопок или транзакций»

## What a specification is here

A specification states how one product feature must behave: its purpose, its rules, what the owner gets from it. It is agreed by the owner and is the reference the code is checked against; where the code differs, the code has a defect or the owner has not decided yet. It is written at the level of the feature (an epic), not of buttons, endpoints, tables or single operations. It does not describe code structure.

## Sources

- The as-built description of the area in `/private/tmp/claude-501/-Volumes-SSD-Dev-tknff-sifty/92c828a0-c188-40b2-b55c-12af52f00517/scratchpad/specs/` (what the code does today, with code references and contradictions). Material, not the answer.
- The owner's decisions: the cards of the feature under `work/tasks/` in the main checkout `/Volumes/SSD/Dev/tknff/sifty` (owner's words, approvals, review notes), `docs/MAP.md`, the existing specs in `docs/specs/`, and the OPS-11 inventory `.../scratchpad/specs/inventory.md`.
- The code only to confirm a status, never as the source of a rule.

## Format

File: `/private/tmp/claude-501/-Volumes-SSD-Dev-tknff-sifty/92c828a0-c188-40b2-b55c-12af52f00517/scratchpad/specs-v2/<name>.md`. English Markdown; every paragraph on one physical line; roles ("owner"), never a person's name. Length: what the feature needs, typically one to three pages; no history narration, no task chronology.

```
# <Feature> — specification

Status: draft by OPS-11, 2026-11-13; not yet approved by the owner.

## 1. Purpose
What the owner achieves with this feature and why it exists.

## 2. Scope
What belongs to this feature and what belongs to which neighbouring spec (links).

## 3. Rules
Numbered rules of intended behaviour, grouped by topic. Each rule ends with one status tag:
- [implemented] — optionally one code reference as evidence;
- [not implemented — CARD-ID];
- [deviates — what the code does instead, one code reference];
- [owner decision needed] — the code does something no owner decision covers, or cards disagree; the rule is written as the writer's best reading and the question goes to section 5.
A rule is a statement of outcome ("A vacancy the owner deleted is never collected again"), not a description of a screen or call.

## 4. Configuration
What the owner can configure for this feature and where (file and key, or screen).

## 5. Questions for the owner
Numbered, each one sentence, each with the options the cards or the code suggest.

## 6. Open cards
Cards that will change this feature, one line each: ID — what it changes.
```

## Boundaries

Write only your one file. No git command that changes anything, no server, no build, no model call, no network. Read `.worktrees/*` only. Do not raise sub-agents. Write the file early and extend it section by section.

Report back in at most 12 lines: file path, number of rules by status, the questions for the owner (one line each).
