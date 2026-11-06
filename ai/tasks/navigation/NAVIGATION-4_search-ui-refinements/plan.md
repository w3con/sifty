---
estimate: 12
---
# NAVIGATION-4 — plan

**Author:** coordinator
**Created:** 2026-11-06

## Statement of work

Refine the Search, Sources and Archive screens from the owner's first day of use. In the backend, change the FTS query so groups combine with AND instead of OR: a role matches when it contains any OR-group term AND all AND-group terms; an empty group is no constraint; the backward-compatible flat `titles`/`keywords` (a single OR group) is unchanged; collection and the ASM-4 rescore keep producing fit scores across the catalogue. In the frontend, rebuild Collection→Search as three columns — OR terms, AND terms, exclusion words — each a column of tag chips with a remove "×" and an add field at the top, where every add and remove persists immediately (OR/AND via `PUT /api/search/expressions`, persisting only non-empty groups; exclusions via the existing stop-word add/remove); the draft model, the Save button and the "Unsaved changes" state are removed. Rebuild Collection→Sources as a table with one row per source (name, URL, enabled toggle, remove) plus an add-row form, and remove the "Source coverage" diagnostic panel entirely (a single unobtrusive collection-running line beside Collect may stay). In Archive, hide the Sort control and the Unread button and sort the applied list by application date, newest first, leaving Triage and Preparation unchanged. Verify with `go test -race ./...`, `tsc --noEmit` and `vite build`; the coordinator checks persistence on staging.

## Steps

### Phase 1 — Backend: AND combination

*Gate: `go test -race ./internal/store/... ./internal/background/... ./internal/server/...` passes.*

- [x] 1.1 — Change `store.BuildFTSQueryExprs` to combine groups with AND (each OR group = OR of its terms, each AND group = AND of its terms); skip empty groups so they are no constraint. Update the doc comment to the new semantics.
- [x] 1.2 — Update or add combinator tests for AND-across-groups; update any test that assumed OR-across-groups; add a test that an OR group plus an AND group narrows and that ASM-4 scores are still produced.

### Phase 2 — Search: three columns, immediate save

*Gate: `tsc --noEmit` passes; adding/removing a term in any column persists without a Save button.*

- [x] 2.1 — Replace the group builder with three columns (OR, AND, Exclusions), each showing existing entries as tag chips with a remove "×" and an add field at the top of the column.
- [ ] 2.2 — Make every add and remove persist immediately: OR/AND write via `PUT /api/search/expressions` (send only the non-empty groups); exclusions via the existing stop-word add/remove. Remove the draft state, the Save button and the "Unsaved changes" indicator. Update the help copy to the AND-between-columns meaning.

### Phase 3 — Sources: table, coverage removed

*Gate: `tsc --noEmit` passes; Sources renders as a one-row-per-source table and the coverage panel is gone.*

- [ ] 3.1 — Rebuild the sites list as a table: one row per source with name, URL, an enabled toggle and a remove action, plus an add-row form.
- [ ] 3.2 — Remove the "Source coverage" panel (the `Sources` per-source coverage table) from the screen; keep the Collect button and at most a single line indicating a collection is running.

### Phase 4 — Archive and verify

*Gate: `go test -race ./...`, `tsc --noEmit` and `vite build` all pass.*

- [ ] 4.1 — In the Archive view only, hide the Sort control and the Unread button, and sort the applied list by application date, newest first; leave Triage and Preparation unchanged.
- [ ] 4.2 — Run `go test -race ./...`, `tsc --noEmit` and `vite build`; record output.
- [ ] 4.3 — (Coordinator) On staging, confirm OR/AND/exclusion edits persist immediately, the Sources table edits and persists, and Archive has no Sort/Unread and is date-ordered.

## Acceptance criteria

- Search shows three columns (OR, AND, Exclusions) of removable tag chips with a per-column add field; every add and remove persists immediately; there is no Save button and no "Unsaved changes" state.
- The backend combines OR and AND groups with AND: a role matches any OR term AND all AND terms, minus exclusions; an empty group is no constraint; the flat `titles`/`keywords` config still behaves as before; `go test -race ./...` proves AND narrows and ASM-4 scores are still produced.
- Sources is a table with one row per source (name, URL, enabled toggle, remove) and an add-row form; the "Source coverage" diagnostic panel is gone.
- Archive has no Sort control and no Unread button and is ordered by application date, newest first; Triage and Preparation keep both controls.
- `go test -race ./...`, `tsc --noEmit` and `vite build` pass; persistence is verified on staging.
