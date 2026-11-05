# NAVIGATION-3 — Search editor — OR/AND builder and exclusion list (frontend)

**Epic:** navigation
**Created:** 2026-11-05
**Status:** founded

## Why this work exists

Once the backend carries an OR/AND term model and a single exclusion mechanism (NAVIGATION-2), the owner needs to edit both from inside the app instead of hand-editing YAML or using two separate moderation panels. This card builds the Collection→Search interface: an editor that composes search terms with OR and AND, and one exclusion-words list that also covers companies. It then removes the now-obsolete moderation UI — the company block rules, the source-block control, and the standalone stop-words panel — because their jobs have moved: dismissed roles live in Archive (NAVIGATION-1), exclusions live in Search, and source control is the Sources list. After this card the Settings drawer holds only Context, Configuration and Queue.

## Must not touch

The backend term model and exclusion semantics delivered by NAVIGATION-2: this card consumes those endpoints and does not re-implement the logic. The ASM-4 ranking. The Apply and preparation flows. The tab shell, the Preparation rename and the Archive Applied/Dismissed toggle delivered by NAVIGATION-1. The Context, Configuration and Queue sections of the Settings drawer.

## Notes

Interview answers from the owner (2026-11-05), recorded by the coordinator:

- Customer and daily user: the owner, for his own daily job search.
- Pain and timing: search terms and exclusions can only be changed by editing YAML or through two overlapping panels; the owner wants to edit them where collection lives. Raised now as the frontend half of the search work, depending on NAVIGATION-2.
- Ideal result: in Collection→Search, add and remove search terms and group them by OR and AND, and keep one list of exclusion words that hides by title or company; the old block-rule and stop-word UIs are gone.
- Rules the solution obeys: the repository work-management harness; depends on NAVIGATION-2's endpoints; frontend only.
- Tried before: hand-editing `config/search.yaml`; the separate stop-words and block-rule panels in Settings.
- What it touches: the Collection→Search section and the Settings drawer in `app/web` (`main.tsx`, `Settings.tsx`, the API client), building on the Search section stub placed by NAVIGATION-1.

The owner's musing about selecting an industry or a title category is deliberately not in scope: source taxonomies differ, so a single cross-source category picker cannot be guessed reliably. It is left as a possible later refinement, not built here.

Dependency: this card cannot be handed to an executor until NAVIGATION-2 is accepted and merged, because it consumes NAVIGATION-2's endpoints.
