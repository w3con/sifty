# GO-1 — Rewrite the backend to Go and create the public repository

**Epic:** go-migration
**Created:** 2026-11-01
**Status:** founded

## Why this work exists

Sifty becomes a public product, and its backend is rewritten from TypeScript to Go. The two are done together because there is no reason to carry the current knowledge-base coupling into new code: the Go rewrite produces a self-contained application from the start, and that application lives in a fresh public MIT repository rather than inside this private one. The public repository is created at the start of this work, with its leak-protection in place before the first commit, so that Go code lands in the right place instead of being migrated later. The outcome is a public, self-contained Go backend that reaches functional parity with the current Stages 1 and 2, with browser automation kept in a thin TypeScript worker running the official Playwright, which Go lacks a binding for. This delivers the product foundation for every later feature in the backlog (the weighted scorer, model separation, the liveness detector), which are separate cards built on top of this one.

The language decision and its reasoning are recorded in `ai/sifty/backlog.md` (Decisions locked) and `ai/sifty/dev-setup-notes.md` (Decision settled — the product backend language). The commit-cleaning approach to copy is captured in `ai/sifty/dev-setup-notes.md` (the pilier section), which was deferred from the harness card to here.

## Must not touch

The current TypeScript application under `app/` stays working until parity is proven and cutover happens; it is not deleted or broken mid-rewrite. The knowledge base under `kb/`. The private work-management harness under `bin/` and its cards under `ai/tasks/`. The `ai/shared` submodule. The private FCTO repository's own git history: the public repository is a fresh `git init` with no history import, never a filtered or rewritten copy of this one. No commits to `main`; work on a branch.

## Notes

The public repository holds only application code; `ai/shared`, the knowledge base, the harness and all cards stay in this private repository (the pilier two-repository pattern). License is MIT. Private candidate data (the CV and profile PDFs, remembered answers, confirmed feedback) stays local, loaded from a configurable path, and never enters the public repository. The product is self-hosted now with a hosted version expected later, so the split must not bake in a single-machine assumption. This is a large card with five phases; the first phase (the public repository and its leak-protection) is a self-contained slice that can be handed over and accepted before the rewrite phases begin.
