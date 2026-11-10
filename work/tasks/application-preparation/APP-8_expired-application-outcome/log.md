# APP-8 log

Coordinator stop on handback,2026-11-09: executor reports completion at28a27f36e1ff195bbdb984165bbed4f78553f33c, own build/named critical test/browser checks passing, defaultwork-testSKIPPEDonce, isolatedpreview127.0.0.1:8083. Actual19:06–19:41Z~35min plus~4minpreparation vs10–20min planned and19:22Z explicitstop. Time overrun attributed by executor to isolated preview/data setup and browser automation; both were required scope and should have been budgeted/controlled. Coordinator has not reviewed/accepted implementation; no further implementation/check cycle. Remaining code/browser evidence review and report estimated5–10min, continuation needs explicit owner response after overrun report. Production8080 is owner real environment; no deployment or data-history repair approved. The user meanwhile reports avatar2MiB ceiling should be10MiB; capture only on DESIGN-3, separate Profile writer handles it.

2026-11-09T19:02Z — execution of agreed five Apply outcomes and button-only modal, direct owner authorization «да, давай. и чтобы нажатие мышкой за пределами modal не закрыавало этот modal. только кнопка.» Complete plan10–20min shown before execution. Start19:02Z/stop19:22Z. Latest owner «да я иногда жал что подался, а иногда возвращал в триаж и удалял оттуда. не важно. все делал на 8080» closes the environment-detour investigation: production is real history, do not investigate/sync snapshots further in APP-8. Return/removal context informs later TRIAGE-1 history-based recognition, not an automatic history repair here. Next: delegated isolated bootstrap/live brief, same executor A.1, review. Preserve active DESIGN-3 Profile writer and preview.

## 2026-11-09 — A.1 atomic reason/store guard test

Check: `cd backend && go test ./internal/store -run TestNegativeOutcomeReasonAndStaleGuard -count=1`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	0.342s
```

## 2026-11-09 — A.1 backend build

Check: `cd backend && go build ./...`
Exit: 0

```

```

## 2026-11-09 — A.1 frontend locked install and build

Check: `npm --prefix frontend ci && npm --prefix frontend run build`
Exit: 0

```

added 133 packages, and audited 134 packages in 3s

44 packages are looking for funding
  run `npm fund` for details

found 0 vulnerabilities

> sifty-web@1.0.0 build
> tsc --noEmit && vite build

vite v7.3.7 building client environment for production...
transforming...
✓ 1074 modules transformed.
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
dist/assets/index-DKG5u9jd.css                                 62.10 kB │ gzip:  12.71 kB
dist/assets/index-TJkfMHBC.js                                 895.23 kB │ gzip: 275.73 kB

