# Quick Match behavior specification

- **Status:** implemented in main (`58af432`) and released to production on 2026-11-10 from pinned main `157fe9f` (build `157fe9f6d0f8-20261110T151309.703877Z`); technical acceptance stands, card review not done. The confirmed staging profile `local@20` was selectively transferred; startup recalculation produced 2777 numeric results while production jobs, 1493 model assessments, workflow/Apply history, config and the disabled queue were preserved. Owner ranking-quality review remains separate.
- **Numerical policy:** v1 defaults adopted by the owner on 2026-11-10; not externally calibrated or validated against outcomes.
- **Next direction:** dual fit (owner goals and published employer requirements) and targeted vacancy-fact extraction were agreed on 2026-11-10. The coordinator-written starting policy in section 13.7 is owner-approved for implementation but is not implemented; the released v1 numerical policy and behavior below remain what runs, and no paid call has run.
- **Implementation:** the shared contract, extractor, retained-fact storage, pure evaluator, versioned result persistence and catalogue/Overview rendering are implemented in main. Normal persisted scored adoption is proven on an isolated snapshot of the owner's actual staging profile (`local@20`): 2271 visible jobs carried numeric results at that profile revision, with 1539 evaluated zero, 732 nonzero and 0 stale. The temporary 8083 review server is stopped. Production now carries that profile and 2777 current persisted results; the earlier empty-test-profile runtime and the labelled synthetic/response-overlay checks remain historical demonstration detail, not the adoption evidence.
- **Last updated:** 2026-11-10.

This is a persistent behavior specification, not a task plan, implementation checklist or estimate journal. It describes what Quick Match must mean and how its result must be produced and explained. The implementation plan, step estimates and acceptance record live in [the ASM-6 task card plan](../../work/tasks/assessment/ASM-6_deterministic-quick-match/plan.md). [docs/MAP.md](../MAP.md) indexes this specification alongside the requirements and code it constrains; user weight calibration is [ASSESSMENT-1](../../work/tasks/assessment/ASSESSMENT-1_user-calibration-of-quick-match-weights/description.md).

## 1. Purpose and scope

Quick Match gives one vacancy a fast, deterministic, explainable fit score against the owner's saved profile. It is the cheap first ranking used in the catalogue and vacancy Overview before the detailed model-based Fit Assessment used in Preparation. Its beneficiary is the owner deciding which roles deserve attention when the active catalogue is larger than one screen.

### Agreed ranking direction — 2026-11-10 (not yet implemented)

Quick Match should prioritize potential suitability to the owner, not reward the completeness of a source site or the success of our extraction. Missing or unsupported extraction is not evidence of a candidate/vacancy mismatch and must not become an implicit ranking penalty. This direction does not authorize inventing missing facts or presenting title overlap as a complete assessment. The current v1 formula and released behavior described below remain implemented; the revised numerical policy and minimum-information handling are approved for implementation in section 13.7. Dual-direction fit and targeted model-assisted extraction are recorded as the agreed next direction in section 13; they are not implemented, and no paid call has run.

The unit of evaluation is one individual vacancy. The score must not depend on which other vacancies happen to be in the catalogue, on the current search terms, on BM25 keyword frequency, on a model call, or on the order in which jobs are processed.

In scope:

- a per-vacancy score in the range 0–100, or an explicit unscored result;
- comparable evidence, reliability, coverage, unknowns, conflicts and stable reasons;
- prepared vacancy facts for the five fit groups and the profile fields they need;
- deterministic rendering of that result in the existing catalogue and Overview surfaces.

Out of scope:

- collecting, recrawling or reclassifying the vocabulary of the catalogue;
- a generic NLP pipeline, a new job taxonomy, or a model-based template learner; the targeted vacancy-fact extraction adapter in section 13 is a separate agreed direction, not generic semantic scoring;
- multiple user profiles, accounts or a shared multi-tenant catalogue;
- user-editable weights or preference-elicitation UI (deferred to a separate calibration card);
- using Quick Match as a hard dealbreaker or silently hiding low-scoring roles;
- changing Inbox filters, queues, collection, or the separate Application LLM eligibility behavior.

## 2. Non-negotiable behavior

1. **No LLM in the score path.** The released Quick Match makes no model, provider, embedding or inference call, and the pure evaluator never makes one. The agreed future direction in section 13 adds a separate targeted vacancy-fact extraction adapter before persisted facts; it is not implemented and does not change the evaluator boundary.
2. **No external I/O in the released score path.** The released Quick Match performs no network fetch, search, collection, source recrawl or queue mutation. It reads saved profile and vacancy records and writes its own result records through the existing persistence layer. The future extraction adapter, when implemented, would use the existing configured provider path; it does not exist now.
3. **Deterministic.** The same profile revision, vacancy fact revision and rules version produce the same result and the same reasons, independent of catalogue size, search query, processing order, locale-dependent formatting or wall-clock time.
4. **Individual, not corpus-relative.** Normalization must not use the minimum, maximum, mean or distribution of other vacancies. A score change for one role must not be caused by another role being added or removed.
5. **Explainable without a model.** The view renders deterministic templates from the result contract in section 6.6; it does not ask a model to explain a number.
6. **No silent filtering.** Quick Match does not introduce new hard-dealbreaker UX. Inbox filters and existing navigation remain unaffected; low or unscored results remain visible.

## 3. Inputs and authority

### 3.1 Profile

