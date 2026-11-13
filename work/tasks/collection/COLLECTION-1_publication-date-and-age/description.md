# COLLECTION-1 — Show publication date and account for vacancy age

**Epic:** collection
**Created:** 2026-11-09
**Status:** review
**Kind:** bug

**Integration status,2026-11-10:** technical implementation accepted; source published to main in commit `3ee6260`, final application source `58af432`. Card remains review, not done; production is not updated. Owner acceptance/release is separate.

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
занеси в бэклог/баглист что надо показывать (и учитывать при оценке) когда вакансия была опубликована.  Много вакансий из web3 jobs 5-6 месячной давности. прямо жесть какая-то
```

## Preparation

**Current result,2026-11-09T21:45Z:** technically accepted source d92112c, records2b8a1d55304b0619538721920b06aa2a57037fb8, worktree .worktrees/COLLECTION-1 in review. Date/completed age, explicit source-publication basis versus unverified legacy dates and Newest ordering live at8083. Schema18, narrow source/date/preservation/migration checks and builds passed; defaultwork-testSKIPPEDonce. Approximately35min including final records vs20–35approved. Full acceptance/incident/coverage record copied to [acceptance.md](acceptance.md); source published to main (`3ee6260`, final `58af432`); owner review and production release separate. Earlier preparation paragraphs below are historical. Next owner priority is Profile, with role profiles for one person and auth/logout deferred.

**Current next proposal,2026-11-09T21:02Z:** [complete plan](plan.md) narrows this first delivery to manual freshness review: source publication+age in list/detail, corrected Newest, and preservation/repair of existing date ingestion semantics. No automatic scorer/age penalty in this step. Complexity3, preparation3–5/implementation12–20/acceptance5–10min, **20–35min total**, superseding the broader80–150min intake estimate below. Reuse accepted APP-8 and its existing independent preview; no repeated setup. Proposed for owner approval; not implemented.

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Observable outcome and evidence to prepare

The owner sees the source publication date and vacancy age during job review; evaluation accounts for that age. The owner reports Web3 Jobs roles published5–6months ago; this is intake evidence, not an independently checked source audit. Keep actual publication, first discovery and later re-discovery distinct; never label first_seen as publication. Missing publication date remains unknown. Preparation must establish source-date coverage and the scoring/evaluation consumer before choosing a rule; no age cutoff, automatic deletion, hiding or penalty weight is approved.

## Provisional estimate

- **Complexity:**5 — source-date normalization, storage, job display and evaluation inputs are coupled; date availability and intended evaluation effect remain uncertain.
- **AI preparation:**15–25min; **AI implementation:**45–85min including builds/corrections; **AI acceptance:**20–40min including real job review and targeted hidden date/provenance checks.
- **Total AI elapsed:**80–150min, excluding owner wait and external source latency. Assumes existing job date fields/adapters are reusable. Provisional backlog estimate, not execution approval. Existing subscription, attributable monetary cost unknown; no paid service assumed.

## Preparation —2026-11-09

Owner: «хорошо, но сначала - где лежит код который про профиль? в frontend его нет?» was followed by «Мой приоритет: сначала свежесть вакансий, затем качество сопоставления. Для первой задачи ещё нужно определить, как возраст влияет на результат: просто отображается и передаётся в подробную оценку или дополнительно влияет на локальное ранжирование. Автоматический штраф либо скрытие старых вакансий пока не согласованы.  - да, давай теперь этим займемся.» The initial priority/policy passage is the coordinator's previous wording quoted by the owner; the explicit new instruction starts preparation of this task, without selecting a penalty or cutoff.

Coordinator read-only findings on main/production database,18:33–18:37Z:

- Of2738 stored jobs,2025 have a nonempty published_at. Counts reflect stored labels, not independently verified date provenance; source formats include ISO/date-only/RFC feed dates and timezone-bearing strings.482 rows with SQLite-parseable dates predate2026-08-09. SQLite parseability is not a complete date-format validator.
- Web3 Careers has132 distinct jobs with dates. Concrete stored example id131, CTO role at D-loopdao: published2025-07-18, first seen2026-10-29; id130 published2025-08-06. Thus newly discovered does not mean newly published. This is stored evidence, not a new fetch confirming the listing is still open.
- Backend model/store/API already carry publishedAt (model/job.go, store/jobs.go). Upsert inserts it but existing-job updates do not fill/update it. EnrichJob parses JobPosting yet only updates description/status, discarding any discovered publication date.
- frontend/filters.ts sortCatalogue('newest') currently orders by firstSeen. RoleContent displays Collected/firstSeen but no publication date. Catalogue age display/sorting belongs to this task; Profile frontend is separate and must remain untouched.
- Source semantics need care: Greenhouse maps UpdatedAt into PublishedAt; feed.go accepts the first pubDate/published/updated in document order. Updated/discovered dates must not become claimed publication. JSON-LD datePosted and genuine feed publication are reusable extraction inputs.
- Local BM25 rescore has no age factor. assess.Job and detailed-assessment DATA omit publication/age/reference time. No inference/search/collection/model operation was started during preparation; database access was readonly.

Open product decision before final plan: preserve relevance/fit score and show/use freshness separately (recommended), or apply an explicit age penalty in local scoring; automatic hiding/deletion is not approved. Coordinator recommendation: actual publication+age in list/detail, genuine-publication Newest sorting with unknown dates last, detailed assessment receives date/age as employer-side freshness context rather than a candidate mismatch. No age filter/cutoff, score formula or source-wide re-fetch is silently added.

The initial80–150min intake estimate is provisional, not the execution estimate. Re-estimate the bounded one-pass plan after this policy answer using the existing date fields; do not turn this into a collector/taxonomy/scoring-engine rewrite. No implementation/worktree/staging change yet. Current Profile implementation .worktrees/DESIGN-3 belongs to its separate frontend handoff; preserve other writers and live data.

Owner's subsequent steering prioritizes a minimal manual workflow, explicit stale/not suitable rejection after Apply, and previously-applied duplicate recognition. See INBOX-1 latest comments, APP-8 and TRIAGE-1. Freshness display/sorting remains part of that discussion; automatic score penalty/cutoff still unselected. Ask about the incoming-list/Triage boundary and manual collection first; do not continue designing a new age-weighted scorer in place of the requested usable loop.
