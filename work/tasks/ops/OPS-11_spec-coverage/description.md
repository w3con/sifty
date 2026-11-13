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

- [ ] **S.1 — Inventory** (30–45 min): table in `inventory.md` of every non-terminal card: area, covering spec, plan state, stale or contradicting statements.
- [ ] **S.2 — Current behaviour per area** (40–60 min, parallel read-only agents): implemented behaviour of the six areas with file:line references, saved in this folder (`area-*.md`).
- [ ] **S.3 — Specs** (90–120 min): `docs/specs/collection.md`, `search.md`, `triage.md`, `application.md`, `settings.md`, `data.md` — implemented behaviour, agreed-but-not-implemented behaviour (with the card), known limits; indexed in `docs/MAP.md`; `bin/map-check` passes.
- [ ] **S.4 — Plans** (20–30 min): every prepared card links its spec; contradictions between a plan and a spec listed for the owner in `log.md`.

## Acceptance criteria

- Each of the six product areas has a spec in `docs/specs/` that distinguishes implemented, agreed-not-implemented and proposed behaviour, with code references for implemented behaviour.
- Every non-terminal card appears in `inventory.md` with its spec or "process — no product spec".
- `docs/MAP.md` lists every spec; `bin/map-check` passes.
- Open contradictions are listed for the owner, not resolved silently.

## Must not touch

Code, configuration, data; `.worktrees/*` (INBOX-2 has a live executor session).

## Estimate

Complexity 5 (six areas, reading across the whole backend and frontend; uncertainty in how much behaviour is undocumented). AI time 3–4 h. No paid services.
