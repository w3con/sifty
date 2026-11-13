# ASSESSMENT-2 — Dual fit and targeted vacancy extraction

**Epic:** assessment
**Created:** 2026-11-10
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
ок, кажется определили. давай обновим по результатам нашей дискуссии что нужно в specs и потом создадим новую карточку?
```

## Why this work exists

The owner prioritizes a real catalogue by two readable meanings of fit — whether a role fits the owner's goals, and whether the published employer requirements match confirmed skills, experience and eligibility — instead of by the completeness of our parsing. The observable outcome is a running catalogue that distinguishes potential suitability from source fragility, shows concrete matching and mismatching facts with honest unknowns, and uses targeted extraction only where a concrete gap justifies it. The owner is the beneficiary and the decision-maker reviewing results in the actual app.

This card is a follow-up to the released v1, not a duplicate of [ASM-6](../ASM-6_deterministic-quick-match/description.md). ASM-6 remains the v1 implementation/release record; [ASSESSMENT-1](../ASSESSMENT-1_user-calibration-of-quick-match-weights/description.md) remains the separate deferred calibration card; ASM-5 remains its own shared-term discussion.

## Behavior authority

The agreed product behavior is recorded in [the Quick Match specification, section 13](../../../../docs/specs/quick-match.md), with the current v1 formula and released behavior preserved as the implemented legacy state. This card links that specification rather than duplicating its requirements.

## Known source gap

Parity tech lead (ID 84) was observed read-only with a saved description of "lead tech lead blockchain +3" and a displayed 100 at 29.4% coverage. That is a source-acquisition problem, not a strong fit: no overall 100 from title alignment alone. Where possible, recover full text through the existing acquisition adapter without inventing facts. Text that is present but not parsed by the strict deterministic extractor must not leave most of the catalogue unscored.

## Existing boundaries and reuse

The released `matchcontract` shared contract, `jobextract` deterministic extractor/source policy, pure `quickmatch` evaluator, persistence/effects and UI boundaries remain authoritative. Future targeted extraction is a separate fact-extraction adapter before persisted facts, not a scorer or hiring judge; it processes a vacancy without the candidate profile and returns validated supporting quotes. Existing configured providers, path resolution, configuration and queue adapters must be reused; no model, provider, port or tunable is hardcoded. A model-facing extraction queue must remain logically distinct from the existing detailed Fit Assessment/preparation queue and must not enable or consume it.

## Exclusions

The approved plan authorizes bounded implementation and staging acceptance with at most three real requests to the already configured extraction provider. No production deployment or queue enablement, provider/model replacement, new generic crawler or mass recrawl, user weight-calibration UI, development-process changes, or changes to the existing assessment queue are included. This work does not rescope ASM-6, ASSESSMENT-1 or ASM-5.

## Approved starting policy

The coordinator's [complete implementation plan](plan.md) records goal-fit weights/order, a requirements interval, narrow insufficient-source handling and bounded use of the existing extraction provider. After the complete proposal and costs were presented, the owner approved them with «запускай». They are approved starting policy, not calibrated preferences or already implemented behavior. Provider/model replacement, preference calibration and selected/opened-role boosts remain outside this implementation; items not fixed by the plan remain open.

## Whole-task acceptance (approved plan)

- In the actual running app, a real saved vacancy shows both fit directions and concrete matching/mismatching facts before and after targeted extraction; unknown values are honest and no fact is invented.
- A completed extraction attempt that finds no new facts leaves scores unchanged and does not endlessly requeue the role on the same input.
- Eligible low-scored roles remain queued rather than permanently excluded, and the same gap is deduplicated.
- Migration preserves profile data, jobs, persisted facts, workflow/Apply history and existing v1 results; stale and unknown states remain distinguishable.
- No target extraction path enables, consumes or mutates the existing detailed assessment/preparation queue, and no paid model call runs without explicit approval.
- Owner reviews the visible comparison panel and wording; the implementation plan's checks cover the hidden invariants above without broad suites.

## Estimate — owner-approved with the complete plan

- **Complexity:** 13 — two directional comparisons of the same confirmed profile, persisted versioned facts, targeted provider work, queue admission/priority and a readable comparison UI are cross-coupled with material uncertainty.
- **AI preparation:** 20–35 minutes including the current discussion, specification and card preparation, and the future implementation-plan and policy confirmation.
- **AI implementation:** 120–210 minutes, including bounded stages expected for extraction, persistence, queue and UI.
- **AI acceptance:** 25–45 minutes for review, running-app checks and corrections.
- **Total AI elapsed:** approximately 165–290 minutes (about 3–5 hours), excluding owner waiting and provider/network latency. Unknown model price and catalogue volume are separate external charges, not included; no calls run now.

Keep one product outcome card because usefulness depends on end-to-end fact extraction → two scores → visible facts. The approved plan divides six bounded, independently reviewable stages rather than creating standalone unconsumed infrastructure. The estimate and the plan's uncalibrated starting numerical policy are approved with «запускай»; provider/model replacement remains excluded.

## Next step

Owner approved the presented [complete implementation plan and costs](plan.md) with «запускай». Prepare the sanctioned worktree and assign Q.1 only, then continue the bounded stages after technical review. Use one executor at a time because a token-neutral 2–3-agent allocation cannot currently be justified. Production release and push are not authorized.

## Preparation record

Documentation preparation and correction: approximately 5–7 minutes writing plus approximately 3 minutes coordinator review. Two transcription issues were corrected: the two-direction behavior had been written as optional ("may show") instead of the agreed mandatory direction, and criterion counts had been described as weighted-point substitutes rather than a communication of evidence scale. No implementation, model call or additional commit was performed.

Further coordinator preparation inspected the current extractor, comparison contract/evaluator, retained-fact persistence, configured inference reuse and one-job description recovery. The requirements comparator's 1-or-unknown behavior is a product-policy issue, so the interval was presented in the complete plan and approved with «запускай». The plan retains the approved 165–290 minute whole-task range and assigns Q.1 as the next bounded stage; no implementation has started. Owner's standing 2–3-agent permission with token-neutrality and approximately 1–2% incidental-overhead target is recorded in `AGENTS.md`; no orchestration changes were made.
