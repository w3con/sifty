---
estimate: 13
---
# ASM-6 — Quick Match implementation plan for approval

## Outcome, source of behavior and approval

The owner ranks actual saved vacancies against their confirmed profile, opens a role and sees comparable criterion scores, evidence and profile gaps. [The behavior specification](../../../../docs/specs/quick-match.md) is authoritative; this card records the change plan and costs. Owner approved this complete plan and its v1 numerical proposals on 2026-11-10 with «запускай». The numerical policy remains uncalibrated, not a probability or proven preference scale. Q.1–Q.5 are authorized as bounded stages, reviewed independently. Execution begins approximately 11:39Z; remaining budget80–125min, upper stop13:44Z, excluding owner waiting. Coordinator ses_e4a7d8341ffe1Xqrv4Rmx52jqw owns work/ASM-6.

Owner preparation authorization: «с остальным согласен. Обновляй specs, и заводит карточку. ну и планируй работу». Architecture steering: «сразу архитектурно разделяй эти модули. чтобы их можно было потом переписывать/заменять». Calibration of weights by the user is separately captured as ASSESSMENT-1, not v1 scope. A Python implementation can be the algorithm/reference basis; scikit-criteria WeightedSumModel is inspected and BSD-3-Clause permits adaptation with notices. Porting mathematics does not require a Python runtime. Record exact source revision/license if copying/adapting implementation; pyDecision GPL code is not an approved dependency.

## Source and ownership

Worktree .worktrees/ASM-6, branch work/ASM-6; application baseline adfdadd, accepted integration records72cc92c, subsequent preparation commits506ffbe/da4af10/9d3cee1. DESIGN-3 has independently advanced toadfdadd and owns Overview; do not edit/rebase/reset its worktree. Coordinate the API/result contract before frontend work; integrate only a fixed finished Overview checkpoint into our own branch if needed. Root main has dirty owner/documentation files and new ASSESSMENT-1 capture4156d24: preserve them, no wholesale tree replacement or implicit commits.

## Architecture and authoritative inputs

- Shared versioned domain-only `matchcontract`: profile projection, source snapshot, vacancy facts/evidence, rules and result. No store/provider/HTML dependencies.
- Replaceable `jobextract`: saved structured source fields/retained text + injected source-field policy → facts. It imports the shared contract, never the scorer. Preserve original information; ambiguous/flattened legacy text stays unknown rather than being reconstructed or fetched.
- Replaceable pure `quickmatch`: confirmed ProfileSnapshot + VacancyFacts + Rules → Result. It imports the contract, never extractor/site adapters/store/providers; no I/O, wall clock or corpus state.
- Existing services/adapters own effective profile projection, config loading, extraction and score orchestration, persistence, cancellation and API. Existing SQLite/profile/Dirs/runtime config remain authoritative. Source usability and weights are explicit data in one existing config extension, not UI constants or a new settings framework.
- Separate production chains: retained-source hash + extractor/schema/source-policy version → facts hash; profile revision + facts hash + score-rules version → result. Profile/weight change reevaluates persisted facts without parsing or downloading. Extractor-policy change can re-extract retained inputs locally and invalidate dependent results. Incompatible schemas are refused with an actionable status.

## First-version boundaries and numerical proposals

Five group defaults proposed: role25, professional30, compensation20, geography15, conditions10. Weighted mean over comparable groups, separate subcriterion-weighted coverage; unknown is not zero and coverage is not multiplied into the score. Rules/defaults are versioned and await approval, not validated user preferences.

- Role: explicit saved target titles versus actual job title; maximum token-F1 over alternative targets after documented normalization/approved small aliases, no corpus frequency. Label this lexical title alignment, not inferred leadership suitability.
- Professional: explicit required/preferred skill evidence, aliases/word boundaries only, no repeated-word inflation. Required/preferred relative weights2:1 proposed. Missing saved skill is not_evidenced and reduces coverage, not a fabricated confirmed lack. Company-stack mentions are context only. Numeric years, unstructured domain experience and inferred seniority stay unknown in v1 unless both sides provide a directly comparable explicit fact; do not infer years of a skill from total employment history.
- Compensation: actual advertised cash ceiling versus confirmed minimum, same currency/unit/tax basis; proposed `min(1, jobCashMax / candidateCashMin)`. A range ceiling means potential, not guaranteed offer. Missing/incomparable salary is excluded, not zero; no estimated salary, currency/tax inference, bonus/equity conversion. Example90k minimum vs80k ceiling →88.9 salary and97.78 overall when every other group fully fits.
- Geography: compare explicit permitted hiring locations with declared preferences, and explicit work-authorization requirement with owner-declared authorization. Confirmed compatible set→1, confirmed incompatible exclusive set→0; omitted/ambiguous restriction or omitted authorization→unknown. Office address is not a hiring restriction, remote is not worldwide, residence implies no work right. Region names compare only with an explicit supported mapping; unsupported frontend-only preferredRegions is not silently consumed as persisted data.
- Conditions: compare explicit work-mode/employment/hours facts with saved allowed alternatives: compatible set/range→1, clearly disjoint→0, missing/ambiguous→unknown. Timezone/availability/free-text restrictions without an actual comparable structured fact remain visible unknowns, not regex-derived penalties.
- Coverage includes missing members inside partially known groups, not merely number of known groups; absent user preference is not_applicable. Score stays numeric for any comparable evidence and is marked uncertain by actual coverage; no usable evidence→null/unscored; evaluated0 remains0. Sort by score, then coverage, then stable job ID; no hidden coverage penalty. Show coverage explicitly so uncertain100 is not mistaken for established fit.