Only the current saved, owner-confirmed profile is authoritative. Document drafts, extracted proposals, unaccepted suggestions, generated application answers and speculative facts must not be scored as confirmed profile facts.

The typed, authoritative profile module already exists in `backend/internal/profile/contract.go`, with `Current` and the `contextProfileData` projection in `backend/internal/profile/ranking.go`. Quick Match consumes an explicit demographic-free matching projection from that existing profile into the shared contract; it does not create a second profile store. The projection preserves profile ID, revision and fact provenance. Legacy arbitrary `preferences` are not the authority and must not become one.

Facts that are absent from the profile are **not evidenced**, not confirmed absent. Demographics, name, photo, nationality proxies, age and similar personal attributes must not be used as fit criteria. Age in particular receives no penalty. The matching projection must exclude demographics rather than relying on a view to hide them.

The profile may carry a role target, skills with an explicit level or context, professional experience, minimum acceptable cash and target, permitted locations and work authorization, remote preference, contract and hours preferences, and time-zone constraints. The exact first-version field list belongs to the implementation plan; this specification fixes the meaning, not a new settings framework.

### 3.2 Vacancy facts

Vacancy facts are prepared separately from the score. Each fact must carry:

- the actual stored field or a supporting quote from the vacancy text;
- the source section: **Required**, **Preferred**, **company context**, or **unknown**;
- the extraction confidence/reliability class;
- the fact and rules version used.

Structured source fields are used first. Where the source preserves headings and lists, that structure is retained in the prepared fact and its excerpt. Legacy flattened description text is not magically restored into sections: if the original heading is unavailable, the section is **unknown** and must not be inferred from prose style. Ambiguity is recorded as unknown or **needs corroboration**, never silently converted into a confident match or mismatch.

Prepared vacancy facts are now a separate stored deliverable. The existing `Job` record keeps title, company, location, tags, description and dates; typed comparison values come from retained source input plus the explicit source policy. Requirements that exist only in the retained legacy text stay unknown rather than being reconstructed at render time.

## 4. Fit groups

Quick Match uses five groups. The group names are approved; the v1 weights are adopted in section 6 and remain uncalibrated.

| Group | What it represents | Typical evidence |
| --- | --- | --- |
| Role | Whether the role itself is the kind of work the owner targets | title, role family, seniority, scope |
| Professional | Competence fit: skills and experience | required/preferred skills, years, domain experience |
| Compensation | Potential declared employer cash compared with the owner's cash minimum | cash amount and range, currency, period, tax basis |
| Geography / eligibility | Where the work may be performed and whether the owner may do it | permitted location, remote mode, authorization, relocation |
| Conditions | Non-pay working conditions | work mode, contract type, hours, time zone |

Reliability is **not** a sixth preference group and must not be turned into an arbitrary per-site coefficient. It is a property of each fact and is reported separately from preference. The three reliability classes are:

- **usable** — explicit, unambiguous and attributable to the vacancy or profile;
- **needs corroboration** — probable but incomplete, conflicting, or not supported by a retained excerpt;
- **unusable** — missing, stale, or not attributable.

Only usable facts enter the numerical comparison. Needs-corroboration facts appear in the result's reliability and unknown/conflict lists; they do not receive a guessed preference weight. Unusable facts do not become zero.

### 4.1 Ambiguity and geography

Remote work mode is separate from permitted location. "Remote" does not by itself prove that the owner may work from their location, and a permitted country does not by itself prove remote eligibility. Both facts are evaluated, with their own provenance.

An ambiguous board location such as a Wellfound location field must not be treated as a confident mismatch. Where explicit job text supplies a location or eligibility statement, that statement is evaluated with its quote and provenance. If the board field and the job text conflict, the conflict is recorded and the fact is at most **needs corroboration**; it is not silently resolved against the owner.

### 4.2 Compensation

A missing salary is not a penalty. When the profile states a salary goal, a missing vacancy salary removes compensation from the score numerator and lowers its evidence coverage; it never becomes zero and never becomes full positive. When the profile states no salary goal, compensation is not applicable rather than missing.

Quick Match uses a **potential compensation alignment**, not a promise and not a verdict on the employer's offer:

```
salaryUtility = min(1, jobCashMax / candidateCashMin)
```

This applies only when both values are positive actual employer cash values in the same currency, period and tax basis:

- `candidateCashMin` is the owner's confirmed minimum acceptable cash, not a target or aspiration;
- `jobCashMax` is the declared cash range ceiling;
- a range lower bound is retained as declared range context but is not silently treated as the guaranteed offer;
- no comparable ceiling means compensation is unknown.

A declared cash range ceiling indicates possible, not promised, pay. The result records basis `range_ceiling` and renders the exact declared range. A stated "up to X" ceiling may prove that the entire band is below the owner's minimum when X is below it; when X is at or above the minimum it supports only possible alignment, not a promised offer. No ceiling is invented from a board estimate, company, location or role. No tax/net figure, equity, bonus or currency conversion is inferred without explicit support.

A rule that maps higher pay to a lower utility is a defect, not an alternative policy. The utility is capped at 1 for pay at or above the minimum.

### 4.3 Skills and experience

A skill absent from the profile means **not evidenced**, not confirmed lack. For example, a Rust requirement whose skill is absent from the saved profile must render as "Rust required — not evidenced in your profile", with the requirement quote and a link to edit the profile. It must not render as "you lack Rust".

