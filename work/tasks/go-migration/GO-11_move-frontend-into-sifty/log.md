# GO-11 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-06 — 1.1

Check: `ls sifty/web/model/ && echo '---' && ls app/src/ | grep -E '^(core|workflow|preparation-error|application-store)\.ts$'`
Exit: 0

```
core.ts
preparation-error.ts
types.ts
workflow.ts
---
application-store.ts
core.ts
preparation-error.ts
workflow.ts
```


## 2026-11-06 — 1.2

Check: `grep -rn '../src/' sifty/web/ --include='*.ts' --include='*.tsx'; echo 'exit:' $?`
Exit: 0

```
sifty/web/model/types.ts:1:// Frontend-owned types extracted from app/src/application-store.ts.
exit: 0
```


## 2026-11-06 — 2.1

Check: `ls sifty/web/package.json sifty/web/vite.config.ts sifty/web/tsconfig.json`
Exit: 0

```
sifty/web/package.json
sifty/web/tsconfig.json
sifty/web/vite.config.ts
```


## 2026-11-06 — 2.2

Check: `npm --prefix sifty/web run build 2>&1 | tail -4`
Exit: 0

```
../dist/assets/unbounded-latin-600-normal-BPdB_PEv.woff           26.40 kB
../dist/assets/index-BaqFdC5W.css                                 28.78 kB │ gzip:  6.87 kB
../dist/assets/index-BHyfT4x_.js                                 288.61 kB │ gzip: 89.05 kB
✓ built in 403ms
```


## 2026-11-06 — 3.1

Check: `grep -n 'sifty/web\|sifty/dist' Makefile`
Exit: 0

```
4:# Output: bin/sifty (binary) and sifty/dist/ (frontend bundle).
8:	npm --prefix sifty/web run build
15:		--dist ./sifty/dist \
23:		--dist ./sifty/dist \
33:	rm -rf sifty/dist
```


## 2026-11-06 — 3.2

Check: `grep -n 'sifty/web\|sifty/dist\|sifty/dist-staging' bin/sift-deploy bin/sift-accept`
Exit: 0

```
bin/sift-deploy:13:#   SIFTY_PROD_DIST          prod React dist dir           (default <repo>/sifty/dist)
bin/sift-deploy:18:#   SIFTY_STAGING_DIST       staging React dist dir        (default <repo>/sifty/dist-staging)
bin/sift-deploy:101:  (b) npm --prefix sifty/web ci && npm --prefix sifty/web run build  → sifty/dist/
bin/sift-deploy:105:  (b) npm --prefix sifty/web ci && BUILD_OUTDIR=<abs> npm --prefix sifty/web run build  → sifty/dist-staging/
bin/sift-deploy:106:  Staging build never touches bin/sifty or sifty/dist.
bin/sift-deploy:142:        DIST_DIR="${SIFTY_PROD_DIST:-$APP_HOME/sifty/dist}"
bin/sift-deploy:149:        DIST_DIR="${SIFTY_STAGING_DIST:-$MAIN_REPO/sifty/dist-staging}"
bin/sift-deploy:172:if npm --prefix "$APP_HOME/sifty/web" ci --silent 2>/dev/null; then
bin/sift-deploy:174:elif npm --prefix "$APP_HOME/sifty/web" install --silent 2>/dev/null; then
bin/sift-deploy:183:# vite.config.ts outDir (../dist relative to sifty/web/ = sifty/dist).
bin/sift-deploy:184:# A staging build never touches sifty/dist or bin/sifty.
bin/sift-deploy:187:    BUILD_OUTDIR="$DIST_DIR" npm --prefix "$APP_HOME/sifty/web" run build \
bin/sift-deploy:190:    log "Building frontend (npm run build → sifty/dist/)..."
bin/sift-deploy:191:    npm --prefix "$APP_HOME/sifty/web" run build \
bin/sift-accept:59:    if ! npm --prefix "$APP_HOME/sifty/web" ci --silent 2>/dev/null; then
bin/sift-accept:60:        npm --prefix "$APP_HOME/sifty/web" install --silent 2>/dev/null \
bin/sift-accept:65:    if npm --prefix "$APP_HOME/sifty/web" run build 2>&1; then
```


## 2026-11-06 — 4.1

Check: `cd sifty && go test -race ./... 2>&1 | tail -5 && cd ../sifty/web && npm run build 2>&1 | tail -3`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/source	(cached)
ok  	github.com/w3con/sifty/internal/store	(cached)
?   	github.com/w3con/sifty/internal/version	[no test files]
ok  	github.com/w3con/sifty/internal/worker	(cached)
?   	github.com/w3con/sifty/scripts/capture-fixtures	[no test files]
../dist/assets/index-BaqFdC5W.css                                 28.78 kB │ gzip:  6.87 kB
../dist/assets/index-BHyfT4x_.js                                 288.61 kB │ gzip: 89.05 kB
✓ built in 399ms
```

