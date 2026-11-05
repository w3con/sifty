# OPS-3 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-05 — 1.1 per-env binary and dist

Check: `bin/sift-deploy --help`
Exit: 0

```
sift-deploy — build, swap and smoke-check a sifty environment.

Usage:
  sift-deploy --env staging|prod [--build-only]
  sift-deploy --help

Options:
  --env <name>    Environment to deploy: staging or prod. Required.
  --build-only    Build Go binary and frontend but do not stop/start any server.
  --help          Print this help and exit.

Per-env config keys (bin/work.config or environment variables):
  SIFTY_PROD_PORT / SIFTY_STAGING_PORT         HTTP port (defaults: 8080 / 8081)
  SIFTY_PROD_DATA_DIR / SIFTY_STAGING_DATA_DIR data directory
  SIFTY_PROD_BINARY / SIFTY_STAGING_BINARY     path to the sifty binary (each env owns its own)
  SIFTY_PROD_DIST / SIFTY_STAGING_DIST         React dist dir (each env owns its own)
  SIFTY_PROD_LOG / SIFTY_STAGING_LOG           server log path
  SIFTY_WORKERS_DIR                            workers dir (shared)

Build (prod):
  (a) cd sifty && go build -o <repo>/bin/sifty.new ./cmd/sifty  → bin/sifty
  (b) npm --prefix app ci && npm --prefix app run build          → app/dist/

Build (staging):
  (a) cd sifty && go build -o <repo>/bin/sifty-staging.new ./cmd/sifty  → bin/sifty-staging
  (b) npm --prefix app ci && BUILD_OUTDIR=../dist-staging vite build     → app/dist-staging/
  Staging build never touches bin/sifty or app/dist.

Deploy (non-build-only):
  1. For staging: refresh data-staging from a consistent prod snapshot.
  2. Back up <data_dir>/store.sqlite (timestamped).
  3. Keep previous binary as <binary>.prev.
  4. Stop the env's server on its port.
  5. Start the new binary detached against the env's data dir and dist.
  6. Poll GET /api/health for HTTP 200 (30 s timeout).
  7. Assert GET /api/jobs returns at least one entry.
  8. POST /api/context (idempotent rescore / profile-save).
  9. On any failure: restore previous binary and backup, restart, exit 1.
 10. Append a log line to <data_dir>/deploy.log.
```

