# GO-3 — Public-rules hooks: enforce the public repository's rules on everything bound for it

**Epic:** go-migration
**Created:** 2026-11-04
**Status:** founded

## Why this work exists

The application code under `sifty/` is published to the public repository `github.com/w3con/sifty`. That repository has rules: no card identifiers, no Russian-language prose, no tool-authorship trailers, no absolute local paths, no private-repository addresses or private project names, and Conventional Commits subjects. The rules are written as a census (`sifty/scripts/leakcheck/census.md`) and enforced by scanners (`commit-msg-check`, `commit-diff-check`, `range-check`, `tree-check`) with hooks in `sifty/.githooks/`.

None of it runs. This private repository's `core.hooksPath` is `.githooks` at the root, which holds only the knowledge-base and harness hooks; nothing points at `sifty/.githooks`. And even if it did, two things would still let the forms through: the scanners enforce the untidy forms only when the repository's own `origin` is public, and this repository's origin is the private one; and the vendored `discover_card_prefixes()` is stubbed to return an empty set, so card identifiers are never recognised. As a result executor commits that touch `sifty/` have carried card identifiers and tool-authorship trailers in their messages, and `sifty/internal/store/store_test.go` carries an absolute local path, with nothing refusing them.

The pilier project solves the same split correctly: one shared hooks directory that both its private and its public repository point at, and a `discover_card_prefixes()` that reads the prefixes from the private repository's own card folders. This card brings that arrangement here.

## Must not touch

- The census's list of forms and the irreversible/untidy distinction: this card changes where and to what the rules apply, not what they forbid. The knowledge-base pre-commit step and the harness self-check step in `.githooks/pre-commit.d`. The private repository's own history and card files, which legitimately carry card identifiers and stay private. Application behaviour.

## Notes

- Reference implementation: `~/Dev/pilier/cloud/githooks/` (`commit-msg`, `pre-commit`, `pre-push`) and `~/Dev/pilier/cloud/bin/leak_census.py` (`discover_card_prefixes`, reading `ai/<kind>/<epic>/<PREFIX-n_slug>` folder names). Read them; do not copy pilier-specific paths.
- Here the cards live at `work/tasks/<epic>/<PREFIX-n_slug>/`. Discovery must find the prefixes from there when the scanner runs inside this repository (prefixes in use today include `GO`, `APP`, `SET`), and return an empty set inside a standalone clone of the public repository, which has no cards.
- The public-bound content in this repository is the `sifty/` subtree. Declare it once (a fenced list in the census, e.g. "Public subtrees", holding `sifty/`) rather than hard-coding it, so the rule is data like the rest of the census.
- Rule to implement: in this private repository, added lines under a public subtree, and the message of any commit that touches a public subtree, are held to the full public rules; a commit that touches only private paths is checked for irreversible forms only. In the public repository itself, behaviour is unchanged.
- The scanner's own directory `sifty/scripts/leakcheck/` stays exempt from the untidy forms, as the census already says. The two Russian regular-expression literals in `internal/assess/assess.go` are already exempt as code literals; keep that.
- Hooks must work inside the harness worktrees (`.worktrees/<id>`): `core.hooksPath` is `.githooks`, resolved against each worktree's own root, and the card folders exist in every worktree.
- After this card the coordinator re-checks the unpushed history with the new `range-check` and rewrites offending messages before pushing; that rewrite is the coordinator's act, not part of this card.
