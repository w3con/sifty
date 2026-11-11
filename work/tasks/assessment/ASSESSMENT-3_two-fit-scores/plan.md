---
estimate: 13
---
# ASSESSMENT-3 — plan

**Author:** coordinator (Claude, Cowork). **Prepared:** 2026-11-11. **Executor:** Sonnet 5.5 (owner's choice for this card). **Branch/worktree:** `work/ASSESSMENT-3` in `.worktrees/ASSESSMENT-3`, created from current `main`.

## Outcome

In the running app, every vacancy with usable text shows two numbers — **Fits you** and **You fit** — and short lists: Matches, Mismatches, Missing from your profile, To clarify. Requirements come from the vacancy text via the local model, are matched to the owner's profile by the local model, and both scores update automatically. No queue mechanics are shown to the owner.

## Architecture (data / logic / presentation — mandatory separation)

```
collect / refresh one vacancy
  → deterministic facts (existing jobextract)                 [pure]
  → VACANCY EXTRACTION (model, vacancy text only)              [effect: provider]
      requirements[] + condition facts, each with exact quote; validated; persisted per (source hash, extractor version)
  → REQUIREMENT MATCHING (model, requirements + profile)       [effect: provider]
      per requirement: match | partial | no | unclear, profile evidence, short reason; persisted per (requirements hash, profile revision, matcher version)
  → EVALUATE (pure, no I/O)
      Fits you (goal groups, unknown = 0.5) + You fit (requirement verdicts) + lists
  → persisted result → API projection → UI renders, never computes fit
```

- **Data / configuration:** profile snapshot (existing), retained vacancy source + hash, deterministic facts, extraction records, match records, rules (weights) in existing match config. Provider for both model passes = `providers.stages.extraction` (existing config; no hardcoded model, endpoint, port or weight).
- **Logic:** `quickmatch` stays pure (inputs → result). Model calls live in a new adapter package (e.g. `backend/internal/vacancyai/`) reusing `assess.InferInto` and the existing inference lock. Worker in `background/`. Store in `store/` with a migration.
- **Presentation:** `frontend/quickmatch.ts` (projection types/formatting only), `QuickMatchPanel.tsx`, catalogue row. Follows the Sifty design system (`theme.css` tokens; message kinds broken / decision / retry / info).

## Scores (owner-approved policy)

**Fits you (0–100).** Groups and existing weights: role 25, compensation 20, location/work mode 15, conditions 10. Each group's utility comes from the existing comparators; a group with no comparable fact contributes **0.5**, not nothing. Score = Σ(weight × utility) / Σ(weight) over all applicable groups. A title-only match gives ≈ 68, never 100. Legal eligibility and skills are not goals.

**You fit (0–100).** Over extracted requirements: required weight 2, preferred weight 1. Verdict utility: match 1, partial 0.5, unclear 0.5, no 0. Score = weighted mean. Shown with counts "N match · N partial · N no · N unclear". If a vacancy has usable text but extraction found **no** requirements, show "No requirements stated" (not 0, not 100). If extraction is still pending, show "Analysing requirements…" and no number.

**Source unavailable** (no description, tags/title only): no numbers; "Full job description is unavailable" + existing Refresh action (keep this ASSESSMENT-2 behaviour).

**Catalogue order:** Fits you desc, then You fit desc, then job id. Pending You fit sorts as if 50 within equal Fits you. (Owner may change later; record it in the spec.)

## Panel content (replaces the current Quick match card)

- Header row: **Fits you 72** · **You fit 64** (with counts). Bucket labels Strong/Partial/Weak per existing thresholds, colours from the design system.
- **Matches** — one line each: requirement or goal → what in the profile matches ("Go — Go (expert)").
- **Mismatches** — known conflicts ("Office in London — you want remote", "Pay €70k — your minimum €100k").
- **Missing from your profile** — requirement the profile cannot confirm, with a link to the profile section ("Rust — not in your skills · Add").
- **To clarify** — unknown goal conditions as "? Salary", "? Contract type", "? Work mode". Only conditions the owner cares about; never parser internals ("description").
- Hide empty sections. No "Current", attempts, daily caps, Retry of the queue, provenance, reason codes, coverage percentages, "this is not a zero" disclaimers.
- One line only when the model is unavailable: retry-kind message "Requirements could not be analysed — the local model is not running" (design-system message, not a queue panel).

## Model passes

**Vacancy extraction prompt (no profile):** input = vacancy text (treated as data). Output JSON: `requirements[]` {text (short normalized), kind: skill | experience | qualification | language | authorization | other, level: required | preferred, quote (exact substring)}; `conditions` {compensation, work_mode, engagement, location, timezone, hours} each {value, quote} or null. Validation: every quote must be an exact substring (whitespace-normalised) of the source; drop invalid items; dedupe; cap list length (config). Glued paragraphs (Parity) must work: no reliance on headings.

**Requirement matching prompt (with profile):** input = requirement list + profile snapshot (skills with levels, experience, roles, languages, authorizations, education). Output per requirement {verdict, profile_evidence (field + value), reason ≤ 12 words}. Validation: profile_evidence must reference an existing profile field/value; otherwise verdict becomes `unclear`. A missing profile fact is `no` only for authorization/language when the profile explicitly answers; otherwise `unclear` + listed under Missing from your profile.

Both: thinking off where supported, temperature low, timeout from provider config, at most 3 attempts on transient errors, typed schema via existing `assess.InferInto`.

## Re-run rules

- New/changed vacancy source → extraction + matching.
- Profile change (new revision) → matching only (extraction reused). Cheap, local.
- Weights/rules change → pure re-evaluation only, no model call.
- Extraction that finds nothing is final for that source hash; never "no gap → done" based on the deterministic gap detector (the ASSESSMENT-2 defect).
- Worker order: newly collected first, then backlog by Fits you desc. No daily cap by default (config value, default 0 = unlimited). One worker, existing inference lock, so it never overlaps the detailed assessment queue's provider use. Pause/resume via config; worker idles cleanly when the provider is down.

## Steps

- [ ] **Q.1 — Spec, worktree, provider check** (20–30 min). Create `work/ASSESSMENT-3` worktree from `main`. Update `docs/specs/quick-match.md`: new section superseding §13.7 with the policy above; amend §2 "No LLM in the score path" → the evaluator stays pure, model passes produce persisted facts/verdicts before evaluation. Record (do not implement yet) the paid-provider guard; it is implemented in Q.3 with the adapters. Check: spec diff reviewed by coordinator.
- [ ] **Q.2 — Contract + pure evaluator** (35–55 min). Extend `matchcontract` with extraction records, match verdicts and the two-score result; change goal-fit aggregation to unknown = 0.5; add You fit computation and the four lists. Pure unit tests: title-only → ≈68 not 100; all unknown → 50; verdicts → expected You fit; "no requirements stated"; source unavailable → no numbers. Stop for review.
- [ ] **Q.3 — Model adapters + persistence** (45–70 min). Vacancy extraction and requirement matching adapters on `assess.InferInto`, prompts in `backend/internal/prompts/`, quote/profile validation, migration with two tables keyed as above, store methods. Tests with injected model responses: invalid quote dropped; unknown profile evidence → unclear; re-run keys behave as in "Re-run rules". Implement the paid-provider guard: bulk extraction refuses a provider not marked local/free unless config explicitly allows it (default config routes `extraction` to luna). Port quote validation from ASSESSMENT-2 `backend/internal/vacancyextract/validate.go` where it fits. Stop for review.
- [ ] **Q.4 — Worker + wiring** (35–55 min). Background worker, ordering, retries, pause, rescore trigger after each pass, API projection extension (no new admin endpoints beyond a status count used by Settings › Assessment queue if needed). Collection triggers extraction for new vacancies. Profile save triggers re-matching. Check: injected end-to-end on a temp DB; existing assessment queue rows untouched. Stop for review.
- [ ] **Q.5 — Panel and catalogue** (40–60 min). Rework `QuickMatchPanel.tsx` per "Panel content"; catalogue row shows both numbers compactly; sorting per policy. Design-system tokens only. `npm run build` passes. Stop for review.
- [ ] **Q.6 — Real verification on staging copy** (40–70 min, owner present). Fresh copy of `data-staging` into `.local/assessment-3/`, isolated review port (e.g. 8094), local Ollama `qwen3.5:9b`. Run the worker on ≥ 30 vacancies including Parity (84), Ethena Labs "Engineering Manager / Tech Lead", one with no description, one in a glued paragraph. Record per-vacancy time (median, p90) → projected backfill time. Screenshots of panels before/after. Data-preservation check: profile, jobs, assessments, assessment queue, Apply history hash-identical except new tables and scores. Owner reviews in the browser.

## Acceptance (owner-visible)

- Parity 84 and Ethena show both numbers, ≥ 5 concrete requirement rows each, Matches/Mismatches/Missing/To clarify filled from real text; no title-only 100.
- Across the verified sample, You fit is a number (or "No requirements stated" / unavailable) — never `not verified` for a vacancy with usable text after processing.
- No queue/attempt/cap/provenance text on the panel.
- Changing a skill in the profile updates You fit without re-extracting.
- The detailed assessment queue and its switch are unchanged.

## Reuse from ASSESSMENT-2 (read-only reference in `.worktrees/ASSESSMENT-2`)

Port selectively, by copying code into the new branch: quote validation (`vacancyextract/validate.go`), prompt structure (`prompts/vacancy-facts.md`), source-unavailable state and the single-vacancy refresh wiring, `useExtractionStatus.ts` ideas only if needed. Do **not** port: the extraction queue with daily caps and per-identity request accounting, the gap-kind admission gate (`len(kinds)==0 → done`), the requirements interval, the panel.

## Executor rules

One step per session; stop at the step boundary and report: files changed, checks run with output, open questions. Separate data / logic / presentation as above. No hardcoded models, ports, weights or paths. No production, merge, push, process-file edits. Do not enable or touch the detailed assessment queue. Commit checkpoints on `work/ASSESSMENT-3` only.
