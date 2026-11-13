# Triage and workflow behavior specification

Status: Written from the code of main `d49de9f` on 2026-11-13 by card OPS-11; not yet reviewed by the owner.

This specification describes what the owner can do with a collected vacancy after it has been stored: look at it in the Inbox, select it, dismiss it, block its company, apply to it, record the outcome of the application, and later find or delete it. Collection (how vacancies arrive), search and filters, the Fits you and You fit scores, the vacancy view layout and the preparation of application answers have their own specifications and are only linked from here. Names of stages in this document are the names shown in the interface (Inbox, Shortlist, Archive with its Applied and Dismissed tabs); older cards and `docs/MAP.md` call the same stages Triage, Pre-selected or Preparation, and the stored values are different again (section 2). Code references are `path:line` relative to the repository root and were read in the main checkout on 2026-11-13; the code of `d49de9f` and of the later documentation-only commit that was checked out are identical. Nothing was run: no server, no build, no browser, no model call; statements about the behaviour of a running screen come from reading the code and are marked "not verified" where that matters.

## 1. Purpose

The owner looks at a large, constantly refilled list of vacancies and must decide quickly which ones deserve an application. The workflow gives the owner four decisions, each reversible except deletion: keep looking later (Inbox), pursue it (Shortlist), reject it (Dismissed, optionally with a reason), or record that an application was really submitted (Applied). It must never record a submission that the owner did not confirm, it must keep a rejected or applied vacancy from coming back into the Inbox when collection runs again, and it must let the owner remove whole groups of unwanted vacancies (a company, a title word) without handling them one by one. The owner's own description of the minimal manual loop (2026-11-09): «надо сделать так, чтобы для начала в ручном режиме можно было получать вакансии, сортировать их, перемещать в triage или браковать. из папки preparation при клике на Apply - должно быть потом не только "ответил/не ответил", но и "вакансия старая/не подошло".» (card INBOX-1, `work/tasks/inbox/INBOX-1_search-catalogue-overview-quick-match/comments.md`). The owner also decided that there is no separate incoming stage: «нет, все сразу попадает в triage (после работы фильтров этом триаже).» (same card, `description.md`).

## 2. Stages and transitions

A vacancy is always in exactly one stage. The interface shows three tabs in the header, each with a count, and a fourth state inside the Archive tab. `frontend/main.tsx:140`

| Interface name | Stored stage | Stored status column | What it means |
|---|---|---|---|
| Inbox | `triage` | `new` | Every newly stored vacancy starts here; it is the default tab on opening the application. `backend/internal/store/migrate.go:32`, `frontend/main.tsx:44` |
| Shortlist | `preselected` | `saved` | The owner chose to pursue the vacancy; the Application tab opens only from here on. `frontend/RoleContent.tsx:56` |
| Archive, Applied tab | `applied` | `applied` | The owner confirmed that the application was submitted; the confirmation date is stored. `frontend/main.tsx:152` |
| Archive, Dismissed tab | `no` | `rejected` | The owner rejected the vacancy, either in the Inbox or with a reason after Apply. `frontend/main.tsx:152` |

The mapping of stage names, old status words (`new`, `saved`, `rejected`) and the alias `shortlist` to the four stages is in `backend/internal/store/core.go:14` and `backend/internal/store/core.go:35`; the same mapping exists in `frontend/model/workflow.ts:11`. An unknown stage or status value is refused. `backend/internal/store/core.go:26`

### Transitions offered in the interface

| From | To | How the owner does it | Stored effect |
|---|---|---|---|
| Inbox | Shortlist | Button "✓ Select" or key Y | status `saved`; enqueue of a model assessment row and, the first time, an inspect-only preparation operation (section 5). `frontend/main.tsx:166`, `backend/internal/store/jobs.go:787` |
| Inbox | Dismissed | Button "✕ Dismiss" or key N | status `rejected`, no reason. `frontend/main.tsx:166`, `backend/internal/store/jobs.go:787` |
| Shortlist | Inbox | Button "Back to Inbox" | status `new`; applied date and reason are cleared. `frontend/main.tsx:166`, `backend/internal/store/jobs.go:779` |
| Shortlist | Applied | "Yes, applied" in the application confirmation | status `applied`, applied date set to the current time, history row marked confirmed. `frontend/ApplicationConfirmation.tsx:94`, `backend/internal/store/jobs.go:780` |
| Shortlist | Dismissed with reason | "No, vacancy is stale", "No, vacancy is closed" or "No, not suitable" in the application confirmation | status `rejected` plus a stored reason `stale`, `closed` or `not_suitable`; no applied date. `frontend/ApplicationConfirmation.tsx:73`, `backend/internal/store/jobs.go:783` |
| Applied | Shortlist | Button "Undo" on an Applied role in the Archive | status `saved`; the applied date is cleared. `frontend/main.tsx:166`, `backend/internal/store/jobs.go:779` |
| Dismissed | Inbox | Button "Restore to Inbox" in Archive, Dismissed tab | status `new`; the stored reason is cleared. `frontend/main.tsx:116`, `frontend/main.tsx:166`, `backend/internal/store/jobs.go:783` |
| Inbox (this session) | Inbox | Button "Undo" on a role just selected or dismissed, while it is still shown in the Inbox list | status `new`. `frontend/main.tsx:166` |