The implemented numerical treatment excludes an unconfirmed absence from the professional mean and counts it in unknowns and reduced coverage. An explicit owner-confirmed absence may eventually score as a confirmed negative; that is a future rule, not an inference from silence.

## 5. Evidence, coverage and unscored results

A result is built from **comparable** criteria only. A criterion is comparable when both the vacancy fact and the profile fact are usable, the comparison scale is defined, and the group is applicable. Missing facts lower evidence coverage; they do not silently receive invented utilities.

**Group applicability is determined before scoring. A group is active only when:**

- the saved profile states a relevant declared goal or preference for that group; or
- the vacancy explicitly lists professional requirements compared with the confirmed profile.

No saved preference or goal makes the group **not_applicable**; it is excluded from both the score and the coverage denominator and is not treated as missing. Ambiguous or conflicting applicability is **unknown**, not silently active. A nonempty configured preference with a missing job fact keeps the group active and unknown; it lowers coverage and does not become not_applicable.

Within an active group, configured subcriteria have fixed weights. For each active group `g`, `C_g` in `[0, 1]` is an **evidence fraction**:

```
C_g = (Σ weight of known/possible members) / (Σ weight of all relevant members)
```

The denominator includes relevant members that are missing, so one known skill out of twenty does not produce full professional coverage. In the professional group the denominator includes the vacancy's explicitly required and preferred members, including members missing from the profile; required weight 2 and preferred weight 1. Requirements found only in an `unknown` section are not presented as an exhaustive list.

The qualitative unknown and applicability behavior here is agreed. `C_g` and the coverage formula are the adopted v1 defaults, approved with the rest of the plan on 2026-11-10. They remain uncalibrated and are not claimed to be the owner's measured trade-offs.

A group's utility `S_g` is a weighted mean over its comparable members only. Groups with `C_g > 0` enter the raw score; groups with `C_g = 0` remain in the coverage denominator but add no zero to the score.

- A role with at least one comparable criterion receives a numeric score with coverage `C_g` for each applicable group.
- A role with no applicable group, or no group with `C_g > 0`, receives **unscored**: `score` is `null`, status is explicit, and the view says why. It must not invent 0 or 50.
- An explicit numeric zero is a real evaluated result with positive evidence and must be distinguishable everywhere from unscored.
- A low-coverage numeric result is not suppressed by Quick Match merely because coverage is low; coverage is shown alongside the score.

The result must prioritize an honest unscored or low-coverage outcome over a confident-looking number built from unknown facts.

## 6. Numerical policy — adopted v1 (uncalibrated)

> **Approval status.** Sections 2–4, 7 and 8 are approved product direction. Section 5's unknown behavior and coverage formula were approved as v1 defaults on 2026-11-10. The numbers and scales below are adopted starting values, still uncalibrated and not validated against outcomes; they must not be presented as a probability or as a proven preference model.

### 6.1 Fixed group weights

Initial candidate weights, summing to 1:

| Group | Weight |
| --- | --- |
| Role | 0.25 |
| Professional | 0.30 |
| Compensation | 0.20 |
| Geography / eligibility | 0.15 |
| Conditions | 0.10 |

These are fixed v1 defaults in one existing configuration location, versioned with the rules. They are not user-editable in v1 and do not create a new settings framework. A separate calibration feature may later expose weight selection; that is deferred.

### 6.2 Group utilities

Each applicable group with at least one comparable member produces a utility `S_g` in `[0, 1]`. Within a group, criterion utilities are combined by a weighted mean over its comparable members using fixed internal rules. The implemented professional distribution is:

- a required requirement has internal weight 2;
- a preferred requirement has internal weight 1.

Only evidenced, comparable requirements enter that mean. An unconfirmed candidate absence is excluded, not assigned 0. A confirmed explicit absence may be assigned 0 under a future rule; v1 does not.

The implemented first-version comparators are:

- **Role:** lexical maximum token-F1 title alignment between the actual title and explicit saved target titles, after documented normalization and a small approved alias set. No corpus frequency and no universal semantic or leadership claim.
- **Professional:** explicit required/preferred skill evidence with aliases and word boundaries only; repeated words do not inflate. Required weight 2 and preferred weight 1. A missing saved skill is unknown/not_evidenced and reduces coverage, not a fabricated confirmed zero.
- **Experience:** numeric years and unstructured seniority inferred from total employment history are unsupported and remain unknown unless both profile and vacancy provide directly comparable explicit facts. Do not infer years of a skill from total history.
- **Geography:** explicit declared hiring country and work-authorization requirement. A confirmed compatible set scores 1; a confirmed incompatible exclusive set scores 0; otherwise unknown. Office address is not a hiring restriction, remote is not worldwide, residence is not a work right, and unsupported region mappings remain unknown.
- **Conditions:** explicit work mode, contract and hours. A compatible set/range scores 1; a clearly disjoint set scores 0; otherwise unknown. Time zone or availability without an actual comparable structured fact remains unknown.

Source extraction supports retained structured source fields and explicit section statements only; there is no generic NLP or semantic inference. Requirements from an `unknown` section are not presented as an exhaustive list. These are implemented v1 comparators, not calibrated preference scales; user calibration is deferred to [ASSESSMENT-1](../../work/tasks/assessment/ASSESSMENT-1_user-calibration-of-quick-match-weights/description.md).

Implemented result ordering is current score descending, then raw score, then evidence coverage descending, then stable job ID ascending; stale and unscored results do not pretend to be current and there is no hidden coverage penalty.

