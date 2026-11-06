# APP-2 — Application tab: spinner, questions before answers, honest empty states

**Epic:** application-preparation
**Created:** 2026-11-03
**Status:** founded
**Short card:** steps and acceptance criteria live here.

## Why this work exists

Form inspection now runs when a job is pre-selected, clicks through applicant-tracking "Apply" controls, and ends in honest outcomes on the backend (questions found; no readable questions; sign-in required; no resolved application form). The Application tab (`app/web/ApplicationDraft.tsx`) still presents all of this through one generic banner — `${operation.status}: ${operation.progress}` while working, and "No prepared answers yet. Inspect the questionnaire or sign in to continue." when an operation is done without answers — so the owner cannot tell reading-in-progress from no-form from needs-sign-in, and questions that were read are not presented as the thing to look at before any answers exist.

The tab should show the chain honestly: an indeterminate spinner while the form is being read (the number of questions is unknown ahead of time, so no percentage); the form's questions as soon as inspection returns, each with its own answer state, so questions appear before answers; and a distinct state for each non-question outcome.

A related defect sits in the same component: the "Preparation provider" selector under More actions starts as a hardcoded `'openai'` and that value is sent with every preparation request, including the automatic one when the tab opens. A request-level provider overrides the stage default, so the owner's choice of preparation model in Configuration is silently ignored for every preparation started from this tab.

## Must not touch

- The inspection and selection mechanics, the backend outcome messages, and the per-question editing, remembering and manual-question features. Answer sourcing (knowledge-base answers are a separate card). The stage-model endpoints.

## Notes

- Backend outcomes to map, as produced by the preparation worker: questions found ("Questionnaire ready for review; nothing submitted"); reachable form with no readable questions ("No readable questions found…"); `questionnaire.status === 'login-required'` ("Sign in to continue…"); no resolved form, including download or non-HTML destinations ("no resolved application form; apply via the original listing"); a failed or interrupted operation with an error. Prefer keying on structured status fields where they exist rather than matching message text; if a needed outcome has no structured marker, record that in the log and match the message text exactly.
- The original listing URL and the inspection destination URL are both available to the tab (job and `draft.questionnaire.url`); the "apply directly" and "apply via the listing" states link to the right one.
- `GET /api/stage-providers` returns the stage selections and the provider list.
- `app/dist` is not tracked; rebuild it in the main checkout for the live check. UI code under `app/` follows the same no-card-identifier rule in comments.

## Steps

- [x] 1.1 — While an inspect operation is queued or running and no questions are known, show an indeterminate spinner with "Reading the application form…" in place of the status/progress banner; check: the UI verification renders the spinner for a running inspect operation and no percentage or raw status text.
- [x] 1.2 — When the draft has questions but no answers, list every question with a per-question "No answer yet" state instead of the "No prepared answers yet" banner; check: for a draft with fields and empty values the tab lists each question with its answer state and the generic banner is absent.
- [x] 1.3 — Show a distinct state for each non-question outcome: no questions on a reachable form ("No questions to answer — apply directly" linking the destination), sign-in required, no resolved application form ("Apply via the original listing" linking the listing), not pre-selected, and failed with a concise error and retry; check: each of the five outcomes renders its own text and link and none shows the generic banner.
- [x] 1.4 — Make the Preparation provider selector default to the configured preparation stage (shown as the default), and send a provider in the request only when the owner picks one explicitly; when no preparation stage is configured, show "Choose a preparation model in Configuration" with a way to open it instead of starting a request that would fail; check: starting preparation from the tab without touching the selector sends no provider and the operation runs on `stages.preparation`, picking a provider sends it, and with the stage unset the tab shows the configuration prompt and sends nothing.
- [x] 2.1 — Rebuild the frontend and verify on the running demo over the owner's data: a job whose form was read shows its questions, a job with no resolved form shows the listing state, a job not pre-selected shows that state, and preparation follows the Configuration stage model; check: a recorded run of those four observations.

## Acceptance criteria

- The Application tab shows an indeterminate spinner while the form is being read, lists the form's questions as soon as they are known with a per-question answer state, and shows a distinct honest state for: no questions, sign-in required, no resolved form, not pre-selected, and failure.
- Preparation started from the tab uses the preparation model chosen in Configuration unless the owner explicitly picks another in the tab.
- The generic `status: progress` / "No prepared answers yet" banner no longer stands in for these states.
