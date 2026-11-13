# Triage and decisions — specification

Status: owner decisions of 2026-11-13 applied (OPS-11); rule tags show the implementation state.

## 1. Purpose

The owner collects a large and constantly refilled list of vacancies and needs to decide quickly which ones to pursue, which to dismiss, and when to record an actual application. This feature gives the owner four reversible decisions — keep for later (Inbox), pursue it (Shortlist), reject it (Dismissed), confirm an application was submitted (Applied) — and makes those decisions stick: a vacancy the owner dismissed or applied to never comes back into the Inbox when collection runs again, and the owner can reach past decisions through the Archive. Blocking a company removes an entire company's postings from the owner's view in one action without handling each vacancy separately.

The owner's own description of the goal (card INBOX-1): "First, in manual mode, the owner should be able to fetch vacancies, sort them, move them to triage or reject them. From the preparation folder, clicking Apply should eventually show not only 'replied / not replied' but also 'vacancy old / not suitable'."

## 2. Scope

This specification covers: the four stages and transitions between them; the in-session undo; the application confirmation and its five outcomes; blocking a company; the Archive (Applied tab and Dismissed tab); permanent deletion and tombstones; what happens to scores on each transition; and how collection handles a vacancy already seen.

Adjacent specifications: [Catalogue](catalogue.md) for exclusion terms, search expressions, sorting and catalogue filters. [Quick Match](quick-match.md) for the Fits you and You fit scores and the model assessment queue. [Vacancy view](vacancy-view.md) for the layout of the vacancy header and the action buttons. [Application preparation](application.md) for drafting answers. [Collection](collection.md) for how postings are fetched and stored. [Profiles](profiles.md) for the agreed per-profile decisions (not yet implemented). [Data](data.md) for data ownership, export and deletion.

## 3. Rules

### 3.0 Canonical screen names

The three browsable screens are named Inbox (new and undecided vacancies), Shortlist (selected for pursuit), and Archive (the Applied and Dismissed views together). Owner, 2026-11-13: accepted the coordinator's recommendation; introduced by DESIGN-3 in production. [implemented — DESIGN-3; cards NAVIGATION-1, APP-8, SETTINGS-5 and docs/MAP.md retain the old names and are to be updated]

### 3.1 Stages

1. Every vacancy is in exactly one of four stages at all times: Inbox, Shortlist, Applied, Dismissed. Every newly collected or imported vacancy starts in the Inbox. [implemented]

### 3.2 Decisions in the Inbox

2. The owner can Select a vacancy (move it to the Shortlist) or Dismiss it from the Inbox. A dismissal from the Inbox carries no reason. [implemented]

3. A vacancy the owner just selected or dismissed stays visible in the Inbox for the rest of the browser session so it can be undone; it disappears from the Inbox on the next reload. [implemented]

4. While a decided vacancy is still visible in the session, the owner can undo the decision; the vacancy returns to the Inbox. [implemented]

### 3.3 Decisions from the Shortlist

5. The owner can return a shortlisted vacancy to the Inbox; this does not affect its scores or assessments. [implemented]

6. The owner can record a confirmed application or dismiss with an outcome reason only through the application confirmation (see section 3.5). [implemented]

7. The owner can open the application confirmation without visiting the employer page, for cases where the application was submitted outside the tool. [implemented]

### 3.4 Decisions from the Archive

8. The owner can undo an Applied vacancy back to the Shortlist from the Archive; this removes the confirmed application date. [implemented]

9. The owner can restore a Dismissed vacancy to the Inbox from the Archive; this removes the stored outcome reason. [implemented]

### 3.5 Apply and the application confirmation

10. Apply works only on a Shortlist vacancy. It opens the employer apply link, or the original listing when no employer link is stored. If no usable URL is available, the vacancy stays in the Shortlist. [implemented]

11. Opening the employer page does not record a submission. The owner must confirm personally after submitting; the tool never applies on the owner's behalf. [implemented]

12. The confirmation has five outcomes: "Not yet" leaves the vacancy in the Shortlist unchanged; "Yes, applied" records the application with the confirmation time; "No, vacancy is stale", "No, vacancy is closed" and "No, not suitable" each dismiss the vacancy with the matching reason. Only "Yes, applied" sets a confirmed application date. [implemented]

13. The "not suitable" outcome has no free-text field for the owner to explain why. [not implemented — no card; owner remark in APP-8 comments: "It doesn't even ask why 'not suitable'."]

### 3.6 Decision integrity

14. A request to move a vacancy to the stage it already occupies changes nothing. [implemented]

15. A negative outcome or a confirmed application is accepted only while the vacancy is currently in the Shortlist; a request that arrives after the vacancy has changed stage is refused, so a confirmed application is never silently overwritten. [implemented]

16. Every stage move is recorded in its history (prior stage, new stage, confirmed flag, outcome reason, time); the history survives until the vacancy is permanently deleted but cannot be read from the interface or any command. [deviates — history is written but never surfaced; related to TRIAGE-1]

### 3.7 Keyboard shortcuts

17. Every triage decision (Select, Dismiss, Apply) and catalogue navigation has a keyboard shortcut. Shortcuts never fire while the owner is typing in a text field or while a dialog is open. [implemented]

18. The Select shortcut works on any keyboard layout (physical key — owner request fulfilled in TRIAGE-2). [implemented] The Dismiss and Apply shortcuts are matched by the typed character and behave differently on non-Latin layouts. [deviates — TRIAGE-2 deliberately scoped to Y only; no card for the remaining shortcuts]

### 3.8 Blocking a company