### 6.3 Weighted mean with separate coverage

Let `A` be the applicable groups, `w_g` the fixed group weight, `C_g` the evidence fraction and `S_g` the group utility. The raw score uses only groups with positive evidence:

```
A+ = { g in A : C_g > 0 }
raw = Σ(g in A+) w_g * S_g  /  Σ(g in A+) w_g
score = 100 * raw
```

Evidence coverage is reported separately over all applicable groups:

```
coverage = Σ(g in A) w_g * C_g  /  Σ(g in A) w_g      (0..1)
```

Coverage is not a preference weight and is not multiplied into the score. A not-applicable group is excluded from both the numerator and the denominator; an applicable group with no known vacancy evidence stays in the coverage denominator and lowers coverage, but contributes no zero to the score.

This conditional mean has a deliberate consequence: if a group becomes not applicable because a profile goal is removed, the denominator shrinks and the remaining score can move up or down. If a vacancy fact becomes missing while the profile goal remains, the group stays in coverage but leaves the score numerator. No universal numeric monotonicity is promised for adding or removing facts. Removing a known compensation fact does not mechanically reduce the score; it lowers compensation's evidence fraction and coverage.

### 6.4 Compensation utility

When, and only when, both values are positive actual employer cash values in the same currency, period and tax basis:

```
salaryUtility = min(1, jobCashMax / candidateCashMin)
```

A vacancy's declared cash ceiling at or above the owner's minimum gets 1.0. A lower ceiling gets a proportional utility, because the declared band may sit entirely below the owner's minimum. The implemented behavior is capped at 1; pay above the minimum is not rewarded indefinitely and is not treated as a guaranteed offer.

Worked example from the review:

- candidate minimum: 90,000;
- vacancy declared cash ceiling: 80,000;
- salary utility: `80,000 / 90,000 = 0.8889` (88.9 / 100 for the compensation group);
- all other groups at full fit (utility 1.0);
- raw total: `0.25 + 0.30 + 0.15 + 0.10 + 0.20 * 0.8889 = 0.97778`;
- displayed score: 97.78 / 100 (integer display 98, raw 0.97778);
- difference from an all-groups-full-fit 100: 2.22 points, because compensation's effective share is 20%.

A ceiling above the minimum is capped at 1.0 and still does not promise that the employer will offer that pay. This is implemented v1 behavior, not an externally calibrated result, and it does not claim that 2.22 points is the owner's true trade-off.

### 6.5 Unscored and zero

- No applicable group, or no group with `C_g > 0`: `status = "unscored"`, `score = null`, `rawScore = null`.
- At least one applicable group with `C_g > 0`: numeric score in `[0, 100]`.
- A genuine evaluated zero: `score = 0`, `status = "scored"`.
- Stale persisted result: `status = "stale"` until recomputed; the previous value is preserved with its provenance.

### 6.6 Result contract

Every persisted and rendered result carries at least:

```
{
  "status": "scored" | "unscored" | "stale",
  "score": 95,                 // integer display, null when unscored
  "rawScore": 0.95,            // 0..1 raw value, null when unscored
  "coverage": {
    "weighted": 0.80,
    "groups": { "role": 1.0, "professional": 0.40, "compensation": 1.0,
                "geography": 1.0 }
  },
  "applicability": {
    "role": "applicable", "professional": "applicable",
    "compensation": "applicable", "geography": "applicable",
    "conditions": "not_applicable"
  },
  "reliability": {
    "usable": 12,
    "needsCorroboration": 1,
    "unusable": 3
  },
  "contributions": [
    { "group": "compensation", "weight": 0.20, "evidenceFraction": 1.0,
      "utility": 0.8889, "weighted": 0.17778, "status": "usable" }
  ],
  "unknowns": [
    { "group": "professional", "fact": "Rust", "section": "Required",
      "kind": "not_evidenced", "quote": "...", "profileEdit": "skills" }
  ],
  "conflicts": [
    { "group": "geography", "fact": "location", "kind": "source_conflict",
      "detail": "board field and job text disagree" }
  ],
  "reasons": ["skill_not_evidenced"],
  "provenance": {
    "profileId": "...",
    "profileRevision": "...",
    "jobHash": "...",
    "factsHash": "...",
    "rawSourceHash": "...",
    "extractorVersion": "...",
    "sourcePolicyVersion": "...",
    "schemaVersion": "...",
    "rulesVersion": "..."
  }
}
```

Stable reason codes drive deterministic UI templates. The view receives the result and named callbacks; it does not recompute utilities, query the profile, or call a model. Human-readable text is rendered from these codes and values, not generated freely.

### 6.7 Staleness and persistence

A persisted result is stale when its score inputs or its fact-production inputs no longer match the stored provenance. Two dependency chains are distinct:

- **Score inputs:** profile ID/revision, `factsHash`, rules version and schema version. A profile or rules change re-evaluates the **persisted facts** without fetching or re-extracting source content.
- **Fact-production inputs:** `rawSourceHash`, extractor version, schema version and source-policy version. An extractor or source-policy change re-extracts the **retained source input** and invalidates every dependent result.

The existing saved result is preserved and marked stale; recomputation happens through the existing persistence/effects path, not from a view. Source-policy, extractor or rule changes must not silently present old numbers as current. If a producer and a consumer disagree about contract compatibility, the result is refused or marked incompatible rather than guessed from old facts.

