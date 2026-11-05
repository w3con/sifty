---
estimate: 12
---
# NAVIGATION-3 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

In the Collection→Search section placed by NAVIGATION-1, build an editor for the OR/AND search expressions — add and remove terms, group them by OR and AND — wired to the NAVIGATION-2 endpoints with validation and save feedback. Replace the standalone stop-words panel with a single exclusion-words list where one term hides a role by title or company, wired to the NAVIGATION-2 exclusion endpoint. In the Collection→Sources section, turn the current read-only coverage view into an editable sources list: add a site, remove a site and toggle its enabled flag through the NAVIGATION-2 sites endpoint, keeping the coverage display alongside the controls. Remove the now-obsolete moderation UI: the company block-rule list, the source-block control, and the "Dismissed & blocked" Settings panel, leaving the Settings drawer with only Context, Configuration and Queue and no source controls anywhere in it. Remove the always-on bottom status footer: fold its queue counts and its "Assess vacancies" toggle into the Settings→Queue section so the queue lives in one place. Verify with `tsc --noEmit` and `vite build`, then a manual check on staging that terms, exclusions and sources edit and persist. This card is frontend only and depends on NAVIGATION-2 being merged.

## Steps

### Phase 1 — Search expression builder

*Gate: `tsc --noEmit` passes; the builder reads and shows current expressions.*

- [x] 1.1 — Build the OR/AND term editor in Collection→Search, reading the current expressions from the NAVIGATION-2 endpoint.
- [x] 1.2 — Wire add, remove and OR/AND grouping edits to the endpoint, with validation and a saved/unsaved indicator.

### Phase 2 — Unified exclusions and cleanup

*Gate: `tsc --noEmit` passes; exclusions edit; old panels are gone.*

- [ ] 2.1 — Replace the stop-words panel with a unified exclusion-words list (add/remove) wired to the exclusion endpoint.
- [ ] 2.2 — In Collection→Sources, add controls to add, remove and enable/disable a source site, wired to the NAVIGATION-2 sites endpoint, keeping the coverage display.
- [ ] 2.3 — Remove the company block-rule list and the source-block control, and remove the "Dismissed & blocked" panel from the Settings drawer; confirm no source control remains in Settings.
- [ ] 2.4 — Remove the always-on bottom status footer; move its queue counts and the "Assess vacancies" toggle into the Settings→Queue section.

### Phase 3 — Verify

*Gate: `tsc --noEmit` and `vite build` pass; staging check done.*

- [ ] 3.1 — Run `tsc --noEmit` and `vite build`; record output.
- [ ] 3.2 — On staging, confirm search terms, exclusion words and source sites can be added, removed and persist across a reload.

## Acceptance criteria

- Collection→Search edits OR/AND search expressions and a single exclusion-words list; both persist through the NAVIGATION-2 backend.
- Collection→Sources can add, remove and enable/disable a source site through the NAVIGATION-2 sites endpoint, with the coverage display retained.
- One exclusion term hides a role by title or company; there is no separate company-block or stop-word UI.
- The company block-rule list, the source-block control and the "Dismissed & blocked" Settings panel are gone; the Settings drawer holds only Context, Configuration and Queue, with no source control anywhere in it.
- The always-on bottom status footer is gone; its queue counts and "Assess vacancies" toggle live in Settings→Queue, which is the only place the queue appears.
- `tsc --noEmit` and `vite build` pass; editing of terms, exclusions and sources is verified on staging.
