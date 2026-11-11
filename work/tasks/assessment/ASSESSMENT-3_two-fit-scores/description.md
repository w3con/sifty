# ASSESSMENT-3 — Two fit scores with model-extracted requirements

**Epic:** assessment
**Created:** 2026-11-11
**Status:** backlog
**Kind:** task

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Мы не это решили делать. должно было быть 2 оценки.
списки по разделам. что у тебя получилось?
```

```text
1. Для исправления запускай Sonnet 5.5
2. Дай ему полное задание, возможно по шагам - спланируй все сам, что должно быть сделано (без имплементациИ). можешь создать новую карточку.
3. сколько вакансий в день - сколько угодно. сейчас их обрабатывает локальная модель, токенов не стоит (но проверь это).
4. какие-то еще вопросы?
```

## Why this work exists

ASSESSMENT-2 was rejected by the owner. On its own review data (2 777 scored vacancies): the second score was `not_verified` for every vacancy, requirements were extracted for only 45 vacancies, 2 310 goal scores rested on a single title comparison (a title match alone gave 100), 280 vacancies were closed as "no extraction needed" although their text held requirements, and the panel showed queue mechanics instead of facts. The detailed rejection record is ASSESSMENT-2's `discussion-handoff.md` in `.worktrees/ASSESSMENT-2/`.

The owner needs, for every vacancy in the catalogue:

1. **Fits you** — does the job match the owner's goals (role, pay, location/work mode, conditions).
2. **You fit** — do the owner's confirmed skills, experience and eligibility meet the published requirements.

Each with short factual lists by section, so the owner can triage the Inbox.

## Owner decisions (2026-11-11)

- **Fits you:** an unknown goal group counts as the midpoint (0.5) instead of being dropped. A title-only match can no longer reach 100; unknown facts never push a role to the bottom.
- **You fit:** one number 0–100, with counts next to it (e.g. "5 match · 1 no · 2 unclear"). Unclear counts as half.
- **Requirement matching:** the configured local model compares each extracted requirement with the owner's profile (synonyms and meaning, e.g. Go ≈ Golang, "led engineering teams" ≈ CTO experience). Profile data goes only to the locally configured provider.
- **Volume:** no daily cap. Every vacancy with usable text is processed. Verified on 2026-11-11: `data/` and `data-staging/` route the `extraction` stage to `qwen3.5-9b` (Ollama, `127.0.0.1:11434`), which has no token cost. The repository default `config/providers.yaml` routes `extraction` to `luna` (OpenCode subscription); new deployments must not silently inherit a paid provider for bulk work, see plan Q.1.
- **Branch:** start fresh from `main`. Port from ASSESSMENT-2 only pieces that serve this card (see plan "Reuse").
- **Executor:** Sonnet 5.5, by the owner's explicit choice for this card (overrides the default executor in AGENTS.md for this card only).

## Must not touch

Production 8080, merge to main, push, the existing detailed Fit Assessment / preparation queue and its switch, profile data and revisions, workflow/Apply history, development-process files (AGENTS.md, bin/work-*, hooks, executor configs), other worktrees including `work/ASSESSMENT-2` (read-only reference).

## Estimate

- **Complexity:** 13 — two model passes (vacancy-only extraction, profile matching), persistence keyed by source/profile versions, a background worker, two scores, a reworked panel, and real-model verification on staging. One outcome: none of the parts is useful alone.
- **AI preparation:** 30–40 min (this card, done).
- **AI implementation:** 180–300 min across Q.1–Q.5.
- **AI acceptance:** 40–70 min (Q.6 with the owner on staging).
- **Total AI elapsed:** about 4–7 hours, excluding owner waits and local-model runtime. Local-model throughput is unmeasured; a full backfill of ~2 800 vacancies is expected to take hours of background time on the Mac (measured in Q.6).
