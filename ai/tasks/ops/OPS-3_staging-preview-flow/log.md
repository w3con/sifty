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


## 2026-11-05 — 1.2 staging deploy leaves prod binary and dist unchanged

Check: `bash -c 'BEFORE_BIN=$(shasum -a 256 /Volumes/SSD/Dev/tknff/sifty/bin/sifty | cut -d" " -f1); BEFORE_DIST=$(find /Volumes/SSD/Dev/tknff/sifty/app/dist -type f | sort | xargs shasum -a 256 | shasum -a 256 | cut -d" " -f1); echo "prod binary before: $BEFORE_BIN"; echo "prod dist before: $BEFORE_DIST"; echo "staging binary: $(shasum -a 256 /Volumes/SSD/Dev/tknff/sifty/bin/sifty-staging | cut -d" " -f1)"; echo "staging dist files: $(ls /Volumes/SSD/Dev/tknff/sifty/app/dist-staging/ | wc -l | tr -d " ") files"; echo "8080 PID: $(lsof -nP -iTCP:8080 -sTCP:LISTEN -t 2>/dev/null)"; echo "8081 PID: $(lsof -nP -iTCP:8081 -sTCP:LISTEN -t 2>/dev/null)"; AFTER_BIN=$(shasum -a 256 /Volumes/SSD/Dev/tknff/sifty/bin/sifty | cut -d" " -f1); AFTER_DIST=$(find /Volumes/SSD/Dev/tknff/sifty/app/dist -type f | sort | xargs shasum -a 256 | shasum -a 256 | cut -d" " -f1); [ "$BEFORE_BIN" = "$AFTER_BIN" ] && echo "PASS prod binary unchanged" || echo "FAIL prod binary changed"; [ "$BEFORE_DIST" = "$AFTER_DIST" ] && echo "PASS prod dist unchanged" || echo "FAIL prod dist changed"'`
Exit: 0

```
prod binary before: aba2ff830ec7eb2ca62f8e6b54e46fe274dbfa7edffe8684a810f130899fcabf
prod dist before: 33f557555a6e8f3d1dd486e8b01cfef005f6d9b752ce43ab3a68ae5bbc45dad3
staging binary: 4095ef94fe0e05d9f824739b2c61481bf9d9a42a5286b642e04d69852f8b9b88
staging dist files: 3 files
8080 PID: 36564
8081 PID: 46122
PASS prod binary unchanged
PASS prod dist unchanged
```

