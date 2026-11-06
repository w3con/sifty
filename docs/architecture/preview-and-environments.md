# Preview and environment runtime

This document records the local preview supervisor and the isolated production
and staging runtime. It describes commands that exist in the repository, not a
target design.

## Continuous preview

`bin/sift-preview` follows one active card worktree continuously:

```sh
bin/sift-preview start --worktree .worktrees/OPS-4 --label OPS-4
bin/sift-preview status
bin/sift-preview status --json
bin/sift-preview stop
bin/sift-preview once            # bounded build/health check, no watch
```

Defaults come from `bin/work.config` or the environment:

| Setting | Default |
| --- | --- |
| `SIFTY_PREVIEW_PORT` | staging port (`8081`) |
| `SIFTY_PREVIEW_DATA_DIR` | staging data directory (`data-staging`) |
| `SIFTY_PREVIEW_STATE_DIR` | `<main checkout>/.local/preview` |
| `SIFTY_PREVIEW_POLL_SECONDS` | `1.0` |
| `SIFTY_PREVIEW_DEBOUNCE_SECONDS` | `1.0` |
| `SIFTY_WORKERS_SOURCE` | `<checkout>/backend/workers` |
| `SIFTY_WORKERS_CACHE_DIR` | `<main>/.local/cache/workers` |
| `SIFTY_FRONTEND_CACHE_DIR` | `<main>/.local/cache/frontend` |

The supervisor scans `frontend/`, `backend/` and the workers source. It reads
files from disk, so uncommitted and untracked sources are included. It excludes
`.git`, `node_modules`, generated `dist*`, build output, caches, logs, runtime
data and work records. A stable scan does not build or restart anything. After a
change it waits for the debounce interval and rescans; a source change during a
build discards that candidate instead of publishing a mixed generation.

Each attempt builds immutable generation directories under
`<state>/generations/`. Frontend dependencies are installed with `npm ci` only
when `frontend/package-lock.json` (or `package.json`) changes. Worker
dependencies use a cache keyed by each worker's lockfile and are linked read-only
into the generation. A frontend-only change swaps the served bundle; a worker
change updates the published worker tree; a backend change stops and restarts
the backend. If any build or restart fails, the previous generation and its
server continue serving, and the failure is recorded in `status.json` and shown
in the preview banner.

The preview proxy listens on the preview port and forwards to a private
loopback port. It injects `/__preview/client.js` into HTML responses only in the
preview path, serves `/__preview/status`, and never changes the production
bundle or server code. The banner shows the requested commit/dirty fingerprint,
the served generation, whether an update is available and the latest build
error. It offers a Reload button; it does not silently reload a page with
unsaved inputs.

### Ownership and stop

`<state>/lock` is an OS file lock held by the supervisor. A second
`sift-preview start` is refused with the existing owner's worktree, pid and
token; it never replaces a running preview. `sift-preview stop` only signals a
process whose recorded owner token, pid, process start time and command line
match a `sift-preview` supervisor. A busy preview port is an error, never a
reason to kill whatever holds it.

### Boundaries

- Preview never refreshes data. It uses the staging data directory as-is, so a
  code update does not overwrite staging config, answers, evidence or feedback.
- Preview writes only under its ignored state directory, the ignored dependency
  caches and the selected staging data directory's normal runtime files.
- No `work-*` command starts or stops a preview. `work-capture` cannot trigger
  one. `WORK_PREVIEW_CMD` remains an unused adapter seam; the generic lifecycle
  commands do not embed sifty policy.

## Isolated environments

Production and staging own separate binaries, frontend bundles, worker trees,
data directories and server pid files. New defaults are `frontend/dist` +
`bin/sifty` + `backend/workers-prod` for production and `frontend/dist-staging`
+ `bin/sifty-staging` + `backend/workers-staging` for staging. Relative
destinations resolve against the main checkout, never a linked worktree, so
artifacts built from `.worktrees/<id>` survive that worktree's removal.
`bin/sift-deploy` resolves symlinks and refuses overlapping production/staging
paths, including after a refresh `--from` override.

