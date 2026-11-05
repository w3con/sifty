---
estimate: 11
---
# OPS-3 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

Correct the two-environment deploy loop so it matches the owner's model: the staging environment on port 8081 shows work in progress (an unmerged branch), production on port 8080 shows only what is merged into main, and the two can run different code at the same time. Today both environments share one compiled binary (`bin/sifty`) and one frontend bundle (`app/dist`), so a staging deploy overwrites what production serves and the two cannot differ; and `bin/work-finish` merges a card to main and then force-deploys staging, so staging shows already-merged code instead of a change under review. This card gives each environment its own binary and its own frontend bundle, makes `bin/sift-deploy --env staging` build and serve whatever checkout it is run from (so a feature branch is previewed on 8081 from that branch's working copy while production keeps serving main on 8080), and removes the automatic staging deploy from `bin/work-finish` so finishing a card only merges to main — production advances to the new main only through `bin/sift-promote` on the owner's confirmation. The application's behaviour is untouched; this is deploy plumbing only.

## Coordinator's implementation decisions (settled)

- Per-environment artifacts: production builds `bin/sifty` + `app/dist`; staging builds `bin/sifty-staging` + `app/dist-staging`. Both pairs are git-ignored build outputs. The names are configurable in `bin/work.config` (`SIFTY_PROD_BINARY`/`SIFTY_PROD_DIST`, `SIFTY_STAGING_BINARY`/`SIFTY_STAGING_DIST`); `SIFTY_DIST_DIR` as a single shared dist is removed.
- Staging deploys the current checkout, not main: `bin/sift-deploy --env staging` builds the working copy it runs in into the staging binary + staging dist and serves them on 8081. Previewing a branch is "make a worktree on that branch, run `bin/sift-deploy --env staging` there"; this is documented, not a new command.
- `bin/work-finish` no longer runs any staging deploy. It keeps the harness contract (fast-forward merge to main). Production is moved to main only by `bin/sift-promote` on the owner's word; `bin/sift-promote` builds main into the production binary + production dist and restarts 8080.
- Both servers run from absolute `--dist` and binary paths so neither depends on a worktree that may be removed.

## Must not touch

The application's behaviour (collection, scoring, serving). The harness one-card-per-conversation gate and the fast-forward-merge-on-finish contract (only the automatic staging deploy is removed). The GO-1 public-repo push. Production is deployed only by `bin/sift-promote`, never automatically.

## Acceptance criteria

- Production (8080) and staging (8081) can serve different code at the same time: with production on main and a different commit built to staging, each port serves its own code, proven by a marker that differs between them.
- `bin/sift-deploy --env staging` run from a feature-branch working copy shows that branch on 8081 and does not change the binary or bundle that production serves on 8080.
- `bin/work-finish` on an app-touching card fast-forward-merges to main and starts/stops no server; staging and production are untouched by finishing.
- `bin/sift-promote` builds the current main into the production binary and bundle and restarts 8080 on main.
- `go test -race ./...`, `tsc --noEmit` and `vite build` pass.
- The corrected loop is documented in `AGENTS.md` and `ai/ROADMAP.md`; no stale "acceptance auto-deploys to staging" wording remains.

## Checkpoints

**1 — Each environment owns its binary and bundle**

*Gate: `bin/sift-deploy --help` shows per-env binary and dist keys; the two builds land in different paths.*

- [x] 1.1 — In `bin/work.config` and `bin/sift-deploy`, give each environment its own binary and its own dist (prod: `bin/sifty` + `app/dist`; staging: `bin/sifty-staging` + `app/dist-staging`); remove the shared `SIFTY_DIST_DIR`; both staging outputs are git-ignored.
- [x] 1.2 — `bin/sift-deploy --env staging` builds the current checkout into the staging binary + staging dist and serves them on 8081, leaving `bin/sifty` and `app/dist` (what prod serves) byte-for-byte unchanged; a check compares their hashes before and after.

**2 — Staging shows the branch; prod stays main**

*Gate: a marker string differs between the two ports; finishing touches no server.*

- [x] 2.1 — Prove isolation: build main to prod (8080) and a working copy carrying a one-line visible marker to staging (8081); a check shows the marker present on 8081 and absent on 8080 at the same time.
- [ ] 2.2 — Remove the automatic staging deploy from `bin/work-finish`; finishing an app-touching throwaway card fast-forward-merges to main and starts/stops no server (prod and staging PIDs unchanged across the finish).
- [ ] 2.3 — `bin/sift-promote` builds the current main into the production binary + production dist and restarts 8080 on main, with the existing backup and rollback; a check shows 8080 serving main after promote.

**3 — Document the corrected loop**

*Gate: the documented commands run from a clean shell.*

- [ ] 3.1 — Update `AGENTS.md` and `ai/ROADMAP.md` to describe the corrected loop (staging = manual preview of an in-progress branch with its own build; finishing only merges to main; production advances only via `bin/sift-promote` on the owner's word); remove the "acceptance auto-deploys to staging" wording.

## Gate evidence

Дословный вывод каждой прогнанной проверки записывается сюда, без сокращений и пересказа.
