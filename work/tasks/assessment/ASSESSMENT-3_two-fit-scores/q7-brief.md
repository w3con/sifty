# Q.7 brief — matcher: complete CV, code skill matching, focused evidence (from coordinator, 2026-11-11)

Start from a clean context (`/clear`), read AGENTS.md, then this card (description, plan, attention, log, this brief) and `docs/specs/quick-match.md` §14.12 — the agreed behaviour you implement. Q.6 and step 0 are accepted.

Keep the separation: synonym table and any thresholds in configuration (`config/` with the existing loader), selection and code matching in Go next to the matcher (`vacancyai` / `profile` as fits), prompts in `backend/internal/prompts/`, nothing hard-coded about the owner.

1. **Complete-CV policy** (§14.12.1): rewrite `requirement-matching.md` and the `no`/`unclear` validation in code. Bump the matcher version.
2. **Code skill matching** (§14.12.2): synonym table in configuration; whole-word matching with care for short terms (Go, C, R, TS); alternatives; a hit is `match` with evidence and skips the model. One targeted test for false positives ("go to market", "R&D").
3. **Computed facts** (§14.12.3): current role / worked in the last year; anything else that follows from dates and titles only.
4. **Focused evidence** (§14.12.4): per requirement the 2–3 closest profile passages plus computed facts; citation outside them → `unclear`.
5. **Re-measure** on the review copy (server 8094, collection disabled): re-match the same 43 vacancies as Q.6 (matching only, no extraction). Report verdict shares before/after, time per vacancy, You fit of the unrelated roles and of Parity / Ethena Labs, a hand check of 20 `unclear` and 10 `no` lines, and how many requirements code matched without the model.
6. Spec: move §14.12 items from "agreed" to "implemented on the branch" and update §14.2/§14.5 in the same commit; record what stays limited.

Do not pull or switch models; §14.12.5 is decided after this measurement. Stop with a report in the usual format.

Coordinator note after reading a30a66c (2026-11-11): the prompt says "a fact that is not offered for it does not exist" and absence is now `no`. If the passage selection misses indirect evidence (e.g. "agent-native development" vs the Validité "own AI delivery system"), the verdict becomes a false `no`, which is worse than the old `unclear`. In the hand check of `no` lines, judge each against the WHOLE profile, not only the offered passages, and report separately: false `no` caused by passage selection vs by the model. Include Ethena Labs "agent-native development" and "Shipped production code in last year" explicitly.
