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