## Must not touch

No weight-calibration UI, hard-dealbreaker editor, User/multiple Profiles/accounts, daily model template learner, general taxonomy/NLP, source-admin framework, recrawl/collection/search, model calls/queue activation or pending-row deletion, automatic age penalty, data refresh, genuine owner-history edits, production release or main publication. FTS remains query/search support; new score must not be overwritten by the old BM25 rescore path. Do not fix unrelated Profile delete/region/avatar issues during this card.

## Estimate, step results and costs

Complexity13: extraction meaning, replacement boundaries, versioned data writes and real catalogue/UI adoption are coupled. One task is retained because extractor/calculator alone would not deliver the owner's prioritization; each stage below is independently reviewable. Preparation15–25min + implementation70–105min + acceptance10–20min = **95–150min total**, approximately80–125min remaining after preparation. Previous65–115min provisional range is revised by30–35min for actual missing vacancy inputs and replaceable/versioned integration. Existing subscription; attributable monetary/external cost unknown for every step; no runtime model/API costs. Owner waiting excluded; tool/network delays recorded separately. This revision requires approval before Q.1.

| Step | Concrete result and why required | AI elapsed |
|---|---|---|
| P.0 | Agreed spec, Python/reference evaluation, calibration capture, complete plan/navigation; supplies one behavior source before coding. |15–25min, approximately completed |
| Q.1 | Versioned shared contract plus independent extractor/source-policy adapter and retained-input fact persistence; supplies real evidence instead of guessed view fields. |25–35min |
| Q.2 | Pure evaluator with proposed scales, exact contributions and unknown/coverage handling; produces deterministic useful results from the contract. |15–25min |
| Q.3 | Profile projection, safe migration/CAS result storage, API and local recalculation triggers replacing old score writes; makes results current and preserves existing data. |20–30min |
| Q.4 | Minimal existing catalogue/Overview rendering and filter/sort adapters, coordinated with Design; owner sees actual scores/reasons and zero/unscored distinction. |10–15min |
| Q.5 | Benchmark300saved jobs, few hidden invariant checks, affected builds, short running-browser review and spec reconciliation; demonstrates usefulness and actual speed. |10–20min, acceptance included |

No optional process/framework work is bundled. Calibration is independently deferred55–100min and not included. Record actual stage time and spend boundary at every handback; stop when approved remaining upper bound is reached and report progress/re-estimate before another stage. Do not silently repeat a browser matrix or build a new environment.

## Checkpoints

- [x] I.1 — Combine fixed Profile/Design/Apply/freshness sources in own branch; preserve originals.
- [x] I.2 — Verify shared base builds, preservation checks and actual browser behavior.
- [x] I.3 — Accept shared application revision and hand off safe continuation to Claude.
- [x] P.0 — Prepare specification, calibration capture, replaceable-module plan and owner-facing approval summary.
- [x] Q.1 — Add shared versioned domain contract, independent extractor and retained evidence facts.
- [x] Q.2 — Implement pure configurable evaluator and deterministic criterion explanations.
- [x] Q.3 — Persist versioned results safely and connect profile/collection/API recalculation without models.
- [x] Q.4 — Render actual Quick Match results in catalogue/Overview without duplicating Design work.
- [x] Q.5 — Measure saved-job throughput, verify hidden invariants and demonstrate actual review use.

## Acceptance and required checks

