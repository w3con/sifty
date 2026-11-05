# OPS-1 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-05 — 1.1 — Write bin/sift-deploy reading per-env config from bin/work.config

Check: `bash bin/sift-deploy --help`
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
  SIFTY_PROD_BINARY / SIFTY_STAGING_BINARY     path to the sifty binary
  SIFTY_PROD_LOG / SIFTY_STAGING_LOG           server log path
  SIFTY_DIST_DIR                               React dist dir (shared)
  SIFTY_WORKERS_DIR                            workers dir (shared)

Build:
  (a) cd sifty && go build -o <repo>/bin/sifty.new ./cmd/sifty
  (b) npm --prefix app ci && npm --prefix app run build  → app/dist/

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


## 2026-11-05 — 1.2 — Build phase: go build and vite build, aborting on error; --build-only builds without touching server

Check: `bash bin/sift-deploy --env staging --build-only`
Exit: 0

```
sift-deploy: env=staging port=8081 data=/Volumes/SSD/Dev/tknff/sifty/.worktrees/OPS-1/data-staging binary=/Volumes/SSD/Dev/tknff/sifty/.worktrees/OPS-1/bin/sifty
sift-deploy: --- BUILD ---
sift-deploy: Building Go binary (sifty/cmd/sifty → bin/sifty.new)...
sift-deploy: Installing frontend dependencies (npm ci)...
sift-deploy: Building frontend (npm run build → app/dist/)...

> fcto-job-catalogue@1.0.0 build
> tsc --noEmit && vite build

vite v7.3.7 building client environment for production...
transforming...
✓ 54 modules transformed.
rendering chunks...
computing gzip size...
../dist/index.html                                                 0.40 kB │ gzip:  0.28 kB
../dist/assets/ibm-plex-mono-latin-400-normal-CvHOgSBP.woff       13.14 kB
../dist/assets/ibm-plex-mono-latin-500-normal-CB9ihrfo.woff       13.16 kB
../dist/assets/ibm-plex-sans-cyrillic-400-normal-BTotfTJu.woff    13.51 kB
../dist/assets/ibm-plex-mono-latin-400-normal-DMJ8VG8y.woff2      14.71 kB
../dist/assets/ibm-plex-sans-cyrillic-400-normal-DZqxrq2p.woff2   14.80 kB
../dist/assets/ibm-plex-mono-latin-500-normal-DSY6xOcd.woff2      14.89 kB
../dist/assets/unbounded-latin-600-normal-oRSANpZr.woff2          20.95 kB
../dist/assets/ibm-plex-sans-latin-400-normal-CYLoc0-x.woff       22.10 kB
../dist/assets/ibm-plex-sans-latin-700-normal-Bth3BMcD.woff       22.28 kB
../dist/assets/ibm-plex-sans-latin-400-normal-CDDApCn2.woff2      22.59 kB
../dist/assets/ibm-plex-sans-latin-700-normal-Bxkt5Cjx.woff2      22.83 kB
../dist/assets/ibm-plex-sans-latin-600-normal-Cu4Hd6ag.woff       23.88 kB
../dist/assets/ibm-plex-sans-latin-500-normal-BgVn5rGT.woff       23.92 kB
../dist/assets/ibm-plex-sans-latin-500-normal-6ng42L7E.woff2      24.18 kB
../dist/assets/ibm-plex-sans-latin-600-normal-CuJfVYMP.woff2      24.25 kB
../dist/assets/unbounded-latin-600-normal-BPdB_PEv.woff           26.40 kB
../dist/assets/index-LzYizq43.css                                 26.52 kB │ gzip:  6.49 kB
../dist/assets/index-CN5R6hXf.js                                 288.46 kB │ gzip: 89.48 kB
✓ built in 427ms
sift-deploy: Build complete. binary hash: 9215bf49fd501af1...
sift-deploy: build-only: binary installed at /Volumes/SSD/Dev/tknff/sifty/.worktrees/OPS-1/bin/sifty; no server touched.
```