### Transitions the server accepts but the interface does not offer

- A move from Inbox, from Dismissed or from Applied straight to Applied is refused by the server: Applied requires that the vacancy is currently in the Shortlist and that the request carries an explicit confirmation. `backend/internal/store/jobs.go:760`
- A negative reason is accepted only when the target is Dismissed and the vacancy is currently in the Shortlist; a stale request (the vacancy is no longer in the Shortlist) is refused with a message asking the owner to reload, and it never changes an application that was confirmed in the meantime. `backend/internal/store/jobs.go:746`, `backend/internal/store/jobs.go:757`
- Every other combination is accepted by the server, for example Dismissed to Shortlist, Applied to Inbox, or Applied to Dismissed without a reason. No screen offers these. `backend/internal/store/jobs.go:741`

### Rules common to every transition

- A request for the stage the vacancy is already in does nothing: no history row, no change. `backend/internal/store/jobs.go:765`
- The write is conditional on the stage observed when the request was read; if the vacancy changed in the meantime the request fails with a message to reload and nothing is written. `backend/internal/store/jobs.go:787`
- Every actual move writes one history row with the old stage, the new stage, whether the move was confirmed, the reason if any, and the time. `backend/internal/store/jobs.go:804`
- Any move that does not end in Dismissed-with-reason clears the stored reason, and any move that does not end in Applied clears the applied date; the history rows keep the earlier values. `backend/internal/store/jobs.go:779`
- The interface blocks a second action while one is being saved and re-reads the vacancy after each move so that the score envelope is not lost. `frontend/main.tsx:96`, `frontend/main.tsx:100`
- Reaching the server through the API keeps the older words working: `{"status":"saved"}` has the same effect as `{"stage":"preselected"}`. `backend/internal/server/routes.go:159`

## 3. The Inbox

- The Inbox lists the vacancies in stage Inbox that are not hidden by a company block and not hidden by an exclusion term; its header count is the same set. `frontend/main.tsx:83`, `frontend/main.tsx:140`
- The list refreshes by itself every 2.5 seconds together with the assessment queue status, so newly collected vacancies appear without a reload; a refresh that started before the owner's own write is discarded. `frontend/main.tsx:64`, `frontend/main.tsx:69`
- A vacancy the owner just selected or dismissed stays visible in the Inbox list until the page is reloaded (a dismissed one is greyed, a selected one carries a ✓), so that it can be undone; the list says "N dismissed this session — they disappear next time." The "session" is in-memory state of the open page. `frontend/workflow.ts:22`, `frontend/main.tsx:101`, `frontend/main.tsx:156`, `frontend/main.tsx:158`
- After Select or Dismiss the selection moves to the next vacancy that is still undecided, searching forward in the current list order and wrapping around; if there is none, the selection stays. `frontend/workflow.ts:27`, `frontend/main.tsx:96`
- Opening a vacancy marks it as read (once; the first read time is kept); the "Unread" toggle hides read vacancies except the one currently held open. Reading does not depend on the stage. `backend/internal/store/jobs.go:701`, `frontend/main.tsx:91`, `frontend/main.tsx:146`
- The order of the list (Best fit, Newest) and the filters (Fits you buckets, Collected within, Hide expired, Remote, Web3, Leadership title and the catalogue filters) are described in [Quick Match](quick-match.md) §14 and in the search area; the Best fit / Newest choice is also saved on the server because the background analysis follows it. `frontend/main.tsx:49`, `backend/internal/server/routes.go:499`
- A vacancy whose source marked it expired shows "Listing expired" and the ◷ icon but is not moved anywhere automatically; "Hide expired" is only a filter. `frontend/main.tsx:156`, `frontend/main.tsx:165`
- Empty states: "Inbox zero. Collect more jobs in Settings → Sources." (Inbox), "Nothing shortlisted yet — press Y in Inbox." (Shortlist), "No applications yet." (Applied). `frontend/main.tsx:157`
- The header actions of the selected Inbox vacancy are Block company, Dismiss and Select; their placement in the redesigned vacancy view is in [Vacancy view](vacancy-view.md). `frontend/main.tsx:166`

## 4. Keyboard shortcuts

| Key | Action | Condition |
|---|---|---|
| J or Arrow Down | Next vacancy in the current list, wrapping from last to first | Any list view. `frontend/main.tsx:124`, `frontend/main.tsx:131` |
| K or Arrow Up | Previous vacancy, wrapping | Any list view. `frontend/main.tsx:124`, `frontend/main.tsx:131` |
| / | Focus the search box | Any list view. `frontend/main.tsx:132` |
| Y (physical key `KeyY`, so also the Russian layout letter on that key) | Select, that is move to the Shortlist | Inbox tab and the selected vacancy is in the Inbox stage. `frontend/main.tsx:129`, `frontend/main.tsx:133` |
| N | Dismiss | Inbox tab and the selected vacancy is in the Inbox stage. `frontend/main.tsx:133` |
| A | Apply: open the employer page and show the application confirmation | Shortlist tab and the selected vacancy is in the Shortlist stage (section 8). `frontend/main.tsx:106`, `frontend/main.tsx:134` |
| Escape | Close the filter popover and take focus out of the current control | Always, except when a modal dialog is open. `frontend/main.tsx:127` |

