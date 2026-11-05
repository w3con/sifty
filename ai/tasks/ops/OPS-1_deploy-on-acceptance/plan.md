---
estimate: 22
---
# OPS-1 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

Build a two-environment deploy loop so that accepting a card is safe and visible without ever risking the running production app. Finishing an app-touching card rebuilds the Go binary and the frontend, runs the application's own tests, and deploys automatically to a staging environment on its own port against its own data; production is a separate environment on the current port against persistent data, and it is updated only when the owner confirms, by a promote command that no automatic path ever calls. A deploy is a single env-aware script that backs up the target's data, swaps the binary, restarts the server, runs a health and smoke check, and rolls back to the previous binary and data on any failure. The data directories move out of `/tmp` to the persistent, outside-the-repository location the architecture already assumes. The result: card acceptance always lands on staging where the owner reviews it, production changes only on the owner's word, and the state where a feature is merged but invisible, or where a bad build takes down the live app, cannot happen.

## Settled with the owner (2026-11-05)

- There is a staging environment and a production environment. Card acceptance auto-deploys to staging; production is updated only on the owner's explicit confirmation (a promote step), never automatically.
- Data moves to where it belongs by the architecture: a persistent directory outside the repository and outside `/tmp`, configured via `SIFTY_DATA_DIR`. Proposed paths: production `~/.local/share/sifty`, staging `~/.local/share/sifty-staging`.
- Ports: production 8080, staging 8081 (moving production off the current 4317).
- Remote CI on GitHub is deferred until the repository is active; the local loop is made to work first.
- A bad deploy must not take down the live app: staging absorbs the risk, and production keeps a backup and automatic rollback.

## Coordinator's implementation decisions (settled, not for the owner)

- One env-aware script `bin/sift-deploy --env staging|prod`; per-env port, data dir, dist path, binary path and log path come from `bin/work.config` and the environment, no hardcoded `/tmp`.
- Staging data is refreshed from a production snapshot at the start of each staging deploy, so the owner reviews the change against real-shaped data without touching production.
- Ports: production 8080, staging 8081 (production moves off the current 4317).
- Production is touched only by `bin/sift-promote`, which deploys the current `main` to the production environment; `work-finish` never calls it.

## Must not touch

The application's behaviour (collection, scoring, serving). The coordinator-gated public-repo push (GO-1). Packaging for distribution (GO-6). Production is never deployed by any automatic path.

## Acceptance criteria

- Finishing an app-touching card rebuilds the Go binary and the frontend, runs the app's tests, and deploys to staging automatically; the owner is given the staging URL. A docs-only or harness-only card deploys nothing.
- Production is updated only by `bin/sift-promote` on the owner's confirmation; no automatic path deploys to production.
- `bin/sift-deploy` backs up the target env's data before swapping; on any build, start, or smoke failure it restores the previous binary and the data backup and exits non-zero.
- A post-deploy smoke test confirms the env's server answers on its port and the catalogue is populated.
- Acceptance runs the application's own suites (`go test -race ./...`, `tsc --noEmit`, `vite build`) for an app-touching card, so an app change that breaks a test is rejected before merge.
- Production and staging data live at persistent paths outside the repository and outside `/tmp`; no hardcoded `/tmp` paths remain.
- Running the deploy by hand and via the acceptance hook produce the same result; a re-run with no code change is a safe no-op and every run is logged with the binary hash, timestamp, env, and smoke result.
- The loop is documented and the old manual recipe in the roadmap is replaced by a pointer to it.

## Checkpoints

**1 — The env-aware deploy script**

*Gate: `bin/sift-deploy --help` and a `--build-only` dry run.*

- [x] 1.1 — Write `bin/sift-deploy --env <name>` reading per-env port, data dir, dist, binary and log paths from `bin/work.config` and the environment, with no hardcoded `/tmp`; `--help` documents every key.
- [x] 1.2 — Build phase: `go build ./cmd/sifty` to a fresh binary and `vite build` in `app/` to `app/dist`, aborting on any build error; `--build-only` builds both without touching any server.
- [ ] 1.3 — Swap and start: back up the target env's data (timestamped copy of `store.sqlite`), keep the previous binary, stop that env's server on its port, start the new binary detached against that env's data dir and dist.
- [ ] 1.4 — Health, smoke and rollback: poll the catalogue endpoint until it answers 200 within a timeout, then assert the catalogue count is greater than zero; on failure restore the previous binary and the data backup, restart, and exit non-zero.
- [ ] 1.5 — Post-deploy data step: run the idempotent rescore (local profile-save) so a migration that needs a one-time backfill completes; it is a no-op when scores are current; deploying against a database with no match score ends with scores populated.
- [ ] 1.6 — Idempotent and logged: a second run with no code change ends healthy without disruption, and every run appends the binary hash, timestamp, env, and smoke result to a deploy log.

**2 — Staging and production environments**

*Gate: both environments start and answer on their ports.*

- [x] 2.1 — Define `staging` and `prod` in `bin/work.config`: prod on port 8080, staging on port 8081, each with its persistent data directory at the home chosen by the move decision (see OPS-2 / pending owner confirmation).
- [x] 2.2 — Move the production data from `/tmp/claude-501/sift-data` to the persistent prod path once, and point the running production server at it; the collected corpus and its config survive the move.
- [ ] 2.3 — Staging deploy first refreshes `~/.local/share/sifty-staging` from a production snapshot, so staging always reviews the new build against current real data.

**3 — Wire into acceptance (staging automatic, production on confirmation)**

*Gate: `bin/work-test`; finish a throwaway app card and a docs card.*

- [ ] 3.1 — After a successful fast-forward merge, `work-finish` runs `bin/sift-deploy --env staging` when the merged card touched `sifty/**` or `app/**`, and never for a docs-only or harness-only card; it reports the staging URL.
- [ ] 3.2 — Make the acceptance check test the application: `work-accept` / `WORK_ACCEPT_CMD` runs `go test -race ./...`, `tsc --noEmit`, and `vite build` for an app-touching card; an app card with a failing Go test is rejected.
- [ ] 3.3 — `bin/sift-promote` deploys the current `main` to the production environment with prod backup and rollback; it is the only path that touches production and is run on the owner's confirmation. A staging deploy never promotes.
- [ ] 3.4 — A staging deploy failure fails the finish loudly, after the merge, reporting the rollback state, so a broken deploy is never silent.

**4 — Documentation**

*Gate: the documented commands work from a clean shell.*

- [ ] 4.1 — Document the two-environment loop in `AGENTS.md` and the README (finish auto-deploys to staging, `sift-promote` to production on confirmation, how rollback works) and replace the manual recipe in `ai/ROADMAP.md` with a pointer to it.

## Gate evidence

Дословный вывод каждой прогнанной проверки записывается сюда, без сокращений и пересказа.
