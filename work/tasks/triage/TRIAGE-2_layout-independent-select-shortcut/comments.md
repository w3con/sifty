# TRIAGE-2 — comments

Captured from the owner's report on 2026-11-10. No plan or work has started.

## 2026-11-10 — K.1 frontend build

Check: `npm --prefix frontend run build`
Exit: 0

```

> sifty-web@1.0.0 build
> tsc --noEmit && vite build

vite v7.3.7 building client environment for production...
transforming...
✓ 1082 modules transformed.
rendering chunks...
computing gzip size...
dist/index.html                                                 0.40 kB │ gzip:   0.28 kB
dist/assets/ibm-plex-mono-latin-400-normal-CvHOgSBP.woff       13.14 kB
dist/assets/ibm-plex-mono-latin-500-normal-CB9ihrfo.woff       13.16 kB
dist/assets/ibm-plex-sans-cyrillic-400-normal-BTotfTJu.woff    13.51 kB
dist/assets/ibm-plex-mono-latin-400-normal-DMJ8VG8y.woff2      14.71 kB
dist/assets/ibm-plex-sans-cyrillic-400-normal-DZqxrq2p.woff2   14.80 kB
dist/assets/ibm-plex-mono-latin-500-normal-DSY6xOcd.woff2      14.89 kB
dist/assets/unbounded-latin-600-normal-oRSANpZr.woff2          20.95 kB
dist/assets/ibm-plex-sans-latin-400-normal-CYLoc0-x.woff       22.10 kB
dist/assets/ibm-plex-sans-latin-700-normal-Bth3BMcD.woff       22.28 kB
dist/assets/ibm-plex-sans-latin-400-normal-CDDApCn2.woff2      22.59 kB
dist/assets/ibm-plex-sans-latin-700-normal-Bxkt5Cjx.woff2      22.83 kB
dist/assets/ibm-plex-sans-latin-600-normal-Cu4Hd6ag.woff       23.88 kB
dist/assets/ibm-plex-sans-latin-500-normal-BgVn5rGT.woff       23.92 kB
dist/assets/ibm-plex-sans-latin-500-normal-6ng42L7E.woff2      24.18 kB
dist/assets/ibm-plex-sans-latin-600-normal-CuJfVYMP.woff2      24.25 kB
dist/assets/unbounded-latin-600-normal-BPdB_PEv.woff           26.40 kB
dist/assets/index-CeMkpW-J.css                                115.29 kB │ gzip:  21.71 kB
dist/assets/index-DWU4qw7K.js                                 940.53 kB │ gzip: 290.38 kB

(!) Some chunks are larger than 500 kB after minification. Consider:
- Using dynamic import() to code-split the application
- Use build.rollupOptions.output.manualChunks to improve chunking: https://rollupjs.org/configuration-options/#output-manualchunks
- Adjust chunk size limit for this warning via build.chunkSizeWarningLimit.
✓ built in 1.25s
```

## 2026-11-10 — K.1 real-browser layout-independent Select (synthetic fixture)

Check: `node /tmp/triage2-browser-accept.mjs`
Exit: 0

```
PASS key=y code=KeyY selects — patches 0->1; last={"stage":"preselected","confirmed":false}
PASS key=н code=KeyY selects (layout-independent) — patches 1->2; last={"stage":"preselected","confirmed":false}
PASS key=y code=KeyZ does not select (strict physical code) — patches 2->2
PASS focused editable input suppresses Select — patches 2->2
PASS Ctrl+KeyY suppressed — patches 2->2

SUMMARY {"passed":5,"failed":0,"results":[{"name":"key=y code=KeyY selects","ok":true,"detail":"patches 0->1; last={\"stage\":\"preselected\",\"confirmed\":false}"},{"name":"key=н code=KeyY selects (layout-independent)","ok":true,"detail":"patches 1->2; last={\"stage\":\"preselected\",\"confirmed\":false}"},{"name":"key=y code=KeyZ does not select (strict physical code)","ok":true,"detail":"patches 2->2"},{"name":"focused editable input suppresses Select","ok":true,"detail":"patches 2->2"},{"name":"Ctrl+KeyY suppressed","ok":true,"detail":"patches 2->2"}]}
```

