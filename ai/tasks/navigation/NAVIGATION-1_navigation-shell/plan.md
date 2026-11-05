---
estimate: 11
---
# NAVIGATION-1 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

Replace the top workflow navigation with four tabs — Collection, Triage, Preparation, Archive — rename the preselected stage's label to "Preparation" throughout the UI, and build a Collection tab that holds two sections, Sources and Search. Move the existing Sources list and the "Collect jobs" button from the Settings drawer into Collection→Sources, and move the existing stop-words panel into Collection→Search, reusing the current components and endpoints unchanged. Give Archive an Applied/Dismissed toggle: Applied shows the current applied list, Dismissed lists the already-persisted `status=no` roles via `GET /api/jobs?status=no` with the same undo and delete actions, and Triage dismissal behaviour stays exactly as today. Trim the Settings drawer to Context, Configuration and Queue. This card changes only the React frontend and CSS; it adds no Go code, no schema change and no endpoint. The block-rule and source-block panels stay in the Settings drawer until NAVIGATION-3 removes them.

## Steps

### Phase 1 — Tabs and rename

*Gate: `tsc --noEmit` passes; the app renders the four tabs.*

- [ ] 1.1 — Replace the three-stage nav in `main.tsx` with Collection · Triage · Preparation · Archive, wiring `view` state and per-tab counts.
- [ ] 1.2 — Rename the preselected stage label to "Preparation" everywhere it shows (nav, empty states, review actions, notices).

### Phase 2 — Collection tab

*Gate: `tsc --noEmit` passes; Collection shows both sections and Collect works via the existing endpoint.*

- [ ] 2.1 — Add a Collection view (`view==='collection'`) with Sources and Search sections (a sub-nav or stacked layout).
- [ ] 2.2 — Move the Sources list and "Collect jobs" button out of the Settings drawer into Collection→Sources.
- [ ] 2.3 — Move the stop-words panel out of the Settings drawer into Collection→Search, unchanged, on the existing endpoints.

### Phase 3 — Archive toggle

*Gate: `tsc --noEmit` passes; Archive toggles the two lists.*

- [ ] 3.1 — Add an Applied/Dismissed toggle to the Archive view.
- [ ] 3.2 — Render the Dismissed list from `GET /api/jobs?status=no` with undo and delete actions; leave Triage dismissal behaviour unchanged.

### Phase 4 — Settings trim and build

*Gate: `tsc --noEmit` and `vite build` both pass.*

- [ ] 4.1 — Remove the Sources and stop-words sections from the Settings drawer nav; keep Context, Configuration, Queue, and the block-rule panel (removed in NAVIGATION-3).
- [ ] 4.2 — Run `tsc --noEmit` and `vite build`; record output.

## Acceptance criteria

- The top nav shows exactly Collection, Triage, Preparation, Archive, each with its count, and Settings is the gear button.
- Collection shows a Sources section (source list + Collect) and a Search section (stop-words), both working through endpoints that already existed.
- Archive toggles Applied and Dismissed; Dismissed lists persisted `status=no` roles with undo and delete; a role dismissed in Triage still disappears from Triage after the session.
- The Settings drawer contains only Context, Configuration, Queue, and the as-yet-unmoved block-rule panel.
- No Go, schema or endpoint change is made; `tsc --noEmit` and `vite build` both pass.