- No shortcut acts while the profile screen or Settings is open, while a modal dialog is open, while Ctrl, Alt or Meta is held, or during text composition. `frontend/main.tsx:126`
- No shortcut other than Escape acts while the focus is in a text field, text area, selection list, editable element or switch. `frontend/main.tsx:128`, `frontend/workflow.ts:28`
- The keys J, K, /, Y, N and A are always swallowed by the page (default action suppressed) even when they do nothing in the current view. `frontend/main.tsx:130`
- The shortcut letters are shown on the buttons as keyboard hints (N, Y, A) and in the search box (/). `frontend/main.tsx:146`, `frontend/main.tsx:166`
- Only Y is independent of the keyboard layout; J, K, N and A are matched by the typed character. `frontend/main.tsx:129`

## 5. Select and the Shortlist

- Select moves the vacancy to the Shortlist (section 2) and, in the same store operation, queues a model assessment of the vacancy at the lowest priority; the queue row is created even when the assessment queue is switched off, and the background worker takes it only when the queue is on. `backend/internal/store/jobs.go:816`, `backend/internal/store/queue.go:159`, `backend/internal/background/assessment.go:117`
- When the previous stage was not the Shortlist, the server also queues a preparation operation that only reads the application form (no model call) so that the questions are ready when the owner opens the Application tab; a repeated Select on a vacancy already in the Shortlist adds nothing. `backend/internal/server/routes.go:176`, `backend/internal/server/preselect_test.go:26`, `backend/internal/server/preselect_test.go:66`
- The Application tab of the vacancy is disabled while the vacancy is in the Inbox ("Shortlist this role to open Application"); in every other stage it opens. Preparing answers is allowed only from the Shortlist. `frontend/RoleContent.tsx:56`, `frontend/RoleContent.tsx:69`
- "Assess now" on the Application tab queues the assessment at priority 100 for any vacancy that has the tab, whatever its stage. `frontend/RoleContent.tsx:54`, `backend/internal/server/routes.go:273`
- Shortlist actions: "Back to Inbox", "Apply ↗ A" and "Confirm application…"; the last one opens the confirmation without opening the employer page, for the case in which the owner already applied. `frontend/main.tsx:166`
- The Shortlist and Archive views list every vacancy in that stage even when its company is blocked; a company block affects only the Inbox. `frontend/main.tsx:83`

Preparation of the application answers is specified separately (application-preparation area); the shortlist re-check on a stronger model is agreed but not built (see the agreed section).

## 6. Dismiss, restore and the Dismissed tab

- Dismiss in the Inbox stores stage Dismissed without a reason. A reason exists only for the three negative outcomes recorded after Apply (section 8). `backend/internal/store/jobs.go:741`, `frontend/main.tsx:166`
- A dismissed vacancy is not shown in the Inbox after the page is reloaded and is not brought back by collection: a posting that matches an existing vacancy keeps that vacancy's stage (section 11). `backend/internal/store/jobs.go:547`
- Archive, Dismissed tab shows the dismissed vacancies, loaded from the server when the tab is opened; the search box and the filters do not apply to this list. Each row shows company, title, the two scores, location and publication text. `frontend/main.tsx:77`, `frontend/main.tsx:155`
- On a dismissed vacancy the header shows "Dismissed" and, when a reason is stored, "Dismissed — Vacancy is stale", "Dismissed — Vacancy is closed" or "Dismissed — Not suitable", with the button "Restore to Inbox". `frontend/main.tsx:166`, `frontend/model/workflow.ts:3`
- Restore moves the vacancy to the Inbox and clears the reason (the history keeps it); it does not touch scores or assessments. `frontend/main.tsx:116`, `backend/internal/store/jobs.go:783`
- Delete role… is available on a dismissed vacancy (section 9). `frontend/main.tsx:168`
- There is no retention rule for dismissed vacancies: they stay until restored or deleted. `backend/internal/store/jobs.go:1089`

## 7. Blocking a company and excluding terms

Two separate mechanisms hide vacancies; they differ in scope and in where the owner manages them.

### 7.1 Block company (from a vacancy in the Inbox)