Migration and recomputation must preserve known profile data, dates, workflow state and existing records. A Quick Match schema change must not erase or rewrite unrelated historical data. Original plain or HTML source is retained where it is already available; legacy flattened text is not restored or refetched automatically.

## 7. Responsibility boundaries

The owner requires these modules to be separated from the start so that either side can later be rewritten or replaced independently. The implementation must keep these concerns separate:

- **Shared domain contract** — planned `backend/internal/matchcontract`: DTOs, types and validation only. It performs no I/O and imports no store, provider, parser or evaluator package. It is the only package both the extractor and the evaluator import.
- **Extraction** — planned `backend/internal/jobextract`: transforms the retained source payload/text plus the source policy into `VacancyFacts`. It imports the shared contract only and must not import the evaluator, store, provider, site-specific parser or UI. It uses deterministic parsing in v1 and makes no model call. The agreed future direction adds a separate targeted model extraction adapter that produces the same persisted facts with validated quotes before evaluation; it is not implemented, and this deterministic extractor remains the current one. No adapter becomes the evaluator or scores the vacancy.
- **Pure evaluator** — planned `backend/internal/quickmatch`: `Evaluate(ProfileSnapshot, VacancyFacts, Rules) Result`, side-effect-free. It imports the shared contract only and never imports the extractor, site-specific parsers, store, provider, network, clock or filesystem. It has no database, locale or catalogue-global state.
- **Profile projection** — existing application code builds a demographic-free `ProfileSnapshot` from the existing `backend/internal/profile` module, preserving profile ID, revision and fact provenance. It may depend on the profile store; the evaluator does not.
- **Persistence/effects** — existing application services handle profile projection, configuration loading, extraction → fact persistence → evaluation → result CAS persistence, staleness and UI. A view owns no requests, persistence or scoring.
- **Source policy** — an extractor-adapter dependency, not a random model/source coefficient or site name embedded in the evaluator.

No plugin framework, microservice, second binary or new settings framework is introduced. All modules run in the same Go binary with explicit package interfaces.

Version chains are distinct:

- **fact production:** `rawSourceHash` + `extractorVersion` + `schemaVersion` + `sourcePolicyVersion`;
- **score:** `profileId`/`profileRevision` + `factsHash` + `schemaVersion` + `rulesVersion`.

A profile or rules change re-evaluates persisted facts without fetching or re-extracting. An extractor or source-policy change re-extracts the retained source input and invalidates dependent results. Replacing the extractor must not require rewriting the evaluator, and replacing the evaluator must not require re-extraction or refetching.

Focused acceptance examples for this structure, not a continuing test framework:

- a fake or replaced extractor produces the same contract and leaves the evaluator unchanged;
- the pure evaluator compiles and runs on fixture facts with no HTML or site-specific dependency;
- replacing the evaluator leaves persisted facts intact and does not refetch the source;
- an extractor-version change invalidates dependent results;
- an incompatible contract version is refused rather than silently guessed from old facts.

Defaults and source-field usability settings belong in one authoritative existing runtime/config extension. Do not embed user/profile values, machine paths, ports, provider/model choices or adjustable weights inside application logic or views. Do not create a new settings framework or a source-management admin UI for this feature.

## 8. Performance and measurement

The owner clarified that Quick Match is pure processing. The initial safety budget is **≤ 0.5 seconds per job**, with a target of **≤ 30 seconds for a 300-job batch**. At 0.5 s/job the arithmetic would be 150 s for 300 jobs, so the 30 s batch target requires about 0.1 s/job average; it is a target, not a measured or guaranteed result.

Measured on the isolated review catalogue (300 saved jobs with text, 1.58 MB of retained title/company/description, darwin/arm64, 14 CPUs, existing config, no network or model call): extraction 0.35–0.40 ms/job, pure evaluation 0.008–0.016 ms/job, complete read/extract/evaluate/save batch 1.0–1.2 ms/job fresh and 0.5–0.6 ms/job warm; the full 300-job batch took 0.30–0.35 s fresh and 0.16–0.18 s warm. Existing GET decoration of the whole 2,273-role listing took about 0.49–0.53 s. Full evidence, including the empty-real-profile and synthetic-fixture distinction, is in the ASM-6 card's acceptance evidence.

No benchmark may call an external provider, search, collection or model. Quick Match prioritizes real catalogue usefulness and a small number of hidden invariant checks, not a broad test suite or new benchmark infrastructure.

## 9. Implemented behavior and honest v1 limitations

Implemented in main (`58af432`): the versioned shared contract (`matchcontract`), the replaceable extractor and explicit source policy (`jobextract`), the pure evaluator (`quickmatch`), profile projection, retained-source/fact storage, compare-and-swap result persistence, local recalculation, the API envelope and the catalogue/Overview rendering. The old FTS5 BM25 `match_score` path no longer overwrites the current Quick Match result.

The released version does not include dual fit, the agreed readable comparison panel, source-acquisition flags, targeted model extraction or its queue. Those are recorded as the agreed direction in section 13; they are not implemented. The owner-approved plan authorizes bounded implementation, while production release and paid-call scope remain separately constrained.

Honest limitations of this v1:

