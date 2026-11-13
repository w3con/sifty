# TRIAGE-4 — Small interface fixes: N and A keys on any layout, Not suitable reason,

**Epic:** triage
**Created:** 2026-11-13
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Заводить карточки? ... одну мелкую про интерфейс: клавиши N и A на любой раскладке, вопрос «почему» у «Not suitable», подпись под поисковыми выражениями - да
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator

Coordinator (2026-11-13): the words above restate the coordinator's proposal ("..." marks the coordinator's shortening) with the owner's answer «да и да». Three small items found by OPS-11: the N and A shortcuts depend on the keyboard layout (Y was fixed by TRIAGE-2; use the physical key); "Not suitable" does not ask why (owner's remark in APP-8 comments: «она даже не спрашивает почему "not suitable"»); the caption under search expressions says they "apply to the next collection run", which is not true (they only rank until SETTINGS-5 S.2). Specs `docs/specs/triage.md`, `docs/specs/catalogue.md`. Not planned, not estimated.