- The button "Block company" on an Inbox vacancy stores a company rule whose value is the company name normalised by lowercasing, joining a few technology spellings and reducing everything to space-separated letters and digits. A repeated block of the same company stores nothing new. `frontend/main.tsx:117`, `backend/internal/store/blocks.go:87`, `backend/internal/store/blocks.go:21`, `backend/internal/store/core.go:106`
- A vacancy is blocked when its normalised company name equals a stored company rule; the flag is computed on every read, so removing the rule un-hides the vacancies and nothing stored on them changes. `backend/internal/store/blocks.go:109`, `backend/internal/store/jobs.go:282`
- A blocked vacancy is hidden from the Inbox list and from the Inbox count; it stays in stage Inbox in the stored data and is not shown anywhere else in the interface (it is not Dismissed, so it is not in the Dismissed tab). Vacancies of the same company that are already in the Shortlist, Applied or Dismissed are not hidden. `frontend/main.tsx:83`, `frontend/main.tsx:140`
- A rule of kind `domain` is understood by the matching code but nothing creates it; rules of kind `source` were removed when exclusions were unified and are refused now (the sources list in `search.yaml` has an enabled flag instead). `backend/internal/store/blocks.go:82`, `backend/internal/server/routes.go:315`, `backend/internal/store/migrate.go:396`
- The server can list and delete company rules (`GET /api/block-rules`, `DELETE /api/block-rules/{id}`); no screen calls them. `backend/internal/server/routes.go:306`, `backend/internal/server/routes.go:330`
- Collection and the assessment queue ignore company rules: a blocked company's postings are still collected, scored and queued. `backend/internal/background/collect.go:203`, `backend/internal/store/queue.go:753`

### 7.2 Exclusion terms (Settings, Search)

- An exclusion term is a word or phrase stored lowercased and trimmed; an empty term is refused; the owner adds and removes terms in Settings → Search, and while typing sees how many stored vacancies the term would newly hide. `backend/internal/store/stopwords.go:20`, `backend/internal/store/stopwords.go:26`, `backend/internal/store/stopwords.go:129`, `frontend/Collection.tsx:58`, `frontend/Collection.tsx:94`
- A vacancy is excluded when any term occurs, as a case-insensitive literal part of the text, in its title or in its company name. `backend/internal/store/stopwords.go:95`, `backend/internal/store/stopwords.go:105`
- An excluded vacancy is hidden from every list, count and the Dismissed tab whatever its stage, including the Shortlist and Applied; the server omits it from the vacancy list, and removing the term brings it back. `backend/internal/store/jobs.go:331`, `frontend/main.tsx:81`
- Collection skips a matching posting before it is stored (checked on the title known from discovery, again on the resolved title and company), so an excluded vacancy found while the term is active is never stored; one stored earlier stays stored and hidden. `backend/internal/background/collect.go:57`, `backend/internal/background/collect.go:73`, `backend/internal/background/collect.go:164`, `backend/internal/background/collect.go:179`
- The assessment queue's periodic fill skips excluded vacancies. `backend/internal/store/queue.go:753`
- The interface for the terms (three columns, immediate save) belongs to the search area; see [Quick Match](quick-match.md) and the search specification.

## 8. Apply and the outcome of an application

- Apply (button "Apply ↗" or key A) works only in the Shortlist tab on a vacancy in the Shortlist. It opens the employer apply link, or the original listing when no employer link is stored, in a new browser tab; only http and https addresses without embedded credentials are used. `frontend/main.tsx:106`, `frontend/model/workflow.ts:21`
- If neither address is usable, the owner sees "No valid employer or original listing URL is available. The role stays in Shortlist." and nothing changes; if the browser blocks the new tab, the owner sees that the page was blocked and the role stays in the Shortlist. `frontend/main.tsx:108`, `frontend/main.tsx:113`
- Opening the employer page is not a submission. The application confirmation then opens with the question "Did you submit your application?", the title and company, and the explanation that opening the page submits nothing, that the owner should confirm only after submitting personally, that the owner can confirm later from the Shortlist, and that a negative outcome is only for the case in which the owner will not apply. `frontend/main.tsx:114`, `frontend/ApplicationConfirmation.tsx:54`
- The confirmation has five buttons: "Not yet" closes it and changes nothing (the vacancy stays in the Shortlist); "No, vacancy is stale", "No, vacancy is closed" and "No, not suitable" dismiss the vacancy with the matching stored reason; "Yes, applied" records the application. `frontend/ApplicationConfirmation.tsx:69`, `frontend/ApplicationConfirmation.tsx:70`
- Only the buttons close the confirmation: Escape and clicks outside do not, and while a choice is being saved all buttons are disabled; if saving fails the dialog stays open with "The outcome could not be saved. The role has not been changed by this action." `frontend/ApplicationConfirmation.tsx:48`, `frontend/ApplicationConfirmation.tsx:63`, `frontend/ApplicationConfirmation.tsx:29`
- The same confirmation opens from the button "Confirm application…" without opening the employer page. `frontend/main.tsx:166`
- Yes, applied stores the confirmation time as the application date; a negative outcome never stores an application date and stores the reason both on the vacancy and in the history row; a negative outcome for a vacancy that is no longer in the Shortlist is refused so that it cannot overwrite a confirmed application. `backend/internal/store/jobs.go:779`, `backend/internal/store/jobs.go:783`, `backend/internal/store/jobs.go:757`, `backend/internal/store/outcome_reason_test.go:13`
- A vacancy that left the Shortlist (Applied or Dismissed) drops out of the Shortlist list at once and the selection moves to the first vacancy that remains. `frontend/main.tsx:87`
- The three reasons and their wording are fixed in code; there is no free-text explanation. `frontend/model/workflow.ts:2`, `backend/internal/store/jobs.go:722`
- Stored columns: `jobs.status`, `jobs.applied_at`, `jobs.outcome_reason`, and one row per move in `workflow_history` (from stage, to stage, confirmed flag, reason, time). `backend/internal/store/migrate.go:161`, `backend/internal/store/migrate.go:431`

