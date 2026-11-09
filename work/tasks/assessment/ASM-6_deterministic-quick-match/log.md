# ASM-6 — resume after compact

## Latest checkpoint — supersedes old dirty DESIGN-3 instructions

Owner supplied Design handback, verified by git:71f949a6c0365673f591e55d30c2ba61f2d86133 (`feat(design): profile page, header profile switcher, filters and stage renames`), parentfe31cc387410f090e5e7ba1736ba6ac31d2fe674. Exactly15frontendfiles,13modified+2new(ListResizer.tsx,profile/ProfileSwitcher.tsx),787insertions/191deletions. Worktree clean. No backend changes. COLLECTION-1 also verified clean at2b8a1d55304b0619538721920b06aa2a57037fb8.

Design's agreed Overview brief: source text stays left with no lost/unrecognized content, right contains only actual facts or Not stated; existing Quick Match shown without invented weights; room for future status/explanations/unknowns after backend contract; LLM presentation in Application only for selected roles. Design will WAIT for branch integration before coding RoleContent.tsx/main.tsx and requests exact integrated base SHA. Picture-only mockup is allowed in parallel. User asked to keep Design in the same worktree for now; yes, preserve it. After integration decide updated-base feature branch/worktree for Overview and give Design that pinned commit, not a moving/dirty tree.

**Immediate post-compact input is now available:** do not ask for the checkpoint again. Prepare/review a preserving integration plan using this Design commit + accepted COLLECTION-1 descendant of corrected APP-8 and SETTINGS1/2/3. Root main remains dirty with owner/Designer/planning material. Avoid wholesale tree replacement (would remove newer captures/settings5/asm6), preserving main-history per-card squash policy and current data/recovery. Integration conflicts include main.tsx,ApplicationConfirmation and shared styling; retain new five outcomes/button-only behavior, source date/basis/Newest/schema18, and intended Design labels/filters/layout. Known unsupported document DELETE/preferredRegions/score0vsunscored need explicit readiness handling, not silently claiming working APIs. No main merge, production release or new scoring implementation has been performed yet.

## Latest owner direction

Owner asks this session to handle deterministic weighted scoring from confirmed profile facts, no LLM; Claude Design handles Overview/Profile/filter UI. Owner then: «с твоей работой - согласен. сейчас сделаем /compact и приступай». Resume after compact; no scoring executor has been raised. Owner's full no-LLM request is captured verbatim in comments.md.

User/Profiles architecture is **deferred**, complete design in SETTINGS-5/plan.md. Owner agreed separate per-profile Triage/Preparation/rejections, common actual-submission history, one person, no accounts/logout now. Consume today's confirmed Profile as scorer input; no new User/multi-profile implementation in ASM-6.

## Accepted source and runtimes, verified2026-11-09T22:47Z

- main HEAD e2c4da36af4d6312084b8fec74668438d99d10fb: latest commits are task captures, NOT accepted application integration. Many root owner/Designer/planning changes dirty/untracked, preserve all. Never stage-all/stash/reset or commit unrelated standing instructions/captures implicitly.
- .worktrees/COLLECTION-1, work/COLLECTION-1: clean2b8a1d55304b0619538721920b06aa2a57037fb8, accepted app source d92112c04a3f40e24fd306a7b19d7c4a422625d5, schema18. Date/completed age + Newest tiers (known publication, legacy unverified source date, missing), basis field, adapter/enrich/date corrections. Full acceptance in that card. About35min total, defaultwork-testSKIPPEDonce, narrow checks/builds passed. Owner review/main integration/release remain separate.
- .worktrees/APP-8 clean d2db849001d465113d794ef2eedebe95b75de880, accepted five Apply outcomes and button-only dismissal, v16→17 migration gate fixed. Old source28a27f has a migration bug; use corrected descendant, not initial feature commit. Owner accepted via «принимай». Why-not-suitable free text remains captured follow-up, not implemented. Overrun~39min plus acceptance reported; do not repeat environment setup/browser matrix.
- .worktrees/DESIGN-3 branchwork/DESIGN-3 HEADfe31cc387410f090e5e7ba1736ba6ac31d2fe674, **active other writer**,13trackedmodified+2untracked files at last check. Do not write/rebase/commit their work yourself. Get a checkpoint from its owner before integration. It predates APP-8/COLLECTION-1; a wholesale checkout/replace would lose accepted changes.
- Production8080PID79846, actualdata/data/store.sqlite; owner says all real submissions were on8080, sometimes applied→return→reject/remove. No environment history reconciliation requested. Production was not changed by recent cards.
- Design frontend8081PID64422; API8082supervisor12715. Their source is DESIGN-3 and data-staging; preserve while the other agent works.
- Independent review8083PID75373, COLLECTION-1 binary/dist, existing isolated app8-iso data reused; test copy, no real submissions recorded into production. Test known-publication fixtures999001/999002 explicitly Collection Fixture/example.invalid. Real copied990008/990009 are StreamingFast, not fixtures; temporary invented basis was caught/restoredNULL. Do not reuse those IDs as synthetic fixtures.
- All our executor sessions are stopped. Actual implementation provider/model verified for APP-8 and COLLECTION-1 including resumes: opencode-go/deepseek-v4.1-flash only. No direct-billed fallback.

