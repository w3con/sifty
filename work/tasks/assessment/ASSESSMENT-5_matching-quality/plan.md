# ASSESSMENT-5 — Matching quality: gold set and method choice (coordinator plan, 2026-11-11)

## Outcome

A measured choice of how the second score's verdicts are made, on a labelled set of the owner's own requirement–profile pairs, so that "You fit" can be shown again (it is hidden after ASSESSMENT-3, spec §14.14). Research card: it ends with a comparison and a decision, not with an implementation; implementation is planned and approved after M.4.

## Steps

- [ ] **M.1 — Atomic profile facts** (45–60 min). Code splits the profile into short single statements (skills with level; each sentence of each experience summary with company, title and dates; computed facts) with stable ids. Used by M.2 and by every candidate. Check: the list for revision 25 reviewed by the coordinator.
- [ ] **M.2 — Gold set** (1.5–2 h AI + about 30 min owner). 150–200 requirement–fact pairs from the 81 vacancies analysed in ASSESSMENT-3 (mixed: relevant and unrelated roles, previous false matches and misses included). Labels: proves / partly / does not prove. First labeller Haiku 5.5 (needs the owner's yes: profile facts go to Anthropic via the subscription), second the executor; the owner resolves every disagreement and labels 30–40 random pairs himself blind; agreement reported. Stored in the card as JSON.
- [ ] **M.3 — Candidates on the gold set** (1.5–2.5 h). Same pairs for each: current 9B prompt; Qwen3.6 35B-A3B; embeddings retrieval with a threshold (local model, "nothing close" → code verdict); a reranker / NLI pair classifier with a confidence threshold; Laya. Needs the owner's yes for downloading the small models (about 1–2 GB). Report precision and recall of "match", false matches on unrelated roles, time per vacancy, memory.
- [ ] **M.4 — Decision with the owner** (20 min). Pick the method; the coordinator then writes the implementation plan (including the old ASSESSMENT-3 Q.8 idea: verdicts keyed by what the judge saw, not by profile revision, so one added skill does not re-judge every vacancy) for separate approval.

## Estimate

Complexity 8. AI time about 4–5.5 h plus about 30 min of the owner. Haiku labelling: a few US cents. No code merged to main in this card except the atomic-facts function if the owner wants it kept.

## Must not touch

Production, staging, merge, the detailed assessment queue, the profile data (read only), other worktrees.

## Re-planned after the owner's pipeline decision (2026-11-11, spec §15)

The owner agreed («2 и 3 - согласен») to rebuild this card around spec §15.2 steps 3–4. The research steps above stay; the implementation that follows M.4 is now known in outline:

- **M.0 — Profile as a fact store** (§15.1): the matcher reads accepted profile facts only (skills, the ✕ list, experience split into atomic facts by M.1, computed facts); the third outcome is shown as "not in your profile". Wording and display, no new utility.
- **M.1–M.2** unchanged (atomic facts, gold set).
- **M.3** candidates: Laya comes measured from ASSESSMENT-7 (run in parallel by a separate agent); add embeddings retrieval and the pair question on the gold set.
- **M.5 — Implementation (after M.4, separate approval):** sections routing for You fit (Requirements / Nice to have only, from INBOX-2 sections), code first (skills searched in those sections with the vocabulary, years, ✓ / ✕), retrieval by meaning, "not in your profile" without a model when no fact is close, the chosen pair judge with its calibrated threshold, You fit shown again; verdicts keyed by what the judge saw (old ASSESSMENT-3 Q.8).

Depends on: INBOX-2 (sections), ASSESSMENT-7 (Laya measurement). Estimate of M.5 to be given at M.4.
