# APP-8 — Add negative outcomes to application confirmation

**Epic:** application-preparation
**Created:** 2026-11-06
**Status:** review
**Kind:** idea

**Integration status,2026-11-10:** technical implementation accepted; source published to main in commit `a3ead8b`, final application source `58af432`. Card remains review, not done; production is not updated. Owner acceptance/release is separate.

## Why this work exists

**Current result,2026-11-09T21:02Z:** owner and coordinator accepted the five-outcome/button-only modal through7a2042e, records checkpointd2db849 in .worktrees/APP-8; review preview8083 refreshed with the corrected backend. Report copied to [acceptance.md](acceptance.md). Existingv16 migration gate fixed at acceptance. Total~39min execution/preparation plus~5min review, versus10–20min estimated. Source published to main (`a3ead8b`, final `58af432`); release outstanding. Owner's missing "why not suitable" explanatory text is captured in the worktree comments as follow-up, not implemented; provisionalcomplexity1,5–10min if approved later. Next product focus is COLLECTION-1, not another modal cycle.

Owner context: «при отправке заявки приложение спрашивает:» followed by the dialog below. Owner request: «не хватает ответа: вакансия уже закрыта (expired)» and «запиши в бэклог».

Owner's clarification: «еще, занеси в backlog - при вопросе applied я или нет - добавить еще опций: нет, вакансия закрыта, и нет, передумал подаваться.»

Reported dialog:

> Did you submit your application?
> Smart Contract Engineer Solidity EVM · Saifuro
> Opening the employer page does not submit an application. Confirm only after you have submitted it yourself. You can also confirm later from Pre-selected.
> Not yet
> Yes, applied

Add two explicit negative responses: “No, vacancy closed” («Нет, вакансия закрыта») and “No, changed my mind” («Нет, передумал подаваться»). The current two answers cannot distinguish these outcomes from “Not yet”. Preparation must establish how each response maps to persisted vacancy state, list/archive visibility and possible later recollection; those details were not decided by this report.

## Preparation update —2026-11-09

New exact wording is captured in comments.md: after Apply from Preparation, the owner also needs «вакансия старая/не подошло». This is part of the minimal manual workflow, not approval of an automatic application system. Coordinator proposes explicit submitted / not yet / stale / not suitable outcomes, preserving the earlier closed-listing request as a distinct reason if represented. Negative outcomes go to dismissed/archive with a durable reason and must not set applied_at or pretend an application was submitted; Not yet stays in Preparation. The final reason/visibility behavior is to be included in the complete approved plan.

Preliminary **complexity2**, preparation2–3min/implementation5–10min/acceptance3–7min, **total10–20min** excluding owner wait. Existing dialog and workflow API/store are reusable; add only missing durable outcome/reason, real browser checks and focused hidden-state preservation coverage. Existing subscription, attributable price unknown, no new services/model calls. This is a separate small outcome from deduplication and the incoming-list decision, not blanket approval of the umbrella. No implementation performed.

## Acceptance criteria

- Application confirmation offers both a closed/expired response and a separate changed-mind response alongside the existing outcomes.
- Neither response falsely records an application as submitted, and both remain distinct from “Not yet”.
- The approved state and visibility behavior persists across reloads and is consistent with collection/reappearance rules.

## Owner answers —2026-11-09

The coordinator proposed the five-outcome mapping; the owner approved it, not authored it: «да, давай. и чтобы нажатие мышкой за пределами modal не закрыавало этот modal. только кнопка.» Product behavior is agreed: Applied records a submission; Not yet keeps Preparation; Stale/Closed/Not suitable dismiss with a stored distinct reason. Only a button dismisses the modal; clicking its backdrop or pressing Escape must not dismiss it. While saving, wait for success; failure stays visible with the original state/draft retained.

Complete bounded implementation and cost are now in plan.md for approval. Original broad manual-loop questions resolved in INBOX-1. This card does not change collection timing or add an incoming stage, and does not merge production/staging histories. Preserve the separate Profile frontend writer.
