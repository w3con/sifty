# APP-8 — Report an expired vacancy from application confirmation

**Epic:** application-preparation
**Created:** 2026-11-06
**Status:** backlog; preparation not started

## Why this work exists

Owner context: «при отправке заявки приложение спрашивает:» followed by the dialog below. Owner request: «не хватает ответа: вакансия уже закрыта (expired)» and «запиши в бэклог».

Reported dialog:

> Did you submit your application?
> Smart Contract Engineer Solidity EVM · Saifuro
> Opening the employer page does not submit an application. Confirm only after you have submitted it yourself. You can also confirm later from Pre-selected.
> Not yet
> Yes, applied

Add an explicit outcome for discovering that the vacancy is already closed/expired after opening the employer page. The current two answers cannot express that outcome. Preparation must establish how this maps to persisted vacancy state, list/archive visibility and possible later recollection; those details were not decided by this report.

## Acceptance criteria

- Application confirmation offers a clear closed/expired response alongside the existing outcomes.
- The response does not falsely record an application as submitted.
- The approved state and visibility behavior persists across reloads and is consistent with collection/reappearance rules.