## 2026-11-10 — K.2 accepted fix parity and actual diff

Check: `test "$(git rev-parse main:frontend/main.tsx)" = "$(git rev-parse work/TRIAGE-2:frontend/main.tsx)" && git diff --stat 39a8745..main -- frontend/main.tsx`
Exit: 0

```
 frontend/main.tsx | 6 +++---
 1 file changed, 3 insertions(+), 3 deletions(-)
```

## 2026-11-10 — K.2 grouped commits and file review

Check: `git log --format="%h %s" 39a8745..main && echo "--- changed_paths ---" && git diff --name-status 39a8745..main | wc -l && git diff --stat 39a8745..main | tail -1 && echo "--- design_assets ---" && git diff --name-only 39a8745..main -- "frontend/Claude outputs" | wc -l && git ls-tree -r -l main -- "frontend/Claude outputs" | awk "{s+=\$4} END {print \"asset_bytes=\" s}"`
Exit: 0

```
e98a5e8 docs(design): add owner-approved design review assets
c2333b4 docs(runtime): reconcile preview and production records
0b2b06e docs(work): record deferred plans and execution evidence
b81d2c5 docs(work): clarify execution boundaries and specifications
e295b21 fix(TRIAGE-2): integrate layout-independent Select shortcut
--- changed_paths ---
     119
 119 files changed, 1255 insertions(+), 24 deletions(-)
--- design_assets ---
      89
asset_bytes=19162852
```

## 2026-11-10 — K.2 commit guards and leak scanning

Check: `python3 backend/scripts/leakcheck/range-check 39a8745..main --repo .`
Exit: 0

```
range-check: 5 commit(s) checked, clean
```

## 2026-11-10 — K.2 source, recovery and worktree preservation

Check: `test "$(git rev-parse refs/work-recovery/TRIAGE-2-preintegration)" = f1754ea9b5720fbdb15d8846448a57bbd0e0f33e && test "$(git rev-parse f1754ea9b5720fbdb15d8846448a57bbd0e0f33e)" = f1754ea9b5720fbdb15d8846448a57bbd0e0f33e && test -d .local/recovery/TRIAGE-2-preintegration-20261110T160409Z && test "$(git worktree list --porcelain | grep -c "^worktree ")" = 14 && test "$(git for-each-ref refs/heads | wc -l | tr -d " ")" = 15 && echo "preoperation source, recovery ref/snapshot, 14 worktrees and 15 branches preserved"`
Exit: 1

```

```

## 2026-11-10 — K.2 source, recovery and worktree preservation (corrected path)

Check: `test "$(git rev-parse refs/work-recovery/TRIAGE-2-preintegration)" = f1754ea9b5720fbdb15d8846448a57bbd0e0f33e && test "$(git rev-parse f1754ea9b5720fbdb15d8846448a57bbd0e0f33e)" = f1754ea9b5720fbdb15d8846448a57bbd0e0f33e && test -d ../../.local/recovery/TRIAGE-2-preintegration-20261110T160409Z && test "$(git worktree list --porcelain | grep -c "^worktree ")" = 14 && test "$(git for-each-ref refs/heads | wc -l | tr -d " ")" = 15 && echo "preoperation source, recovery ref/snapshot, 14 worktrees and 15 branches preserved"`
Exit: 0

```
preoperation source, recovery ref/snapshot, 14 worktrees and 15 branches preserved
```

## 2026-11-10 — K.2 production runtime unchanged

Check: `test "$(lsof -nP -iTCP:8080 -sTCP:LISTEN -t)" = 71548 && curl -sS -m 5 http://127.0.0.1:8080/api/version && echo && curl -sS -m 5 -o /dev/null -w "health=%{http_code}\n" http://127.0.0.1:8080/api/health`
Exit: 0

```
{"version":"157fe9f6d0f8-20261110T151309.703877Z"}

health=200
```

## 2026-11-10 — K.2 main checkout clean and diff check

Check: `test -z "$(git -C ../.. status --porcelain)" && git -C ../.. diff --check && echo "main checkout clean; git diff --check clean"`
Exit: 0

```
main checkout clean; git diff --check clean
```