- Extraction understands narrow explicit headings and labels. Requirements found only in legacy flattened text, or that need inferred years, semantic matching or broad aliases, remain unknown and are not claimed as extracted.
- Source availability is distinct from parse failure. A saved vacancy whose description is effectively absent can currently display a high score from title alignment and partial coverage; that is a known v1 behavior to be replaced, not an approved ranking of the role. The observed read-only Parity example is recorded in section 13.3.
- There is no currency or tax inference, no salary estimation, no bonus/equity conversion, and unsupported country names, regions or frontend-only preferred locations remain unknown rather than being guessed.
- Experience is compared only when both profile and vacancy provide a directly comparable explicit fact; total employment history is never converted into inferred years or seniority.
- Weights and scales remain uncalibrated, and there is no user weight editor. User calibration, multiple profiles/accounts and a redesigned model queue are excluded.
- The model assessment queue is not changed by this specification. Its existing background behavior and pending rows remain as they were; this document does not claim the queue is now selected-only or backend-gated, and Quick Match does not activate, retry or delete it.
- The earlier isolated review runtime held an old empty test copy, not the owner's current profile. A new isolated snapshot of the actual staging profile (typed role, skill and compensation facts) was used for the persisted-adoption demonstration: 2271 visible jobs carried numeric results at profile revision20, with 0 stale. The old runtime was not re-pointed and the temporary 8083 server is now stopped. The labelled synthetic-profile fixture and real-profile response overlay are historical labelled demonstrations, not the adoption evidence.
- The profile status now reports the existing compatibility revision independently from candidate-context status, so a missing candidate context no longer makes Quick ranking look stale. Retained ambiguities carry an explicit kind: only genuine board/text disagreement becomes a source conflict, while missing structure stays a generic ambiguity/unknown.

## 10. Deferred and separate work

- **Dual fit and targeted vacancy-fact extraction** — approved for bounded implementation in the [ASSESSMENT-2 plan](../../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/plan.md) and section 13.7; not implemented. The extraction queue starts paused, no paid call has run, and production release remains separate. It is logically separate from the existing detailed Fit Assessment/preparation queue.
- **User calibration of weights** — captured as a separate backlog card. It may use preference elicitation and case comparisons; it is not part of v1 and must not be silently added to this feature.
- **Application LLM eligibility** — the detailed model Fit Assessment and its gating in Preparation remain separate, unresolved behavior. Quick Match does not decide it.
- **Multiple profiles/accounts/shared catalogue** — future architecture, not this specification.
- **Source administration UI** — not required; source-field usability is configuration in v1.
- **Generic taxonomy/NLP/corpus recrawl** — explicitly excluded.

## 11. Research notes and reuse constraints

The owner explicitly asked that ready Python/other libraries be considered as a reference or basis rather than only a hand-written evaluator. Narrow review of two official implementations and the methodology guide found the following.

### 11.1 Government Analysis Function, *An Introductory Guide to Multi-Criteria Decision Analysis (MCDA)*

- URL: https://analysisfunction.civilservice.gov.uk/policy-store/an-introductory-guide-to-mcda/
- Published 1 May 2024.
- Supports a linear additive value model of the form `V = Σ(v_i × w_i)`, with component values on an interval scale and weights normalized to 1.
- Emphasizes **mutual preferential independence**; a linear model is valid only when trade-offs between two criteria do not depend on a third.
- Distinguishes global scales from local scales; direct rating defines local limits from the options themselves, which is exactly the corpus-dependent normalization this specification rejects for a stable individual score.
- Treats cost separately from benefit and warns that simple name-based weighting and arbitrary percentages are not a valid weight-elicitation method. It recommends swing weighting and sensitivity analysis.
- Implication: the proposed fixed weights are a v1 prior, not validated preference elicitation. They must not be described as the owner's true trade-offs until calibrated.

### 11.2 scikit-criteria

