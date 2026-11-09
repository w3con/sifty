# ASM-6 — Deterministic Quick Match from profile facts

**Epic:** assessment
**Created:** 2026-11-09
**Status:** backlog
**Kind:** task
**Planning:** design preparation only; complete implementation steps/costs and scoring semantics require approval before execution.

## Why this work exists

**Resume checkpoint:** owner «с твоей работой - согласен. сейчас сделаем /compact и приступай» accepts the proposed no-LLM direction and wants resumption after compact. Exact source/runtime/ownership findings and next actions are in log.md/attention.md. No scorer implementation or integration has started; full concrete rules/steps/cost plan still required. Designer stays in DESIGN-3 for now and must supply a fixed checkpoint before integration; never overwrite its active worktree.

Owner's exact request is in comments.md. The owner wants instant, deterministic vacancy prioritization using weights and confirmed current-profile facts, with no LLM. The practical use is manual Inbox/Triage review: see a score and the concrete reasons, keep/reject/shortlist a vacancy. This is distinct from ASM-4 BM25 and ASM-5 aliases; do not implement an unused generic scoring framework.

## Provisional estimate

- **Complexity:**8 — fact meaning, criterion/unknown semantics, persistent versioned results, existing ranking and explanation UI are coupled. Keep the working scoring outcome together, with independently reviewable parts; separate broad vacancy-section extraction/multi-profile changes rather than hide them inside it.
- **AI preparation:**10–20min; **AI implementation:**40–70min; **AI acceptance:**15–25min; **total65–115min**, explicitly provisional until the input contract/rules and integrated baseline are agreed. Existing subscription, attributable monetary cost unknown; no runtime model/provider costs. Owner waiting/external delays excluded. This is not an execution approval/stop clock.
- The minimum useful version must actually rank the current catalogue from the current confirmed profile and expose explanations, not just return scores in self-tests. Broader salary/location extraction or corpus-wide reparsing is separate if the agreed inputs are not available.

## Initial design and objections

Separate persisted/source facts/configuration, calculation logic and rendering. Proposed pure evaluator: current job facts + explicit current-profile projection + one authoritative weights/rules configuration → score/status/coverage/matched criteria/confirmed conflicts/unknowns/evidence. Reuse existing config, path/provider adapters and profile revision mechanisms. Future User/multiple role Profiles are SETTINGS-5 and deferred; consume today's confirmed profile through an explicit input adapter, not a new User/accounts system.

Deterministic means identical inputs and rule version produce identical results independent of other jobs being added or text keyword repetition. Do not retain corpus min-max BM25 normalization as a supposed absolute fit score. Existing FTS can remain search support; the new output must be honestly labelled and versioned.

Each criterion distinguishes match, mismatch, unknown and not-applicable. Missing a profile skill does not prove lack of knowledge; a mention in company boilerplate does not prove a required skill. Known aliases and word boundaries matter (e.g. Go as a language vs the verb); no LLM/semantic-learning/ESCO expansion hidden in the task. One repeated word must not accumulate unlimited points. Typed skill levels may affect only explicitly defined compatible criteria; do not invent required proficiency/years from a bare mention.

Structured source salary, employment/work mode/location/requirements are preferred. Preserve source field/quote and unknown status; remote is not automatically worldwide, and 'not remote' must not become a positive remote fact. Salary comparisons require comparable currency/period/tax basis; do not compare incompatible values or include equity as guaranteed cash. No general regex-guessed fact may silently become a hard penalty. Free-text Other requirements must remain unexecuted/unknown unless a deterministic supported rule is explicitly defined.

Propose separate coverage/completeness plus the score explanation: a single known positive should not masquerade as a reliable100% while everything else is unknown. The exact formula/unknown treatment/category weights require owner-facing examples and approval. Weights live in one explicit existing configuration location, not magic values in views. Score0 and not-scored are separate states; current DESIGN-3's score<=0→unscored bucket is not a sound new-score contract.

No automatic age penalty/hiding from COLLECTION-1; date/age and Newest remain their own approved behavior until an age scoring rule is separately agreed. No automatic Profile preference editing or changing confirmed facts from a rejection comment. Store result provenance: profile/revision, job/input hash, rule/weight version and computation time, with current/stale/not-scored/error status.

## Design/UI coordination and current baseline

Claude Design owns the Overview/layout/Profile/filter work. Agree the Quick Match response before it renders numerical contributions: score, status, coverage, matched/conflicting/unknown criteria and source evidence. No fake '+20 Go' or made-up salary/contract values. Input/data parsing and score effects remain backend logic; Overview renders API data/callbacks.

First integrate accepted SETTINGS/Profile/APP-8/COLLECTION-1 and current DESIGN-3 checkpoint into a coherent base. DESIGN-3 is still based on fe31cc3 and has uncommitted Profile/filter/navigation/resize changes overlapping main.tsx/ApplicationConfirmation; do not overwrite source wholesale and lose five Apply outcomes/date basis/schema18. Preserve dirty main and the other live writer. This card has no worktree/executor until a complete plan is approved.

## LLM boundary requiring a decision

Moving Fit Assessment into Application is presentation only. Current background/assessment.go periodically calls EnqueueAll, and store.EnqueueAll scans all jobs rather than only selected roles. With the queue enabled this does not enforce selected-only inference. Do not claim the no-LLM Inbox boundary based on a hidden tab. Propose explicit LLM action in Application after Select; owner must decide automatic-on-Select versus explicit action. Existing pending queue must be preserved and execution eligibility rechecked rather than bulk-deleted/enabled during this work. Quick Match itself never calls a model.

## Acceptance criteria

- Actual current catalogue is ranked using the agreed deterministic rules and saved confirmed profile. Explanation cites the exact criterion/profile value/vacancy evidence; missing facts are not false mismatches and zero is not 'not scored'.
- Same inputs/rule version produce the same score; adding unrelated jobs or repeating a keyword cannot change it. Rule changes and profile/job changes invalidate results honestly.
- Real running-app demonstration on representative copied vacancies plus a few critical hidden invariants (determinism, unknown handling, no model calls, version/preservation). Broad suites/test infrastructure are not implied.
- Complete step/results/time/known costs and actual input availability must be shown and approved before implementation. Current notes are not executable checkpoints.

## Must not touch

No User/multiple-profile/account/logout implementation (SETTINGS-5 deferred), application auto-submit, new provider/model, general-purpose NLP/extraction framework, speculative dictionary import, broad source scrape, automatic age penalty or unapproved hiding. Do not interfere with Claude Design's active worktree/preview or release an entire cumulative branch as a scorer-only change.
