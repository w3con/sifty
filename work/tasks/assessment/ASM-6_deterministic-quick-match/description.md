# ASM-6 — Deterministic Quick Match from confirmed profile facts

**Epic:** assessment
**Created:** 2026-11-09
**Status:** review
**Kind:** task

**Integration and production status,2026-11-10:** technical implementation accepted; source published to main in commit `58af432` as the final application source. Pinned main `157fe9f` was then released to production at <http://localhost:8080> (build `157fe9f6d0f8-20261110T151309.703877Z`, PID71548). The confirmed staging profile was transferred selectively (`local@20`, 20 revisions, 8 documents, 22 facts, referenced PDFs byte-identical, avatar added); startup recalculation produced 2777 numeric Quick Match results. Production jobs, 1493 historical model assessments, workflow/Apply history, configuration and the disabled queue were preserved by ordered hash. The temporary 8083 review server is stopped. Card remains review, not done; owner review of ranking quality remains separate. Release detail: `production-release.md`.
**Planning:** owner «запускай» (2026-11-10) approves the full Q.1–Q.5 plan and presented v1 numerical defaults; bounded execution starts11:39Z, whole remaining upper stop13:44Z. Main publication/production release not included.

## Outcome and actual use

The owner reviews an actual catalogue ranked by a deterministic no-LLM Quick Match, opens Overview, sees criterion scores/evidence/unknowns and can identify a missing profile fact. The result is an aid to prioritization, not a hiring probability or an automatic rejection. Acceptance uses the existing isolated review catalogue and a confirmed test profile; production data and the separate model queue remain untouched.

Persistent behavior lives in [docs/specs/quick-match.md](../../../../docs/specs/quick-match.md). This card owns the change plan, cost, checkpoints and evidence, not a competing behavior definition. Exact original request is in comments.md; later decisions and research are in log.md.

## Authorization and existing work

Owner: «с остальным согласен. Обновляй specs, и заводит карточку. ну и планируй работу». This authorizes the specification, backlog capture and planning; not the application's implementation. Owner further requires extraction and scoring to be separate replaceable modules. Sources/types/contracts must allow replacing one without rewriting the other. Python or other implementations may inform a Go port; no assumption that only Go libraries are relevant.

The integration prerequisite is already technically accepted: application source adfdadd4c362fbc183199c04cea9ce0b689a6064 combines DESIGN-3/APP-8/COLLECTION-1 and Profile. Records through72cc92c and later documentation commits are in .worktrees/ASM-6. Designer independently fast-forwarded DESIGN-3 toadfdadd and owns subsequent Overview work. Main application is not integrated/released. Preserve root dirty material and all original branches/data.

## Scope and exclusions

Deliver normalized evidence-bearing vacancy facts, a pure evaluator, safe versioned persistence/API and minimal rendering through the existing catalogue/Overview. The existing typed saved Profile is authoritative; do not build a new profile store. Separate the shared domain contract, extractor/source adapters, evaluator, effects/storage and views. Reuse existing config/Dirs/services; adjustable defaults in one configuration location. No user/mockup data, provider choices, addresses or weights in views.

No user weight-calibration UI (ASSESSMENT-1), hard-dealbreaker UI, User/multiple Profiles/accounts, broad NLP/taxonomy, daily LLM source learner, recrawl/new collection, currency/tax inference, source-admin UI, automatic age penalty, model queue redesign, application submission, main publication or production release. Unknown inputs stay visible and are not invented to make all five groups score.

## Estimate for approval

- **Complexity:**13 — evidence extraction, explicit contracts, evaluator rules, migration/stale-write protection and actual catalogue integration are coupled. Keep one outcome because an unconsumed extractor or isolated calculator would not deliver prioritization; implement bounded independently reviewed stages rather than create standalone infrastructure cards.
- **AI preparation:**15–25min for this specification/research/plan; **AI implementation:**70–105min; **AI acceptance:**10–20min. **Total95–150min**, owner waiting excluded; approximately80–125min remaining after preparation. Existing subscription, attributable monetary cost unknown; no runtime LLM/API charges introduced.
- Earlier65–115min was explicitly provisional before input audit. Revised range adds30–35min for prepared facts/source reliability, replaceable contracts and lifecycle integration rather than assuming structured vacancy inputs already existed. Scope/cost must be approved before Q.1. Calibration remains separate provisional55–100min, not part of this total.
- Specification preparation approximate actual20–25min; no application execution clock started. Record final actual stage times in log. At approved upper bound stop and preserve progress before any further work.

## Acceptance

- A real confirmed profile ranks the existing isolated catalogue and yields criterion contributions, exact evidence and actionable unknowns in the running app, not only self-tests.
- Same inputs/rule version → same result, independent of corpus/search/repeated terms. Missing salary is not scored zero; ambiguous source location is not a false confirmed conflict; zero differs from unscored.
- Extractor and evaluator compile against the shared versioned contract, not each other; changing weights/profile reuses persisted facts without fetching/extracting again. Incompatible fact schema is refused honestly.
- Profile/job/rule changes invalidate results; stale in-flight writes cannot mark old scores current. Migration preserves schema18 data, known publication dates/basis, profile revisions, workflow history and Apply reasons.
- Affected builds, few critical hidden invariant checks and real browser review; broad suites skipped. Benchmark pure processing and storage separately on300saved jobs without network/model calls, report actual measurements against proposed targets.
- Complete plan and numeric proposals in plan.md are approved before executor application work. No claim that this spec already describes released functionality.
