# GO-9 — MCP server entry point

**Epic:** go-migration
**Created:** 2026-11-04
**Status:** founded — approach revised (see Decision)

## Decision (2026-11-06, owner)

MCP stays TypeScript; it is not rewritten in Go. It lives as a small standalone TypeScript adapter under `sifty/mcp` that drives the Go backend over its HTTP API (not in-process service calls). It is an independent part of the tree, built and run on its own. Any backend operation an MCP tool needs that the Go HTTP API does not yet expose is added to the Go HTTP API and consumed from `sifty/mcp` over HTTP. This supersedes the "rewrite the MCP server in Go" premise in the section below and the planning question about a Go MCP library and stdio transport.

This card is also where the project's remaining operational/entry-point needs are scoped: the owner does not use a CLI or cron (GO-8 is descoped), so the only shell-driven surface still wanted is whatever MCP requires, and it is handled here as HTTP endpoints plus the `sifty/mcp` adapter. For `app/src` to be deleted in GO-5, the MCP implementation must first be carved out of `app/src/mcp.ts` into `sifty/mcp`.

## Why this work exists

The retired TypeScript backend exposed itself as a Model Context Protocol server over stdio (`app/src/mcp.ts`), offering the services as tools (`jobs_list`, `jobs_detail`, `jobs_search`, `jobs_import`, `jobs_read`, `jobs_assess`, `sources_health`, `candidate_context`, and the application-preparation tools) so an external assistant could drive Sift directly. The Go backend has no MCP entry point of any kind (confirmed in the 2026-11-04 parity audit).

This matters on two counts. The architecture document names the MCP server as one of two alternate entry points (alongside the CLI) over the same orchestration, so it is part of the target the port must reproduce. And the product plan (`ai/research/sift-public-service-research.md`) treats MCP access, how a user reaches Sift from their own assistant, as a feature across the paid tiers, not an incidental. It is also a blocker for deleting `app/src` in GO-5, since the TypeScript `mcp.ts` is the only MCP implementation that exists.

## Must not touch

- The HTTP server and its routes, the background workers, and the services and store methods themselves (the MCP tools reuse them).
- The grounding and safety invariants the HTTP side enforces: no application submission, exact-quote grounding, and the same read/write boundaries per tool.
- The public-subtree rules for `sifty/` (no card identifiers, tool-authorship trailers, absolute local paths or Russian prose in code or in commit messages touching `sifty/`).

## Notes

- Source of the tool set and each tool's shape: `app/src/mcp.ts`.
- Planning questions for `plan.md`: which Go MCP library and transport (stdio to match the TypeScript server), full tool parity versus a first useful subset, and how a tool maps onto an existing service call so no orchestration is duplicated.
- The search/import/read/assess tools depend on collection working, so this is best sequenced after COL-1, or those tools scoped to a clear "collection not yet available" response until then.
- `plan.md` to be written and approved before execution (this card is founded with a description only, like GO-5 and GO-6).

## Steps

- [ ] To be defined in plan.md.

## Acceptance criteria

- To be defined in plan.md.