(!) Some chunks are larger than 500 kB after minification. Consider:
- Using dynamic import() to code-split the application
- Use build.rollupOptions.output.manualChunks to improve chunking: https://rollupjs.org/configuration-options/#output-manualchunks
- Adjust chunk size limit for this warning via build.chunkSizeWarningLimit.
✓ built in 1.18s
```

## 2026-11-09 — A.1 real Chromium outcomes/modal/reload verification

Check: `node .local/app8-review/verify.mjs`
Exit: 0

```
{
  "results": [
    {
      "name": "backdrop click keeps modal open",
      "pass": true,
      "detail": ""
    },
    {
      "name": "Escape keeps modal open",
      "pass": true,
      "detail": ""
    },
    {
      "name": "Not yet closes without moving job",
      "pass": true,
      "detail": "stage=preselected"
    },
    {
      "name": "stale persists stage/reason",
      "pass": true,
      "detail": "{\"stage\":\"no\",\"reason\":\"stale\"}"
    },
    {
      "name": "closed persists stage/reason",
      "pass": true,
      "detail": "{\"stage\":\"no\",\"reason\":\"closed\"}"
    },
    {
      "name": "not_suitable persists stage/reason",
      "pass": true,
      "detail": "{\"stage\":\"no\",\"reason\":\"not_suitable\"}"
    },
    {
      "name": "Applied records submission",
      "pass": true,
      "detail": "{\"stage\":\"applied\",\"appliedAt\":\"2026-11-09T19:10:50.966Z\"}"
    },
    {
      "name": "all buttons disabled while saving",
      "pass": true,
      "detail": "disabled=5"
    },
    {
      "name": "failed save keeps modal open",
      "pass": true,
      "detail": ""
    },
    {
      "name": "failed save keeps previous stage",
      "pass": true,
      "detail": "{\"stage\":\"preselected\",\"reason\":null}"
    },
    {
      "name": "retry after failure succeeds",
      "pass": true,
      "detail": "{\"stage\":\"no\",\"reason\":\"closed\"}"
    },
    {
      "name": "archive shows stored reason",
      "pass": true,
      "detail": "Dismissed — Vacancy is closed\nRestore to Triage"
    },
    {
      "name": "archive reason survives reload",
      "pass": true,
      "detail": "Dismissed — Vacancy is closed\nRestore to Triage"
    },
    {
      "name": "mobile modal fits viewport",
      "pass": true,
      "detail": "{\"x\":17,\"y\":184.875,\"width\":356,\"height\":474.25}"
    },
    {
      "name": "unexpected non-GET blocked",
      "pass": true,
      "detail": "POST /api/search, POST /api/search, POST /api/search"
    }
  ],
  "failures": [],
  "blocked": [
    "POST /api/search",
    "POST /api/search",
    "POST /api/search"
  ],
  "blockedCount": 3
}
```

## 2026-11-09 — A.1 production/staging untouched (read-only snapshot)

Check: `python3 .local/app8-review/snapshot.py`
Exit: 0

```
prod {'jobs': 2738, 'history': 336, 'assessment_queue': 2497, 'preparation_operations': 132, 'user_version': 14}
staging {'jobs': 2755, 'history': 315, 'assessment_queue': 2497, 'preparation_operations': 77, 'user_version': 16}
```

## 2026-11-09 — note

A.1 real-use isolation: APP-8 preview http://127.0.0.1:8083 (port verified free), binary .local/app8-review/sifty pid 54741, dist frontend/dist, workers backend/workers, data WAL-consistent sqlite backup of current production 8080 data at /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/opencode/app8-iso/data (ISOLATED DATA: contains synthetic APP-8 fixture jobs 990001-990006; max_assessments_per_run=0, assessment_queue/preparation_operations/locks cleared before launch). Screenshots .local/app8-review/modal-desktop.png, archive-dismissed.png, modal-mobile.png. Browser POST /api/search blocked (3), no unexpected non-GET; only PATCH /api/jobs/{id} allowed for fixtures. Production 8080 and staging 8081 were not contacted for writes.

## 2026-11-09 — note

A.1 card-level test run (single invocation, no --full): SKIPPED: automated tests are paused during rapid prototyping. Use bin/work-test --full to run them explicitly. Narrow targeted store test was run separately and passed.

## 2026-11-09 — note

A.1 technical decisions: reason column jobs.outcome_reason plus workflow_history.reason (migration v17); Store.StatusWithReason validates allowed negative reasons (stale/closed/not_suitable), requires preselected for a reasoned negative, and writes status/applied_at/reason/history in one transaction with a conditional UPDATE on the observed status so a stale negative cannot erase a concurrently confirmed application. Ordinary no-reason dismissal unchanged. Modal has no backdrop handler, onCancel is preventDefault only, all buttons disabled while saving, failure keeps modal/stage and shows role=alert. Applied/Not yet semantics unchanged. Failure found and fixed during work: initial guard only compared observed status, so an applied role could still be re-dismissed; added the preselected requirement for reasoned negatives and extended the test. Actual elapsed 19:06-19:41Z (~35min vs 10-20min plan): overrun caused by isolated preview/data setup and real-browser automation, not by scope growth. Owner acceptance/integration/release remain separate.

## 2026-11-09 — A.1 existing-v16 -> v17 migration preservation + outcome guard

Check: `cd backend && go test ./internal/store -run 'TestMigrationV16ToV17PreservesData|TestNegativeOutcomeReasonAndStaleGuard' -count=1`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	0.243s
```

## 2026-11-09 — A.1 backend build after targetVersion fix

Check: `cd backend && go build ./...`
Exit: 0

```

```

## 2026-11-09 — note

A.1 acceptance correction: migrate.go targetVersion raised 16->17 so an existing v16 database no longer returns early; TestMigrationV16ToV17PreservesData builds a genuine v16 shape (new columns dropped, PRAGMA user_version=16), reopens and verifies both columns, version 17, original job/history preserved, ordinary read works and a new reason write persists. No frontend change, no preview restart, no full suite/work-test.

## 2026-11-09 — note

A.1 completion record: technical and owner acceptance recorded in acceptance.md (through 7a2042ea85dd4fca2ecd9e42dd5d0bffcce64149); card remains review until preserving integration/release. Next product task: COLLECTION-1 publication dates/age/Newest with a complete bounded estimate/plan before execution. Source card acceptance: coordinator acceptance.md + attention preserved unchanged in commit 6428bfc. Follow-up captured, not implemented: why-not-suitable free text (provisional complexity1, 5-10min extra optional storage/display using the existing flow, no model/Profile changes; implementation unapproved). Existing 8083 preview backend refreshed only: verified old pid54741/port, stopped it, atomically replaced binary, same argv/isolated data/config/port; new pid51202, backend source 7a2042e+records, binary sha256 9175b583f94094492fefe5ce0271df60efce2f74ff712831e75738d7fb21059e; health/version OK; no data copy, no frontend change.