## 9. Archive, retention and deletion

### 9.1 Archive, Applied tab

- The Applied tab lists applied vacancies, newest application date first; the Unread toggle and the Best fit / Newest control are not shown here, and the search box and filters still apply. `frontend/main.tsx:84`, `frontend/main.tsx:146`, `frontend/main.tsx:150`
- Each row shows "Applied <date>"; the vacancy header shows "Applied <date>" with "Undo" (back to the Shortlist, which clears the application date). `frontend/main.tsx:156`, `frontend/main.tsx:166`
- A note under the tabs says: "Applications are permanently deleted 30 days after the confirmed application date. Entries without a valid date are kept." and each applied vacancy shows either the confirmed date with the time of permanent removal, or "No valid past confirmed application date; automatic deletion is disabled." A date is valid only in the exact form `YYYY-MM-DDTHH:MM:SS.mmmZ`. `frontend/main.tsx:153`, `frontend/main.tsx:168`, `frontend/model/workflow.ts:10`

### 9.2 Retention of applied vacancies

- The store has a purge operation that deletes (by the full delete of section 9.3, with tombstones) every applied vacancy whose application date is at least 30 days before a given clock time; vacancies with no date, a date that cannot be read, or a date in the future are kept; a vacancy that cannot be deleted at that moment (work is running) is reported as skipped. `backend/internal/store/jobs.go:1088`, `backend/internal/store/core.go:55`
- Nothing calls the purge operation: no timer, no endpoint, no command. Applied vacancies are therefore not removed automatically today, although the interface text says they are. The wiring is the agreed card GO-7. `backend/internal/store/jobs.go:1088`, `docs/MAP.md:109`
- The retention window of 30 days is a constant in the Go code and another in the frontend; it is not read from configuration. `backend/internal/store/core.go:55`, `frontend/model/workflow.ts:9`

### 9.3 Delete role… (permanent)

- The button "Delete role…" appears in the Shortlist, Applied and Dismissed views, never on a vacancy in the Inbox tab; the server also refuses to delete a vacancy in stage Inbox with the message "Dismiss roles in Triage; permanent Delete is available outside Triage". `frontend/main.tsx:168`, `backend/internal/store/jobs.go:836`
- A modal asks "Delete <title>?", states that the role, its answers and application history are deleted permanently, that this cannot be undone, that automatic collection will not restore it, and that remembered candidate facts and shared files remain; its default focus is "Keep role". The server additionally requires an explicit confirmation flag. `frontend/DeleteRole.tsx:5`, `backend/internal/server/routes.go:194`
- Delete is refused while a collection or an assessment process holds its lock, or while the model assessment of this vacancy is running; the owner is told to wait and delete again. `backend/internal/store/jobs.go:843`, `backend/internal/store/jobs.go:847`, `backend/internal/store/jobs.go:994`
- Delete writes a tombstone for the vacancy's canonical original address, for its apply address, and for every address under which any merged duplicate of it was seen; collection and manual import skip a posting that carries a tombstoned address. A tombstone is by address only; the company and title are not part of it. `backend/internal/store/jobs.go:862`, `backend/internal/store/jobs.go:910`, `backend/internal/store/jobs.go:416`, `backend/internal/store/core.go:119`
- Delete removes the vacancy and all of its rows in sightings, snapshots, assessments, assessment queue, feedback index, workflow history, application drafts and revisions, questionnaire snapshots, preparation operations and aliases; the stored fit results (Fits you and You fit) and extracted facts go with it through cascading rules; the deleted number is never reused; the vacancy is removed from the cached discovery lists; the full-text index entries are removed. `backend/internal/store/jobs.go:917`, `backend/internal/store/jobs.go:929`, `backend/internal/store/jobs.go:944`, `backend/internal/store/jobs.go:986`, `backend/internal/store/migrate.go:511`
- After a successful delete the interface removes the vacancy from every list and shows "Role and application history permanently deleted." `frontend/main.tsx:174`
- No screen and no endpoint removes a tombstone; the store can clear a tombstone only through an option that no caller sets. `backend/internal/store/jobs.go:416`, `backend/internal/server/enrich.go:72`

## 10. What happens to scores and assessments on each transition

