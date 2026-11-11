# ASSESSMENT-7 — Pair judge trial: Laya and small local judges

**Epic:** assessment
**Created:** 2026-11-11
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Laya не тестировали. ... - я могу поднять отдельного агента и дать ему задание - если есть карточка.

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Purpose

Measure whether Laya (already downloaded on the owner's Mac) can be the "pair judge" of spec `docs/specs/quick-match.md` §15.2 step 4: given one requirement and one profile fact, answer proves / partly / does not prove, with a confidence. Research only: a measurement and a report, nothing merged into the product. Runs in parallel with INBOX-2 under its own agent; it touches no shared code paths.

## Inputs that already exist (main, released with ASSESSMENT-3)

- `backend/internal/vacancyai/live_export_test.go` (`TestLiveExportMatchInputs`): writes, for vacancies of a data COPY, what the matcher shows the model — requirements code did not decide, the profile facts chosen for each, the computed facts. Env: `SIFTY_PROBE_DATA_DIR` (a copy), `SIFTY_PROBE_JOB_IDS`, `SIFTY_EXPORT_OUT`.
- Hand-judged cases from ASSESSMENT-3 Q.7/Q.7b (on branch `work/ASSESSMENT-3`, card folder: `q7-analysis.md`, `q7-analysis-run2.md`, `q7b-matching-comparison.md`, `q7b-conversation.md`): the known false matches (Growth Lead: "A/B testing" ← testing standards, "customer lifecycle marketing" ← product lifecycle records, "user acquisition" ← game production; Product Designer: visual design), misses ("2+ years" with 17 years; "Programming language" with Go and Python; "agent-native development" vs the own AI delivery system) and the 9B / Qwen verdicts on 81 vacancies.
- `scripts/model-bench/` (README) for running comparisons on a copy.

## Plan

- [ ] **P.1 — Pairs** (30–45 min). Make a fresh copy of `data-staging` into `.local/assessment-7/` (never the live `data/` or `data-staging/`). Export the inputs for the 81 vacancies of ASSESSMENT-3. Turn them into (requirement, single fact) pairs. Mark the pairs that correspond to the hand-judged cases above with the expected answer; that is the small labelled set for this trial (expect 40–60 pairs). Keep everything in the card folder.
- [ ] **P.2 — Laya** (45–75 min). Run Laya on all pairs with a fixed prompt or its native typed-decision format; record answer, confidence, time per pair, memory. Two runs for stability.
- [ ] **P.3 — Baselines on the same pairs** (30–45 min). The current 9B matcher prompt reduced to one pair; and, only if the owner allows the download in chat, one small local embedding model (cosine between requirement and fact, threshold sweep). No hosted models.
- [ ] **P.4 — Report** (20–30 min). On the labelled pairs: correct / false proof / missed proof per judge; how many of the known false matches each judge rejects; how many known misses it finds; stability across runs; time per vacancy projected (pairs per vacancy × time per pair). A recommendation, clearly marked as the executor's, and the open questions. Stop.

## Must not touch

Production (8080), staging (8081), `data/`, `data-staging/`, `main` (no merge, no push), other worktrees (INBOX-2 is active), the detailed assessment queue, the profile. No hosted model calls. No downloads without the owner's yes in chat.

## Estimate

Complexity 4, AI time about 2.5–3.5 h. Cost: none (local only). The full gold set and the decision stay in ASSESSMENT-5; this card gives ASSESSMENT-5 a first measured candidate.

## Owner approval

Owner, 2026-11-11: «я могу поднять отдельного агента и дать ему задание - если есть карточка.» — card created for that; start with `bin/work-handover ASSESSMENT-7` on the Mac.
