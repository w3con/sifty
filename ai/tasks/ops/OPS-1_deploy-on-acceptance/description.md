# OPS-1 — Auto-build and deploy on card acceptance

**Epic:** ops
**Created:** 2026-11-05
**Status:** founded (plan.md written; awaiting "Execute the plan")

## Why this work exists

Accepting a card today (`work-finish`) fast-forward-merges its branch into main and nothing else: no build, no deploy. The running application is a Go binary built by hand into `/tmp/claude-501/sifty-main`, serving a frontend bundle built by hand into `app/dist`, launched by hand on port 4317 against a data directory in `/tmp`. The only record of how to rebuild and restart it is a prose recipe in the roadmap. So "accepted" does not mean "running": accepted work stays invisible until a human remembers to run a multi-step manual rebuild and restart. For card ASM-4 that step was never run, so the owner opened the app, saw 2452 roles with no score, and concluded the tool was broken, when in fact the feature was built, tested, and verified working on a copy of the live data but simply not deployed anywhere he could see. This card closes that gap: after a card is accepted, the application is rebuilt and relaunched automatically, so acceptance means the change is live and visible on the same URL.

## Must not touch

The application's own behaviour: this card is build and deploy plumbing only, it changes nothing about how the app collects, scores, or serves. The first push of the public repository stays coordinator-gated as decided for GO-1. Packaging the app for other people to install (GO-6) stays out of scope and deferred; this card automates the owner's own local run loop, not a distributable package.

## Notes

The current manual run recipe lives in `ai/ROADMAP.md` (build the Go binary from `sifty/`, `vite build` from `app/`, swap `/tmp/claude-501/sifty-main`, relaunch on 4317 against `/tmp/claude-501/sift-data` with `--dist .../app/dist`). This card turns that recipe into a script and wires it into acceptance.

ASM-4 is the concrete case that exposed this gap. It is verified working (a rescore over a copy of the live data produced match scores across the full 0–100 range, correctly ranked) but intentionally left undeployed and unmerged until this pipeline exists; it is the pipeline's first real deploy. The acceptance check that `work-finish` runs today is the harness self-test, not the application's own test suite, so a broken app change can pass acceptance; fixing that is part of this card.