- The two fit scores (Fits you, You fit), their requirement verdicts and the old match number are computed from the vacancy text and the profile; the stage is not an input, and no transition recomputes, clears or hides them. No stage condition was found in the score recalculation or in the model work planner. `backend/internal/store/match_fit.go:104`, `backend/internal/matchsvc/service.go:254`
- Select queues a model assessment (section 5); Back to Inbox, Dismiss, Restore, Apply outcomes and Undo leave an existing assessment, its queue row and any drafted application answers in place. `backend/internal/store/jobs.go:816`, `backend/internal/store/jobs.go:787`
- Moving a vacancy back into the Shortlist from the Archive (Undo) queues the assessment and the form inspection again like a first Select. `backend/internal/store/jobs.go:816`, `backend/internal/server/routes.go:176`
- Every Archive and Dismissed row still shows both scores. `frontend/main.tsx:155`, `frontend/main.tsx:156`
- The periodic fill of the assessment queue is not limited to the Shortlist: while the queue is switched on, the worker adds every stored vacancy that is not excluded, in any stage, every 30 seconds. The queue is switched on by default and the setting is in Settings → Queue; on 2026-11-13 the production store holds it switched off (read-only query). `backend/internal/background/assessment.go:117`, `backend/internal/background/assessment.go:126`, `backend/internal/store/queue.go:733`, `backend/internal/store/queue.go:243`
- A dismissed or applied vacancy keeps its stored assessment, so reopening it shows the old summary (marked stale when the vacancy text or the profile changed). `backend/internal/store/jobs.go:146`, `frontend/RoleContent.tsx:53`
- A permanent delete removes the assessment together with the vacancy (section 9.3). `backend/internal/store/jobs.go:919`

## 11. Vacancies seen before, and duplicates

- When collection or manual import brings a posting, the store looks for an existing vacancy by three tests: the same canonical address was already seen for a vacancy; the same canonical apply address belongs to a vacancy; or the normalised company and title are equal, the stored vacancy's company is not "Unknown", the stored vacancy's address is on a different host than the new one, and the two apply addresses do not conflict. `backend/internal/store/jobs.go:442`, `backend/internal/store/jobs.go:399`
- The address normalisation removes the fragment, a leading `www.`, tracking parameters (`utm_*`, `ref`, `source`, `gh_src`) and a trailing slash. `backend/internal/store/core.go:80`
- If a match exists, the posting is added to that vacancy as another source and its text is refreshed only when the new text is richer; the vacancy keeps its stage, so a vacancy already Applied, Dismissed or in the Shortlist does not return to the Inbox. `backend/internal/store/jobs.go:510`, `backend/internal/store/jobs.go:547`, `backend/internal/store/jobs.go:593`
- If several stored vacancies match at once they are merged into the one with the lowest number; the strongest stage wins (Applied, then Shortlist, then Dismissed, then Inbox); the application date is carried over only when the result is Applied; drafts, assessments, history, sightings and feedback move to the surviving vacancy and the old numbers become aliases of it. `backend/internal/store/jobs.go:512`, `backend/internal/store/jobs.go:547`, `backend/internal/store/jobs.go:565`
- Not recognised as duplicates: two listings on the same host with different addresses, a company named "Unknown", listings whose apply addresses conflict, and a listing under a new address whose earlier vacancy was permanently deleted (the tombstone is by address, so a different address is a new vacancy). `backend/internal/store/jobs.go:446`, `backend/internal/store/jobs.go:416`
- An earlier submission is recognised only through the current stage and application date of the matched vacancy. If the owner moved it out of Applied, the date is gone and nothing in the interface reads the history; if the owner deleted it, the history is gone as well. `backend/internal/store/jobs.go:779`, `backend/internal/store/jobs.go:921`, `backend/internal/store/queue.go:361`

## 12. Configuration and stored data

| Item | Where it lives | Notes |
|---|---|---|
| Stage and application date, reason | `jobs.status`, `jobs.applied_at`, `jobs.outcome_reason` in the store | `backend/internal/store/migrate.go:32`, `backend/internal/store/migrate.go:431` |
| History of moves | `workflow_history` | No screen or endpoint reads it. `backend/internal/store/migrate.go:161`, `backend/internal/store/queue.go:361` |
| First read time | `jobs.seen_at` | `backend/internal/store/jobs.go:706` |
| Company blocks | `block_rules` (kind, value, created time; unique per kind and value) | `backend/internal/store/migrate.go:204` |
| Exclusion terms | `stop_words` | `backend/internal/store/migrate.go:212` |
| Deleted listings | `deleted_listings` (hash of the canonical address) | `backend/internal/store/migrate.go:202` |
| Merged duplicate numbers | `job_aliases` | `backend/internal/store/migrate.go:97` |
| Retention window (30 days) | Go constant and frontend constant, not configuration | `backend/internal/store/core.go:55`, `frontend/model/workflow.ts:9` |
| Best fit / Newest order | `runtime.yaml`, key `quick_match.model_passes.order` | `backend/internal/server/routes.go:499`, `backend/internal/config/order.go:13` |
| List filters and sort in the browser | browser local storage keys `fcto.catalogue.query`, `fcto.catalogue.filters`, `sift.quick`, `sift.unread`, `sift.sort` | `frontend/main.tsx:78` |

