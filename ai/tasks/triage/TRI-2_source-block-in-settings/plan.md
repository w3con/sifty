---
estimate: 6
---
# TRI-2 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

TRI-1 put two actions on the per-role "Block…" control in Triage: block the company, and block the source. The owner wants blocking a source to be a Settings action, not a per-role one, because a source is blocked once and for all rather than from whichever role happens to be open. This work removes "Block source" from the Triage control, leaving only "Block company" as a direct action, and adds to Settings the ability to create a source block: Settings already lists active block rules with a remove control (from TRI-1) but has no way to create one, so without this change removing the Triage button would leave no way to block a source at all. Settings gains a list of the sources present in the collected roles and a control to block one; the existing list and remove controls handle the rest.

## Decisions settled with the owner (2026-11-05)

- The per-role control in Triage blocks only the company. Blocking a source is done in Settings.

## Coordinator's implementation decisions (settled, not for the owner)

- The Triage "Block…" control collapses to a single "Block company" action (no submenu), calling the existing `POST /api/jobs/{id}/block {dimension:"company"}`. The `source` dimension stays supported by that endpoint but is no longer offered from Triage.
- Creating a source block from Settings is not tied to a job, so a new `POST /api/block-rules {kind,value}` inserts a rule directly via the existing `AddBlockRule`; it is scoped to `kind:"source"` for this card. `GET /api/block-rules` and `DELETE /api/block-rules/{id}` already exist and are reused.
- A new `GET /api/sources` returns the distinct source names present in the stored roles, each with its role count, so Settings can offer a pick list rather than a free-text field. A source already blocked is marked so in the list.
- No change to how a source block is matched (TRI-1: a role is source-blocked only when every one of its sources is blocked).

## Out of scope — do not touch

The TRI-1 block-matching semantics in `blocks.go` (company by normalised name, source only when all sources blocked). The stop-word work (COL-6). The dedup, stage, assessment and collection paths. The navigation restructure (separate card).

## Acceptance criteria

- The Triage per-role control offers only "Block company"; there is no "Block source" option in Triage.
- Settings shows the sources present in the collected roles and lets the owner block one; the blocked source's roles then disappear from Triage exactly as a TRI-1 source block does.
- A source blocked from Settings appears in the active block-rule list and can be removed there, which restores its roles.
- Blocking a company from Triage still works.
- `go test -race ./...` and `bin/work-test` pass.

## Checkpoints

**1 — Backend: sources list and direct block creation**

*Gate: `cd sifty && go test -race ./internal/server/...`*

- [x] 1.1 — `GET /api/sources` returns the distinct source names in the store with a role count each and a blocked flag; a handler test covers it.
- [x] 1.2 — `POST /api/block-rules {kind,value}` inserts a rule via `AddBlockRule` (scoped to `kind:"source"`), returns the stored rule; a handler test covers creation and that a matching role is then hidden from Triage.

**2 — Frontend: Triage control and Settings creation**

*Gate: `cd app && npx tsc --noEmit`*

- [x] 2.1 — Collapse the Triage "Block…" control to a single "Block company" action, removing the "Block source" option.
- [x] 2.2 — Add `api` calls for `GET /api/sources` and the direct `POST /api/block-rules`; in Settings, list the sources with their counts and a block control, wired into the existing block-rule list and remove.

**3 — Verify end to end**

*Gate: `bin/work-test`*

- [x] 3.1 — `bin/work-test` passes.
- [x] 3.2 — Build the web bundle and the Go binary and confirm by hand against a copy of the live data: Triage offers only "Block company"; blocking a source from Settings hides its roles from Triage; removing it there restores them. (Gate: manual, recorded in `log.md`.)

## Gate evidence

Дословный вывод каждой прогнанной проверки записывается сюда, без сокращений и пересказа.
