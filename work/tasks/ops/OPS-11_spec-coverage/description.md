# OPS-11 — Every product feature has a behaviour spec; all open plans checked against specs

**Epic:** ops
**Created:** 2026-11-13
**Status:** in progress
**Kind:** task
**Short card:** steps and acceptance criteria live here.

## Why this work exists

Owner, 2026-11-13: «проверь все планы. обнови specs если нужно. каждая фича должна иметь specs.»

Coordinator's own work (AGENTS.md: the coordinator maintains `docs/specs/` and card plans). Read-only agents gather the current behaviour from code; the coordinator writes the specs. No code changes.

Scope decided by the coordinator: product behaviour areas only. Process areas (R-REL, R-HARNESS, R-PLATFORM, R-LEGACY, R-PKG, R-MCP, R-DESIGN) are described by `AGENTS.md`, `work/WORKFLOW.md` and their cards and get no product spec. Product areas without a spec on 2026-11-13: collection (R-COL), search and catalogue (R-SEARCH beyond Quick Match), triage and Apply outcomes (R-TRI), application preparation (R-APP), settings and configuration (R-SET beyond profiles), local data ownership and backup (R-DATA).

## Steps

- [x] **S.1 — Inventory**: `as-built/inventory.md`, 45 non-terminal cards.
- [x] **S.2 — Current behaviour per area**: as-built descriptions in `as-built/` (material, not specs; owner 2026-11-13: a spec is per feature, not per task, button or transaction).
- [x] **S.3 — Feature specs** (owner's rework): `collection.md`, `catalogue.md`, `triage.md`, `application.md` (not designed), `data.md`, updated `vacancy-view.md` and `profiles.md`, per `brief-specs.md`; reviewed by the coordinator against the owner's decisions; questions for the owner collected in one list.
- [x] **S.4 — Into the repository**: an executor moves the approved specs into `docs/specs/`, indexes them in `docs/MAP.md` (replacing the old R-* prose where it duplicates them), links each open card to its spec; `bin/map-check` passes.

## Acceptance criteria

- Each product feature has a spec in `docs/specs/` written as rules of intended behaviour, each rule tagged implemented / not implemented / deviates / owner decision needed.
- Every non-terminal card appears in `inventory.md` with its spec or "process — no product spec".
- `docs/MAP.md` lists every spec; `bin/map-check` passes.
- Open contradictions are listed for the owner, not resolved silently.

## Must not touch

Code, configuration, data; `.worktrees/*` (INBOX-2 has a live executor session).

## Estimate

Complexity 5 (six areas, reading across the whole backend and frontend; uncertainty in how much behaviour is undocumented). AI time 3–4 h. No paid services.
