# ASM-6 — integration prerequisite; scorer plan remains pending

estimate: 8

## Approved observable outcome

Owner: «хорошо, делай что надо и скажи когда Claude может продолжить.» Following the coordinator's summary, this authorizes integrating the finished Profile/Apply/freshness/Design sources and checking the shared base so Claude can resume Overview in its existing worktree. The mockup is already finished. This stage does not authorize scorer semantics or implementation. Beneficiary: owner and paused Overview designer; actual use is continuing Overview from one checked revision containing both branches.

## Scope and boundaries

Prepare work/ASM-6 in .worktrees/ASM-6 as a descendant of DESIGN-3 71f949a6c0365673f591e55d30c2ba61f2d86133, COLLECTION-1 2b8a1d55304b0619538721920b06aa2a57037fb8 and current committed main e2c4da36af4d6312084b8fec74668438d99d10fb. Integration on this work branch is expressly assigned Git work; no base-branch publication, work-finish/release bypass or feature-history transfer into main. Preserve all original branches/checkpoints, root dirty/untracked files, current DESIGN-3 worktree and live production/staging/data. Once accepted, Claude can fast-forward its own clean worktree to the resulting descendant; no reset/rebase required. Do not mutate DESIGN-3 from the executor.

Retain five Apply outcomes, button-only dismissal, persisted reasons and corrected schema18 migrations; source-date basis/age/Newest ordering; Designer's Profile/filter/resize/navigation changes. Resolve conflicts with both features, not wholesale file replacement. Preserve newer main captures and accepted worktree evidence. Existing unsupported document deletion/preferredRegions/profile deletion and score-zero semantics remain named limitations for subsequent work, not authorization for implementing those APIs or new scoring now.

Keep domain data in existing store/profile/config services, effects in existing API/hooks/workflow adapters and rendering in views. Config/settings sources remain existing Runtime/config/Dirs and persisted profile; no mockup data, ports, provider choices or new settings machinery in product logic. Do not refactor unrelated existing defaults for this integration.

## Estimate and stopping boundary

- Complexity: 3 — bounded branch integration with frontend conflicts and existing evidence.
- AI preparation: approximately5min; implementation/checks12–18min; acceptance/handoff3–7min; total20–30min including shared preparation. Existing subscription; monetary attribution unknown, no runtime model costs. Stop at30min from stage launch, preserve partial work and report rather than adding another cycle. Owner/external waiting excluded.
- I.1 result: one recoverable common-source revision; necessary to remove divergent frontend bases;10–15min including preparation/bootstrap.
- I.2 result: affected builds, focused hidden migration/outcome checks and short actual Chromium review; necessary to avoid handing Overview a broken baseline;7–10min.
- I.3 result: exact accepted SHA and safe fast-forward handoff to Claude; necessary for real continuation;3–5min, coordinator acceptance included, no duplicated cost.

## Checkpoints

- [ ] I.1 — Integrate fixed Design, accepted cumulative application and committed main in own worktree; preserve originals.
- [ ] I.2 — Build backend/frontend, run named preservation checks, inspect integrated UI in real Chromium and record limitations.
- [ ] I.3 — Coordinator accepts the base and records the exact safe continuation instruction for Claude.

## Acceptance criteria

- Both source checkpoints and current committed main are ancestors of the handed-off revision; root dirty contents unchanged except explicitly maintained ASM-6 planning records. DESIGN-3 remains clean and pinned until its owner advances it.
- Apply modal retains five outcomes and cannot close via backdrop/Escape; Shortlist/Inbox labels remain consistent in merged visible text. Date/basis/age and Newest behavior survive alongside Designer filters/resizing/Profile.
- Go build and frontend typecheck/build pass. Reuse only named critical hidden tests for v16→17/18 data preservation and stale outcome guards; broad suites skipped. Record exact test names/results, not blanket test claims.
- Actual browser renders integrated catalogue/detail/Profile, with collection/model/nonessential mutations blocked. Reuse existing isolated8083data/runtime where possible, no fresh environment setup/matrix, no model/search/collection calls, no production/staging data copies or writes.
- Handoff names the integration worktree, commit and fast-forward command for the existing DESIGN-3 branch. No claim of main publication/production release or implemented scorer.

## Later scorer work

No executable scorer steps yet. Rules/unknown/formula/weight examples, complete implementation scope and revised estimate require separate approval. SETTINGS-5 remains deferred.
