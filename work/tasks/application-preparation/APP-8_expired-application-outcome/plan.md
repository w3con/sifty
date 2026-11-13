---
estimate: 2
---
# APP-8 — Manual Apply outcomes and explicit modal dismissal

**Approval:** owner «да, давай. и чтобы нажатие мышкой за пределами modal не закрыавало этот modal. только кнопка.» authorizes the five-outcome/button-only summary. This plan is the coordinator's implementation of that agreed scope; complete10–20min step/cost breakdown shown before execution. Latest owner clarification «все делал на 8080» establishes production as real-use environment and ends the two-environment forensic detour; it does not authorize history merging. Start2026-11-09T19:02Z, stop19:22Z, one executor/combined A.1. Coordinator ses_e4a7d8341ffe1Xqrv4Rmx52jqw.

## Outcome and real use

After opening an employer page from Preparation, the owner can record Applied, Not yet, Stale, Closed or Not suitable. Negative decisions persist in dismissed/archive with their reason. The modal waits for an explicit button and never disappears because of an outside click or Escape. Demonstrate in the real running app, including reload, failed save and reopening Archive. No actual application is submitted by this task.

## Boundaries and sources

Reuse existing ApplicationConfirmation, transition API/store workflow and history. Persist domain outcome/reason in SQLite, separately from scraped vacancy metadata; do not invent source publication/expiry dates or set applied_at for a rejection. Domain/store owns validation and atomic status/reason/history, the API and container own effects, the modal/list render data and named callbacks. Existing configuration/path adapters remain authoritative; no new settings system, user/profile literals, host paths, ports or provider choices embedded in application logic/views.

Preserve prior submitted applications/history and unrelated fields on retries/stale actions. Failure must not partially change a stage or discard the decision. Existing ordinary dismissals without a reason remain valid. Not yet is not a persisted rejection. Keep current collection-on-app-load with existing cooldown and existing Triage→Preparation/No. No extra incoming stage, scoring change, collector/dedup rewrite, data refresh/history sync, Profile/frontend edits, production release or main integration. Use a separate worktree from an inspected committed baseline retaining accepted Profile; never edit the other agent's DESIGN-3 worktree or take its8081preview. Review on isolated current-data copy/independent preview; any later shared-runtime transition needs coordination and a recoverable snapshot.

## Estimate and complete steps

**Complexity2:** small extension of existing dialog and workflow persistence. AI preparation2–3min, implementation5–10min, acceptance3–7min; **total10–20min**, owner wait/external delays excluded. OpenCode Go subscription, attributable monetary cost unknown; no new paid services/runtime model calls. Upper20min is a stop/re-estimate boundary from executor preparation start. One executor, one combined pass and final review; no component-library or test-framework work.

| Part of A.1 | Concrete result and necessity | AI elapsed, shared preparation included | External cost |
|---|---|---|---|
| Outcomes | Five explicit answers, atomic stored dismissal reason and visible Archive reason; owner retains the decision |5–10min including preparation|No new service|
| Modal | Button-only dismissal, no backdrop/Escape dismissal, saving/error retained |2–3min|None|
| Final review | Builds, real browser outcomes/reload/modal interactions, narrowly verified state/history preservation |3–7min|None|

All parts mandatory; total10–20min, not three separate executor stages.

## Checkpoints

- [x] A.1 — Persist five Apply outcomes, make dismissal button-only, verify atomic history and real browser behavior.

## Acceptance criteria

- Applied still needs explicit confirmation and the existing Preparation condition. Not yet closes by button without moving the job. Stale/Closed/Not suitable leave Preparation for dismissed/archive, preserve the selected reason across reload and never record a submission.
- Outside click and Escape do not close the modal. Button activation works with mouse/keyboard. Failed save leaves the modal and previous persisted stage intact; in-progress actions cannot be double-submitted.
- Previously submitted application/history cannot be erased by a stale negative confirmation. Only1–2 targeted hidden-state checks for atomic stage/reason/history and stale preservation; visible controls are checked in a real browser, not a broad UI suite.
- Backend/frontend builds pass. Defaultwork-test once reports its actual SKIPPED status; no full suites/model/collection calls. Real demonstrations use isolated data and do not submit applications or change owner history.
- Card report records source revision, real browser evidence, actual elapsed, technical decisions and any corrected mistakes. Owner acceptance and integration/release are separate.
