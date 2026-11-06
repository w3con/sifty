# APP-5 — Respectful browser automation: avoid getting the inspection worker banned

**Epic:** application-preparation
**Created:** 2026-11-03
**Status:** founded

## Why this work exists

APP-4 turned the Playwright inspection worker from a passive page reader into something that actively drives real applicant-tracking sites (Ashby, Greenhouse, Lever): it clicks the "Apply" control, waits for the SPA to load, and reads the revealed form. The worker does this with bare `chromium.launch({ headless: true })` — no realistic browser fingerprint, no pacing between requests, no recognition of a block or challenge page. The only protections that exist anywhere are unrelated to it: a browser-like User-Agent on a single HTTP fetcher in the candidate-context code, single-concurrency inside the preparation worker, and exponential backoff on assessment-queue failures. None of these shields the browser worker.

The business risk is a ban: an ATS or its anti-bot layer (Cloudflare, 403/429 throttling, a captcha challenge) flags the automated browser, and the host is blocked for the owner's real IP. A ban is expensive and slow to undo, and it defeats the whole inspection chain. This card builds a shared, conservative automation discipline so the worker behaves like a careful human and backs off the moment a site pushes back, instead of retrying into a block.

## Must not touch

- The POST-request blocking in the worker (only data-API paths are allowed; form submission stays blocked). The APP-4 inspection reading logic and its honest states. Answer generation (APP-3) and the knowledge-base answer store (Settings epic).

## Notes

- The worker is `sifty/workers/playwright/index.mjs`; it currently launches plain headless chromium and sets no context options.
- Existing, reusable precedent: `sifty/internal/candidate/context.go` sets a browser-like `User-Agent` on its HTTP fetches. The same string can seed the worker's fingerprint.
- Timing and concurrency values should live in `runtime.yaml` alongside the other operational settings, with conservative defaults, so the owner can tune them without a rebuild.
- "Honest state" here mirrors APP-4: a blocked or throttled host ends the operation in a descriptive `done`/backed-off state and puts that host on cooldown, rather than failing or retrying immediately.
