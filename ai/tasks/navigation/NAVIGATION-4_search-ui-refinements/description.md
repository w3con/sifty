# NAVIGATION-4 — Search, Sources and Archive UI refinements

**Epic:** navigation
**Created:** 2026-11-06
**Status:** founded

## Why this work exists

The navigation restructure shipped the function but the owner's first day of use exposed rough edges that make the daily screens awkward. The Search section still carries a draft/Save model with an "Unsaved changes" state; the owner wants every change to persist immediately with no Save button. The Search layout (an OR/AND group builder with a dropdown and an exclusion list below) is to become three plain columns of deletable tag chips — one for OR terms, one for AND terms, one for exclusion words — each with an add field at the top. The Sources list renders each site as a stacked multi-line list item; it should be a proper table with one row per source. The read-only "Source coverage" diagnostic panel under the sites list is debug noise on a daily screen and comes off entirely. In Archive the sort control and the "Unread" button are meaningless and come off, with the list sorted by application date.

## Must not touch

The ASM-4 ranking formula and its weights, beyond feeding it the new query combination so fit scores keep being produced across the catalogue. The Apply, preparation and dismissal flows, and Triage behaviour. The tab shell (Collection · Triage · Preparation · Archive) and the Preparation rename. The Settings drawer contents (Context, Configuration, Queue) delivered earlier. The stop-word (exclusion) HTTP API and the sites HTTP API contracts. The Sort control and the Unread button in Triage and Preparation — they stay; only Archive loses them.

## Notes

Owner requests and coordinator decisions (2026-11-06), recorded by the coordinator:

- Immediate save: every add and remove in Search (OR terms, AND terms, exclusions) persists at once; no Save button, no "Unsaved changes" state.
- Three columns in Search — OR, AND, Exclusions — each a column of tag chips with a remove "×" and an add field at the top of the column.
- Search combination semantics (owner chose "AND between columns"): a role matches when it contains ANY of the OR terms AND ALL of the AND terms, minus the exclusion words. An empty column is no constraint (not "match nothing"). This requires a backend change: group combination in the FTS query becomes AND across groups instead of OR. The backward-compatible flat `titles`/`keywords` config is a single OR group, so AND across one group is unchanged.
- Sources is a table: one row per source with name, URL, an enabled toggle and a remove action; an add-row form.
- The "Source coverage" diagnostic panel (owner chose "remove entirely") comes off the Sources screen. A single unobtrusive line showing whether a collection is currently running may stay beside the Collect button; the per-source coverage detail does not.
- Archive (owner): no Sort control and no Unread button in the Archive view; the applied list is sorted by application date, newest first. Triage and Preparation keep Sort and Unread.

Grounding read by the coordinator (current state on main):
- Search and Sources UI live in `app/web/Collection.tsx`; the coverage panel is `app/web/Sources.tsx` (the `Sources` component and its per-source `source-table`; `RunProgress` is the collection-status line).
- The OR/AND draft model, the `Save` button and the `Unsaved changes` indicator are in `Collection.tsx`. Expressions persist through `PUT /api/search/expressions` (body `{"expressions":[{"or":[...]},{"and":[...]}]}`; GET returns the bare array). Exclusions use the existing `addStopWord`/`removeStopWord`. Sites use `addSearchSite`/`removeSearchSite`/`toggleSearchSite`.
- The FTS query combination is in `store.BuildFTSQueryExprs` (`internal/store`), fed to collection (`internal/background/collect.go`) and the ASM-4 rescore (`internal/server/routes.go`). It currently combines groups by OR; this card changes that to AND across groups.
- The Archive Sort control and Unread button are in the shared list toolbar in `app/web/main.tsx`; `sortCatalogue` and the `sort`/`unread` state drive them. Applied rows already carry `appliedAt`.
