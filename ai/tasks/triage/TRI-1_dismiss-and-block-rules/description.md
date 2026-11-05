# TRI-1 — Sticky dismissals and company/source block rules

**Epic:** triage
**Created:** 2026-11-04
**Status:** founded (awaiting owner details; not scheduled)

## Why this work exists

Triage has Dismiss and Select, but a dismissal is not durable against collection. Once collection runs daily (COL-1 plus COL-3), a role that is re-collected or updated comes back into the inbox even though the owner already dismissed it, and there is no way to say "never show me anything from company X" so that such postings stay out of triage entirely. The owner wants dismissals to stick and wants a way to block a company or source.

The owner has not settled the details yet. This card holds the problem and the coordinator's proposed shape so it survives a context reset; it is not scheduled, and its `plan.md` is written only once the owner has decided the open questions below.

## Proposed shape (coordinator's proposal, for discussion)

Two capabilities over one small layer that sits between collection and the triage inbox:

- **Sticky dismissal.** Store a dismissal as a durable decision keyed on the role's stable identity (the canonical-URL dedup key COL-1 already computes), so a re-collected or re-assessed posting with the same identity stays out of the inbox instead of returning.
- **Block rules.** A small owner-managed list of rules (by company name, by domain, by source) that filters matching postings out of triage, addable from a role in triage ("never show this company again") and manageable in settings.

## Open questions for the owner (decide before `plan.md`)

1. When a dismissed role is re-collected with a materially changed posting (new description, new salary, reopened), does it resurface for re-triage with its prior dismissal shown, or stay dismissed silently? (Proposal: resurface only on a material change.)
2. Block at collection (a blocked company's postings are never stored) or block at display (stored but hidden from triage)? (Proposal: hide at display with a managed, reversible list, so a block can be undone and nothing is silently lost.)
3. Match a company block by exact name, by normalised name, or by domain? Company names vary across sources; a domain is more stable but not always present.

## Owner's further thoughts (2026-11-04, to think through before plan.md)

The owner raised these as open directions for triage, not yet decided — they widen this card from "dismiss + block" toward triage filtering generally:

- Show what the owner has already applied to — an "applied" state/track, so a role already acted on is not re-triaged as if new.
- Filters for applications/roles that should NOT be shown (the block/suppress direction), and where those filters live.
- Those suppress filters likely belong in the roles/applications column (the catalogue list), where the owner scans.
- Possibly also surface/add to the filters from within the assessment tab (add "block this" / "hide like this" while looking at a role's assessment).
- The owner's own words: "надо думать" — this needs a design conversation before a plan. Treat the proposed shape above as a starting point, not settled.

This card is the next scheduled work, but it starts with that design pass, not with building.

## Must not touch

- To be defined once scheduled.

## Notes

- Depends on the collection dedup identity from COL-1 (canonical URL) for sticky dismissal.
- Overlaps with COL-3 (collection triggers): block-at-collection, if chosen, would hook the ingest path COL-3 drives.
- Relevant UI surfaces for the filtering thoughts above: `app/web/CatalogueFilters.tsx` (the catalogue filter groups), the roles list in `app/web/main.tsx`, and the assessment tab in `app/web/RoleContent.tsx`.

## Steps

- [ ] To be defined in plan.md.

## Acceptance criteria

- To be defined in plan.md.
