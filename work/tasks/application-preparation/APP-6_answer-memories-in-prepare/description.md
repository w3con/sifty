# APP-6 — Inject remembered answers into application preparation

**Epic:** application-preparation
**Created:** 2026-11-04
**Status:** founded
**Short card:** steps and acceptance criteria live here.

## Why this work exists

The owner can mark an application answer as remembered, and the backend stores it: `records.WriteAnswer` writes the file, `records.ReadAnswers` reads it back, and the HTTP routes `POST /api/jobs/{id}/application/remember` and `GET /api/jobs/{id}/application/memories` are wired. But when the Go backend drafts answers for a role's application form, it never loads those memories — `records.ReadAnswers` is called only in `selfcheck` and in the memories GET route, not in the preparation path (confirmed in the 2026-11-04 parity audit). So every draft is generated blind to what the owner has already confirmed, which defeats the purpose of remembering an answer. The retired TypeScript backend's `bindAnswerMemories` (`app/src/application-memory.ts`) injected the saved answers as owner-confirmed evidence immediately before answer generation; the Go port dropped that step.

This card restores it: remembered answers become available to generation again, so the owner does not re-answer the same question for every role.

## Steps

- [ ] 1.1 — In the preparation path (`internal/background/preparation.go`, `defaultPrepareJob`, before it calls `prepare.Prepare`), load the owner's remembered answers via `records.ReadAnswers`, apply the same scope/selection the TypeScript `bindAnswerMemories` used, and inject them as owner-confirmed-answer evidence sources into the context given to generation; check: a Go test runs a prepare over a role with one saved memory present and asserts the memory's text is among the evidence sources passed to `prepare.Prepare`, and that with no memory saved the evidence set is unchanged.

## Acceptance criteria

- Application answer generation has access to the owner's remembered answers: a saved memory is injected as owner-confirmed evidence before drafting, so a question the owner has already answered is drafted from that answer rather than from scratch. With no memory saved, behaviour is unchanged.

## Must not touch

- The on-disk format of answer records and the remember/memories HTTP routes (they already work).
- The assessment pipeline, the collection layer, and the answer-generation grounding rules: evidence must remain an exact quote, so this card adds a source without relaxing validation.
