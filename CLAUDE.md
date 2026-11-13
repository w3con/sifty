# Sifty

Local-first job triage application. The Go backend is under `backend/`, the React frontend under `frontend/`, the retired TypeScript implementation under `legacy/app/`, and config templates under `config/`. One build command (`make build`) produces the binary and the frontend bundle. Production runs on port 8080 against `./data`, which is git-ignored. Staging (8081, `./data-staging`) was retired on 2026-11-13: cards are reviewed on preview servers against a fresh copy of production data (see AGENTS.md).

OpenCode is the primary tool. This file is retained for existing installations; use AGENTS.md and its startup reading order for current project rules. Further Claude Code integration is not part of the development plan.

**Path rule:** resolve `PROJECT_ROOT="$(git rev-parse --show-toplevel)"` — never hardcode absolute paths.

## Build and run

```
make build          # build Go binary to bin/sifty + vite build in frontend/
make run            # serve production on :8080 against ./data
```

The Makefile run targets share build artifacts. Production is updated only by the release command on the owner's word; see AGENTS.md.

## Work-management harness

Development work is managed by the `bin/work-*` commands and the card-folder convention. The rules are maintained inline in `AGENTS.md`, which is the cross-tool source of truth read by every tool. This section is a thin twin: do not restate the mechanism here, follow `AGENTS.md`.

`tools/shared` is a git submodule holding the shared agent tooling (`github.com:w3con/ai`); `tools/bin`, `tools/harness` and `tools/templates` are relative symlinks into it. On a fresh clone: `git submodule update --init tools/shared`.

## Go module

Module path: `github.com/w3con/sifty` (unchanged from origin in FCTO repository).

## Writing

Every rule about how to write, in chat and in documents, lives in `~/.claude/STYLE.md` and in no other file. Do not restate any of it here.
