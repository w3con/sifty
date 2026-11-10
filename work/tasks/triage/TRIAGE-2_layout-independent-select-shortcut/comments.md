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
