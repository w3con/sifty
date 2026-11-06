# APP-4 — Robust form inspection: click through ATS Apply buttons and honest states for unresolved URLs

**Epic:** application-preparation
**Created:** 2026-11-03
**Status:** founded

## Why this work exists

APP-1 wired form inspection into the live chain, but on real jobs it does not yet reach the questions. Two real-world cases, found during the APP-1 end-to-end verification, fail:

- A job on an applicant-tracking system (Ashby, Greenhouse, Lever) hides the application form behind an "Apply" button. The worker reads the landing page as-is and finds zero fields, so a job with a real form (e.g. 458, Catena on Ashby) ends in "no readable questions". Most reputable employers use such systems, so this is the common case, not an edge case.
- A job whose apply URL is unresolved (e.g. 758) falls back to the original listing URL, which may return a download or non-HTML; the inspect then fails with a raw Playwright error (`page.goto: Download is starting`) instead of an honest state.

This card makes inspection actually read ATS forms, and makes unreadable destinations honest states rather than raw failures.

## Must not touch

- The APP-1 inspection wiring, the model-free inspect path, the selection trigger, and the queue-status fix. This card changes the worker's reading behaviour and the inspect outcome handling, not the plumbing around them.
- Answer generation (owned by APP-3) and the knowledge-base answer store (owned by the Settings epic).

## Notes

- The Playwright worker is `sifty/workers/playwright/index.mjs`. It already classifies the ATS family (Ashby / Greenhouse / Lever / generic) but does not click an Apply control or wait for the revealed form.
- The worker's dependencies are not installed by any current setup step; a reliable `sifty/workers/playwright/node_modules` (playwright + its chromium) must exist for inspection to run. Fold the install into this card or a setup step.
- Honest-state handling lives in `sifty/internal/background/preparation.go` (the inspect phase, around the `len(q.Fields) == 0` branch and the inspect-error branch). Today a worker error marks the operation failed; an unreadable or unresolved destination should instead end `done` with a descriptive message, matching the honest no-questions / login-required outcomes APP-1 already produces.
- Verify on the owner's real database: 458 (Ashby) should yield the form's questions after click-through; 758 (unresolved apply URL) should end in an honest "no resolved application form; apply via the original listing" state, not a raw error.
