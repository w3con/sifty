# Quick Match behavior specification

- **Status:** implemented in main (`58af432`) and released to production on 2026-11-10 from pinned main `157fe9f` (build `157fe9f6d0f8-20261110T151309.703877Z`); technical acceptance stands, card review not done. The confirmed staging profile `local@20` was selectively transferred; startup recalculation produced 2777 numeric results while production jobs, 1493 model assessments, workflow/Apply history, config and the disabled queue were preserved. Owner ranking-quality review remains separate.
- **Numerical policy:** v1 defaults adopted by the owner on 2026-11-10; not externally calibrated or validated against outcomes.
- **Next direction:** two fit scores, Fits you and You fit, with model-extracted requirements are agreed in section 14 (2026-11-11), implemented by card ASSESSMENT-3 and not yet implemented. Section 14 supersedes section 13.7 and the ASSESSMENT-2 implementation notes; ASSESSMENT-2 was rejected by the owner on 2026-11-11. The released v1 numerical policy and behavior in sections 6 and 9 remain what runs until ASSESSMENT-3 is accepted.
- **Implementation:** the shared contract, extractor, retained-fact storage, pure evaluator, versioned result persistence and catalogue/Overview rendering are implemented in main. Normal persisted scored adoption is proven on an isolated snapshot of the owner's actual staging profile (`local@20`): 2271 visible jobs carried numeric results at that profile revision, with 1539 evaluated zero, 732 nonzero and 0 stale. The temporary 8083 review server is stopped. Production now carries that profile and 2777 current persisted results; the earlier empty-test-profile runtime and the labelled synthetic/response-overlay checks remain historical demonstration detail, not the adoption evidence.
- **Last updated:** 2026-11-11.

This is a persistent behavior specification, not a task plan, implementation checklist or estimate journal. It describes what Quick Match must mean and how its result must be produced and explained. The implementation plan, step estimates and acceptance record live in [the ASM-6 task card plan](../../work/tasks/assessment/ASM-6_deterministic-quick-match/plan.md). [docs/MAP.md](../MAP.md) indexes this specification alongside the requirements and code it constrains; user weight calibration is [ASSESSMENT-1](../../work/tasks/assessment/ASSESSMENT-1_user-calibration-of-quick-match-weights/description.md).

## 1. Purpose and scope

Quick Match gives one vacancy a fast, deterministic, explainable fit score against the owner's saved profile. It is the cheap first ranking used in the catalogue and vacancy Overview before the detailed model-based Fit Assessment used in Preparation. Its beneficiary is the owner deciding which roles deserve attention when the active catalogue is larger than one screen.

### Agreed ranking direction — 2026-11-10 (not yet implemented)

Quick Match should prioritize potential suitability to the owner, not reward the completeness of a source site or the success of our extraction. Missing or unsupported extraction is not evidence of a candidate/vacancy mismatch and must not become an implicit ranking penalty. This direction does not authorize inventing missing facts or presenting title overlap as a complete assessment. The current v1 formula and released behavior described below remain implemented; the revised numerical policy and minimum-information handling are approved for implementation in section 13.7 (superseded by section 14, 2026-11-11, where an unknown goal group counts 0.5 and a title-only match can no longer reach 100). Dual-direction fit is recorded in sections 13 and 14; it is not implemented.

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

1. **No LLM in the score path.** The evaluator stays pure and deterministic and never makes a model, provider, embedding or inference call. Model passes (section 14.5: vacancy-only requirement extraction, then requirement matching against the profile) run before evaluation in a separate adapter and produce persisted, validated facts and verdicts; the evaluator consumes only those persisted inputs, so a score is reproducible from persisted inputs without any model call. The released v1 makes no model call at all; the model passes of section 14 are agreed and not yet implemented.
2. **No external I/O in the released score path.** The released Quick Match performs no network fetch, search, collection, source recrawl or queue mutation. It reads saved profile and vacancy records and writes its own result records through the existing persistence layer. The section 14 model adapters, when implemented, are separate from the score path and use the existing configured provider path; they do not exist yet.
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

The released version does not include dual fit, the agreed readable comparison panel, source-acquisition flags, targeted model extraction or its queue. Those are recorded as the agreed direction in sections 13 and 14 (section 14 governs); they are not implemented. The owner-approved plan authorizes bounded implementation, while production release and paid-call scope remain separately constrained.

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

- **Two fit scores with model-extracted requirements** — section 14, card [ASSESSMENT-3](../../work/tasks/assessment/ASSESSMENT-3_two-fit-scores/description.md); not implemented. This supersedes the ASSESSMENT-2 plan and section 13.7 (ASSESSMENT-2 was rejected by the owner on 2026-11-11). Earlier text: approved for bounded implementation in the [ASSESSMENT-2 plan](../../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/plan.md) and section 13.7; not implemented. The extraction queue starts paused, no paid call has run, and production release remains separate. It is logically separate from the existing detailed Fit Assessment/preparation queue.
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

