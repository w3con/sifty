# APP-8 — acceptance

> **Integration status,2026-11-10:** source published to main in commit `a3ead8b` (final application source `58af432`). Card is review, not done; production is not updated. The report below remains the technical acceptance record at its original date.

2026-11-09T21:00Z. Coordinator ses_e4a7d8341ffe1Xqrv4Rmx52jqw. Owner accepted the demonstrated five-outcome/button-only result with «принимай, и давай уже дальше»; technical acceptance completed through7a2042ea85dd4fca2ecd9e42dd5d0bffcce64149 after the correction below. No main integration/production release performed.

## Delivered and demonstrated

After Apply: Applied / Not yet / Vacancy stale / Vacancy closed / Not suitable. Negative outcomes store distinct SQLite reasons in jobs and workflow_history and show them in Archive after reload; they do not record submission. Modal dismissal requires a button; backdrop and Escape do not dismiss. Saving/error state stays in the modal. Store validates and atomically persists stage/reason/history; API/container owns effects, modal renders callbacks/data. No adjustable settings or owner/profile data embedded in presentation.

Executor backend/frontend builds and the narrow negative-outcome/stale-applied guard check passed. Real Chromium evidence shows five outcomes, stored Archive reasons/reload, failure/retry, saving controls and mobile layout. Defaultwork-test ran once: SKIPPED, not a full-suite pass. Coordinator independently inspected source/screenshots and opened the actual8083 app in Chromium: all five buttons, backdrop/Escape retention and Not yet button closing confirmed with all non-GET requests blocked. The intercepted App-mount search request was not executed by that review. OpenCode message metadata verified66assistant messages at the initial check, all opencode-go/deepseek-v4.1-flash.

## Defect found at acceptance and correction

Phase17 columns were present but targetVersion was still16. Existingv16 databases would skip migration and fail reads, although the v14-derived preview/fresh tests passed. Returned to the same executor; targetVersion now17. TestMigrationV16ToV17PreservesData starts from a genuinely v16-shaped database lacking both new columns, reopens it, verifies version/columns/job/history preservation, ordinary reads and a new reasoned write. This and TestNegativeOutcomeReasonAndStaleGuard passed; affected backend build passed. Correction20:57:04–20:57:52Z, no full/front-end suite repeated.

## Owner feedback still pending

Owner: «она даже не спрашивает почему "not suitable"». The coordinator's original plan defined category-only reasons, not an explanatory text field. Record this as a follow-up: ask for/store an explanation and display it in history/archive, without silently running a model or changing Profile/scoring. It is not implemented in the accepted five-button result. Do not prolong this acceptance by building it unapproved.

## Time, friction and recovery

Execution35min plus preparation~4min exceeded the10–20min estimate/19:22stop; executor continued despite the brief's explicit deadline. Overrun was reported and work preserved, then owner authorized acceptance. Coordinator review/correction20:55–21:00Z~5min. Existing preview/data/browser setup consumed disproportionate effort and was poorly budgeted; reuse the existing preview for the next feature instead of repeating setup or expanding browser matrices. Migration gating was a real missed hidden failure and required its targeted correction. No repair of the harness/timer is included.

Source .worktrees/APP-8 onwork/APP-8, review preview http://127.0.0.1:8083 uses isolated copied data. Original production/staging data and active DESIGN-3 Profile writer preserved. Screenshots .local/app8-review/{modal-desktop,archive-dismissed,modal-mobile}.png; verbatim checks in log.md. Integrating only this card must preserve unrelated dirty main and the unintegrated accepted Profile baseline; never release the entire cumulative worktree as a modal-only change.

Next product task: COLLECTION-1 publication dates/age/Newest, with a complete bounded estimate/plan before execution. Prior-application recognition remains TRIAGE-1 and follows actual8080 history, not an environment-reconciliation detour.
