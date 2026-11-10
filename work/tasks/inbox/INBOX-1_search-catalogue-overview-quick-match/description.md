# INBOX-1 — Общий каталог, Overview и объяснимый Quick Match

**Epic:** inbox
**Created:** 2026-11-08
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
ызывает EnqueueAll, а тот выбирает вакансии без ограничения на Preparation. Поэтому утверждать, что нынешний код оценивает моделью только прошедшие triag  - ну ты пиши (это и другое, что раньше обсудили) в карточку для отдачи в работу позже, как закончим обсуждение.
```

## Initial preliminary estimate — superseded by plan.md

- **Complexity:** 21 (provisional) — an umbrella of several independent outcomes (shared catalogue, Overview, explainable Quick Match); not executable as one task and requires an outcome split after the ongoing discussion.
- **AI preparation:** 30–60 minutes to finish the discussion and split the umbrella into bounded outcomes.
- **AI implementation:** 6–10 hours provisionally across the resulting tasks; not authorized by this capture.
- **AI acceptance:** 1–2 hours for builds, code review and real-use review of the split outcomes.
- **Total AI elapsed:** 7.5–13 hours excluding owner replies; highly uncertain until the discussion concludes. This is not approval. Monetary usage cost is unknown; the existing subscription covers it and no paid calls are made now.

Updated proposed scope/cost is in plan.md: profile → search → scoring → Overview, main budget10–16AI hours, optional later default search+0.5–1h. Do not add the old umbrella estimate or the Profile estimate to this total: they describe overlapping work.

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Discussion checkpoint

### User / role-profile backend design —2026-11-09, proposal

**Deferred canonical design:** owner answered per-profile Triage/Preparation/rejections and explicitly deferred implementation. Complete preserved design is now [SETTINGS-5](../../settings/SETTINGS-5_user-data-and-role-profiles/plan.md), including common User data/documents/application history, explicit profile contexts/scores/queues, migration and API. Future implementation planning belongs there; paragraphs below are discussion history, not another executable plan.

Owner agreed the hierarchy/page split (common «Мои данные», role «Профили») and asked «давай спроектируем? какие изменения нужны в бэкенде?». One person; auth/logout deferred. This is design preparation, not implementation approval. Source audit: backend in accepted .worktrees/COLLECTION-1.

Measured singleton dependencies: profile.LocalProfileID="local"; store.CurrentProfile selects ORDER BY id LIMIT1 and CreateProfile returns that first row; ProfileData mixes identity/factual skills/experience/current eligibility with role preferences; documents/facts attach to profile_id; RescoreForProfile overwrites jobs.match_score and stamps only profile revision; store.Context returns the globally latest snapshot; assessment queue conflicts only on job_id, assessment writes jobs.assessment; application_drafts is keyed only by job_id. Adding profile rows/dropdown alone would silently mix these inputs/results.

Coordinator's proposed boundary: one current User owns identity/contact/avatar, factual skills/experience/languages/current location/work authorization, documents and shared personal revisions. Role Profiles own names/targets/conditions and selections of document/skill/experience IDs. Reuse existing JSON+revision approach and stable row IDs; do not normalize every nested field or build account/session machinery. Availability/notice and preference placement must follow factual versus desired meaning, not copy the whole old Geography/Availability sections blindly.

Proposed backend consequences: profile CRUD and explicit-ID section saves; User CRUD limited to /api/me; document owner moves to User with profile-document links, originals/files retained once; fact proposals carry explicit user/profile target scope; BuildProfileContext(userId,profileId) composes only selected data/documents. Per-profile quick-score rows and detailed assessments/queues carry profileId, profileRevision, userRevision and job/input fingerprint; active-profile preference is not a worker input. Switching profiles reads their results and never silently starts model calls. Existing algorithm/provider/path/config adapters reused.

Preparation and submissions pin the used profile/revisions/context/documents; old saved drafts do not silently change when the active profile changes. User application ledger retains actual submitted events and listing/profile snapshots even if a role is returned/rejected/removed or the profile is archived. Selecting/opening Apply is not a submission. Profile archive detaches selections, preserving common documents/facts and application history.

**Open product boundary:** per-profile versus common Triage/Preparation/rejections. Coordinator recommends per-profile review decisions (a salary/role rejection in one profile may fit another), with actual submissions common for the User and visible across all profiles. Existing global blocks remain global unless separately changed. Do not assume the earlier agreement to shared application history also approved shared rejection/pipeline state. Need owner answer before a complete implementation plan/estimate.

Proposed migration: one User+initial role profile from current stored data, preserving IDs/originals/fact provenance/revisions/history; bind existing selections, recalculate local scores per profile, retain old detailed assessments as attributable/stale where inputs cannot be proven, no model rerun or live source refresh. New-profile comparison must distinguish not-scored/stale/current, not substitute zero. Targeted acceptance should cover isolation between profiles, exact worker input across switching, shared fact/selection preservation, migration, and persistent prior submission after removal. This remains a design proposal, not a full checkpoint/cost plan.

**Role-profile scope answered,2026-11-09:** owner confirms one person with several role profiles; logout/accounts deferred. Global dropdown/Profiles UI work is already in DESIGN-3. Coordinator proposes common personal facts/original files with profile-specific selection/goals/conditions, per-profile versioned scores and common prior-application history carrying the used profile. This is a model proposal, not approval of a new whole-app multi-user or scoring rewrite. Freshness technically accepted separately; next Profile completion begins by inspecting the active other agent's actual handback. Preserve current one-profile data/accepted revisions and avoid UI pretending several backends exist.

**Owner answers,2026-11-09:** «нет, все сразу попадает в triage (после работы фильтров этом триаже).» — no new incoming stage. «только загрузка приложения запускает сбор. если был недавно - не запускает.» — preserve current App-mount collection and cooldown, not manual-button-only collection. For duplicate examples: «любая вакансия от chainsync, coinbag, uniswap». Five Apply outcomes accepted, with new button-only dismissal; APP-8 plan prepared separately. The previous two workflow questions are answered; remaining question is which environment owns real application history after the concrete production/staging divergence below. Do not expand into the old umbrella/automatic scoring work.

**Latest steering —2026-11-09:** owner now prioritizes a minimal manual cycle: obtain jobs, sort them, move to Triage or reject; Apply from Preparation also offers stale/not suitable; recognize already-applied duplicates. Exact wording is captured in comments.md. This narrows the immediate discussion; old10–16h umbrella/Overview/scoring plans are not the proposed next implementation. Existing COLLECTION-1 supplies publication/age, APP-8 owns durable negative Apply outcomes (provisional10–20min), TRIAGE-1 owns duplicate/prior-application recognition (provisional25–45min). Total manual-flow estimate depends on whether the owner wants a new incoming stage before current Triage or is referring to the existing Triage→Preparation loop; do not silently build an additional stage. Likewise clarify whether manual means disabling App-mount auto-collection. Await these answers and a real duplicate example, then prepare bounded steps/results/costs for approval. No implementation/model/collection/staging change yet; Profile frontend worktree is handed to another agent.

Latest planning checkpoint: owner accepted the simpler initial approach with «ок» and requested a plan, then proposed «начать лучше с профиля, потом перейти к поиску, затем модуль оценки, затем карточка вакансии». Proposed plan.md follows exactly that order. Free-form dealbreakers are stored in Profile and evaluated by LLM after Select/Preparation, not a universal quick-rule engine. Earlier open alternatives below remain discussion history. Implementation is still not approved; see plan.md for costs, boundaries and remaining decisions.

Dealbreaker boundary for the proposed first version: ordinary typed conditions may be compared only against known vacancy facts; free-form preferences are checked in detailed model Fit Assessment during Preparation. Missing/ambiguous conditions remain unknown. No custom chip is advertised as an executed Quick Match rule; the earlier universal/typed-rule alternatives are deferred.

Owner explicitly requires profile data to influence scoring. SETTINGS-1 must include an actual integration with the existing scoring module and demonstrated rescore, not merely store fields for this umbrella to consume someday. This umbrella retains the broader explainable formula/Overview/catalogue outcome; resolve precise scoring ownership before execution so no second engine or missing integration results.

Profile ownership split: the separate card `work/tasks/settings/SETTINGS-1_user-profile/description.md` now owns the editable user profile/documents/skills/preferences and its audit. References below retain discussion history; they do not duplicate implementation ownership. This umbrella keeps catalogue, vacancy Overview, Quick Match, Preparation-only model gating and coverage status. User request for a default search derived from the profile is a later outcome here, dependent on the Profile data contract; never silently overwrite manual search edits. CV inputs are not generated application CVs; the latter is separate future work.

The owner requested this record for later handover after discussion. This is an umbrella, not an executable or approved implementation plan. The coordinator authored the summary; distinguish owner direction from recommendations and unresolved questions. No code, schema, tests, provider calls or deploy is authorized here.

### Owner direction

- Retain collected vacancies in our database and apply user search locally. Collection/Search terms need enabled/disabled state without deletion or a new source-site fetch. Full-site downloads are not mandatory; source coverage can be bounded.
- Full job descriptions must precede scoring. Process the whole active collected catalogue rather than only the first screen: the best vacancy may be among unprocessed records. Queues control processing rate, not permanent eligibility.
- Overview should show persisted extracted facts from the proposed design. Detailed model Fit Assessment is for roles that pass triage into Preparation; general-catalogue Quick Match is separate.
- Explainable deductions matter: users should see unconfirmed/mismatching requirements and update their profile. The proposed inline editor includes “Not in your profile”, “I know this”, production/familiar/learning levels, optional context and “Add & rescore”.
- A status line can show incomplete description/scoring coverage. Previous model-queue counts are not equivalent to catalogue readiness.
- A Settings user/profile tab is now under discussion, including multiple CVs, skills, desired pay, work type, and countries where the person can/wants to work. Tab name and exact scope are not yet settled.
- A shared server catalogue and user-specific API views are a future architectural direction, not authorization to build hosting/accounts now.

### Verified current code and contradictions

- `frontend/Collection.tsx` has string-array OR/AND terms and add/remove only; exclusions also lack enable toggles. `.q-chip.off` in `frontend/theme.css` is a style, not working state/persistence.
- `backend/internal/background/collect.go` skips excluded postings before resolve/Upsert, so they may never reach the database. `store/jobs.go` reversibly hides already stored exclusions on read, but a local toggle cannot restore a row never collected. This conflicts with retaining a shared searchable catalogue.
- `source/greenhouse.go` can get complete descriptions in one board API response; `source/generic.go` may need detail requests. Do not assume a full-site scrape is always cheaper.
- `backend/internal/model/job.go` has description/location/dates/company/tags, not the proposed structured compensation/work mode/contract/seniority/requirements/benefits fields.
- `store/rescore.go` computes BM25, normalized against the current corpus. It does not implement the design's per-term bonuses/penalties or preference checks. Zero defaults/resets do not distinguish unprocessed from assessed nonmatches. Numeric mockup contributions are not an implemented formula or probability.
- `store/jobs.go` Status enqueues assessment on transition to preselected. BUT `background/assessment.go` calls EnqueueAll every30sec; `store/queue.go` EnqueueAll scans all roles, skipping stopped ones but not limiting to Preparation. Thus Preparation-only automatic model assessment is not enforced. This is source inspection, not a new provider-call experiment.
- Prior footer in `app/web/main.tsx` showed running/pending/failed/missing-description/current and model labels; removal is visible in commit `0285f3c`. Similar UI remains in `frontend/Settings.tsx` Queue and CSS in `frontend/queue-control.css`. These counts describe model assessment, not full-catalogue description/Quick Match readiness.
- `backend/internal/candidate/context.go` already stores versioned evidence snapshots, preferences and owner-confirmed feedback with provenance. That is not yet a full structured editable Profile model or managed multi-version CV library.

### Coordinator recommendations, not approval of every detail

- Separate shared vacancy facts from per-user profile, preferences, workflow, quick scores and detailed assessments. User exclusions should not erase the shared catalogue. Define active/retained catalogue boundaries and freshness/closure handling.
- Preserve full source text alongside normalized facts and their supporting excerpts/source revision. Salary needs amount/range/currency/period and base versus bonus/equity; contract and working hours are separate. Remote includes geography/time-zone constraints. Unknown differs from a confirmed negative.
- Prefer structured source facts; whether free-text extraction needs a separate shared model stage remains an open cost/quality question. Do not invent company facts, mandatory requirements or benefits.
- Define an explainable Quick Match formula, with different required/optional weights. Distinguish unconfirmed profile evidence, explicitly absent skill and insufficient experience. Missing evidence may lower the preliminary score without proving the person lacks that skill.
- Add & rescore saves a confirmed profile fact with level/context and recomputes the user's relevant catalogue scores without recollection. Production/familiar/learning should not have identical meaning. A changed profile makes old model assessments stale; it should not silently rerun all paid inference.
- Coverage should distinguish description pending/failed, extracted, scoring pending/ready/failed. Show description coverage, fast-score coverage and Preparation model queue separately; do not use zero as “not processed”.
- Proposed Profile sections: CV/document versions and links; experience/skills/languages; target roles and seniority; compensation; work format/contract; desired and eligible geography, work authorization and relocation; availability/time zones; explicit exclusions and preference importance. A tag cloud is a view of structured skills, not the only source of truth. Exact first-version fields remain under discussion.

### Open decisions and handover prerequisites

- Active catalogue/retention/coverage boundaries, description completeness and failed-retrieval policy; treatment of already skipped exclusions.
- Deterministic versus model fact extraction and costs; explainable scoring weights, stable normalization, unknown/required/preferred rules and all-terms-disabled semantics.
- Profile minimum scope, CV selection/version meaning, evidence/conflict handling, skill-level vocabulary and hard versus soft constraints.
- Preparation-only automatic model gating, manual exceptions, leaving Preparation, stale/retry behavior and existing non-Preparation queue rows.
- Status-line contents/visibility and honest coverage denominators.
- Split this21-point umbrella into independently useful outcomes, refine estimates and get complete plan approval after discussion. Do not dispatch this record as one executor stage.
- Reconcile prior ASM-3/ASM-4 and COL-6 decisions deliberately. Review COL-4 observability, SET-1 setup questions, APP-6/APP-7 deferred preparation limitations and the separate unintegrated DESIGN-2 refactor. Update affected R-COL/R-SEARCH/R-ASM/R-SET/R-TRI requirements and task dependencies before execution.