- Shared contracts, extractor and evaluator are distinct Go packages. Evaluator has no extractor/HTML/site/network/store dependencies. A different extractor implementing the same contract can supply facts without editing evaluator; changing rules/profile reuses saved facts without fetching/parsing. Use a small targeted contract/invariant check, not a plugin framework or exhaustive suite.
- Few critical hidden checks: determinism/corpus independence/repeated tokens; missing salary versus zero/partial coverage; v18 migration and failed/stale write preservation; no model calls and separate invalidation chains. Group related cases in a handful of focused tests. Do not add visible-label interaction tests or broad preserved-suite/race runs.
- Migration retains profile revisions/fact provenance, jobs/workflow history, Apply reasons and publication dates/basis. Existing detailed model assessments stay historical; no silent inference rerun or pending-queue deletion/activation.
- Real saved catalogue and confirmed profile produce scores/contributions/unknowns. In an actual browser the owner can select a role, inspect evidence, distinguish missing salary/source-location ambiguity from incompatibility, and find the profile section to edit. This is required adoption evidence, not only passing self-tests.
- Reuse independent8083 and isolated data; do not copy/refresh production/staging. Block App-mount search/nonessential mutations during browser review. Build affected backend/frontend. Default work-test reported SKIPPED separately from named critical checks; no broad tests unless owner asks.
- Benchmark on300saved representative jobs without network/model calls: measure extraction separately, calculation separately and complete read/evaluate/save batch. ≤0.5s/job initial budget; ≤30s/300 is a proposed local batch target, not a measured guarantee. Record hardware/config, input sizes, warm/cold state, total and tail latency. If target fails, report result and smallest justified change within approved time; no unlimited optimization.
- Reconcile implemented limitations and released status in spec/MAP. No done/main/production claim until normal acceptance/release approval. Preserved checkpoints remain recovery material.

## Resume and execution gate

Owner «запускай» approves the presented plan/numerical proposals. Obtain WORK_SESSION=1 work-resume and assign **Q.1 only** with25–35min bound. Later bounded stages follow technical acceptance. Q.1 deadline12:14Z; whole remaining-work upper stop13:44Z. Any new source parser, unsupported comparison or material estimate increase returns to scope review before execution. Main publication and production release remain outside this approval.

## Owner-authorized completion and main integration — 2026-11-10

Owner: «Claude design - закончил. весь дизайн и что-то по фронтенду - завершено.; Все что есть на design-3 worktree можно сводить в main. Ты заканчивай свою работу и своди тоже. 8083 - мне не нужен, можешь останавливать / использовать». This supersedes the earlier main-publication exclusion for the completed source union, authorizes taking the finished DESIGN-3 source, and permits reusing/stopping our review server. Production release and pushes remain separate. The coordinator accepts the B/C/D code corrections in8677836, but normal persisted actual-profile use remains unproven until J.1.

Observable outcome: the owner's main checkout contains the complete finished Design and Quick Match source, with existing Profile/Apply/date prerequisites preserved; the scored catalogue is demonstrated through the normal persisted API on an isolated snapshot, without writing original staging/production data. Preserve all dirty root material and original work branches/worktrees. No automatic work-finish/release, fabricated done.md, broad suites or model/collection requests. Existing runtime/config adapters remain authoritative; merge presentation with presentation, retain domain/config/effects boundaries and current Quick Match semantics.

Incremental complexity5: source overlap, dirty-main preservation and card-granularity publication. AI preparation5–7min, execution20–30min, acceptance5–8min; total30–45min, excluding owner waits. Start approximately13:30Z, upper stop14:15Z. Existing subscription, attributable monetary cost unknown, no runtime model charges.

| Step | Concrete result and necessity | AI elapsed |
|---|---|---|
| J.1 | Combine pinned finished DESIGN-3 2f6c026 with ASM-6 8677836 in our branch; build/browser review and demonstrate actual persisted scoring using a consistent read-only staging snapshot in a new isolated review directory. Inventory dirty main and propose exact per-card publication sources; original data untouched. |10–15min union plus5–10min persisted-use acceptance|
| J.2 | Publish reviewed source to main in meaningful per-card squash commits, reconciling root dirty/untracked content without losing it; preserve source/recovery histories. Required to deliver the explicitly requested main result. |10–15min|
| J.3 | Verify the integrated main result, stop the verified8083server, reconcile navigation/spec/status and save the seven-section coordinator acceptance report. Required for a truthful durable handback. |5min|

- [x] J.1 — Combine completed design, demonstrate persisted actual-profile scoring and inventory main preservation.
- [x] J.2 — Publish reviewed source into main with per-card history and preserved root work.
- [x] J.3 — Verify main, stop8083 and record final acceptance and remaining release boundary.

Assign J.1 only first; stop before main writes and return exact per-card integration proposal for coordinator review. J.2 Git mechanics require a separate explicit coordinator assignment after review. Do not repeat default work-test (already SKIPPED); only affected builds and a small actual-browser acceptance of the merged result.
