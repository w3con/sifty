# ASSESSMENT-9 — attention

Coordinator: the Claude Code session on the owner's Mac that took over coordination on 2026-11-13. No executor is active in this worktree.

No open items. Known limits after acceptance (not blocking):

- A phrase is matched anywhere in the location's quote, so wording about the company ("We're based in Sofia, Bulgaria", 1131; "Based in Austin, TX but remote-first", 1212) is read as a restriction. Harmless today because those labels do not resolve in the region vocabulary; adding such countries or cities to the vocabulary would make them decide verdicts. Recorded in `docs/specs/quick-match.md` §4.1.
- The built-in phrases include bare "only" and "within" (the plan said "within <region>"), which also match "within one hour of CET" (1313) and commuting-distance wording (1311). Same reason they are harmless today.
- Nine remote vacancies with an explicit restriction stay "?" because `resolveRestrictionLabel` in `internal/quickmatch` does not resolve their labels ("Remote (US only)", "US & Canada", "US and Canada", lists joined by "or", US state lists): 17, 1119, 1153, 1175, 1196, 1201, 1279, 1281, 1294. Candidate for a separate card.
- The older `quickMatch.result` path does not merge model conditions (known since ASSESSMENT-8); only the `fit` path shows this change.
