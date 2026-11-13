# OPS-2 — Relocate Sifty to its own project home

**Epic:** ops
**Created:** 2026-11-05
**Status:** founded (plan.md written; awaiting "Execute the plan")

## Why this work exists

The Sifty application is buried inside the job-search knowledge-base repository, built by hand into `/tmp`, with no single build command and no single URL to open. The owner wants it to be a normal standalone project in its own home, `/Volumes/SSD/Dev/tknff/sifty`, alongside his other projects: everything Sifty in one place, one way to build it, one place to look. This card makes that move, so the confusion of "the app lives inside a knowledge base, nobody knows how to build it or where to see it" ends.

## Must not touch

The job-search knowledge-base content (`kb/`, the CV and website cards) stays in the FCTO repository untouched. The application's behaviour does not change. The first push of the new repository stays gated to the owner (GO-1); this card creates the local repository but does not push it.

## Notes

The FCTO repository currently mixes three things: the Sifty app (`sifty/` Go backend, `app/` frontend, config and data directories), the job-search knowledge base (`kb/`, CV and website work), and the shared development harness (`tools/shared` submodule, `bin/` symlinks). The move separates the first into its own home and leaves the second in place; the harness moves with the app as its own copy. The Go module path is already `github.com/w3con/sifty`, matching the public-repo plan in GO-1.

Depends on ASM-4 being folded in first (its fast-ranking code is verified but sits on an open worktree inside FCTO). The deploy loop OPS-1 runs after this move and targets the new home.
