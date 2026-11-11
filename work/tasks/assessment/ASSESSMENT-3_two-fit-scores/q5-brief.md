# Q.5 brief — panel and catalogue (from coordinator, 2026-11-11)

Q.3b and Q.4 are accepted (see log.md).

0. Refresh the copy: copy `data-staging` again into `.local/assessment-3/data` (profile revision 21 with 11 experience entries). Review server on 8094 only; do not touch 8080/8081.
1. `QuickMatchPanel.tsx` and `frontend/quickmatch.ts` per "Panel content" in plan.md: two scores (Fits you; You fit with match/partial/no/unclear counts); lists Matches (partial marked), Mismatches, Missing from your profile, To clarify. States: "No requirements stated", "Analysing requirements…", source unavailable → no numbers, model not running → a plain message. No queue, limits or provenance in the UI.
2. Catalogue row: both numbers compactly; `sort=fit` as in the spec.
3. Design-system tokens only. `npm run build` clean.
4. Screenshots of the panel for Parity, Ethena Labs "Engineering Manager / Tech Lead" and one vacancy without requirements, before and after re-matching with the new profile, saved in this card folder.
5. In attention.md: a proposal (do not implement) for keeping production and staging from using Ollama at the same time — pause on staging or a shared lock.

Stop after Q.5 with a report in the same format as the Q.4 report.
