---
estimate: 8
---
# GO-3 — plan

**Author:** coordinator
**Created:** 2026-11-04

## Statement of work

The public repository's rules (no card identifiers, Russian prose, tool-authorship trailers, absolute local paths, private names; Conventional Commits subjects) are written in `sifty/scripts/leakcheck/census.md` and implemented by the vendored scanners, but nothing in this private repository runs them: `core.hooksPath` is the root `.githooks`, which has no leak-check hooks, the scanners enforce untidy forms only when the repository's own origin is public, and `discover_card_prefixes()` is stubbed empty. This card makes this repository's own hooks run the scanners so that everything bound for the public repository is held to its rules: a census-declared public subtree (`sifty/`), whose added lines and whose touching commits' messages get the full public rules, while commits touching only private paths get the irreversible checks only. Card-prefix discovery reads `work/tasks/<epic>/<PREFIX-n_slug>` folder names in this repository and returns empty in a standalone public clone. Existing violations in the `sifty/` tree are fixed. The census's list of forms, the knowledge-base and harness pre-commit steps, and application behaviour are unchanged; rewriting existing commit messages is the coordinator's act after this card.

## Steps

### Phase 1 — Scanner: know the cards and the public subtree

- [x] 1.1 — Make `discover_card_prefixes()` read prefixes from `work/tasks/<epic>/<PREFIX-n_slug>` folder names of the repository it runs in, empty when there are none; check: inside this repository it returns at least GO, APP and SET, and in a scratch repository without `work/tasks` it returns an empty set.
- [x] 1.2 — Add a census-declared list of public subtrees (holding `sifty/`) and make `commit-diff-check` apply the full public rules to staged added lines under a public subtree in a private repository, irreversible-only elsewhere; check: in a scratch clone, a staged line naming a card identifier under `sifty/` is refused and the same line under `ai/` passes.
- [x] 1.3 — Make `commit-msg-check` hold the message of a commit whose staged paths touch a public subtree to the full public rules (card identifiers, tool-authorship trailer, Conventional shape), irreversible-only otherwise; check: in a scratch clone, a `sifty/` commit whose message names a card identifier or carries a `Co-Authored-By` trailer is refused, and a private-only commit with the same message passes.
- [x] 1.4 — Make `range-check` apply the same per-commit rule by the paths each pushed commit touches; check: a scratch range containing one offending `sifty/` commit is refused naming that commit, and a range of private-only commits with card identifiers passes.

### Phase 2 — Wire the hooks here

- [x] 2.1 — Add `commit-msg` and `pre-push` hooks to the root `.githooks`, and a `pre-commit.d` step, each delegating to the scanners in `sifty/scripts/leakcheck/`, leaving the existing pre-commit steps intact; check: in a scratch clone of this repository with `core.hooksPath=.githooks`, an offending `sifty/` commit is refused by the hooks and a clean one succeeds, and the same holds inside a `git worktree`.

### Phase 3 — Clean the tree and prove the scanner

- [x] 3.1 — Fix every existing form `tree-check` reports in the `sifty/` subtree under the public rules (the absolute local path in `internal/store/store_test.go` at least) without weakening any rule; check: `tree-check` in public mode over `sifty/` exits 0.
- [x] 3.2 — Run the scanner self-test and the harness self-check; check: `sifty/scripts/leakcheck/leakcheck-selftest` and `bin/work-test` both pass.

## Acceptance criteria

- Committing in this repository runs the leak checks: added lines under `sifty/` and messages of commits touching `sifty/` are held to the full public rules; private-only commits are checked for irreversible forms only.
- Card identifiers are recognised from this repository's own card folders; a standalone public clone sees none.
- `pre-push` re-checks every pushed commit by the same rule, so a `--no-verify` commit cannot leave the machine unchecked.
- `tree-check` over `sifty/` in public mode is clean; the scanner self-test and `bin/work-test` pass; the census's forms and the existing pre-commit steps are unchanged.