The current live layout predates these defaults: production runs the old binary
on 8080 with `legacy/app/dist` and the shared `backend/workers`, and the live
preview owns 8081 with `data-staging`. Migration is explicit. The new commands
refuse a busy port, an unverified pid, or a live preview on the same port/data;
they do not kill an unknown occupant. For the first production transition,
`bin/sift-promote --source <sha> --approve <sha> --adopt-legacy` is the one
explicit adoption operation: it requires a single listener, verifies that the
process runs the configured production binary and data directory, reads the
actual `--dist` and `--workers-dir` from that process, captures those exact
assets plus a consistent SQLite snapshot for rollback, and records its pid.
It refuses an ambiguous owner, a different binary, or a missing bundle/worker
tree, and refuses to adopt again after a completed migration. No live adoption
is performed merely by implementing or documenting it.

## Approved-source release

`bin/work-finish <id> --approve <source-sha>` is the normal completion path.
`WORK_RELEASE_CMD` (this repository: `python3 bin/sift-promote`) is read from
the target checkout, with the approved source worktree as fallback, so the card
that first introduces the adapter can release itself. The adapter runs in the
source worktree with `WORK_SOURCE_SHA`, `WORK_SOURCE_TREE`, `WORK_SQUASH_SHA`,
`WORK_TARGET_SHA`, `WORK_CARD_ID`, `WORK_WORKTREE` and `WORK_APPROVAL` in its
environment. `work-finish` prepares the one squash commit without moving a ref,
runs the adapter against the immutable source revision, re-verifies that the
source and target did not move, and only then fast-forwards main to the prepared
commit. The commit tree is the checked source tree plus `done.md`; no separate
finish commit is written. A failed adapter leaves main, `done.md`, the source branch/worktree and
recovery untouched. If the adapter already succeeded and a later check or
publish fails, `work-finish` sends the same adapter `WORK_RELEASE_RECOVERY=1`
with the reason; `sift-promote` then verifies the release receipt keyed by this finish's
source, squash, approval and build, checks the currently served build, restores
that pinned recovery point and prints a `RECOVERY result=` receipt. Missing,
stale or mismatched receipts are refused rather than rolling back a later
release. The refusal
reports the actual main revision and never claims it was unchanged. A verified success records
`refs/work-recovery/<id>` before removing the checkpoint branch.

`bin/sift-promote --source <sha> --approve <sha>` resolves both arguments to the
same commit, then runs `bin/sift-deploy --env prod --source <sha>`. A legacy
rollback restarts the captured process's exact original argv and accepts
`/api/health` or `/`; it does not add `--port`/`--public-port` or require
`/api/version`, which the pre-isolation binary lacks. The deploy
extracts that committed tree with `git archive` into a private directory, builds
and publishes it, verifies the served build identity and health, and records
`sourceSha`/`buildId` in deploy state. `--check` resolves and prints the plan
without touching production. `--force` is accepted only for compatibility and
never bypasses the source/approval match.

`bin/sift-data-refresh` is the explicit staging data operation. It stops the
recorded staging server, preserves the old staging data under
`.refresh-backup/<timestamp>/`, takes a consistent SQLite backup of production,
and mirrors the production payload: `config/`, `config-backups/`, `kb/`,
`evidence/`, `answers/`, `feedback/` and any other root-relative candidate file
or directory. It removes stale WAL/SHM files, refuses a live preview on staging,
and fails loudly if copying or recovery fails. A normal deploy never runs this
operation; `bin/sift-deploy --env staging --refresh-data` is the explicit
combined code-and-data operation.

All builds go to fresh candidate directories under `.local/deploy/<env>/`.
One per-environment operation lock serializes deploy, build-only and refresh
mutations. `--build-only` publishes only when the target port is free; it
refuses to overwrite a running environment's artifacts and reports failure
instead of silently leaving stale or partially replaced files. A normal deploy
copies the current binary/dist/workers before the stop, then takes the final
SQLite snapshot only after the server is verifiably stopped and before
publication. It verifies the serving process through `GET /api/version` against
the build identity, uses `GET /api/health` rather than requiring a non-empty
catalogue, and performs no `POST /api/context` write. Rollback stops the
candidate, restores the binary, bundle, worker tree and a consistent database,
removes stale WAL/SHM files, and restarts the previous server. If the target
cannot be stopped, or artifact/data restore is incomplete, it reports
`ROLLBACK_INCOMPLETE` or `REFRESH_INCOMPLETE`, preserves the backups and does
not restart an incoherent mixture.