Measured on 2026-11-13 with read-only queries on the production store: Inbox 2588, Shortlist 59, Applied 23, Dismissed 163 (one with reason `not_suitable`), 15 company blocks, 5 exclusion terms, 0 tombstones, schema version 22, assessment queue switched off. These numbers are a snapshot, not a rule.

## Implemented on a branch, not released

- Nothing of this specification changes on the INBOX-2 branch (vacancy view and description sections, not merged): the stage rules, shortcuts, block, dismiss, apply and delete code is unchanged there except that block, status and delete additionally advance a catalogue change counter that the list polling uses, and the list requests get timeouts and a "server unreachable" message. Only the diff of `frontend/main.tsx`, `backend/internal/store/blocks.go` and `backend/internal/store/jobs.go` was read; not verified line by line. Card INBOX-2 (`work/tasks/inbox/INBOX-2_listing-markup/`).
- The main-checkout code includes everything that cards APP-8 (five outcomes), TRIAGE-2 (physical Y key) and COLLECTION-1 (publication date) added to this area; APP-8 and TRIAGE-2 are still in state "review" or "in progress" on their cards, and whether the running production build contains each of them was not verified (the quick-match specification records a release from `157fe9f`, which contains APP-8 but not the TRIAGE-2 commit `e295b21`; later releases of ASSESSMENT-8 and ASSESSMENT-9 are recorded on their cards).

## Agreed, not implemented

- **Triage, Shortlist and rejections per profile.** The owner agreed that each profile has its own Triage, Preparation and rejections, with a submitted application as a person-level fact visible in every profile: «да отдельные. сохрани все это в карточке, но сейчас мы это делать не будем. отложим на потом.» Interface: Inbox, Shortlist and Archive show the active profile's vacancies. Stage SETTINGS-5 S.4 and S.5, approved 2026-11-12 to start after INBOX-2; details in [Profiles](profiles.md) §4. Card SETTINGS-5.
- **Free-text reason on "not suitable".** The owner, on accepting the five-outcome dialog: «39 минут на модалку? она даже не спрашивает почему "not suitable"». Captured as follow-up, no plan and no separate card; provisional complexity 1. Card APP-8 (`comments.md` in its worktree).
- **Recognise a vacancy applied to before.** The owner: «любая вакансия от chainsync, coinbag, uniswap» and, on the real history: «да я иногда жал что подался, а иногда возвращал в триаж и удалял оттуда. не важно. все делал на 8080». Production is the authoritative history; recognition must look at earlier submissions even after the vacancy was returned, rejected or removed; no merging of histories without a separate approved plan; no plan written yet. Card TRIAGE-1.
- **Archive retention actually running.** Applied vacancies are to be purged after 30 days by a timer; the store method exists. Card GO-7.
- **Dismiss a requirement, not a vacancy.** The owner agreed that a requirement shown in the vacancy panel can be dismissed as "not important to me" (with "Dismissed · Undo" for a few seconds and a profile list); this changes You fit, not the stage of a vacancy. Mentioned here only to avoid confusing it with Dismiss in this specification. Card ASSESSMENT-4.
- **Shortlist re-check on a stronger model.** On entering the Shortlist both model passes run again on an owner-chosen model; backlog idea, the first line of the card is the executor's record and not the owner's wording, the second is verbatim: «кажется все это вышло из под контроля. нам нужны новые карточки, обновления specs.» Open: default model, route, automatic or on request. Card ASSESSMENT-6.
- **Not decided.** Whether free-form dealbreakers from the profile are shown highlighted in the Inbox (never hiding a vacancy) or act as exclusions; whether a duplicate-recognition result shows the earlier application date in the interface. Cards INBOX-1 and TRIAGE-1.

## Known limits and defects

- Applied vacancies are never purged although the interface says they are (sections 9.1 and 9.2). Card GO-7.
- A company block cannot be seen or undone from the interface: the Settings panel that listed company blocks and recently dismissed vacancies was removed deliberately, and the server endpoints that list and delete the rules have no screen. Fifteen such blocks exist in production. No card.
- A blocked vacancy stays in stage Inbox but is hidden from the Inbox and shown nowhere else, so it cannot be reached from the interface until the rule is deleted through the server. No card.
- "Block company" derives the rule from the company text of the vacancy without a guard; for a vacancy whose company is the placeholder "Unknown" this would hide every vacancy that carries the placeholder (inferred from the code, not tried). No card.
- Only the Y shortcut survives a non-Latin keyboard layout; N, A, J and K do not. Card TRIAGE-2 deliberately limited itself to Y; no card for the rest.
- The history of moves (`workflow_history`) is written but not read by any screen, endpoint or command; a permanent delete erases it together with the vacancy, so an earlier application cannot be recognised after a delete. Related to TRIAGE-1.
- Moving an Applied vacancy back with Undo clears its application date; only the history keeps the fact. Related to TRIAGE-1.
- A permanent delete protects only the addresses of the deleted listing; the same vacancy listed under another address on another site is collected again as a new vacancy in the Inbox. No card.
- A tombstone can never be removed (no caller of the restore option), and importing a URL whose address is tombstoned silently stores nothing (the import answers with number 0). No card; the second point comes from reading `ImportURL` and `Upsert`, not verified by running.
- The note "N dismissed this session — they disappear next time" does not say that they can be found under Archive, Dismissed. The Dismissed tab ignores the search box and the filters, and its list is read once when the tab is opened. No card.
- The server accepts stage moves the interface never offers (for example Applied to Dismissed without a reason, or Dismissed to Shortlist). No card.
- A permanent delete returns an empty "cleanup errors" list in every case, so the notice "a local technical log could not be removed" in the interface cannot appear. No card.
- Ordinary dismissal has no reason; the owner cannot say why a vacancy was dismissed in the Inbox. No card.
- Expired vacancies are not removed from the Inbox automatically. No card (liveness detection is not planned).