19. Blocking a company creates an exclusion entry in a unified exclusion list managed in Settings. A blocked company's vacancies are hidden everywhere (Inbox, Shortlist, Applied and Dismissed views) and future collection of that company's postings is also stopped. Blocking the placeholder company name "Unknown" is refused. Owner, 2026-11-13: accepted the coordinator's recommendation; the unused domain-block kind is also removed under this decision. [not implemented — no card; today company blocks are Inbox-only, do not stop collection, and are stored in a separate `block_rules` table]

20. A company exclusion entry is reversible: removing the entry from Settings restores all hidden vacancies immediately and collection resumes. [not implemented — no card; today removal requires direct database access]

21. The owner can review and remove all company exclusion entries from a management screen in Settings. [not implemented — no card; today no screen exposes the 15 existing blocks in production]

### 3.9 Dismissed

22. Dismissed vacancies are shown in the Archive, Dismissed tab. The catalogue search and filters do not apply there. A dismissed vacancy has no automatic retention limit and stays dismissed until restored or permanently deleted. [implemented]

23. A vacancy dismissed directly from the Inbox carries no reason. A dismissal through the application confirmation carries one of three reasons (stale, closed, not suitable), shown in the Archive. [implemented]

### 3.10 Applied and retention

24. Applied vacancies are shown in the Archive with their confirmed application date. [implemented]

25. Applied vacancies are not deleted automatically; there is no retention limit. Owner, 2026-11-13, choosing option (b) of the coordinator's question (never delete automatically, remove the promise): "On question 7, option (b): we need to add the ability to delete entries in Shortlist and Archive — a button?" The owner can permanently delete an applied vacancy at any time using the "Delete role…" button available in Shortlist and both Archive tabs. [deviates — the application interface still shows text promising 30-day automatic deletion; that text must be corrected; GO-7 step 1.2 (the purge wiring) is superseded by this decision]

### 3.11 Permanent deletion

26. The owner can permanently delete a vacancy from the Shortlist, Applied or Dismissed views; it is not available in the Inbox. [implemented]

27. Permanent deletion requires explicit confirmation. The vacancy, all its application history, its assessments and all application drafts are deleted; the owner's shared facts and documents are not affected. This action cannot be undone. [implemented]

28. Delete is refused while a collection or assessment process is running on the vacancy; the owner is told to wait and try again. [implemented]

29. Delete writes a tombstone for each URL the vacancy was ever seen under; collection skips any posting whose address is tombstoned. [implemented] The same vacancy listed under a different address on another site is not covered and would be collected again as a new Inbox entry; the deletion dialog states that automatic collection will not restore the vacancy without that qualification. [deviates — dialog states the promise unconditionally]

30. A tombstone cannot be removed; there is no way to allow a tombstoned address to be collected again. [implemented — permanent constraint]

### 3.12 Scores on transitions

31. No stage transition recalculates, hides or clears the two fit scores; scores are visible for every vacancy regardless of stage. [implemented]

32. Moving a vacancy to the Shortlist triggers a model assessment and a form inspection. Moving it back to the Shortlist from Applied also triggers both, as if for the first time. All other transitions leave existing assessments and application drafts in place. [implemented]

33. The model assessment queue, while switched on, adds only Shortlist vacancies, not the whole catalogue. Owner, 2026-11-13: accepted the coordinator's recommendation. [deviates — the code adds every non-excluded vacancy regardless of stage every 30 seconds while the queue is On; no card for the fix]

### 3.13 Vacancies seen before

34. When collection or import brings a posting whose address or company-and-title already matches a stored vacancy, that vacancy keeps its stage, scores and application history; it does not return to the Inbox. [implemented]

35. When several stored vacancies match a new posting, they merge into one; the strongest stage wins (Applied > Shortlist > Dismissed > Inbox), and application history, assessments and drafts move to the surviving vacancy. [implemented]

36. The owner can find whether a vacancy was applied to through the Applied tab, the Dismissed tab (for negative outcomes), and the stage of a matched vacancy. If the vacancy was returned to the Inbox and later deleted, the application history is gone; the history records every past move but no screen or command can read it. [deviates — TRIAGE-1 owns this gap]

### 3.14 Per-profile decisions (agreed, not implemented)

37. Inbox, Shortlist and rejections are per profile: a vacancy dismissed for one profile may be shortlisted in another; the active profile's decisions are shown. [not implemented — SETTINGS-5; profiles.md §4; the code keeps one stage per vacancy]

38. A submitted application is a fact about the person and is visible across all profiles, recorded once with the profile used at the time. [not implemented — SETTINGS-5; profiles.md §4]

## 4. Configuration

| Item | Where the owner sets it |
|---|---|
| Sort order (Best fit / Newest) in Inbox and Shortlist | Toggle in the application header; persisted on the server and followed by the background analysis worker |
| Assessment queue on/off | Settings → Queue |

## 5. Questions for the owner

All questions from this draft were answered by the owner on 2026-11-13 (see log.md). There are no open questions for this feature.

## 6. Open cards

GO-7 — its step 1.2 (wire the 30-day archive purge) is superseded by the owner's decision of 2026-11-13 (rule 25); the interface text promising deletion is to be removed (no card yet).
TRIAGE-1 — recognising a vacancy the owner previously applied to, including after the vacancy was returned, rejected or deleted; every move is recorded in history but no screen reads it.
SETTINGS-5 — per-profile Inbox, Shortlist and rejections, with submitted applications shared across profiles.
ASSESSMENT-6 — Shortlist re-check on a stronger model; not yet planned.
APP-8 follow-up — free-text reason on "not suitable" outcome; no card, provisional complexity 1.
