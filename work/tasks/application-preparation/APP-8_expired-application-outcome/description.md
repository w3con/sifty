# APP-8 — Add negative outcomes to application confirmation

**Epic:** application-preparation
**Created:** 2026-11-06
**Status:** backlog; preparation not started
**Kind:** idea

## Why this work exists

Owner context: «при отправке заявки приложение спрашивает:» followed by the dialog below. Owner request: «не хватает ответа: вакансия уже закрыта (expired)» and «запиши в бэклог».

Owner's clarification: «еще, занеси в backlog - при вопросе applied я или нет - добавить еще опций: нет, вакансия закрыта, и нет, передумал подаваться.»

Reported dialog:

> Did you submit your application?
> Smart Contract Engineer Solidity EVM · Saifuro
> Opening the employer page does not submit an application. Confirm only after you have submitted it yourself. You can also confirm later from Pre-selected.
> Not yet
> Yes, applied

Add two explicit negative responses: “No, vacancy closed” («Нет, вакансия закрыта») and “No, changed my mind” («Нет, передумал подаваться»). The current two answers cannot distinguish these outcomes from “Not yet”. Preparation must establish how each response maps to persisted vacancy state, list/archive visibility and possible later recollection; those details were not decided by this report.

## Acceptance criteria

- Application confirmation offers both a closed/expired response and a separate changed-mind response alongside the existing outcomes.
- Neither response falsely records an application as submitted, and both remain distinct from “Not yet”.
- The approved state and visibility behavior persists across reloads and is consistent with collection/reappearance rules.
