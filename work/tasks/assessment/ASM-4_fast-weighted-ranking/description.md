# ASM-4 — Fast weighted ranking before LLM assessment

**Epic:** assessment
**Created:** 2026-11-05
**Status:** founded (plan.md written; awaiting "Execute the plan")

## Why this work exists

Owner's statement (2026-11-05): "Мы уберём начало оценки с помощью LLM в тот момент, когда вакансия открыта. До этого, оценка должна делаться весовыми коэффициентами, желательно через существующую библиотеку. Идея в том, что оценка должна быть очень быстрой. Пришло 100 или 300 новых вакансий — их нужно сразу же отранжировать, а не ждать 3 часа, пока LLM отработает."

Today ranking depends on the LLM assessment queue, which is slow: a batch of fresh roles can take hours before anything is scored, so a newly collected inbox is not immediately ordered by fit. This work gives every role an instant relevance score computed with BM25, so 100–300 roles arriving at once are ranked the moment they land. That score becomes the primary fit number and drives the "Best fit" sort. The LLM deep analysis is no longer what produces the first ranking: the mass queue is removed, and the LLM runs only for roles the owner selects, plus an explicit "assess now" button on any open role.

## Settled with the owner (2026-11-05)

- The fast score is BM25 text relevance via SQLite's built-in FTS5 `bm25()`, already compiled into the project's `modernc.org/sqlite` driver (verified by probe on 2026-11-05). No new dependency, no database change, no hand-rolled algorithm; BM25 is instant at this corpus size.
- The BM25 score is the primary 0–100 fit number shown in the list and used by the "Best fit" sort.
- The LLM assessment no longer runs as a mass queue on collection. It runs only when a role is moved to Selected, plus a manual "assess now" button on any open role. There is no on-open auto-start.

## Must not touch

The collection fetch/source internals, the stop-word/block filtering (COL-6, TRI-1/2), and the dedup/stage model. The LLM assessment logic itself (prompt, provider plumbing, the stored result shape) stays; only what triggers it changes.

## Notes

The remaining design choices are the plan author's to settle and are recorded in plan.md: what the BM25 query is built from (the candidate profile plus the configured search terms), how the raw BM25 score is normalised to 0–100, the new persisted `matchScore` field, and back-filling existing roles. Existing stored LLM assessments are kept and still shown; only auto-queuing stops.
