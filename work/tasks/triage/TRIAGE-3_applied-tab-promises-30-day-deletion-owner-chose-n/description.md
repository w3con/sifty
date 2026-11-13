# TRIAGE-3 — Applied tab promises 30-day deletion; owner chose never to delete automa

**Epic:** triage
**Created:** 2026-11-13
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
по 7 - (б) - тогда нужно добавить возможность удалить сообщения в shortlist и archive. Кнопка?
---
Карточки заводить? (одна про обещание автоудаления) - да
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator

Coordinator (2026-11-13): the words above are the owner's answer to question 7 of the OPS-11 combined questions and the owner's «да» to capturing this card («да и да»). Decision: applied vacancies are never deleted automatically. Work: remove the Applied tab text promising permanent removal after 30 days (`frontend/main.tsx:168`), remove the unused `PurgeArchive` and the retention constant, re-plan GO-7 (its step 1.2 is superseded). The owner's question about a delete button is answered: "Delete role…" already exists in Shortlist and both Archive tabs. Spec `docs/specs/triage.md` rule 25. Not planned, not estimated.