## Actual DESIGN-3 review findings

Diff is not only Profile/CSS: ApplicationConfirmation, ApplicationDraft, CatalogueFilters, main.tsx, workflow.ts, theme and Profile files; new ListResizer.tsx/ProfileSwitcher.tsx. About669insertions/191deletions in the inspected tracked diff, plus new files.

Behavior changes: Triage/Preparation labels become Inbox/Shortlist; Roles/Skills filter groups removed and stored selections reset; new fit buckets/Collected7/30days filters; list resize; import form removed from list. These may be owner-approved Design decisions — preserve/inventory, do not automatically revert as 'unrelated'. Newest publication and Collected filter dates have different meanings and must stay distinct.

Integration overlaps main.tsx/ApplicationConfirmation with our five outcomes/date labels; must preserve both desired UI changes and accepted backend/behavior. New fitBucket(score<=0→unscored) conflates a genuine0 with no evaluation; ASM-6 needs explicit ready/stale/unscored status. Profile's preferredRegions exists only frontend, not the current Go Geography contract; DELETE profile/documents/{id} UI calls a not-yet-implemented endpoint. Current ProfileSwitcher supports one profile, browser-only rename, Add disabled; not real multi-profile persistence. Review actual current diff, not only prior acceptance of oldfe31cc3.

## Scoring/Overview design boundary

ASM-6 inputs: confirmed current profile + supported vacancy facts + single authoritative weights config; output score/status/coverage/matches/conflicts/unknowns/evidence and version/fingerprint. Missing profile skill ≠ confirmed inability; skill mention ≠ requirement; repeated text can't accumulate unlimited weight. Remote≠worldwide. Salary only compared for compatible units/currency/tax basis. Unknown≠mismatch;0≠unscored. Same inputs/rule version stable independent of unrelated corpus additions; don't disguise BM25 corpus normalization as absolute fit.

No invented keyword contributions in Overview. Structured facts require actual source fields/quotes; uncertain regex results remain unknown. Current job input lacks structured salary/contract/requirements; CleanHTML flattens headings/lists, so section ordering needs a real backend structure while original text remains. Unknown text should not all be collapsed as About: it can contain requirements. Separate persisted source data, extraction/calculation logic and view rendering. No broad source scraping, generic NLP framework, dictionary project or hidden User/Profile migration.

LLM presentation in Application is not a server-side gate: background.AssessmentWorker periodically calls Store.EnqueueAll; its SELECT id FROM jobs includes all jobs and queue execution isn't restricted to selected stage. Queue currentlydisabled was preserved; never enable old pending rows during review. Explicit LLM button after Select was recommended; automatic-vs-explicit trigger was not separately finalized. Keep Quick Match itself model-free and settle any change to paid execution behavior in the complete plan.

## Next precise actions

1. Ask/get immutable DESIGN-3 checkpoint SHA (including new files) without taking ownership from the live Design session. It can remain in the same worktree for now; later updated-base branch/worktree choice follows integration. Preserve all its ongoing work.
2. Re-read root status/diff and prepare a preserving integration plan for accepted SETTINGS1/2/3, DESIGN3 UI, APP8 and COLLECTION1. Main is dirty, source has cumulative unintegrated Profile history and newer canonical task captures exist only in main. Normal work-finish must not be bypassed and a whole-branch tree replacement is unsafe. Keep meaningful per-card squash history, existing guards/data preservation, no unapproved production release.
3. Agree the minimum JobFacts/QuickMatch response with Design and finalize deterministic criteria/unknown/coverage/weights, then complete ASM-6 step/results/time/cost plan before an executor. Estimate currently65–115min provisional, not permission to spend it unseen. Reuse review environment, few hidden invariant checks and short actual browser review.
4. Designer can build Overview on current API with honest unknowns; subsequent feature should use the integrated base rather than remain indefinitely on a diverged snapshot. No concurrent edits of the same worktree/backend models.

AGENTS role split remains: coordinator plans/accepts only, all implementation/mechanical Git/build/deploy by configured visible DeepSeek executor, bounded stage/exact step IDs/liveworkbrief, actualmodelverification. No broad suites by default. Raise a power assertion before long work. Do not start anything long before the user's requested compact.

## 2026-11-10 — note

Bootstrap stage start (executor): 2026-11-09T22:59:52Z; hard deadline 2026-11-09T23:29:52Z (30min incl. continuation and coordinator acceptance). Scope: commit only this card's planning files, then obtain the live work-handover brief; no implementation.
