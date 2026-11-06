# COL-6 — Stop-word collection filter

**Epic:** collection
**Created:** 2026-11-05
**Status:** founded (plan.md before execution)

## Why this work exists

The triage inbox fills with whole categories of roles the owner never wants: administrative, sales, and other role types that are wrong by their title alone. Today the only tools are dismissing each role one at a time (TRI-1 block rules hide by company or source, not by role type) and the per-listing permanent Delete. Neither says "I never want any role whose title contains this word, and stop spending collection effort on them." This card adds that: a stop-word is a reversible rule that hides every already-collected role whose title matches it and makes collection skip matching postings from then on. Removing the stop-word brings the hidden roles back and resumes collecting them.

This is the first of two cards. This one builds the stop-word capability with its management placed in Settings for now. The second card restructures navigation (a single Collection tab with Sources and Search sections, a single Archived tab with an Applied/Dismissed toggle, Pre-selected renamed to Selected) and relocates this card's stop-word management into the Search section.

## Must not touch

The `deleted_listings` tombstone and `DeleteRole` (that is the permanent, irreversible delete path; stop-words are reversible and separate). The dedup and status carry-forward in `store.Upsert`. TRI-1 block rules and the `block_rules` table. The stage model, the assessment path, and the collection fetch/source internals (this filters which postings are ingested, it does not change how pages are fetched).

## Notes

Match is a case-insensitive substring of the role title (owner's choice): a stop-word is a literal character sequence such as `risk engineer`, matched anywhere in the title. Before a stop-word is committed, the UI shows how many stored roles it would hide. The collection skip belongs in the collection loop (`internal/background/collect.go`), not inside `Upsert`, so a manually added or enriched role is never silently dropped.
