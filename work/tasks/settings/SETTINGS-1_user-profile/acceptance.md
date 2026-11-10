# SETTINGS-1 — technical acceptance, 2026-11-09

> **Integration status,2026-11-10:** source published to main in commit `564f334` (final application source `58af432`). Card is review, not done; production is not updated. The report below remains the technical acceptance record at its original date.

## Result and access

Technical implementation accepted through `3c959df`; all approved implementation steps completed. Open **http://localhost:8081 → Profile**. Vite serves `.worktrees/SETTINGS-1/frontend`; the API preview on8082 follows the same task source with the documents worker/Mammoth. Existing `data-staging` is preserved. Production8080 is healthy and unchanged; main source has not been integrated. Formal successful closure remains blocked, not done.

## Verified usefulness

- Isolated real-browser Save/Cancel/reload and concurrent edits preserve input, show latest values and require an explicit section rebase/discard. Partial rows no longer disappear silently.
- Saving target roles/skill changed a full-description fixture's existing BM25 score from0 to100 while unrelated roles stayed0. Context and score attribution recorded the saved profile revision; no new ranking engine or automatic LLM assessment.
- Exactlyone synthetic extraction request through the existing configured Luna adapter returned4 attributable proposals. Real-browser accept/reject and both conflict choices worked; accepted skill retained document provenance. No second paid call. Exact provider token usage/cost was unavailable.
- Profile/context/job metadata makes old detailed assessments stale. Critical tests cover atomic profile/history/fact saves, late/cancelled publication, profile-version attribution and failed rescore currentness. Broad suites were not run; default work-test reported SKIPPED.
- Real staging Chromium rendered all sections/editors and original legacy documents with no page/console errors; Cancel left owner revision1 unchanged. Final screenshot review caught a false unavailable message and hidden Analyze on imported originals; the same UI executor corrected it and verified Analyze+Original on existing PDFs, absent actions for genuinely missing files. No owner document was analyzed and no synthetic data was saved into owner staging.

## Decisions and limitations

SQLite confirmed profile is authoritative; candidate contexts are derived. Data/configuration, service/hook effects and presentation were reviewed separately. Existing path/config/provider adapters are reused. Unknown levels/currency/eligibility remain unknown; free-text dealbreakers belong to explicit detailed assessment, not the fast scorer. Ordinary Save never starts extraction or bulk model assessment.

Ollama requests timed out during the earlier check; this does not establish that Ollama is down. Owner-authorized Luna verified the extraction route without changing the normal staging provider. Staging starts with unconfirmed/empty structured fields and an honestly stale derived context until the owner saves/refreshes; legacy documents remain available as inputs.

## Problems and repairs

- Preparation lost about80min to a stalled shell invocation; exact cause unproven. Repeated local model attempts and overbroad package checks exceeded their verification briefs and were recorded; subsequent stages used bounded checks and one Luna call.
- Review returned provenance/CAS/source-version/upload-limit, partial-row and stale-section defects to their original executors; corrected and checked.
- A fresh-database queue response with rows:null initially blanked the app; normalized at the existing frontend response boundary and verified with zero queue rows.
- Preview defaulted workers to main and omitted the new document worker. Worktree worker-source configuration fixed activation. A workers-only generation restart failed its own version check; a fresh preserved state fixed staging. These preview defects remain recorded limitations rather than an unrelated harness repair.

## Closure blocker and exact next action

Normal work-finish requires a clean target checkout; main has20 preserved owner/Designer paths. The task tree includes its initial main snapshot and unrelated task/instruction records. Releasing that entire tree would fold unrelated work into the Profile squash. The configured finish adapter additionally deploys production, excluded from the approved implementation plan. No force, release bypass, stash, unrelated commit, merge or production update was used to manufacture closure.

Next: settle the existing main baseline and prepare a filtered Profile-only integration source, preserve this branch/worktree, then run normal checked squash/release against the exact approved source. Provisional integration/closure estimate20–40min AI after baseline/release scope is settled; no new product feature is needed. Do not rerun Luna or broad suites. Owner staging/release decision remains outstanding; do not label review as done.

## Elapsed and calibration

Approximately4h10–20min from20:35CET to final acceptance, within approved3–5h, including the observed80min tool stall. Formal integration/release is not counted as delivered. Closure cleanliness and branch composition should have been resolved at preparation rather than discovered after staging; next task preparation must inspect that dependency before promising unattended done. An initial coordinator acceptance-record patch failed its context match and applied no changes; corrected without touching application code.
