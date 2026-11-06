---
estimate: 10
---
# APP-4 — plan

**Author:** coordinator
**Created:** 2026-11-03

## Statement of work

APP-1 wired form inspection into the live chain, but on real jobs the worker does not reach the questions: applicant-tracking systems (Ashby, Greenhouse, Lever) hide the form behind an "Apply" control the worker never clicks, so a job with a real form reads as empty; and a job with an unresolved or non-HTML apply destination fails with a raw Playwright error instead of an honest state. This card makes the Playwright worker (`sifty/workers/playwright/index.mjs`) click through the ATS Apply control and wait for the revealed form before reading it, and makes the inspect phase in `sifty/internal/background/preparation.go` end an unreadable, unresolved, download or non-HTML destination in an honest `done` state with a descriptive message, the same way APP-1 already handles the no-questions and login-required cases. It also ensures the worker's own dependencies are installed so inspection runs at all. It does not touch the inspection plumbing, the selection trigger, the queue fix, or answer generation.

## Steps

### Phase 1 — Reach the ATS form

- [x] 1.1 — In the Playwright worker, when the destination is a known ATS family, click the Apply control and wait for the application form to render before reading controls; check: a recorded Ashby page yields its form fields where the pre-change worker read zero.
- [x] 1.2 — Guard the click-through with a bounded wait and fall through to reading the page as-is when no Apply control is present, so non-ATS forms are unaffected; check: a generic inline-form fixture still reads its fields unchanged.

### Phase 2 — Honest states for unreadable destinations

- [x] 2.1 — In the inspect phase, treat a navigation that returns a download or non-HTML, and a job with no resolved apply URL, as an honest `done` outcome with a descriptive message ("no resolved application form; apply via the original listing"), not a failed operation; check: fixtures for a download destination and an empty apply URL each end `done` with the documented message.

### Phase 3 — Worker dependencies and live verification

- [x] 3.1 — Make the Playwright worker's dependencies (playwright and its chromium) reliably present, via a setup step the clone can run; check: a fresh checkout can install and the worker starts.
- [x] 3.2 — End-to-end on the owner's real database: job 458 (Ashby) yields the form's questions after click-through; job 758 (unresolved apply URL) ends in the honest no-resolved-form state; the catalogue does not crash; check: a recorded run shows 458 questions and 758 honest state.

## Acceptance criteria

- The work is complete when inspecting a job on a known ATS reaches and reads the application form behind its Apply control, and a generic inline form still reads unchanged.
- An unresolved, download or non-HTML apply destination ends in an honest `done` state with a descriptive message, never a raw Playwright failure.
- The Playwright worker's dependencies install via a setup step a fresh clone can run.
- On the owner's real database, job 458 yields its form questions and job 758 ends in the honest no-resolved-form state, without crashing the catalogue.
