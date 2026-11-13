# GO-8 — CLI command parity with the TypeScript backend

**Epic:** go-migration
**Created:** 2026-11-04
**Status:** descoped (see Decision)

## Decision (2026-11-06, owner)

The owner confirmed he does not operate the backend from a command line or cron, so a general CLI port is not wanted and this card is not planned as written — do not port the fifteen commands. This also removes the CLI as a blocker for deleting `app/src` in GO-5: the deletion no longer waits on a Go CLI. The only shell/entry-point surface the project still needs is whatever the MCP integration requires; that need is owned by the MCP card (GO-9), which exposes any missing operation through the Go HTTP API rather than through a CLI. This card is kept only as a record of what the old TypeScript CLI did, in case a specific command is ever wanted later.

## Why this work exists

The Go binary exposes only `version`, `serve`, `help` and `selfcheck` (`cmd/sifty/main.go`). The retired TypeScript backend's command-line entry point (`app/src/cli.ts`) exposed roughly fifteen commands that were available to operate and script the backend from a terminal or a cron job: `validate`, `context`, `search`, `list`, `detail`, `status`, `import`, `read`, `assess`, `health`, `daily`, `provider-check`, `session`, `backup`, and a launchd template. (The owner has since confirmed he does not use a CLI or cron — see Decision above; this section is retained only as a record of the old command set.) Without them the Go backend is operable only through the HTTP interface. Confirmed in the 2026-11-04 parity audit.

Each command wires an existing Go service or store call to a verb; this is surface, not new orchestration.

## Must not touch

- The `serve` path and the HTTP server, the background workers, and the services and store methods themselves (the CLI reuses them, it does not duplicate their logic).
- The public-subtree rules for `sifty/` (no card identifiers, tool-authorship trailers, absolute local paths or Russian prose in code or in commit messages touching `sifty/`).

## Notes

- Source of the command set: `app/src/cli.ts`. Read each command there for its exact arguments and output.
- A planning question for `plan.md`: which commands are still worth carrying. Some may be vestigial on this target (the launchd template is macOS-specific; `session` may be obsolete under the Playwright-worker model). The plan decides the kept set and justifies each drop, rather than porting all fifteen reflexively.
- Several commands depend on collection actually working (`search`, `import`, `read`, `daily`) — those are only useful once COL-1 lands, so sequence this after COL-1 or scope its collection-touching commands to no-ops-with-a-clear-message until then.
- `plan.md` to be written and approved before execution (this card is founded with a description only, like GO-5 and GO-6).

## Steps

- [ ] To be defined in plan.md.

## Acceptance criteria

- To be defined in plan.md.
