# COLLECTION-2 — Remotive category pages collected as vacancies

**Epic:** collection
**Created:** 2026-11-12
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Находка вне карточки: в production 72 страницы категорий Remotive («Software Development», компания «Unknown», id 2270 и дальше) — тот же дефект сбора, что у 40 строк Remote3. Шаг выпуска их не удаляет. Нужна отдельная карточка и ваше решение при выпуске.
(report of the INBOX-2 lead session, relayed by the owner)

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator check (2026-11-12, production)

73 rows from Remotive with company "Unknown" are category pages, not vacancies (e.g. 2270 "Software Development" → https://remotive.com/remote-jobs/software-development, 2271 "Artificial Intelligence", 2272 "Information Technology"); all in status `new`, no history. Same class of defect as the 40 Remote3 rows removed by INBOX-2's release step.

## Proposed (to plan)

- [ ] **R.1 — Collector:** the Remotive discovery must accept only vacancy pages (a vacancy URL pattern or the presence of a JobPosting), never category listing pages; one targeted test on saved pages. (20–30 min)
- [ ] **R.2 — Removal:** the 73 rows removed by the normal role deletion (so the deleted-listings record keeps them out), with the same dry-run/apply script as INBOX-2 R.1, parametrised by source and URL pattern; run at release with the owner's word. (15–20 min)
- [ ] **R.3 — Guard for every source:** at collection, a page whose title is a known category name and whose company is "Unknown" is not stored as a vacancy; counted in the per-source quality numbers (link to COL-2). (20–30 min)

Estimate: complexity 3, AI time about 1 h. Can follow INBOX-2 right away (same release mechanics).

- [ ] **R.4 — We Work Remotely foreign salary blocks** (found by INBOX-2 L.5r0): 195 stored texts contain salary blocks of other jobs from the page; keep only the job's own content block (RSS description preferred). (20–30 min)
