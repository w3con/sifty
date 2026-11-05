# NAVIGATION-2 — Search semantics — OR/AND terms and unified exclusions (backend)

**Epic:** navigation
**Created:** 2026-11-05
**Status:** founded

## Why this work exists

Two things about search are weaker than the owner needs. First, search terms (the `titles` and `keywords` lists in `config/search.yaml`) combine only by OR: today they are flattened into one FTS query used both to collect jobs and, since ASM-4, to rank them. The owner wants to compose terms with OR and AND so a query can require a combination, not only any single term. Second, two moderation mechanisms overlap and confuse: stop-words hide a role by a case-insensitive substring of its title, while block rules hide a role by its company name and, separately, by its source. The owner's model is simpler: excluding a company is the same act as adding an exclusion word, and blocking a source is just not keeping that source in the Sources list. This card delivers the backend for both: an OR/AND term model honoured by collection and by ASM-4 ranking, and a single exclusion mechanism that matches a term against the title or the company, with existing company block rules migrated into it and the separate source-block retired. It ships no UI; NAVIGATION-3 builds the editor on top of it.

## Must not touch

The frontend: this card is Go and schema only, and the editor UI is NAVIGATION-3. The ASM-4 ranking formula and its weights beyond the minimum needed to feed it the new term model so that fit scores keep being produced across the catalogue. The Apply and preparation flows. Config files unrelated to search. The dismissal and Archive behaviour delivered by NAVIGATION-1.

## Notes

Interview answers from the owner (2026-11-05), recorded by the coordinator:

- Customer and daily user: the owner, for his own daily job search.
- Pain and timing: search terms cannot express AND; moderation is split across two mechanisms that do the same job. Raised now as the backend half of the navigation work, which the owner chose to split from the frontend.
- Ideal result: search terms compose with OR and AND; one exclusion list covers both titles and companies; a source is controlled purely by whether it is in the Sources list.
- Rules the solution obeys: the repository work-management harness; backend changes are explicitly allowed for this card; keep tests race-clean.
- Tried before: hand-editing `config/search.yaml` and the two separate moderation UIs are the current state; no prior unification attempt.
- What it touches: `config.Search` and its parsing (`internal/config`), `store.BuildFTSQuery` and its callers in `internal/background/collect.go` and `internal/server/routes.go`, the stop-word matching (`store.TitleMatchesStopWords`, `ListStopWords`), the block-rule storage and migrations, and the source-block code path.

Grounding read by the coordinator: `config.Search` holds `Search.Titles` and `Search.Keywords` as flat `[]string` (`internal/config/config.go`); both are flattened and OR-combined by `store.BuildFTSQuery(append(titles, keywords...))`, called at collection (`collect.go:87`) and at rescore/ranking (`routes.go:500`) — so the term model is shared with ASM-4 and this card must keep ranking working. Stop-words match title only (`collect.go` → `store.TitleMatchesStopWords`). Block rules are a separate table (company-normalised and source kinds), surfaced today in the Settings "Dismissed & blocked" panel.

Open interpretation the owner approved on 2026-11-05: an exclusion term hides a role when it matches the title OR the company name, so the unified mechanism keeps both old behaviours; existing company block rules are migrated into exclusion terms and source-block rules are dropped because the Sources list is now the only source control.
