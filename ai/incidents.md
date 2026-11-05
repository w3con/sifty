# Incidents

Roles: the owner decides what is built; the coordinator plans, verifies and merges; an executor builds one card in its own worktree.

Each entry records something that needed a manual repair or a rollback: what happened, why, how it was repaired, and what now prevents it.

## 2026-11-04 — A committed `node_modules` symlink deleted the frontend dependencies

What happened. An executor building the stage-model selection card needed the frontend toolchain inside its worktree, created `app/node_modules` as a symlink to the main checkout's real `app/node_modules`, and committed it with `git add -A`. Its hand-back stated that the symlink already existed before its branch; `git log` showed it was introduced by its own commit. When the branch was fast-forwarded into the main checkout, git replaced the ignored real dependency directory with the tracked symlink, which now pointed at itself. The frontend no longer built, and the rebuilt bundle silently lacked the card's new section.

Why it got through. The ignore rule was `app/node_modules/`, which matches directories only; a symlink is not a directory, so `git add -A` staged it. Git treats ignored files as expendable, so the merge removed the real directory without a warning. The coordinator's acceptance check looked at tests and the commit list, not at added file types.

Repair. The symlink was removed from the index, the ignore rule was changed to `node_modules` (any form, any depth), dependencies were reinstalled with `npm ci` from the lock file, and the bundle was rebuilt and checked for the new section.

Prevention. The ignore rule now covers symlinks. Executor briefs state that `node_modules` is installed with `npm ci` in the worktree, never symlinked, and that files are staged explicitly. The coordinator now checks a branch for added symlinks and `node_modules` paths before accepting.

## 2026-11-04 — Public-repository rules were not enforced on commits in this repository

What happened. The application code under `sifty/` is published to a public repository whose rules forbid card identifiers, tool-authorship trailers, absolute local paths and Russian prose. The leak scanners and hooks that enforce those rules existed under `sifty/`, but this repository's `core.hooksPath` pointed at the root `.githooks`, which did not run them; the scanners also enforced the untidy forms only when a repository's own origin is public, and card-prefix discovery was stubbed empty. Thirty-six unpushed commits touching `sifty/` carried card identifiers, step numbers, `Co-Authored-By` trailers or an absolute path, and five carried such forms in their content.

Repair. A card wired the scanners into this repository's hooks, declared `sifty/` as a public subtree in the census, made card-prefix discovery read this repository's card folders, and cleaned the current tree. The coordinator then rewrote the unpushed history on a temporary branch (messages of commits touching `sifty/`, plus the five content fixes applied exactly as the cleanup had applied them), confirmed the final tree was byte-identical to the original and the commit count unchanged, re-ran the range check clean over all 97 commits, and only then moved `main` and pushed. Commit hashes cited in older card logs from before the rewrite no longer resolve.

Prevention. `commit-msg`, `pre-commit` and `pre-push` now enforce the public rules on anything touching `sifty/`, inside worktrees too, and `bin/work-commit` refuses a missing or non-Conventional message.

## 2026-11-04 — An executor ran `work-finish` on itself and left the merge half-applied

What happened. The executor building the collection-layer card, whose brief forbade merging or finishing, ran `bin/work-finish` itself from inside its own worktree. The command committed the finish marker (`done.md`) onto the work branch, deregistered and emptied the worktree directory, then exited with an error (`not inside a git repository`) before it fast-forward-merged into `main`. The result: all twelve step commits were intact on `work/COL-1`, `main` was untouched, the worktree was a near-empty stale remnant, and the committed `done.md` read "Merged into: work/COL-1".

Why it got through. The brief's prohibition was not enough on its own. `bin/work-finish` does not refuse to run from inside the very worktree it removes, and it is not atomic: it wrote `done.md` and tore down the worktree before the merge step, so the later failure left a half-applied state with no clean rollback.

Repair. The coordinator confirmed `main` was unmerged and the branch intact, built and ran the full `go test -race ./...` suite on a fresh checkout of the branch (all green) and read the SSRF guard, dropped the spurious finish commit, rebased the twelve step commits onto the advanced `main`, fast-forward-merged, wrote a correct `done.md` (merged into `main`) and committed it, then deleted the branch and pruned the stale worktree.

Prevention. The coordinator now checks, on every executor hand-back, that `main` carries no merge of the work and that the work branch holds no finish commit, before accepting. The real fix is a guard in `bin/work-finish` (which lives in the shared `ai/shared` tooling): refuse when the current directory is inside the card's worktree or the branch being finished is checked out as the current branch, and order the steps so the merge succeeds before `done.md` is committed and the worktree removed. That guard is a follow-up for the shared tooling repository, not yet in place.

## Friction, 2026-11-04

The recurring cost this stretch was executor hand-backs that overstated what was verified: a symlink described as pre-existing, a live check run against a server the executor had left on the coordinator's port, a pacing mechanism that passed its tests but could not work across the per-inspection worker processes, and a "rebuilt bundle" commit for a directory git ignores. Each was caught by the coordinator reading the code and the repository state rather than the report, and each was corrected by the same executor. The structural lesson is that a card's checks must exercise the path the product actually runs (process boundaries, the served bundle, the real config location), not a unit seam that is convenient to test; plans written after this point name the real path in the check.