## Contradictions

For the coordinator to decide; none is resolved here.

- **Stage names.** `docs/MAP.md`, the cards NAVIGATION-1, APP-8 and SETTINGS-5 history, and the quick-match specification speak of Triage, Pre-selected and Preparation; NAVIGATION-1 records the owner's word "Preparation" for the middle tab (2026-11-05). The interface shows Inbox, Shortlist and Archive (introduced with the redesign in commit `5bedae3`, card DESIGN-3); I found no card recording an owner decision on these names. `docs/MAP.md:99`, `frontend/main.tsx:140`
- **Model assessment only after Select.** `docs/MAP.md` (R-ASM outcome), the comment in `backend/internal/store/jobs.go:814`, the text in the vacancy panel ("Detailed model assessment is shown in Application for shortlisted roles") and the card MAP entries say the detailed assessment starts when a vacancy is moved to the Shortlist or on "Assess now"; the code also queues every non-excluded vacancy in any stage every 30 seconds while the queue is on. Card INBOX-1 records the same finding. `backend/internal/background/assessment.go:126`, `backend/internal/store/queue.go:733`, `frontend/RoleContent.tsx:77`
- **Company block and its management.** Cards TRI-1 and TRI-2 (both marked done) require a Settings list of company blocks with removal, recently dismissed vacancies with restore, and a source block created in Settings; the cards NAVIGATION-2 and NAVIGATION-3 removed the panel and the source blocks and say the company rules move into the single exclusion list. In the code the migration copied the existing company rules into exclusion terms once, but "Block company" still writes to the separate block table, with a smaller scope (Inbox only) than an exclusion term (everywhere, and it stops collection). `docs/MAP.md:99` lists the TRI cards as the basis of "blocks a company"; nothing there says the management screen is gone. `backend/internal/store/blocks.go:87`, `backend/internal/store/migrate.go:385`, `work/tasks/navigation/NAVIGATION-3_search-editor-frontend/plan.md:28`
- **Retention text.** The Archive note and the per-vacancy line say applied vacancies are deleted after 30 days; `docs/MAP.md` says the method exists and no timer calls it, and the code agrees with the MAP. `frontend/main.tsx:153`, `docs/MAP.md:110`
- **"Automatic collection will not restore it".** The delete dialog promises this without qualification; the code keeps a deleted listing out only by address, not by company and title. `frontend/DeleteRole.tsx:5`, `backend/internal/store/jobs.go:416`
- **MAP status of the area.** `docs/MAP.md` calls R-TRI "implemented, with one timer gap"; besides the timer gap the company-block management is missing from the interface and the history is unreadable (see the known limits). `docs/MAP.md:101`
- **State of TRIAGE-2.** The MAP lists it as in progress; the card file says review; the physical-key fix is in main. `docs/MAP.md:318`, `frontend/main.tsx:129`
- **Per-profile decisions versus code.** `docs/specs/profiles.md` §4 says Triage, Shortlist and rejections are per profile; the code keeps one stage per vacancy for the single profile (agreed, not implemented). `backend/internal/store/jobs.go:741`

## Related specifications

- [Quick Match](quick-match.md): the two scores, sorting (Best fit, Newest), the model passes and their order.
- [Vacancy view](vacancy-view.md): layout of the vacancy header with Block company, Dismiss and Select, and the key facts.
- [Profiles](profiles.md): per-profile Triage and decisions, and the shared application history.
- Collection, search and the exclusion terms editor, application preparation, settings and data ownership: specifications written by the same card OPS-11 (`docs/specs/collection.md`, `docs/specs/search.md`, `docs/specs/application.md`, `docs/specs/settings.md`, `docs/specs/data.md`; not yet present in the checkout at the time of writing).
- Cards read for this specification: APP-8, TRIAGE-1, TRIAGE-2, TRI-1, TRI-2, ASSESSMENT-4, ASSESSMENT-6, GO-7, INBOX-1, INBOX-2 (diff only), NAVIGATION-1 to NAVIGATION-4, SETTINGS-5, COL-6.
