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
  one. Phase 6.3 will connect an approved work handover to
  `sift-preview start --worktree <path> --label <id>` through a configurable
  adapter (`WORK_PREVIEW_CMD`), not by embedding sifty policy in the generic
  lifecycle commands.

## Isolated environments

Production and staging own separate binaries, frontend bundles, worker trees,
data directories and server pid files. Relative destinations in
`bin/work.config` resolve against the main checkout, never a linked worktree,
so artifacts built from `.worktrees/<id>` survive that worktree's removal.
`bin/sift-deploy` refuses overlapping production/staging paths.

`bin/sift-data-refresh` is the explicit staging data operation. It stops the
recorded staging server, preserves the old staging data under
`.refresh-backup/<timestamp>/`, takes a consistent SQLite backup of production,
copies production config/evidence/answers/feedback, removes stale WAL/SHM
files, and leaves staging stopped unless asked to restart. A normal deploy never
runs this operation; `bin/sift-deploy --env staging --refresh-data` is the
explicit combined code-and-data operation.

All builds go to fresh candidate directories and are published only after they
succeed. A deploy verifies the serving process through `GET /api/version`
against the build identity, uses `GET /api/health` rather than requiring a
non-empty catalogue, and performs no `POST /api/context` write. Rollback stops
the candidate, restores the binary, bundle, worker tree and a consistent
database, removes stale WAL/SHM files, and restarts the previous server.
