# NAVIGATION-1 — Navigation restructure — tabs and sections (frontend)

**Epic:** navigation
**Created:** 2026-11-05
**Status:** founded

## Why this work exists

The navigation grew ad hoc. The top row carries three workflow tabs (Triage, Pre-selected, Archive), while a Settings drawer mixes genuine settings (Context, Configuration, Queue) with collection controls (the Sources list and the Collect button) and moderation (block rules, stop-words, a recently-dismissed list). The owner lives in this tool daily to find work, and the current shape scatters related actions across unrelated places. This card delivers a cleaner shell the owner navigates by intent: one Collection tab for everything about getting jobs in, the decide loop (Triage), a Preparation tab for selected roles, and an Archive that holds both applied and dismissed roles. It is a pure frontend reshuffle on existing data and endpoints; it introduces no new capability. The new search-term editor and the unified exclusion mechanism that will eventually fill the Collection→Search section are deliberately out of this card and land in NAVIGATION-2 (backend) and NAVIGATION-3 (frontend).

## Must not touch

The Go backend, the database schema, and the HTTP endpoints: this card changes only the React frontend and its CSS and uses endpoints that already exist. The ASM-4 assessment ranking and the fit score it produces. The Apply flow and the role review pane logic. The Triage dismissal behaviour: a role dismissed in Triage still disappears from Triage after the session, exactly as today. The existing search-term configuration (titles and keywords) and the block-rule semantics: the block-rule and source-block UI stays where it is for this card and is relocated or removed only in NAVIGATION-3.

## Notes

Interview answers from the owner (2026-11-05), recorded by the coordinator:

- Customer and daily user: the owner, for his own daily job search.
- Pain and timing: navigation is scattered; collection controls and moderation live inside a Settings drawer that should hold only settings. Raised now because the core loop works and the owner wants the tool pleasant to use daily rather than to keep building it.
- Ideal result: four tabs read by intent — Collection, Triage, Preparation, Archive — with a Settings gear holding only Context, Configuration, Queue.
- Rules the solution obeys: frontend only for this card; the repository work-management harness; no backend or schema change.
- Tried before: the current scattered Settings drawer is the workaround in use; no prior restructure attempt.
- What it touches: the top nav, the Settings drawer, the Sources component, the stop-words panel, and the Archive view, all in `app/web`.

Current entry points for grounding: the top nav and the three stages are in `app/web/main.tsx` (the `nav.stages` block and the `view` state); the Settings drawer and its sections (Sources, Context, Configuration, Queue, "Dismissed & blocked") are in `app/web/Settings.tsx`; the Sources list is `app/web/Sources.tsx`; dismissed roles are already persisted and readable via `GET /api/jobs?status=no`.

The rename is Pre-selected → **Preparation** (the owner's word on 2026-11-05). This differs from the earlier roadmap line, which said "Pre-selected → Selected"; the owner's later choice of "Preparation" supersedes it.
