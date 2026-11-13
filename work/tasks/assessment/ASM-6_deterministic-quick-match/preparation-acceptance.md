# ASM-6 preparation acceptance record

**Date:** 2026-11-10. **Status:** documentation preparation complete for owner review; not application acceptance and not Q.1 authorization.

## Work completed

- Aligned `docs/specs/quick-match.md` with the root implementation plan: precise proposed v1 comparators for role, professional, experience, geography and conditions; retained structured-field/explicit-section extraction only; proposed sort by score, then coverage, then stable job ID.
- Refined applicability and evidence coverage: active only for a declared owner goal/preference or an explicit professional requirement; configured preference with missing job fact stays active/unknown; no preference is not_applicable; professional denominator includes required/preferred vacancy members missing from the profile; unknown-section lists are not exhaustive.
- Marked section 5 numerical coverage and section 6 numerical policy as **PROPOSED**, not approved by the section header. Added the actual scikit-criteria `WeightedSumModel` reference and exact source-pin notice condition.
- Mirrored the updated spec, plan, description, attention, MAP, ROADMAP and the narrow AGENTS persistent-spec paragraph into `work/ASM-6`; merged new root log sections while preserving all worktree integration/preparation evidence.
- Ticked P.0 as a coordinator-delegated mechanical record only. Q.1–Q.5 remain open and unapproved.

## Source paths

- `docs/specs/quick-match.md` (root and worktree mirror)
- `work/tasks/assessment/ASM-6_deterministic-quick-match/{plan,description,attention,log}.md`
- `docs/MAP.md`, `work/ROADMAP.md`, `AGENTS.md` (worktree mirrors; root docs remain dirty for coordinator)
- `work/tasks/assessment/ASSESSMENT-1_user-calibration-of-quick-match-weights/description.md` (deferred calibration link)

## Corrected drafting defect

The earlier draft inverted compensation utility (`candidateFloor / jobGuaranteedFloor`) and treated a band ceiling as a guaranteed offer. Final proposal is `min(1, jobCashMax / candidateCashMin)` for comparable positive employer cash, basis `range_ceiling`, no promise or unsupported tax/currency inference. Example: 90k minimum vs 80k ceiling → 0.8889 salary utility; 0.20 weight with others full → 97.78. This was a documentation defect caught before implementation.

## License and reference

scikit-criteria is BSD-3-Clause. Actual reference: https://github.com/quatrope/scikit-criteria/blob/master/skcriteria/agg/simple.py (`WeightedSumModel`). Porting weighted aggregation mathematics into Go is possible without a Python runtime and without a corpus-relative scaler; pin the exact revision and retain the notice if code is adapted later. pyDecision GPL-3.0-or-later code is not used.

## Checks and tests

Targeted documentation checks only: exact root/worktree spec mirror, navigation links, no inverse salary formula, 90/80 example, 1/20 coverage not full, proposed numerical boundaries and module dependencies present, targeted `git diff --check`. No broad map harness, product tests, build, browser run or default `work-test`; Q steps are open. **Actual application implementation: NO.**

## Elapsed

Agent preparation start: `2026-11-10T10:38:39Z`; documentation alignment through approximately `2026-11-10T10:56Z` (about17min), plus coordinator preparation5–8min, approximately22–25min total. User waiting excluded. Within the original15–25min preparation boundary if measured at handback; no new cycle started. Full application plan95–150min includes preparation15–25min; no execution clock is running.

## Limitations

Numerical policy and exact applicability/comparator choices remain proposed for owner review. Q.1–Q.5 are not approved. No application code, schema, runtime, data, build, model call, main publication or production change was made.

## Final link correction — 2026-11-10

Final review caught one wrong relative calibration link in section 6.2 (four levels up instead of two). Corrected in root and worktree. The earlier navigation check was incomplete; all four relative Markdown links now resolve from the spec parent and exist in both copies.
