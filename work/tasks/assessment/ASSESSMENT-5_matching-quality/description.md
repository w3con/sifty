# ASSESSMENT-5 — Matching quality: gold set and method choice

**Epic:** assessment
**Created:** 2026-11-11
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
да, потому тебе и пишу-  кажется все это вышло из под контроля. нам нужны новые карточки, обновления specs.

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Context (coordinator, 2026-11-11)

Grew out of ASSESSMENT-3 Q.7/Q.7b. Record of the research conversation between the owner and the executor: `work/tasks/assessment/ASSESSMENT-3_two-fit-scores/q7b-conversation.md` (on branch work/ASSESSMENT-3 until merged). Owner's position there (as recorded by the executor): matching quality is unacceptable (three false matches of four on Growth Lead); the first line must be free, fast and good; a hosted model (Haiku) only for the shortlist (ASSESSMENT-6) and for labelling. Measured: model size does not fix it (9B, Qwen3.5 35B, Qwen3.6 35B-A3B agree on 73–79 % of verdicts, no winner in a hand check); a shared-word rule does not separate true from false matches (140 of 340 true matches share no word).

Spec: `docs/specs/quick-match.md` §14.13–14.14.
