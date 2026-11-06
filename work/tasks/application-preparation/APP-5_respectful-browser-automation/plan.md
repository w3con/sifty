---
estimate: 9
---
# APP-5 — plan

**Author:** coordinator
**Created:** 2026-11-03

## Statement of work

The Playwright inspection worker now actively drives real applicant-tracking sites, but runs as bare headless chromium with no realistic fingerprint, no pacing, and no recognition of a block or challenge, so a site's anti-bot layer can ban the owner's IP. This card gives the worker (`sifty/workers/playwright/index.mjs`) a single shared, conservative automation discipline: a realistic browser context (non-headless user-agent seeded from the existing candidate-context string, viewport, locale, `navigator.webdriver` suppressed, standard launch args), a minimum interval with jitter between navigations to the same host, a bounded global concurrency for browser navigations, and detection of block signals (HTTP 403/429 and known challenge/captcha pages) that puts the offending host on a cooldown and ends the operation in an honest backed-off state instead of retrying into a ban. The timing, jitter, concurrency and cooldown values live in `runtime.yaml` with conservative defaults. It does not change the inspection reading logic or honest states from APP-4, does not relax the POST-submission blocking, and does not touch answer generation.

## Steps

### Phase 1 — Look and pace like a human

- [x] 1.1 — Launch the worker's browser context with a realistic fingerprint (user-agent seeded from the candidate-context string, viewport, locale, `navigator.webdriver` suppressed, standard launch args), built in one shared place; check: a UA-echo fixture receives a non-headless user-agent and `navigator.webdriver` reads false.
- [x] 1.2 — Enforce a minimum interval with jitter between navigations to the same host, so repeated inspects on one ATS are spaced; check: two inspects to the same host fixture are at least the configured interval apart.
- [x] 1.3 — Bound the global concurrency of browser navigations explicitly (not left to incidental single-threading); check: a test driving several inspects observes at most the configured number running at once.

### Phase 2 — Back off when a site pushes back

- [x] 2.1 — Detect block signals (HTTP 403/429 and a known challenge/captcha page) and end the operation in an honest backed-off `done` state with a descriptive message, never an immediate retry; check: a fixture returning 429 ends in the documented state and no second request is made within the cooldown.
- [x] 2.2 — Put a blocked host on a cooldown so further inspects to it are deferred until the cooldown lapses; check: a second inspect to a host just blocked is deferred, and one after the cooldown proceeds.

### Phase 3 — Make it tunable

- [x] 3.1 — Read the interval, jitter, concurrency and cooldown from `runtime.yaml` with conservative defaults when unset; check: overriding the interval in config changes the observed spacing, and an absent key uses the default.

## Acceptance criteria

- The worker presents a realistic, non-headless browser fingerprint built from one shared place, with `navigator.webdriver` suppressed.
- Navigations to one host are spaced by a jittered minimum interval, and browser navigations are globally concurrency-bounded.
- A 403, 429 or challenge/captcha response puts that host on cooldown and ends the operation in an honest backed-off state, never an immediate retry.
- The interval, jitter, concurrency and cooldown are configurable in `runtime.yaml` with conservative defaults; the POST-submission blocking and the APP-4 honest states are unchanged.
