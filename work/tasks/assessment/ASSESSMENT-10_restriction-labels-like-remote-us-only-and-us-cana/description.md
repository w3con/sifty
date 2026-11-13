# ASSESSMENT-10 — Restriction labels like 'Remote (US only)' and 'US & Canada' are not res

**Epic:** assessment
**Created:** 2026-11-13
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
тдельный вопрос, без срочности. По ASSESSMENT-9 девять вакансий с явным ограничением («Remote (US only)», «US & Canada») по-прежнему получают «неизвестно», потому что словарь регионов не разбирает такие формулировки места. Это кандидат на отдельную небольшую карточку. Заведу её, если скажете. - да
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator (2026-11-13)

After ASSESSMENT-9 (released 2026-11-13), 38 model-read locations became usable, but nine remote vacancies with an explicit restriction still get "?" because `resolveRestrictionLabel` in `backend/internal/quickmatch` does not resolve their labels: 17 "Remote (US only)", 1119 "REMOTE (US only) or NYC/Dallas", 1153 "US only, US time zones", 1175 US state list, 1196 "US and Canada", 1201 "US-based (remote within the US)", 1279 "UK, Ireland, Germany, Portugal, Spain or The Netherlands", 1281 "ET timezone in the US", 1294 "US & Canada" (production copy of 2026-11-13; list in ASSESSMENT-9 `attention.md`). Likely work: strip wrapper words ("Remote", parentheses, "-based"), split on "&", "and", "or", "/", ",", resolve each part through the region vocabulary, map US state names to US; separators and wrapper words in configuration next to the regions, not in code. Spec quick-match §4.1.

Not planned yet; steps and acceptance criteria to be written when the coordinator prepares the card.

Provisional estimate: complexity 2 (one pure function in quickmatch plus vocabulary; risk of over-splitting city names). AI time: preparation 10 min, implementation 30–45 min, acceptance 15 min; total about 1 h. No paid services.
