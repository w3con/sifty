---
estimate: 8
---
# DESIGN-2 — Plan

## Statement of work

Separate layout and styling from application behavior across all ten current frontend TSX files. Keep main.tsx as bootstrap; use App as composition and small screen/section components for presentation. Put API requests, polling, persistence, workflow mutations and substantial event handling into named feature hooks/controllers. Keep formatting and display-only conditions in presentation where they are readable. Presentation must not become a second application controller, and a controller must not become a copy of the current monolith. Format touched source normally, with one declaration/statement per line and multiline JSX. Preserve appearance and behavior.

## Boundaries

Use existing React and CSS, without a framework change or speculative abstraction. Preserve element order, CSS selectors, accessibility labels, component keys/lifetime, unsaved draft retention, modal focus/cancellation behavior, same-gesture popup opening, localStorage keys, collection triggering, polling cadence and the existing ordering guards around reads and writes. No generic form engine, global state store, new automated-test infrastructure or broad schema cleanup. Extract only necessary types at the new presentation/controller boundaries. Static inline presentation styles move to scoped CSS rules without introducing the separate planned token layer; genuinely runtime-dependent styles remain explicit in the view.

## Planned stages, results and cost

All stages are required; no optional process work is included. Shared preparation and acceptance are counted once.

| Stage | Concrete result | Why required | AI elapsed time | Monetary/external cost |
|---|---|---|---|---|
| Preparation | Inventory, scope, preservation constraints and complete card | Cover every TSX and avoid conflicting with existing changes | 10 min | Coordinator usage; exact cost unknown |
| 1 | Bootstrap, composed App, header/tabs/search, catalogue/list/row/import/detail actions, catalogue hooks | Remove the central UI and state monolith | 20–30 min | Existing subscription; exact attributable cost unknown |
| 2 | Separate Collection and Settings views/sections from feature controllers | Make source/search editing and settings layout independently editable | 20–30 min | Existing subscription; no planned paid external calls |
| 3 | Separate RoleContent and application draft/field views from behavior | Make the largest nested views editable while retaining draft and request ordering | 25–40 min | Existing subscription; no paid model generation required for review |
| 4 | Separate dialog behavior, readable Filters/Sources/Icon and extracted static styles; built staging candidate | Complete coverage of every TSX and make the result available for real use | 10–15 min | Existing subscription; local builds only |
| Acceptance | Review boundaries and behavior, inspect staging and collect owner review | Demonstrate usable design editing boundaries and functional parity | 10–25 min | Coordinator usage; exact cost unknown |
| **Total** | **All current TSX covered and result available on staging** | | **95–150 min** | **No new paid service; exact model cost unknown** |

## Checkpoints

- [ ] 1.1 — Preserve local changes and extract the shell, catalogue presentation and application hooks; build frontend.
- [ ] 2.1 — Separate Collection and Settings controllers from readable section views; build frontend.
- [ ] 3.1 — Separate role content, feedback, draft and field-editor behavior from presentation; build frontend.
- [ ] 4.1 — Complete dialogs, filters, sources and icons; extract static styles and publish staging evidence.

Assign one numbered stage per executor session, with coordinator review before the next stage. Return corrections to the same executor. Log checks and preservation evidence, close the assigned checkbox only after its required checks, and commit checkpoints on the work branch. Check elapsed time at each handback and before another stage; stop at 150 minutes with a preserved checkpoint and revised estimate if unfinished.

## Acceptance criteria

- Every original TSX is accounted for: decomposed where it mixes responsibilities, otherwise retained as a readable presentation component. No stale parallel monolith remains.
- main.tsx only imports startup styles/dependencies and mounts the application. App composes named sections rather than embedding complete page layouts and business handlers.
- Presentation modules contain no direct API/fetch calls, browser storage access, polling or asynchronous workflow/save/delete handlers. Their event adapters call named callbacks. Feature hooks/controllers own those operations and their state; small DOM-only presentation behavior is isolated clearly where necessary.
- Header, workflow tabs, search, catalogue toolbar, job list/row, import form, role header/actions, collection sections, settings panels, assessment/feedback sections, draft/field editor and dialogs are discoverable by responsibility. Components are not split merely one per HTML tag.
- Existing selection/filter semantics, stage counts, read marking, keyboard shortcuts, archive behavior, queue toggle, URL import, search autosave, source edits, settings saves, modal lifecycle and application draft revisions/dirty fields remain intact.
- Existing appearance, content, classes/selectors and accessibility are preserved. Static inline styles move to CSS; runtime dimensions/values remain explicit where appropriate. No design-token work is imported from the other worktree.
- All changed/new TSX and support modules use readable multiline source, explicit props and named substantial handlers; no replacement file is another compressed monolith.
- `npm run build` succeeds for each independently completed frontend stage; record output via work-note. Do not run preserved automated suites or add tests. On the final checkpoint `bin/work-test` must report its actual default SKIPPED result, never a passing suite.
- Staging serves the refactored worktree with a verified build/generation. Coordinator inspects available running-app evidence and the diff; owner reviews appearance and behavior on staging before completion/release. If browser tooling is unavailable, state that limitation and leave owner visual review open.
- Demonstrate actual usefulness with a short acceptance record identifying the presentation files for changing the header layout, source table, settings panel and answer editor, and their separate behavior modules, together with staging evidence. Builds alone do not establish adoption or visual parity.
- Main-checkout local changes and all unrelated worktrees remain preserved. Production is unchanged. Integration/release requires the owner's completed-result approval and a separate explicit executor mechanics assignment.

## Review scenarios

Inspect all four navigation stages, catalogue search/filter/selection and keyboard entry points, Collection source/search sections, each Settings panel, selected role description/assessment/application tabs and both dialogs. Review confirmation/deletion/draft mutation paths in code and use only reversible staging interactions during acceptance; do not invoke paid assessment/provider checks or destructive user-data actions simply to demonstrate a structural refactor. Preserve any known broken preparation behavior as an existing limitation rather than claiming end-to-end success.