- Repository: https://github.com/quatrope/scikit-criteria
- Documentation: https://scikit-criteria.quatrope.org/en/latest/
- Actual reference implementation: [`skcriteria/agg/simple.py` WeightedSumModel](https://github.com/quatrope/scikit-criteria/blob/master/skcriteria/agg/simple.py). Porting its weighted aggregation mathematics into Go is possible; a port must not import a corpus-relative scaler. Pin the exact source revision and retain the BSD notice if code is adapted in future.
- License: **BSD-3-Clause** (`LICENSE.txt`, SPDX `BSD-3-Clause`), permitting reuse with copyright notice and disclaimer.
- Provides MCDA aggregation, normalization/scalers, weighting and missing-data handling, including weighted-sum-style aggregation.
- Reuse constraint: **adapting the mathematics into Go does not require a Python runtime**; only importing the Python package at runtime would. The owner's explicit porting direction is honored: this BSD-3-Clause implementation is a relevant reference for weighted aggregation and domain-independent normalization only. If any code is adapted, pin the exact source revision and retain the BSD copyright notice and disclaimer. Its API and domain model are not copied wholesale.

### 11.3 pyDecision

- Repository: https://github.com/Valdecy/pyDecision
- License: **GNU General Public License v3.0 or later** (`LICENSE`, GPL-3.0-or-later).
- Implements SAW/WSM and many other MCDA methods with cost/benefit criteria handling.
- Reuse constraint: GPL-3.0-or-later is a strong copyleft license. Do **not** copy or port its code into this repository without a licensing review; use it as a conceptual reference only. A Go rewrite of a mathematical method is still a licensing question when the implementation is derived from GPL code, so the safe default is an independent implementation from public method definitions.

### 11.4 Conclusion for implementation

Use the standard weighted-sum/SAW mathematics and the MCDA guide's warnings. A Go implementation may adapt the BSD-3-Clause scikit-criteria weighted aggregation and domain-independent normalization as a reference; adapting the mathematics to Go does not require a Python runtime. Do not derive code from the GPL-3.0-or-later pyDecision implementation. The evaluator remains a small, independently replaceable pure-Go module behind the shared contract. No library, package or code was installed, copied or added during preparation.

## 12. Decisions requiring owner approval

Approved product direction: the five groups, determinism, no-LLM/no-external-I/O, existing typed profile authority with a demographic-free projection, fact provenance, reliability classes, missing-not-negative rules, separate coverage, unscored versus zero, no hard-dealbreaker UX, and strong shared-contract module boundaries between extraction, pure evaluation, persistence/effects and views.

Adopted v1 numerical defaults (owner-approved 2026-11-10; uncalibrated):

- the fixed v1 weights 0.25 / 0.30 / 0.20 / 0.15 / 0.10;
- the conditional weighted-mean formula, per-group evidence fraction and exact coverage denominator;
- the exact applicability rules for ambiguous inputs and the criterion-weight denominators;
- salary utility `min(1, jobCashMax / candidateCashMin)` with basis `range_ceiling`;
- required weight 2 and preferred weight 1 inside the professional group;
- excluding unconfirmed candidate absence from the mean rather than scoring it 0;
- the remaining role, geography/eligibility, conditions and experience scales;
- the batch performance target of 30 s for 300 jobs.

No formula here claims to solve the domain's preference scales, and no weight is externally validated by this document. The v1 defaults are adopted for implementation even where an example is worked; calibration remains deferred to the separate backlog card.

Agreed direction and starting policy recorded on 2026-11-10: dual fit, a readable factual comparison panel, source-availability handling, targeted extraction after concrete gap flags, and a separate extraction queue (section 13). The coordinator-written numerical/runtime policy in section 13.7 was approved for implementation on 2026-11-10 (the owner's exact wording is in the [ASSESSMENT-2 plan](../../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/plan.md)); it is not implemented, and the v1 defaults and released behavior above remain current. Items not stated in section 13.7 remain open.

## 13. Agreed next direction — dual fit and targeted extraction (2026-11-10, not implemented)

This section records product direction agreed with the owner. The coordinator-written starting policy in section 13.7 was approved for implementation on 2026-11-10 (the owner's exact wording is in the [ASSESSMENT-2 plan](../../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/plan.md)); it is not implemented, and no paid call has run. The released v1 behavior in sections 6 and 9 remains what runs until the approved policy is implemented and verified. Items not stated in section 13.7 remain open. Follow-up work is tracked by [ASSESSMENT-2](../../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/description.md).

### 13.1 Two meanings of fit

Quick Match's primary meaning is **potential suitability to the owner**, not the quality, completeness or coverage of our source extraction. A missing fact is not proof of a mismatch. Evidence coverage must not silently multiply or penalize the score; the v1 conditional mean with its five fixed weights is legacy implemented behavior, not the approved new policy.

The result shows two distinct directions:

- **Fits your goals** — the owner's desired role, content, pay, location and working conditions.
- **Matches requirements** — the published employer requirements compared with confirmed skills, experience and eligibility.

Neither direction is a hiring probability. A role can show high goal fit while requirements remain unconfirmed. Each direction keeps its own unknown, unscored or stale status according to the evidence actually available.

### 13.2 Readable factual panel

The vacancy panel should show concrete, readable facts. Matches and mismatches must both be visible in the same row format; criterion counts communicate the scale of available evidence, they are not the score formula and do not imply equal importance. Partial matches must be marked partial, not exact. A condition must never be labelled "vacancy did not state" merely because the extractor failed. "Missing from your profile" may name only a published job requirement that is not confirmed in the saved profile, with an edit link; it must not claim a missing fact where no supported comparison exists. A relevant condition whose vacancy value is unknown (salary, contract, work mode) appears as a question-mark row, not as a technical diagnostic dump. Unknowns stay concise and are not presented as mismatches. Empty groups are hidden. Remove the redundant "Strong fit …" reasons paragraph, the Compared/usable labels, duplicate Role/Role Title headings, the Evidence/Weight/Utility/Weighted/Share-of-group math, source-field labels, a separate repetitive "Why this result" block and user-facing result provenance; provenance stays in storage. Conflicts are shown only when one actually exists, with the contradicting statements; the currently detectable case is a board location versus the job text, and no broader conflict classes are promised. Coverage, if shown, is not statistical confidence or a probability. Wording is provisional and this is not a layout specification. The existing "Criterion comparisons + group math" rendering remains the implemented legacy surface until a new one is approved.

### 13.3 Source availability versus extraction failure

A saved vacancy with no meaningful description is a source-acquisition problem, not a high fit. The observed read-only example is Parity tech lead, ID 84, saved description "lead tech lead blockchain +3", displaying 100 with 29.4% coverage; it must not receive an overall 100 from title alignment alone. When text is available but the strict deterministic parser cannot identify the Requirements section, that must not leave most of the catalogue unscored. The approved plan does not impose an arbitrary character minimum or require a Requirements heading; a genuine short description can still be evaluated. Where possible, recover the full text through the existing acquisition adapter, without inventing facts. A new generic crawler or mass recrawl is not approved.

### 13.4 Pipeline

Agreed process, not implemented:

1. collect the vacancy description;
2. run deterministic fact extraction and persist the facts;
3. compute preliminary scores for both fit directions;
4. when concrete gap flags exist, run targeted additional model extraction;
5. validate the returned facts and persist them with provenance;
6. recompute both fit directions and refresh the comparison panel.

The deterministic extractor cannot know a true completeness percentage. Gap flags are concrete, per relevant section or fact kind, where evidence text is detected but the information is not extracted or is ambiguous. A section heading is not required; matching text signals can qualify. A flag must not be raised merely because salary is absent. Examples: an unparsed requirements section, an incompletely parsed salary range, unextracted remote or geographic restrictions. Each flag saves why it exists and the supporting source fragment, and distinguishes a short or failed source from a true unknown, and from an actual extraction error. A flag is not a guarantee that a model can recover the fact. Abstract source-quality percentages are not used.

### 13.5 Admission to the targeted extraction queue

Admission requires usable text and at least one actionable, unattempted gap flag; a high preliminary score is not required. Ranking uses preliminary usefulness together with the number and importance of distinct gap flags; the same gap is deduplicated and does not inflate the count. A high preliminary score is a priority signal, not a hard admission gate: at comparable gaps a score-50 role is processed before a score-10 role, while eligible low-scored roles remain queued rather than permanently excluded. The approved starting priority band, gap weights and every-fifth fairness rule are recorded in section 13.7 and the plan; calibration of those defaults remains open. Optional prioritization by user-opened or user-selected roles was discussed as a proposal, not an owner decision, and remains open. A model-facing extraction queue is logically distinct from the existing detailed Fit Assessment/preparation queue and must not enable or consume it.

### 13.6 Extraction adapter and rescoring

The future model is an independent fact-extraction adapter, not a scorer or hiring judge. It processes one vacancy without the candidate profile, must return actual supporting quotes validated against the retained input, and persists fact provenance. The pure deterministic evaluator keeps its no-LLM/no-external-I/O boundary and consumes only persisted facts. The adapter reuses existing configured providers, path resolution, configuration and queue adapters; no model, provider, port or tunable is hardcoded. Current v1 makes no LLM calls. The approved plan reuses the existing `providers.stages.extraction` selection with bounded request caps and allows at most three real staging requests during acceptance; no call has run, and the queue starts paused.

New facts automatically rescore both directions, up or down, and refresh the comparison lists. When an attempt completes with no new facts, scores stay unchanged, the attempt is done even if unknowns remain, and the role is not endlessly requeued on the same inputs. A changed input version may trigger one new attempt under the approved attempt cap. A temporary execution failure gets a limited retry and is distinguished from a completed attempt that found no new data. Storage should retain the input, extractor and model-production identity; this specification does not prescribe new schema fields.

### 13.7 Owner-approved implementation policy — 2026-11-10 (approved, not implemented)

The coordinator presented the complete six-stage policy and cost; the owner approved it on 2026-11-10 (exact wording and approval are in the plan). The approved plan is [ASSESSMENT-2](../../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/plan.md). The starting policy is approved for implementation but is not implemented; released v1 behavior remains current, and production release, push and queue enablement remain separately unauthorized.

- **Goal fit:** a 0–100 weighted mean over comparable saved goals using role 25, compensation 20, location preference 15 and conditions 10, normalized over comparable active groups. Coverage stays separate and is never a multiplier. Skills and legal eligibility are not goal preferences.
- **Requirements fit:** a supported interval, not a third aggregate score. Extracted, deduplicated published requirements keep required/preferred weight 2:1; confirmed matches contribute 1, confirmed incompatibilities 0, and unconfirmed comparisons the [0,1] interval. The lower bound is the confirmed-match share and the upper bound adds unresolved weight; the lower bound is not used for ranking. With no supported comparison show "Requirements not yet verified", not 0–100. Explicit work authorization belongs here; residence is not authorization.
- **Catalogue order:** current goal-fit raw value descending, then stable job ID. Both directions stay visible; stale and unscored values remain distinguishable and are not zero; coverage and requirement uncertainty do not break ties.
- **Insufficient source:** no numeric fit for missing, error, pending-detail or metadata-only descriptions; show "Full job description is unavailable" and reuse the existing single-vacancy refresh. Do not impose an arbitrary character minimum or require a Requirements heading; a genuine short description can still be evaluated. No automatic mass recrawl.
- **Extraction queue:** usable text with actionable, deduplicated, unattempted gap kinds; no score admission threshold. Priority uses a 20-point goal-fit band, then distinct gap impact, raw goal fit, enqueue time and job ID; approved gap weights are 3 for requirements, 3 for hiring/authorization restrictions, 2 for compensation and 1 for work conditions, one flag per kind. Every fifth dispatch selects the oldest eligible job to avoid starvation. The queue starts paused and must not consume the detailed assessment queue. Optional selected/opened boosts are excluded.
- **Runtime provider:** reuse the existing `providers.stages.extraction` selection and `assess.InferInto` transport with a vacancy-only prompt/schema; do not choose or replace the model, endpoint or credentials. Approved caps are one worker, at most 20 physical requests per UTC day including failures, and at most three attempts per input/producer identity with bounded transient retries; no silent fallback. Acceptance allows at most three real staging requests. Provider price is unknown; request caps are not a monetary budget.

The plan's data/config/logic/presentation separation, existing store/migrations and provider/path/configuration/queue adapters, exclusions and targeted checks apply. No new framework or process change is introduced. Items not stated in this section remain open.
