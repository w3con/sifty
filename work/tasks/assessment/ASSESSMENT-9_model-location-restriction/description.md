# ASSESSMENT-9 — A quoted regional restriction read by the model stays unknown

**Epic:** assessment
**Created:** 2026-11-13
**Status:** review
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Почему «неизвестно»: место вакансии 1136 взято из модели и помечено как требующее подтверждения (jobextract/modelconditions.go:110), а по спецификации такой факт остаётся неизвестным. ... Это ограничение стоит вынести в отдельную карточку, если нужно считать такие случаи несовпадением.
(ASSESSMENT-8 release report, relayed by the owner)

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator (2026-11-13)

After ASSESSMENT-8, vacancy 1136 ("Remote (US/Canada only)") is "?" instead of ✕ because every location the model reads is stored as `needs corroboration` (`jobextract/modelconditions.go` ~l.106–110) and an unconfirmed fact never decides. The model is the main source of locations (L.5r0: 71 % with code alone, 93 % with the model), so most regional restrictions will stay "?".

Proposed rule: a model location whose quote passed the exact-excerpt check **and** whose quote states the restriction explicitly ("only", "based in", "must reside / be located in", "eligible to work in", "within <region>") is `usable` for the restriction check; a bare place name stays `needs corroboration`. The restriction phrases live in the region vocabulary configuration (ASSESSMENT-8 G.1). Spec quick-match §4.1 / §14.5 updated.

- [x] **M.1 — Rule and phrases** (25–35 min): as above; one targeted test (explicit restriction quote → usable; bare place → needs corroboration).
- [x] **M.2 — Check** (15–20 min): on a fresh production copy: vacancy 1136 becomes "not covered" with a mismatch line; count of vacancies that change from "?" to ✕ / ✓.

Estimate: complexity 2, AI time about 45–60 min. Backend only.