Section 14 (2026-11-11) supersedes the 13.7 policy described next; ASSESSMENT-2 was rejected. Earlier record: agreed direction and starting policy recorded on 2026-11-10: dual fit, a readable factual comparison panel, source-availability handling, targeted extraction after concrete gap flags, and a separate extraction queue (section 13). The coordinator-written numerical/runtime policy in section 13.7 was approved for implementation on 2026-11-10 (the owner's exact wording is in the [ASSESSMENT-2 plan](../../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/plan.md)); it is not implemented, and the v1 defaults and released behavior above remain current. Items not stated in section 13.7 remain open.

## 13. Agreed next direction — dual fit and targeted extraction (2026-11-10, not implemented)

> **Superseded in part (2026-11-11).** ASSESSMENT-2, which implemented sections 13.4–13.7, was rejected by the owner. Section 14 replaces 13.7 and the gap-flag admission and daily-cap queue of 13.4–13.6 with two model passes over every vacancy with usable text. 13.1–13.3 (two meanings of fit, factual panel, source availability) remain direction, refined by section 14. History is kept below.

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

### 13.7 Owner-approved implementation policy — 2026-11-10 (SUPERSEDED by section 14, 2026-11-11)

> **Superseded.** ASSESSMENT-2 was rejected by the owner on 2026-11-11; this policy and the ASSESSMENT-2 implementation notes are retained as history only. Use section 14. In particular the requirements interval, the "Requirements not yet verified" state, goal-fit group dropping, the gap-kind queue, the 20 requests/day cap and the every-fifth-dispatch rule no longer apply.

The coordinator presented the complete six-stage policy and cost; the owner approved it on 2026-11-10 (exact wording and approval are in the plan). The approved plan is [ASSESSMENT-2](../../work/tasks/assessment/ASSESSMENT-2_dual-fit-targeted-extraction/plan.md). The starting policy is approved for implementation but is not implemented; released v1 behavior remains current, and production release, push and queue enablement remain separately unauthorized.

- **Goal fit:** a 0–100 weighted mean over comparable saved goals using role 25, compensation 20, location preference 15 and conditions 10, normalized over comparable active groups. Coverage stays separate and is never a multiplier. Skills and legal eligibility are not goal preferences.
- **Requirements fit:** a supported interval, not a third aggregate score. Extracted, deduplicated published requirements keep required/preferred weight 2:1; confirmed matches contribute 1, confirmed incompatibilities 0, and unconfirmed comparisons the [0,1] interval. The lower bound is the confirmed-match share and the upper bound adds unresolved weight; the lower bound is not used for ranking. With no supported comparison show "Requirements not yet verified", not 0–100. Explicit work authorization belongs here; residence is not authorization.
- **Catalogue order:** current goal-fit raw value descending, then stable job ID. Both directions stay visible; stale and unscored values remain distinguishable and are not zero; coverage and requirement uncertainty do not break ties.
- **Insufficient source:** no numeric fit for missing, error, pending-detail or metadata-only descriptions; show "Full job description is unavailable" and reuse the existing single-vacancy refresh. Do not impose an arbitrary character minimum or require a Requirements heading; a genuine short description can still be evaluated. No automatic mass recrawl.
- **Extraction queue:** usable text with actionable, deduplicated, unattempted gap kinds; no score admission threshold. Priority uses a 20-point goal-fit band, then distinct gap impact, raw goal fit, enqueue time and job ID; approved gap weights are 3 for requirements, 3 for hiring/authorization restrictions, 2 for compensation and 1 for work conditions, one flag per kind. Every fifth dispatch selects the oldest eligible job to avoid starvation. The queue starts paused and must not consume the detailed assessment queue. Optional selected/opened boosts are excluded.
- **Runtime provider:** reuse the existing `providers.stages.extraction` selection and `assess.InferInto` transport with a vacancy-only prompt/schema; do not choose or replace the model, endpoint or credentials. Approved caps are one worker, at most 20 physical requests per UTC day including failures, and at most three attempts per input/producer identity with bounded transient retries; no silent fallback. Acceptance allows at most three real staging requests. Provider price is unknown; request caps are not a monetary budget.

The plan's data/config/logic/presentation separation, existing store/migrations and provider/path/configuration/queue adapters, exclusions and targeted checks apply. No new framework or process change is introduced. Items not stated in this section remain open.

## 14. Two fit scores with model-extracted requirements (ASSESSMENT-3, agreed 2026-11-11)

Supersedes section 13.7 and the ASSESSMENT-2 implementation notes. ASSESSMENT-2 was rejected by the owner on 2026-11-10/11 (second score `not_verified` for every vacancy, requirements extracted for 45 of 2 777 vacancies, title-only goal scores of 100, vacancies closed as "no extraction needed" although their text held requirements, queue mechanics shown instead of facts). Sections 13.1–13.6 remain the product direction; where 13.4–13.7 differ from this section, this section governs. Implementation is card [ASSESSMENT-3](../../work/tasks/assessment/ASSESSMENT-3_two-fit-scores/description.md). Released v1 behavior (sections 6 and 9) stays what runs until ASSESSMENT-3 is finished and released. The numbers below are owner-approved starting policy, uncalibrated.

Status of the subsections at the close of step Q.9 (nothing in this section is released; "on the branch" means implemented on `work/ASSESSMENT-3`, not in `main`):

| Subsection | Status |
|---|---|
| 14.1 Fits you, 14.3 catalogue order, 14.4 panel, 14.6 re-run rules, 14.8 responsibility boundaries | implemented on the branch; 14.2, 14.3 and 14.4 describe You fit as shown only while the matching pass is on (14.14) |
| 14.2 You fit | implemented on the branch; not shown by default (14.14) |
| 14.5 two model passes, 14.9 machine-wide lock, 14.10 collection switch | implemented on the branch |
| 14.7 paid-provider guard | implemented on the branch; the guard is about cost only (14.13.3) |
| 14.11 verification of Q.6 | measurement, a fact of 2026-11-11 |
| 14.12 matcher changes | implemented on the branch; its measured quality is what 14.13 and 14.14 judge insufficient |
| 14.13 model trial | measurements are facts; the engagement check, the matching stage, the `think` option and the removal of the local-only rule are implemented on the branch; the candidate fixes, the pipeline steps and the sections are agreed directions moved to cards |
| 14.14 scope split | implemented on the branch (step Q.9) |

### 14.1 Fits you (goals)

0–100 weighted mean over the goal groups with the existing fixed weights: role 25, compensation 20, location / work mode 15, conditions 10 (the professional group is no longer part of this score). A group with no comparable fact contributes the midpoint 0.5 instead of being dropped, so score = Σ(weight × utility) / Σ(weight) over the groups where the owner has a goal. The four weights are divided by their own sum and need not add up to 1; the professional group carries no weight and its key is no longer part of the rules. A title-only match therefore gives about 68, never 100, and unknown facts never push a role to the bottom. Legal eligibility and skills are not goals. Weights stay in the existing match configuration, versioned with the rules.

### 14.2 You fit (requirements)

0–100 over the requirements extracted from the vacancy text. Required weight 2, preferred weight 1. Verdict utility: match 1, partial 0.5, unclear 0.5, no 0. Score = weighted mean. The score is shown with counts, e.g. "5 match · 1 partial · 1 no · 2 unclear".

- Usable text but extraction found no requirements: "No requirements stated" (not 0, not 100).
- Extraction pending: "Analysing requirements…" and no number.
- Where each verdict is listed (section 14.4): match and partial under Matches (a partial line keeps its verdict so the view marks it as partial), no under Mismatches, unclear under Missing from your profile.
- Source unavailable (no description, tags or title only, error, pending detail): no numbers for either score; "Full job description is unavailable" and the existing single-vacancy Refresh action. No automatic mass recrawl.

A verdict `no` follows the complete-CV policy of section 14.12 (implemented on the branch): it is given for a specific skill, technology, experience, qualification or language that the profile does not show, so an unrelated role can fall below 50.

Neither score is a hiring probability.

### 14.3 Catalogue order

Best fit (the default of the Inbox toggle): Fits you descending, then You fit descending, then job id. A pending You fit sorts as 50 within equal Fits you. Coverage and requirement uncertainty are not tie-breakers. The owner may change this order later; the change is recorded here.

Newest (the other position of the toggle): the source publication date, not the discovery time. The key has three ranks: 0 a publication date with a recorded basis (`published_basis` set), 1 a date without one, 2 no usable date. Within ranks 0 and 1 the later date comes first; rank 2 and equal dates fall back to job id ascending. A date is usable when it is non-empty, a real calendar date, in a format the sources store (ISO 8601 with or without fraction and zone, a space instead of `T`, `YYYY-MM-DD hh:mm:ss UTC`, RFC 1123, or a plain `YYYY-MM-DD`), and not later than now; a zone-less value is read in the machine's local zone. This is the one definition of "newest" in the product. It is implemented twice, in `frontend/dates.ts` (`newestKey`, used by the Inbox) and `backend/internal/background/newestkey.go` (used by the model work order), and both are checked against one fixture, `backend/internal/background/testdata/newest-key.json`, which was generated from the browser implementation. Limit: the server reads a zone abbreviation only when it is UTC or GMT; a date with another abbreviation counts there as unknown, where a browser might read it.

### 14.4 Panel sections

Header: **Fits you N** and **You fit N** with counts and the existing Strong / Partial / Weak bucket labels. Sections, one line per fact, empty sections hidden:

- **Matches** — a requirement or goal and what in the profile or goals matches it ("Go — Go (expert)"); a partly met requirement is listed here too, marked partial.
- **Mismatches** — known conflicts ("Office in London — you want remote", "Pay €70k — your minimum €100k").
- **Missing from your profile** — a published requirement the profile cannot confirm, with a link to the profile section ("Rust — not in your skills · Add").
- **To clarify** — goal conditions the owner cares about whose vacancy value is unknown, as "? Salary", "? Contract type", "? Work mode".

The icons of section 14.12.7 sit on requirement lines of kind skill; a requirement the profile lacks is shown without the model's explanation (14.12.9).

Never shown: queue state, attempts, caps, Retry of the queue, provenance, reason codes, coverage percentages, parser internals (field names such as "description"), "this is not a zero" disclaimers. The only operational message is a single design-system retry-kind line when the model is unavailable ("Requirements could not be analysed — the local model is not running").

Implemented in Q.5 (`frontend/QuickMatchPanel.tsx`, `frontend/quickmatch.ts`; the view only formats the stored `fit` and computes nothing):

- A goal line reads "Work mode: Office in London — you want remote"; a requirement line reads "Go — Go (expert)", with a "partial" tag on a partly met one and a "preferred" tag on a preferred one. The vacancy's own sentence is the hover text of a requirement line; a long profile fact is shortened to one line and shown in full on hover. A list longer than six lines shows six and "Show N more".
- A "Missing from your profile" line links to the profile section that can answer it (skills, languages, experience, work authorization); a requirement of kind qualification or other has none, so it has no link.
- Each score without a number says why in words: "No requirements stated", "Analysing requirements…" (only while the worker is working), "Not analysed yet" (model not running, provider refused, paused or limit reached), "Full job description is unavailable" with a Refresh action that fetches the text again. A model that is not running adds the retry-kind line above; a provider that may not be used adds a broken-kind line. A stale result keeps its numbers, drawn muted, with an info line.
Implemented in Q.5b on the branch (not released):

- **The counts belong to You fit.** "N match · N partial · N no · N unclear" sits directly under the You fit number, in that score's own column, so it reads as the breakdown of You fit and not of the whole panel. Fits you has no counts line.
- **Goal lines and requirement lines in one list.** In a list that holds both (Matches and Mismatches), the goal lines (Role, Contract type, ...) come first and a thin rule separates them from the requirement lines. The list header counts both kinds, while the counts line under You fit counts requirements only; the rule makes that difference visible without a second heading.
- **Alternatives.** A requirement that offers a choice carries its members (`alternatives` on the list line, absent for a single demand); the line shows them after the requirement, "Backend language (Kotlin · Python · Go)", and the evidence names the member the profile meets. A result stored before Q.5b has no alternatives on its lines until it is evaluated again; on the review copy every stored result was evaluated again when the server started, without a model call.

- Catalogue row: the two numbers side by side (Fits you filled, You fit outlined; the hover text spells both out). Order and the Fit filter follow 14.3 on Fits you; a stale result or one without a Fits you number follows every current one. The "Best fit" button is that order.

### 14.5 Two model passes

The extraction pass uses the provider selected for the `extraction` stage and the matching pass the provider selected for the optional `matching` stage (the extraction provider when none is named), both in the existing provider configuration (a provider may also set `think: low|medium|high` for models that cannot switch reasoning off); no model, endpoint, port, limit or weight is hardcoded. Both run through the existing inference transport and lock, at low temperature with thinking off where supported, with the provider timeout and at most three attempts on transient errors.

1. **Vacancy extraction (no profile).** Input is the vacancy text, treated as data. Output: requirements, each with short normalized text, kind (skill, experience, qualification, language, authorization, other), level (required or preferred) and an exact quote; and condition facts (compensation, work mode, engagement, location, timezone, hours), each a value with an exact quote or null. Every quote must be an exact excerpt of the retained source: typographic quotes and apostrophes, dashes, non-breaking and repeated spaces are compared as their plain forms, the words themselves must match, and the original fragment is stored. A list of examples or choices inside one demand ("a backend language such as Kotlin, Python or Go") is one requirement with an alternatives list, not one requirement per item; "strong interest or background in" is a preferred requirement. A work mode is accepted only from a sentence about where the work is done ("a 50/50 hybrid role" describing a mix of duties is not one), and hours only from an explicit full-time, part-time or amount of hours, and an engagement only from a sentence that names a kind of agreement (contract, freelance, permanent, full-time, part-time and the like; the word "employment" alone, as in equal-opportunity text, benefit lists and work-authorization notices, is not one; see section 14.13.2). Invalid items are dropped, the list is deduplicated and its length capped by configuration. It must work on text without headings, including paragraphs glued together. Persisted per (description hash, extractor version): the key is the hash of the description text alone, so a new sighting, tag or view of the same text never asks the model again (the title and company go into the request as orientation but not into the key). A response whose every requirement failed validation is retried and then refused; it is never stored as "no requirements stated". An empty list for a text that plainly states requirements ("requirements", "qualifications", "N+ years") is asked again, and only an answer that stays empty through every attempt is final. Conditions the model found are used as facts before evaluation only where the ordinary parse found no fact for that field and the quote passed validation; when both state the field and disagree, the deterministic fact stands and the disagreement is kept for audit and never shown.
2. **Requirement matching (with profile).** The profile is treated as the owner's complete CV (section 14.12). Input is the extracted requirements and the profile facts: the headline, skills with levels, experience entries, languages and work authorizations; goals, demographics, name and photo are not sent, and the profile has no education section. Five more facts are computed by code from the experience entries and offered with every requirement: total years of experience; years in roles titled CTO, VP, Head, Director, Lead, Manager, Chief or Founder; years in roles titled engineer, developer, architect or programmer; whether the owner founded a company; and whether there is a current role or one that ended within the last twelve months, read as "worked in the last year". They are read from dates and titles only; a fact that cannot be computed is absent and means unknown, never "no". The pass has two parts. First, code: a requirement of kind skill or experience that names technologies is matched against the profile through the configured vocabulary (section 14.12.2); a hit is a match decided by code, with the profile fact as its evidence and the reason "Named skill found in the profile", and the model never sees that requirement. Second, the model: for each remaining requirement it receives only the profile facts closest to it (by word overlap with the requirement, its alternatives and the synonyms of the technologies it names; at most `quick_match.model_passes.evidence_passages`, default 3) and the computed facts, and returns a verdict (match, partial, no, unclear), the number of the fact it relies on and a short reason. A match or partial verdict must cite a fact that was shown for that requirement or a computed fact; any other citation turns it into `unclear`. A verdict `no` on a specific skill, technology, experience, qualification or language needs no cited fact, because a complete CV that does not show it does not have it. A `no` on authorization or on a requirement of kind other needs a shown fact that contradicts it, otherwise it is `unclear`. A `no` on what a CV does not usually state (soft skills and working style, team or company size, work authorization and eligibility, availability, travel) is `unclear`; the list of such topics is configuration. A requirement that lists alternatives is met when any one of them is met. A requirement the model did not judge is `unclear`. Semantic equivalence ("led engineering teams" ≈ CTO experience) is the model's task. Persisted per (requirements hash, profile revision, matcher version); the matcher version is a digest of the prompt, the schema, the skill vocabulary, the number of passages and the model name, so a change to any of them (a different model included) runs the matching pass again and never the extraction. The profile facts go to the provider configured for the matching stage; which provider that is, is the owner's choice (section 14.7).

### 14.6 Re-run rules

- New or changed vacancy source: extraction, then matching.
- New profile revision: matching only; extraction is reused. Until step Q.8 (section 14.12.10) every stored verdict is stale after any profile change and the worker matches every vacancy again; a vacancy the owner is looking at, after adding a skill from it, is matched first.
- Changed weights or rules: pure re-evaluation only, no model call.
- Changed skill vocabulary (the built-in default or `skills.yaml` in the data directory's config folder) or number of evidence passages: matching only; extraction is reused and no vacancy text is read again.
- An extraction that finds nothing is final for that source hash. It is never closed because a deterministic gap detector found no gap.
- Order: one list for all model work, extraction and re-matching alike, in the order of the Inbox sort toggle (section 14.3). Best fit takes Fits you descending, then job id, a vacancy without a Fits you yet counting as the midpoint; Newest takes the Newest key, then job id. A newly collected vacancy has no priority of its own: it takes the place its Fits you or its publication date gives it. The toggle is saved on the server (`quick_match.model_passes.order` in `runtime.yaml`, values `fit` or `newest`, default `fit`; `GET` and `PUT /api/quick-match/order`; the frontend saves it on every change and reads it on load, and the server's value wins). The worker reads it before every vacancy, so a switch takes effect at the next one. Saving it recomputes nothing and writes a timestamped backup of `runtime.yaml` each time. `quick_match.model_passes.new_window_hours` no longer exists; a leftover key in an old file is ignored. A vacancy without text is never queued.
- No daily cap by default (`daily_limit`, 0 = unlimited). One worker, taking the machine-wide model lock (section 14.9) for each model call so it never overlaps the detailed assessment queue's provider use, in this or any other Sifty process; that queue is neither enabled nor consumed. `paused` stops it. When the provider cannot be reached it backs off (15 s doubling to 5 min) and keeps the work; a vacancy whose answer cannot be used is set aside for an hour or until its text changes. After every pass the stored two-score result is recomputed with no model call (conditions the model read are merged into the deterministic facts first). Collection, a refreshed vacancy, a profile save and a rules change wake the worker; a profile save therefore re-runs only matching. The result is stored per vacancy and projected on each job as `fit`; `GET /api/quick-match/analysis` reports the worker's state and counts; `sort=fit` orders by section 14.3.

### 14.7 Paid-provider guard

Bulk extraction and matching refuse a provider that is not marked local or free unless configuration explicitly allows it. The guard is about cost only. Sifty has no rule about where data may go: the owner decides that by choosing the provider of each stage, and a provider that receives the profile facts or the vacancy texts is the owner's configuration, not something the application restricts (owner's decision, 2026-11-11; it replaces the earlier rule that matching must use a local provider). The guard exists because the repository default routes the `extraction` stage to a subscription provider while the owner's deployments route it to a local model. A new deployment must not silently inherit a paid provider for bulk work. Implemented in ASSESSMENT-3 Q.3. A provider counts as local when it is an `ollama` or `compatible` endpoint on a loopback address, and as free when `providers.yaml` marks it `free: true`. Extraction refuses any other provider unless `quick_match.model_passes.allow_paid_bulk_provider` is true, and the refusal names how to allow it. Matching follows the same rule as extraction (`allow_paid_bulk_provider` allows a paid provider for either pass). Both passes read `quick_match.model_passes` (requirement cap, attempt limit, temperature) from `runtime.yaml`.

### 14.8 Responsibility boundaries

Model calls live in an adapter package separate from the evaluator, persistence and views. The evaluator is pure and consumes persisted facts and verdicts; the API projects the persisted result; the frontend renders it and never computes fit.

### 14.9 Machine-wide model lock

Production, staging and review servers each have their own data directory but use the same local model, so a lock stored in a data directory (the earlier `inference` row in the `locks` table) could not stop two servers overlapping. Every model call of the fit worker, the detailed assessment queue and application preparation now runs under one operating-system file lock (`flock`, package `internal/modellock`) on a path that does not depend on any data directory: the `SIFTY_MODEL_LOCK` environment variable, otherwise `sifty/model.lock` under the user's cache directory, otherwise the same beneath the system temporary directory (`dirs.ModelLockPath`). This is the only file the application keeps outside its data directory. The lock is taken without waiting: a holder that finds it busy skips and tries again later (the fit worker reports the work as owed, the assessment queue waits for its next tick, preparation returns to the queue). The operating system drops the lock when its holder exits or crashes, so a dead process never leaves it held. A test starts a second process with another data directory and proves it is refused, and that a killed holder frees the lock.

Limits: the lock holds only among processes built from a version that includes it; an older binary (production and staging run `main` today) neither takes nor honours it, and still uses the former per-data-directory row, until it is redeployed. The lock is Unix-only (`flock`). It serialises whole model calls; it does not prioritise one process over another.

### 14.10 Collection switch for review copies

`runtime.yaml` accepts `collection_disabled: true`. While it is set, the process starts no vacancy collection: `POST /api/search` answers `skipped` without recording a run, the daily scheduler does nothing, and a direct trigger is skipped. It exists so that a server running on a copy of the data, which analyses vacancies with the local model, can never fetch from the internet. It is a per-process, per-data-directory setting, off by default, and it does not affect any model work or any other request.

### 14.11 Verification on a review copy (Q.6, 2026-11-11) and what stays limited

Run: a copy of the staging data (profile revision 25, 2 781 vacancies, 2 328 with usable text) analysed by the worker with `qwen3.5:9b` through the local Ollama, `daily_limit` 0, collection disabled (section 14.10). The first 45 vacancies were taken while the worker still put newly collected vacancies first, so the sample leans to the newest vacancies, which are mostly unrelated to the profile. Full numbers and the hand check are in the card folder (`q6-analysis.md`).

Measured:

- **Time.** One vacancy, extraction plus matching: median 24 s, p90 43 s, mean 26 s, longest 60 s (extraction alone median 17 s, matching alone median 10 s). Backfill of the 2 279 vacancies then still pending at one vacancy at a time: about 15 hours at the median, 16 at the mean, 27 at the p90. The run was not isolated: the owner was browsing the copy and other Sifty servers were running on the machine, and whether any of them used the model during the run was not checked. The worker is sequential by design, so the time cannot be shortened by waiting; the order (section 14.6) decides which vacancies are analysed first.
- **Verdicts.** Of 363 requirements in 38 vacancies: match 30.9 %, partial 1.9 %, no 0 %, unclear 67.2 %. Six of 45 vacancies had no requirements stated.
- **Scores.** You fit ranged from 50 to 100 (median 69) over 42 vacancies; Parity 88, Ethena Labs 57.
- **Data preservation.** The profile, jobs (without the two score columns), assessments, assessment queue and its state, Apply drafts and revisions, workflow history and feedback hashed identically before and after, except the viewed-at time of two vacancies that the owner opened.

Limits found (the first two describe the Q.6 build; Q.7, section 14.12, changed the matcher and re-measured them):

- **You fit has a floor of 50.** The verdict `no` is kept only when a profile fact contradicts the requirement (section 14.5), and it appeared 0 times in 363 lines; `unclear` counts 0.5 (section 14.2). An unrelated role (a senior accountant, a people partner) therefore scores 50, the same as a role the profile half answers, and the Mismatches list never holds requirement lines. This follows the written policy; whether the policy is right is the owner's decision.
- **The matcher is literal.** In a hand check of 10 "unclear" lines taken from vacancies with Fits you of 50 or more, 2 were clearly answered by the profile and 2 more partly (for example "agent-native development" against a profile that describes the owner's own AI delivery system; "led a small team of 3-8" against Chief Architect and Founder & CTO roles); on unrelated vacancies none of 10 was. The facts sit in prose summaries and the model asks for a literal statement. A larger local model or a profile with explicit skills and team sizes is the lever; neither was tried, because no local model larger than `qwen3.5:9b` is installed.
- **The shared model lock reaches only builds that contain it.** Production and staging run an older build until they are redeployed; they neither take nor honour the lock (section 14.9).
- **Newest on the server is a copy.** It is checked against the browser's behaviour for the date formats the sources store today (section 14.3), not for every string `Date.parse` accepts.
- **The toggle writes a backup each time.** Every save of the order goes through the configuration save and keeps a timestamped copy of `runtime.yaml`.
- **Not exercised on the copy:** a real collection waking the worker (collection is disabled there), and the paid-provider refusal against a live provider.

### 14.12 Matcher changes after Q.6 (owner-approved 2026-11-11; implemented on the branch in step Q.7, not released)

Status: implemented on the branch `work/ASSESSMENT-3` and measured on the review copy; it replaces the parts of sections 14.2 and 14.5 that were marked as superseded, and those two sections now describe it. Not in `main`, not in production. The numbers below were measured with the local model `qwen3.5:9b`; what stays limited is listed at the end of this section.

1. **The profile is a complete CV.** The matcher treats the profile as the owner's full CV. A specific skill, technology, experience, qualification or language that the profile does not contain, directly or through a computed fact, is `no` (listed under Mismatches). `unclear` is kept only for what a CV usually does not state: soft skills and working style, team or company size, work authorization and location eligibility, availability, and anything the matcher explicitly marks as not answerable from a CV. Effect: unrelated roles fall below 50 in You fit; the floor described in section 14.11 disappears. Utilities in section 14.2 are unchanged (no 0, unclear 0.5). *Implemented:* the prompt (`backend/internal/prompts/requirement-matching.md`) states the policy; in code a `no` on a skill, technology, experience, qualification or language needs no cited fact and carries no attached fact, a `no` on authorization or on a requirement of kind other keeps the rule that a shown fact must contradict it, and a `no` on what a CV does not usually state (soft skills, team or company size, work authorization, availability, travel, and degrees, because the profile has no education section) becomes `unclear` through a list in `skills.default.yaml`. The matcher version moved from `vacancyai.match/2` to `/3`.
2. **Named skills are matched by code.** Before the model, a requirement that names a technology, language or tool (or a list of alternatives) is matched against the profile's skills and experience text through a small synonym table (for example Go ≈ Golang, TS ≈ TypeScript, K8s ≈ Kubernetes) with whole-word matching; short or ambiguous terms need a technology context. A hit is `match` with the profile fact as evidence and is not sent to the model. The synonym table is configuration, not code, and reuses ASM-5 material where it fits. A miss is not a verdict: the requirement goes to the model. *Implemented:* the vocabulary is `backend/internal/config/skills.default.yaml` (built into the binary; an optional `skills.yaml` in the data directory's config folder replaces it), read by the existing configuration loader; it holds the synonym groups, the words that show an ambiguous name is a technology, the words ignored when choosing passages, the topics a CV cannot answer and the categories of section 14.12.4. Short or ambiguous names (Go, C, R, TS, Node, Spring, REST) count only when spelled as configured and with a technology context: they are the whole requirement, a member of a list of alternatives, or stand within three words of a technology-context word or within two of another technology; a hyphen or `&` joining them to a letter (`Go-to-market`, `R&D`) disqualifies them. The rule also decides a partial: a requirement that names several technologies of which the profile lists some but not all is `partial` by code. Years of one technology, an unknown word in an all-of requirement and a group in which no member is listed are left to the model. There was no synonym material in ASM-5 to reuse (the card holds only a description).
3. **More computed facts.** In addition to total years, leadership years and founder (section 14.5): a current role exists (an experience entry marked current or ending in the current or previous month), read as "worked in the last year"; further facts that follow from dates and titles only. Absent means unknown, never `no`. *Implemented:* `experienceCurrent` (a current role, or one whose span reaches into the last twelve months) and `experienceEngineering` (years in roles titled engineer, developer, architect or programmer), beside the three of section 14.5.
4. **Focused evidence for the model.** For each requirement the model receives only the two or three profile passages closest to it (selected by code by word overlap with the requirement, its alternatives and synonyms) plus the computed facts, not the whole profile. Citations still refer to profile facts by number; a citation outside the offered passages turns the verdict into `unclear`. *Implemented:* the choice is by shared word stems with the requirement, its alternatives, the synonyms of the technologies it names and the categories of the vocabulary (the word "cloud" brings AWS, GCP and Azure; without this a requirement for cloud experience was shown nothing and answered `no` although the profile lists AWS). The number of passages is `quick_match.model_passes.evidence_passages` (default 3). A requirement with no overlapping passage is shown only the computed facts.
5. **Model size.** The default stays `qwen3.5:9b` (configuration, per installation). A larger model is considered only if the Q.7 measurement still shows the matcher missing evidence the profile states. This was tried in Q.7b on two larger local models; the result is in section 14.13.1: no better, and the owner calls the remaining error rate unacceptable (section 14.13.3).
6. **Re-run cost.** Only the matching pass changes (new matcher version): extraction is reused, no vacancy text is read again. *Implemented:* the matcher version is a digest of the prompt, the schema, the vocabulary and the passage count, so changing any of them re-runs the matching pass and nothing else.

7. **Actions on a vacancy line (added at the owner's request during Q.7).** A requirement line of kind skill offers, for each skill it names, two icons: ✓ adds that skill to the profile (level Proficient, entered by hand) and ✕ records that the owner does not have it. A line that lists several skills ("Kubernetes, IaC and observability") or a group of alternatives offers one pair per skill, split at commas, semicolons, "and", "or" and "&" (a slash stays inside a name); a click acts on that one skill and never on the whole line. The page does not change; the server saves the skill and asks the model worker to match this vacancy first (`POST /api/profile/skills` and `POST /api/profile/absent-skills`, both with an optional `jobId`), and the page reloads until the vacancy's You fit is back. The tooltips of the icons appear after 0.7 s. The level of a skill is shown to the model as part of its text and is not otherwise used: the code ignores it when it decides a match, and a level asked by a vacancy is not extracted.
8. **Skills the owner does not have.** The profile's skills section keeps a second list, `absentSkills` (key and name), shown under the skills as "Skills you do not have", each entry removable with a cross; the list and the skills the owner has never overlap (adding a skill removes it from the list, and naming one in the list removes it from the skills). The matcher is given each entry as a fact (field `skillsAbsent`, "Does not have: ..."); a requirement all of whose named skills are on the list is answered `no` by code with the reason "You said you do not have this skill", without a model, and such a line is not offered the icons again. Other requirements may be shown the fact by the passage choice.
9. **What a line shows.** The model's explanation of a verdict is never shown (a requirement the profile lacks is just the requirement); a fact that contradicts a requirement is shown as the note, as before.
10. **Planned, not built: Q.8.** A change to the profile still marks every stored verdict stale (section 14.5: the key contains the profile revision), so adding one skill makes the worker ask the model again for every vacancy. The owner chose to remove this in a separate step Q.8: a verdict is kept per requirement under a digest of what the model was shown for it (the requirement, the passages the code chose and the computed facts) and the matcher version; after a profile change the code chooses the passages again, and the model is asked only about requirements whose shown facts changed.

Measured in Q.7 (2026-11-11, review copy on port 8094, the same profile revision 25 as in Q.6, collection disabled, matching only: the 43 vacancies of Q.6 and 38 more that the worker had analysed since; full tables and the lists in `q7-analysis.md` in the card folder, the two earlier runs in `q7-analysis-run1.md` and `q7-analysis-run2.md`):

- **Verdict shares** over the 396 requirements of the 43 Q.6 vacancies, before and after: match 32.6 % to 37.9 %, partial 2.5 % to 6.3 %, no 0 % to 34.3 %, unclear 64.9 % to 21.5 %.
- **You fit** of the 43 vacancies: minimum 50 before and 8 after, median 70 and 62, 15 of 43 below 50 (none before). Of the eleven unrelated roles, nine are below 50 (accountant 11, product marketing 8, marketing compliance 17, people partner 25, business development 33, attorney 33, support operations 40, copywriter 42, trader 44); a Growth Lead (73) and a Product Designer (71) are not. Parity stays 88; Ethena Labs goes from 57 to 62 (76 in the two earlier runs of the same vacancy: one vacancy gave different verdicts on different runs).
- **Requirements decided by code** without the model: 44 of 764 (5.8 %), 10 of 396 (2.5 %) in the Q.6 set; in no vacancy did code decide every requirement.
- **Time** of the matching pass: median 8.9 s, p90 14.4 s per vacancy (10.1 s and 16.4 s before). With the extraction pass of Q.6 (15.2 s) the backfill of 2 279 vacancies is still about 15 hours.
- **Hand check** of 20 `unclear` lines: 13 right (soft skills, authorization, degrees); 7 wrong, of which 4 are answered by the profile ("Programming language" with Go and Python listed, "2+ years experience" with 17 years, "Team leadership and coaching", "Cloud and enterprise technologies") and 3 should have been `no`. Of 10 `no` lines, 9 right and 1 wrong ("Agile Execution experience": the profile says the owner introduced Scrum).

What stays limited:

- **The model invents some matches and misses some answers.** It accepts a shared word as evidence ("testing standards" for A/B testing, "product lifecycle records" for customer lifecycle marketing), so unrelated roles can still reach 70; and it answers `unclear` or `no` for things the profile states. A prompt line against matching on a shared word changed nothing. This is the case section 14.12.5 names for trying a larger local model; none was tried and the default was not changed.
- **Verdicts differ between runs** of the same vacancy even at the low temperature, so a profile change that re-asks every requirement also moves verdicts that did not need to move (step Q.8, 14.12.10).
- **Passage choice is by shared words.** "Agile" does not reach a passage that says "Scrum"; a synonym or category in the vocabulary fixes one such case at a time.
- **The code decides few requirements** (above), so it saves little time; its value is that a skill the profile lists by name is not left to the model's judgement.
- **Skill level is not used** (14.12.7), a degree cannot be judged (no education section), and the vocabulary is a short list of common technologies that the owner can extend in `skills.yaml`.
- **Not exercised:** a vacancy list longer than the 81 analysed. The matcher with other models was exercised in Q.7b (section 14.13.1).

### 14.13 Model trial, pipeline direction and open decisions (Q.7b, 2026-11-11; the measurements are facts, the directions are proposals unless marked as the owner's)

Status: measured on review copies of the data, matching on the same 81 vacancies and the same profile revision 25 as in Q.7; implementation of the configuration changes is on the branch and not released. The raw files are in the card folder (`q7b-matching-comparison.md`, `q7b-runs/`).

**14.13.1 Which model reads and compares**

Extraction (six vacancies chosen for what they test, 52 expected requirements and 27 expected conditions written by hand before the runs; two runs each):

| Model | Requirements found (of 52) | Conditions correct (of 27) | Alternative groups kept (of 6) | Seconds per vacancy (median) |
|---|---|---|---|---|
| Qwen3.5 9B (local) | 42 | 23 to 24 | 4 to 5 | 11.0 |
| Qwen3.6 35B-A3B (local, about 24 GB) | 41 | 24 | 5 | 6.4 |
| Claude Haiku 5.5 (hosted, subscription CLI) | 52 | 27 | 6 | 20.7 |
| Luna (hosted, opencode CLI) | 52 | 25 to 26 | 5 to 6 | about 16 |

The two local models are within noise of each other on quality and well below the hosted ones; the Qwen3.6 model is 1.7 times faster at extraction. Cost of a hosted model, estimated from token counts (about 5 000 input and 1 500 output tokens per vacancy; Haiku 5.5 at 0.10 and 0.50 US dollars per million tokens): about 0.1 to 0.2 US cent per vacancy, about 0.5 US dollar per 400 vacancies, about 3 US dollars for a backlog of 2 279. Reasoning tokens are billed as output, so reasoning is kept low. Models that cannot switch reasoning off (gpt-oss) were not usable: with default reasoning the extraction returned empty answers, and the model was dropped from the comparison by the owner as too slow.

Matching (81 vacancies, same extracted requirements, same prompt; the model name is part of the matcher version):

| Model | Same verdict as 9B | You fit below 50 (of 81) | Growth Lead | Product Designer | Seconds per vacancy (median) |
|---|---|---|---|---|---|
| Qwen3.5 9B | - | 29 | 73 | 71 | 8.9 |
| Qwen3.5 35B-A3B | 73 % | 27 | 69 | 71 | 8.4 |
| Qwen3.6 35B-A3B | 75 % | 20 | 69 | 71 | 5.6 |

A hand check of 20 random requirements where 9B and Qwen3.6 differ found Qwen3.6 right in six, 9B right in five or six, the rest disputed or wrong in both. The shared fault is the one named in section 14.12: the model takes the nearest profile fact as proof on a shared word or a neighbouring topic ("identity and SSO" supported by smart-contract security standards, "fundraising" by token economics, "network isolation" by "network token economics", "A/B testing" by testing standards, "customer lifecycle marketing" by product lifecycle records). For the Growth Lead vacancy three of the four matches are false; the true You fit is about 0 to 20, not 69 to 73, and for the Product Designer about 40 to 50, not 71. The target roles in the profile do not explain this: they feed Fits you (41 and 27 for these two vacancies, correct), not You fit. Conclusion: a larger local model does not remove the fault. The cause is the design: the model sees the three closest facts and, when none fits, still picks one.

**14.13.2 What produces the condition facts, and the engagement check**

The deterministic parse (`jobextract`) reads only labelled lines ("Salary: ...") and exact standalone words ("Remote"), by design. On 97 vacancies it found a place in 76, a work mode in 2, and no pay, engagement, hours or time zone. The conditions of Fits you therefore come almost entirely from the model (`MergeModelConditions`: the model fills only what the parse did not find; the deterministic fact wins on a disagreement; a quote that is not an exact excerpt is dropped). Merged with the conditions of the 9B model, Fits you changes on 52 of 92 vacancies, always by 9 or 10 points (the conditions group, weight 10, goes from unknown to known): work mode moves 29 down and 23 up, engagement 46 down and 5 up; compensation, location, hours and time zone moved no score.

A hand check of 15 vacancies where Haiku named no engagement and Qwen3.6 named "employment" showed the cause: eight quotes were not about the contract at all (equal-opportunity text, "competitive salary package", "open until filled", a salary range), two were boilerplate about all employees, three were benefit lines for full-time employees, and two stated it ("Job type Full-Time"). A profile that wants `contract` treats any "employment" as a mismatch, so these false values lowered Fits you by about 9 points. Implemented on the branch: an engagement is kept only when its quote names a kind of agreement (`GroundEngagement`, like the existing checks for work mode and hours). Applied to the stored answers it cut the vacancies whose score moves because of engagement from 51 to 21 of 92 (9B) and from 18 to 8 of 30 (Qwen3.6); Haiku unchanged at 3. After it, the scores of different extraction models differ by 0.4 to 2.2 points on average (25 common vacancies). Not measured: whether the work-mode values, which carry the other half of the changes, are right.

**14.13.3 Owner's decisions and direction**

- **No restriction on where data goes (owner, 2026-11-11).** The rule "profile data goes only to a local provider" was a mistake. Sifty does not restrict providers by data; the owner configures each stage. Implemented: `ErrProfileNotLocal` is removed; matching follows the cost rule of section 14.7.
- **The matcher's false matches are unacceptable (owner, 2026-11-11).** Candidates, none decided or built: (a) a code check that downgrades `match` to `unclear` when the requirement and the cited fact share no meaningful word; (b) a second, narrow question per pair ("does this fact itself prove the requirement") answered by the model, at the cost of more calls; (c) a separate small decision model per (requirement, fact) pair, for example the open Laya typed-decision models, as an independent module that Sifty can switch to in configuration (the owner asked for a pluggable matcher; an experiment on the exported inputs is prepared); (d) a stronger hosted model for the matching stage on the shortlist, now allowed by the decision above. The choice needs a measurement against the hand-judged cases of this section.
- **Proposed pipeline (owner, 2026-11-11, with the status found in the code).** (1) Fetch vacancies, split into sections and store the sections: the fetch exists; sections are not stored (see below). (2) Extract requirements and conditions with a local model: exists, on the whole text in one call. (3) Deterministic first score by Go code: exists. (4) Second score by the local model, in the order of the first score: exists as the background queue; its accuracy is the open problem above. (5) When the owner puts a vacancy on the shortlist, run it again on a stronger model (Haiku recommended): not built. Re-extraction on a hosted model sends only the vacancy text; re-matching sends profile facts and is now the owner's choice; a new extraction changes the requirements hash, so matching follows. (6) Compose the resume from blocks (the model for polishing is undecided: Laya or Haiku): not started; the proposed shape is code orders the blocks by the evidence counts of the matcher and the model only polishes, validated by code.
- **Sections (owner, 2026-11-11).** The vacancy view in the interface is to show sections, so the division is needed independently of the model. It should be built once in Go (beside `jobextract`), stored, used by the interface and, if it helps accuracy, by extraction; the model is not required to read section by section. Measured on 97 vacancies: 18 carry heading markup and 38 more have bare label lines ("Requirements", "What you'll do"), 56 in all; 16 are flattened ("About UsParity is..."). Section 3.2 still applies: only explicit headings divide a text, nothing is inferred from prose. Time: across 30 Qwen3.6 extractions the time was about 0.3 s plus 0.4 s per thousand characters plus 0.7 s per requirement (R squared 0.93), so splitting into five calls costs about 1.5 s per vacancy (20 % of the median) and dropping boilerplate blocks saves about 0.6 s; whether it improves accuracy is untested. The plan: the six hand-labelled vacancies, whole text against text without boilerplate blocks against section by section, on the same model.
- **A benchmark for other users.** `scripts/model-bench/` (README in English) lets anyone run the extraction and the matching comparison on a copy of their own data and print a report; hosted models need `--allow-paid`.

**14.13.4 Open**

Which model extracts and matches by default; the fix for the false matches (above); the shortlist re-run as its own card; the sections card; Q.8 (section 14.12.10); four tests that fail on `main` before this card (`TestMigrationV14`, `TestMigrationV16ToV17PreservesData`, `TestSchema_FreshDatabase`, `TestRouteDrift`); whether a hosted model may see the resume blocks (the owner's call, now that no rule forbids it).

### 14.14 Scope split and what ships with ASSESSMENT-3 (owner and coordinator, 2026-11-11; implemented on the branch in step Q.9, not released)

Status: implemented on the branch `work/ASSESSMENT-3`, not released. Supersedes the open items of section 14.13.4 by moving them to cards.

- **You fit is not shown as a number until a matching method is chosen.** The model's requirement verdicts are not good enough (section 14.13.1). Configuration `quick_match.model_passes.matching` (default `false`) switches the model's matching pass off; the extraction pass stays on. With it off: no You fit number and no counts line in the panel or the catalogue; the panel lists the vacancy's requirements under one heading ("Requirements"), each with the ✓ / ✕ actions; a line already decided by code (a named skill found in the profile through the vocabulary of section 14.12.2, or listed under "Skills you do not have") shows that state; every other line is neutral. Fits you, its order and its lists are unchanged. With it on, the behaviour of sections 14.2 and 14.12 applies (for comparison on review copies). *Implemented:* the setting is `quick_match.model_passes.matching` in `runtime.yaml` (a missing key reads as `false`); the worker builds no matching pass and plans no matching work while it is off, and a vacancy is finished after extraction. The lines decided by code are computed when the result is built, from the current profile and the vocabulary, and stored nowhere (`vacancyai.NewCodeMatcher`), so adding a skill or marking one as not yours shows at once, without a worker pass. The stored result carries `matchingOff: true` (for every usable vacancy, analysed or not) and, where requirements exist, the You fit status `off`; a view draws no You fit while the flag is set; the stored result's key carries the state of the setting, so switching it recomputes every result at the next start or profile save. The panel and the vacancy's Application tab show Fits you alone; the catalogue row shows one number; the order of the catalogue is Fits you, then id, because every You fit counts as the midpoint. The requirements are listed under "Requirements", required before preferred, with ✓ for a match decided by code, ✕ for a skill marked as not yours, and a neutral dot for every other line; the ✓ / ✕ actions of section 14.12.7 are offered on skill lines as before.
- **Matching quality** — gold set, method comparison (local models, embeddings with a threshold, a pair classifier, Laya), the decision, and the old Q.8 (verdicts keyed by what the judge saw, not by the profile revision): card ASSESSMENT-5.
- **Shortlist re-check on a stronger model**: card ASSESSMENT-6.
- **Sections**: one splitter in Go, stored with the vacancy, used by the view (By section) and available to the extraction: card INBOX-2 (step L.5).
- **The research tools stay** on the branch: diagnostic live tests run only with their environment variables; `scripts/model-bench/`.

## 15. Agreed pipeline direction — profile as a fact store, vacancy by sections (owner, 2026-11-11; agreed, not implemented)

Status: agreed direction. It governs the planning of INBOX-2 (L.5), COL-2, ASSESSMENT-5, ASSESSMENT-6 and ASSESSMENT-7. Sections 14.2, 14.12 and 14.14 stay the implemented behaviour until those cards replace them; each card moves its part from "agreed" to "implemented" in this section.

### 15.1 The profile is a store of facts

The profile is a store of accepted facts. A model runs only when a new file or link is added, to propose facts; the owner accepts them. Every comparison afterwards runs on accepted facts only, never on a document. For a requirement of a vacancy there are three outcomes: a fact supports it; a fact says the owner does not have it (the "Skills you do not have" list, ✕); no fact addresses it. The third is shown as "not in your profile" with the action to add it (✓) and counts as not met in You fit; it does not claim the owner lacks it. (This replaces the "complete CV" wording of section 14.12.1; the verdict utilities are unchanged.)

### 15.2 A vacancy, step by step

1. **Keep the original.** The fetched HTML (or feed item) is stored as received, so it can be parsed again when the parser improves, without a new fetch. It is never rendered directly (scripts, inline styles, trackers, broken markup); the view shows the sanitised structured form (INBOX-2), styled only by Sifty's CSS.
2. **Sections by code.** The parser splits the text into sections by headings and by label lines, using the rules of the vacancy's source (15.3) and a general heading vocabulary (About, Role, Responsibilities, Requirements, Nice to have, Compensation, Benefits, Location, Process, Legal/EEO), including headings glued to the following text ("About UsParity…"). Each section is classified by that vocabulary; an unmatched heading keeps its own text and the class "other". A text without any detectable heading is a parsing failure of its source, not a normal state (15.3).
3. **Route sections.**
   - About the company, Compensation, Benefits, Location and Conditions feed Fits you. Code reads labels and numbers first; the model reads only these sections, and only for fields the code did not find.
   - Requirements and Nice to have feed You fit. Code first: the owner's skills (with the synonym vocabulary) are searched in these sections; a hit is a supported requirement without a model. Then the model splits only these sections into single requirements (required / preferred from the section class).
   - Legal/EEO and Process are not used for scoring.
4. **Score.** Fits you by code over goal facts (sections 6 and 14.1). You fit per requirement: code first (skill vocabulary, years from dates, the owner's ✓ / ✕); for the rest, the nearest accepted facts are found by meaning (embeddings, not shared words); no close fact → "not in your profile" without a model; a close fact → a narrow question "does this fact prove this requirement" answered by the chosen pair judge with a confidence threshold calibrated on a labelled set (ASSESSMENT-5, candidates in ASSESSMENT-7).
5. **Shortlist.** On entering the shortlist, steps 3–4 run again on the model configured for the shortlist, or once more on the same model when none is configured (ASSESSMENT-6).
6. Preparing an application is out of scope here.

### 15.3 Parsing rules per source and their repair

Each source has its own parsing rules (where the description is: page body block, JSON-LD, feed item; how headings look), stored as configuration with the source, applied to every vacancy of that source. The parser measures its own result per source (share of vacancies with sections, glued or broken text, leftover entities). When a source falls below the threshold, a repair run analyses a few stored originals of that source (a model may propose the rules), the proposed rules are accepted only if they parse a held-out sample of stored originals at least as well as the current ones, and then every stored vacancy of the source is parsed again from its original. Because originals are stored (15.2.1), repair never needs a new fetch. Card: COL-2 (its existing `HintStore` seam).
