# Q.6 brief — real verification on a staging copy, with the owner (from coordinator, 2026-11-11)

Start from a clean context (`/clear`), read AGENTS.md, then this card (description, plan, attention, log, this brief). Q.5b is accepted.

Order matters: the owner wants to look at the app on his computer early, so step 0 and step 1 come first.

0. **Model work follows the Inbox sort (owner decision, see log.md 2026-11-11).** One list for all model work (extract and re-match), no "new first" group; remove `new_window_hours` (config, runtime.yaml comment, spec §14).
   - Best fit → Fits you descending, then job id; a vacancy without Fits you yet counts as the midpoint.
   - Newest → exactly the Inbox Newest key (source publication date, unknown dates last, then job id; `frontend/filters.ts` sortCatalogue). Put the key computation where both can use it or mirror it on the server with one shared definition in the spec — do not invent a second meaning of "newest".
   - The toggle is browser-only today (localStorage `sift.sort`). Persist it on the server through the existing settings/preferences storage (no new settings framework); the frontend saves it on change and reads it on load; the worker reads it at every plan step, so a switch takes effect at the next vacancy. Default Best fit.
   - Update the existing ordering test for both orders. Spec §14 in the same commit.

1. **Review server for the owner, first.** Build the branch (frontend and backend) and start it on port 8094 against `.local/assessment-3/data` with POST /api/search blocked. Make the block permanent, not a session-only proxy: preferred is a configuration switch read by the server (e.g. collection disabled in the copy's runtime.yaml) if one exists or is a small change; otherwise keep the proxy script in this card folder with its start command. Set the copy's `daily_limit` to 0 (no limit) for the run. Tell the owner the URL (http://localhost:8094) and leave the server running; record the exact start and stop commands in q6-run.md.
2. **Shared model lock.** Every Sifty process on the machine (production, staging, review servers) uses the same Ollama. Implement one lock shared across processes for model calls (fit worker and the detailed assessment queue), located through the existing path resolution / configuration, not a hard-coded path. One targeted test: two stores with different data dirs cannot hold it at once. Spec §14 updated in the same commit.
3. **Run on ≥ 30 vacancies** including Parity (84), Ethena Labs (22), one without description, one glued paragraph. Record per-vacancy time (median, p90) and the projected backfill for all vacancies with text.
4. **Quality numbers:** share of requirements per verdict (match / partial / no / unclear) across the run, and a hand check of 10 "unclear" lines: how many the profile actually answers (e.g. Ethena "Led small team of 3-8", "Shipped production code in last year"). This decides whether a larger local model is needed.
5. **Data preservation:** hashes of profile, jobs, assessments, assessment queue, Apply history identical except new tables and scores.
6. Spec: reconcile §14 with what is implemented and what remains limited.

Stop with a report in the usual format. Do not merge; acceptance and merge are the coordinator's act after the owner's review.
