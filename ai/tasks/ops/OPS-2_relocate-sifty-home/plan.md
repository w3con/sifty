---
estimate: 23
---
# OPS-2 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

Extract the Sifty application from the job-search knowledge-base repository and give it its own project home at `/Volumes/SSD/Dev/tknff/sifty`, a standalone git repository that holds the whole app: the Go backend, the frontend, its own copy of the development harness and the Sifty task cards, and its configuration. The app's data (the collected corpus, candidate evidence, remembered answers, feedback) lives under that home in a git-ignored data directory, off `/tmp` and out of version control. There is one documented build command that produces the binary and the frontend bundle, and one place to look: `http://localhost:8080` for production, `http://localhost:8081` for staging. After the move the old knowledge-base repository at `/Volumes/Documents/Work/FCTO` keeps only the job-search material (strategy, market, outreach, the CV and website work) and its documents point at the new Sifty home. The outcome ends the current confusion where the app is buried inside a knowledge base, built by hand into `/tmp`, with no single build and no single URL.

## Settled with the owner (2026-11-05)

- Everything Sifty moves to `/Volumes/SSD/Dev/tknff/sifty` as a standalone project with its own repository. No part is left behind in the knowledge-base repo, and nothing is deferred to "decide later".
- One build command and one place to look (production `http://localhost:8080`, staging `http://localhost:8081`); the data lives under the new home, persistent and git-ignored.

## Coordinator's implementation decisions (settled, not for the owner)

- The new repository keeps the Go module path `github.com/w3con/sifty` (unchanged) and matches the public-repo intent of GO-1; whether history is carried with `git filter-repo` or started fresh from a snapshot is chosen at execution by whichever is clean.
- The Sifty task epics move with the app (assessment, collection, go-migration, triage, application-preparation, settings, ops, sifty-dev-setup); the knowledge-base epics stay in FCTO (the cv and website work).
- The development harness moves as the app's own: the `ai/shared` submodule, the `bin/` symlinks, `AGENTS.md`, `.githooks`, and a thin project `CLAUDE.md`.
- Data lives at `<home>/data` (production) and `<home>/data-staging` (staging), both git-ignored; the corpus is moved once out of `/tmp`.
- The new repository's first remote push stays gated to the owner (GO-1); this card does not push.

## Must not touch

The job-search knowledge-base content (`kb/`, the CV and website cards). The application's behaviour. The GO-1 public-push gate (no push without the owner's word).

## Acceptance criteria

- `/Volumes/SSD/Dev/tknff/sifty` is a standalone git repository that builds and runs the whole application from one documented command, from a clean shell.
- Production runs on port 8080 and staging on 8081; opening `http://localhost:8080` shows the app against the moved data, scored.
- The data lives under the new home, is persistent across a reboot, and is git-ignored; nothing depends on the old `/tmp` binary or data any more.
- The FCTO repository no longer contains the application, its config, its data, or its task cards; it keeps the job-search knowledge base intact, and its `CLAUDE.md` and `README.md` point at the new Sifty home.
- `go test -race ./...` and the frontend build pass from the new home.

## Checkpoints

**1 — Park ASM-4 first**

*Gate: `git -C /Volumes/Documents/Work/FCTO log --oneline -1` shows ASM-4 folded in; no stray worktree.*

- [x] 1.1 — Fold the verified ASM-4 branch into the current line so none of its work is lost in the move (code only, no deploy), and remove its worktree; the fast-ranking code is present on the branch that will be moved.

**2 — Stand up the new home**

*Gate: `git -C /Volumes/SSD/Dev/tknff/sifty status` is clean and `go build ./...` is green there.*

- [x] 2.1 — Create `/Volumes/SSD/Dev/tknff/sifty` as a new git repository and move the application into it: `sifty/` (Go), `app/` (frontend), the config templates, and the app's prompts/site assets; the Go module path stays `github.com/w3con/sifty`.
- [x] 2.2 — Carry the development harness as the app's own: the `ai/shared` submodule, the `bin/` symlinks, `AGENTS.md`, `.githooks`, a thin project `CLAUDE.md`, and the Sifty task epics; `bin/work-test` runs from the new home.
- [x] 2.3 — One build command: a top-level `Makefile`/`bin/build` that produces the Go binary and runs `vite build`; `go test -race ./...` and the build both pass from the new home.

**3 — Data home**

*Gate: the server starts from the new home against the moved data and lists roles.*

- [x] 3.1 — Create `<home>/data` and move the corpus out of `/tmp` into it once; git-ignore `data` and `data-staging`; the config and candidate evidence come with it.
- [x] 3.2 — Run the app from the new home on port 8080 against `<home>/data`; `http://localhost:8080` serves the catalogue.

**4 — Clean the knowledge-base repo**

*Gate: `git -C /Volumes/Documents/Work/FCTO status` clean; the app is gone, the KB intact.*

- [x] 4.1 — Remove the moved trees and the Sifty task cards from FCTO; keep `kb/` and the CV/website work untouched.
- [x] 4.2 — Update FCTO's `CLAUDE.md` and `README.md` to describe it as the job-search knowledge base and point at the new Sifty home.

**5 — Verify end to end**

*Gate: manual, recorded in `log.md`.*

- [x] 5.1 — From a clean shell: one build command builds the app in the new home, production serves on 8080 and staging on 8081, the data is persistent and git-ignored, and nothing references the old `/tmp` setup.

## Gate evidence

Дословный вывод каждой прогнанной проверки записывается сюда, без сокращений и пересказа.
