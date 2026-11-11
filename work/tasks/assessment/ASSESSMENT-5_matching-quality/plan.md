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
