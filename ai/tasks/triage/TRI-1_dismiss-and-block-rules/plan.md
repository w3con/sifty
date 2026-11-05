---
estimate: 14
---
# TRI-1 — plan

**Author:** coordinator
**Created:** 2026-11-04

## Statement of work

The triage inbox has no way to say "never show me anything from this company or this source". The owner clears such roles one at a time with Dismiss, and they return whenever a new posting from the same company is collected. This work adds durable block rules by company, domain, or source, evaluated at display time and fully reversible: a blocked role is hidden from Triage, never deleted, and reappears the moment its rule is removed. The owner creates a block directly from a role in Triage and manages the list of active blocks, together with recently dismissed roles, in Settings.

Two facts found in the code bound this work. Sticky dismissal already exists at the storage level: `store.Upsert` dedups a re-collected posting to its existing row by canonical URL, apply URL, or company+title identity, and carries the `rejected` status forward, so a dismissed role that dedups does not return to Triage. Per-listing permanent blocking already exists too: `DeleteRole` writes a tombstone to `deleted_listings` that collection skips forever. What is genuinely missing, and all this card builds, is blocking a whole company or source rather than one listing, plus a visible place to review and undo blocks and dismissals.

## Decisions settled with the owner (2026-11-04)

- Scope: block rules by company / domain / source, plus a visible managed list in Settings of active blocks and recently dismissed roles, with unblock and restore-from-dismiss. Dedup, stages, and the assessment tab are out of scope.
- Match key for a company block: always the normalised company name (`normalizeText`). The `domain` rule kind stays in the schema for a future explicit block-domain action, but the Block-company button never derives a domain (host identifies the board, not the employer, for ATS-hosted roles).
- Enforcement: at display, reversible. Blocked roles are stored and hidden, never deleted; removing the rule makes them visible again.
- Placement: the block is created only from a role in Triage; the managed list lives in Settings. The assessment tab is not touched.

## Coordinator's implementation decisions (settled, not for the owner)

- The server computes a `blocked` flag (and a `blockedBy` reason string) on each job in the `GET /api/jobs` payload. The frontend hides blocked roles from the Triage view and the catalogue listing but keeps them in Pre-selected and Archive: a role the owner already advanced is a committed choice and is not retroactively hidden.
- A company block stores a `company` rule holding `normalizeText(company)`; a source block stores a `source` rule holding the source name.
- A role is blocked-by-source only when every one of its current sources is blocked, so a role still reachable through a trusted board stays visible. Company and domain rules match a single role field, so they hide on any match.
- Storage is a `block_rules` table `(id, kind, value, created_at)`, `kind ∈ {company, domain, source}`, unique on `(kind, value)`, added idempotently with a `user_version` bump.
- Restoring a dismissed role reuses the existing stage machinery: Settings lists `stage='no'` roles, newest first and capped, and Undo is the existing `PATCH /api/jobs/{id}` back to `triage`. No new endpoint is added for that.

## Out of scope — do not touch

The dedup and status carry-forward logic in `store.Upsert`; the stage model and its transitions; `DeleteRole` and the `deleted_listings` tombstone; the assessment tab (`RoleContent.tsx`); the collection and ingest path. No change to how dismissal already persists across collection.

## Acceptance criteria

- Blocking a company or a source from a role in Triage immediately hides every matching role from the Triage view and the catalogue listing, and the Triage count reflects the smaller set.
- A role already in Pre-selected or Archive is not hidden by a block that matches it.
- No blocked role is deleted; removing its rule in Settings makes the matching roles reappear in Triage.
- A company block uses the normalised company name; a role is source-blocked only when all of its sources are blocked.
- Settings shows the active block rules with a remove control, and the recently dismissed roles with a restore-to-Triage control.
- Existing dismissal persistence is unchanged: a re-collected dismissed role that dedups to its row stays dismissed.
- `go test -race ./...` and `bin/work-test` pass.

## Checkpoints

**1 — Backend: storage and rule matching**

*Gate: `cd sifty && go test -race ./internal/store/...`*

- [x] 1.1 — Add the `block_rules` table to the base schema (`kind`, `value`, `created_at`, `UNIQUE(kind,value)`) and bump `user_version`; `go build ./...` is green.
- [x] 1.2 — Add store methods `AddBlockRule`, `ListBlockRules`, `RemoveBlockRule`, and a `blockMatch(job)` helper with the domain-else-company and all-sources-blocked semantics; a unit test covers each match case.
- [x] 1.3 — Populate `Blocked` and `BlockedBy` on the job returned by `Get`/`List`; `go test -race ./...` is green.

**2 — Backend: API**

*Gate: `cd sifty && go test -race ./internal/server/...`*

- [x] 2.1 — `POST /api/jobs/{id}/block {dimension:"company"|"source"}` derives the rule value server-side, inserts it, and returns the stored rule; a handler test covers company and source.
- [x] 2.2 — `GET /api/block-rules` and `DELETE /api/block-rules/{id}`; handler tests cover list and removal, including that removal clears the `blocked` flag on affected jobs.

**3 — Frontend: Triage block action and hiding**

*Gate: `cd app && npx tsc --noEmit`*

- [x] 3.1 — Carry `blocked`/`blockedBy` on the `Job` type and add `api` calls for block, list-rules, and remove-rule.
- [x] 3.2 — Hide blocked roles from the Triage view and the catalogue listing, exclude them from the Triage count, and leave Pre-selected/Archive unaffected.
- [x] 3.3 — Add a "Block…" control on the selected role in Triage offering "Block company" and "Block source"; after it runs, the list refreshes and the matching roles are gone.

**4 — Frontend: Settings management**

*Gate: `cd app && npx tsc --noEmit`*

- [x] 4.1 — Add a "Dismissed & blocked" section to Settings listing the active block rules with a remove control that restores the matching roles.
- [x] 4.2 — In the same section, list the recently dismissed roles with a restore-to-Triage control.

**5 — Verify end to end**

*Gate: `bin/work-test`*

- [x] 5.1 — `bin/work-test` passes.
- [x] 5.2 — Build the web bundle and the Go binary and confirm by hand: block a company in Triage so its roles vanish, see the rule and a dismissed role in Settings, remove the rule, and watch the roles return. (Gate: manual, recorded in `log.md`.) [coordinator-verified against the real binary on a copy of the live data]

## Gate evidence

Дословный вывод каждой прогнанной проверки записывается сюда, без сокращений и пересказа.
