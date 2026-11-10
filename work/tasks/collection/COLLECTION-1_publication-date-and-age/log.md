# COLLECTION-1 preparation log

2026-11-09T21:10Z — owner «Делай.» approves the complete20–35min F.1 plan. Start21:10Z/stop21:45Z includes preparation/review; preserve old main/worktrees. Bootstrap from accepted APP-8 d2db849, same executor continues one F.1 pass, reuse independent8083data/preview. Owner's next instruction: «Как закончишь со свежестью вакансий - надо будет завершить с профилем. там дизайн уже поменялся». Queue that after freshness; current DESIGN-3 has active dirty Profile/theme files owned by the other agent, untouched.

2026-11-09T21:02Z — APP-8 owner/technical acceptance complete; next plan narrowed to actual source publication/age and Newest for manual review. Existing parsers/store/API fields inspected; old provisional80–150min scope withdrawn in favour of20–35min complete single-pass proposal, existing8083preview/data reused. Automatic age penalty/scoring/hiding and duplicate/history work excluded from this first step and remain explicit later scope. Next: owner approval, then isolated worktree/live brief and one F.1 executor stage. No source/runtime edits or provider/collection calls during planning.

## 2026-11-09 — F.1 — publication date, age and genuine Newest

Check: `(cd backend && go test ./internal/model -run TestNormalizePublishedAt -count=1 && go test ./internal/collect -run 'TestParseFeedPublicationIgnoresUpdated|TestParseFeedParity|TestStructuredJobsParity|TestHTMLLinksGetroParity' -count=1 && go test ./internal/source -run TestGreenhouse -count=1 && go test ./internal/store -run TestUpsertFillsAndPreservesPublishedAt -count=1 && go build ./...)`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/model	0.230s
ok  	github.com/w3con/sifty/internal/collect	0.182s
ok  	github.com/w3con/sifty/internal/source	0.217s
ok  	github.com/w3con/sifty/internal/store	0.309s
```

## 2026-11-09 — F.1 — real Chromium review on reused 8083

8083 PID 63591 serves this worktree's binary, frontend/dist and backend/workers against the existing isolated data copy (app8-iso/data); no new environment or data copy. Previous APP-8 server argv/binary preserved in .local/collection1-review/recovery.json and sifty-app8-recovery. Browser routes blocked every non-GET; only the app-mount POST /api/search was attempted and aborted. Screenshots: .local/collection1-review/list-newest.png and detail-coinbag.png. No schema/migration change, so the existing v17 database is used unchanged; no model/search/collection/worker calls were made.

Check: `node .local/collection1-review/verify.mjs`
Exit: 0

```
{
  "results": [
    {
      "name": "recent publication precedes a later-discovered old publication",
      "pass": true,
      "detail": "2@512 2057@1478 (firstSeen order would reverse this)"
    },
    {
      "name": "recent publication precedes old Espresso posting",
      "pass": true,
      "detail": "2746@27 463@1500"
    },
    {
      "name": "unknown publication dates sort last",
      "pass": true,
      "detail": "⬡ | Mesh | 8 | Senior Engineering Manager, Cloud Infrastructure | Location unstated | Publication date not reported"
    },
    {
      "name": "old posting row shows publication and age",
      "pass": true,
      "detail": "Antares | — | General Application for Engineers | Los Angeles · California · USA | Published Sep 10, 2024 · 2 years old"
    },
    {
      "name": "old publication is not labelled auto-expired",
      "pass": true,
      "detail": "Antares | — | General Application for Engineers | Los Angeles · California · USA | Published Sep 10, 2024 · 2 years old"
    },
    {
      "name": "old Coinbag row shows publication and age",
      "pass": true,
      "detail": "⬡ | Coinbag | 59 | Software Engineer | Europe · Ireland · Thailand | Published Apr 25, 2022 · 5 years old"
    },
    {
      "name": "detail shows Published separately from Collected",
      "pass": true,
      "detail": "LOCATION |  | Europe · Ireland · Thailand |  | EMPLOYMENT |  | Not separately verified — see listing text |  | COMPENSATION |  | Not separately verified — see listing text |  | PUBLISHED |  | Apr 25, 2022 · 5 years old |  | COLLECTED |  | 10/29/2026, 5:11:54 PM"
    },
    {
      "name": "only the app-mount POST /api/search was blocked",
      "pass": true,
      "detail": "POST /api/search"
    }
  ],
  "failures": [],
  "blocked": [
    "POST /api/search"
  ]
}
```

## 2026-11-09 — F.1 corrections — source date semantics, query identity, honest legacy label

Corrections: accept 'YYYY-MM-DD HH:MM:SS+07:00' (actual Web3 stored format); keep date-only values date-only so a stated calendar day does not shift across browser zones; reject impossible calendar dates in the frontend; completed calendar age (Coinbag 4y 6m, not 5y); matcher keeps identity query parameters and returns no match for ambiguous multi-post pages, so callers keep partial fields instead of guessing; legacy rows are visibly labelled 'unverified source date'. Legacy coverage gap: old rows have no origin metadata, so source-update vs genuine publication cannot be recovered within this scope; raw values are preserved without mass rewrite and are not asserted as verified publication. No schema/migration change.

Check: `(cd backend && go build ./... && go test ./internal/model -run TestNormalizePublishedAt -count=1 && go test ./internal/collect -run 'TestParseFeedPublicationIgnoresUpdated|TestParseFeedParity|TestStructuredJobsParity|TestHTMLLinksGetroParity' -count=1 && go test ./internal/source -run 'TestGreenhouse|TestMatchStructuredJob|TestSamePostingURL' -count=1 && go test ./internal/server -run TestMatchEnrichJob -count=1 && go test ./internal/store -run 'TestUpsertFillsAndPreservesPublishedAt|TestUpsertNormalizesWeb3PublicationFormat' -count=1)`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/model	0.170s
ok  	github.com/w3con/sifty/internal/collect	0.203s
ok  	github.com/w3con/sifty/internal/source	0.205s
ok  	github.com/w3con/sifty/internal/server	0.256s
ok  	github.com/w3con/sifty/internal/store	0.241s
```

## 2026-11-09 — F.1 corrections — frontend date helper and Chromium

Check: `node --experimental-strip-types .local/collection1-review/dates-check.mjs && node .local/collection1-review/verify.mjs`
Exit: 0

```
{
  "results": [
    {
      "name": "date-only keeps its stated calendar day",
      "pass": true,
      "detail": "2025-07-17T22:00:00.000Z"
    },
    {
      "name": "invalid calendar 2026-02-31 is rejected",
      "pass": true,
      "detail": ""
    },
    {
      "name": "future date is rejected",
      "pass": true,
      "detail": ""
    },
    {
      "name": "Web3 space/no-separator zone is accepted",
      "pass": true,
      "detail": ""
    },
    {
      "name": "Coinbag age uses completed units (4 years 6 months, not 5 years)",
      "pass": true,
      "detail": "4 years 6 months old"
    },
    {
      "name": "unknown source date is labelled honestly",
      "pass": true,
      "detail": "Unverified source date not reported"
    }
  ],
  "failures": []
}
{
  "results": [
    {
      "name": "recent publication precedes a later-discovered old publication",
      "pass": true,
      "detail": "2@512 2057@1478 (firstSeen order would reverse this)"
    },
    {
      "name": "recent publication precedes old Espresso posting",
      "pass": true,
      "detail": "2746@27 463@1500"
    },
    {
      "name": "unknown publication dates sort last",
      "pass": true,
      "detail": "⬡ | Mesh | 8 | Senior Engineering Manager, Cloud Infrastructure | Location unstated | Unverified source date not reported"
    },
    {
      "name": "old posting row shows unverified source date and completed age",
      "pass": true,
      "detail": "Antares | — | General Application for Engineers | Los Angeles · California · USA | Unverified source date Sep 10, 2024 · 2 years 1 month old"
    },
    {
      "name": "old publication is not labelled auto-expired",
      "pass": true,
      "detail": "Antares | — | General Application for Engineers | Los Angeles · California · USA | Unverified source date Sep 10, 2024 · 2 years 1 month old"
    },
    {
      "name": "old Coinbag row shows unverified source date and completed age",
      "pass": true,
      "detail": "⬡ | Coinbag | 59 | Software Engineer | Europe · Ireland · Thailand | Unverified source date Apr 25, 2022 · 4 years 6 months old"
    },
    {
      "name": "detail shows unverified source date separately from Collected",
      "pass": true,
      "detail": "LOCATION |  | Europe · Ireland · Thailand |  | EMPLOYMENT |  | Not separately verified — see listing text |  | COMPENSATION |  | Not separately verified — see listing text |  | UNVERIFIED SOURCE DATE |  | Apr 25, 2022 · 4 years 6 months old |  | COLLECTED |  | 10/29/2026, 5:11:54 PM"
    },
    {
      "name": "only the app-mount POST /api/search was blocked",
      "pass": true,
      "detail": "POST /api/search"
    }
  ],
  "failures": [],
  "blocked": [
    "POST /api/search"
  ]
}
```

## 2026-11-09 — F.1 correction — publication basis and 3-way Newest

Publication basis contract: optional jobs.published_basis (v17→v18 additive); source parsers set datePosted/pubDate/published/first_published/published_date/created_at only when a valid genuine date exists; legacy rows keep NULL basis and are shown as 'Source date … (unverified)'; known rows show 'Published'; missing shows 'Publication date not reported'; Newest orders known (desc), legacy unverified (desc), missing last. Store fills a missing date, retains an established date/basis, and never stamps a new origin onto a different date. Existing v17 isolated DB migrated to 18 in place. Synthetic known fixtures 990008/990009 (existing APP8 synthetic rows in the isolated copy only) were given basis first_published for the browser check; no real legacy row was marked known. Pre-existing TestMigrationV14 is broken independently (seeds a v13 DB without a jobs table) and was not made part of this check.

Check: `(cd backend && go build ./... && go test ./internal/store -run 'TestMigrationV16ToV17PreservesData|TestSchema_FreshDatabase|TestSchema_ExistingDatabase|TestNegativeOutcomeReasonAndStaleGuard|TestUpsertFillsAndPreservesPublishedAt|TestUpsertNormalizesWeb3PublicationFormat|TestUpsertAssociatesPublicationBasisWithoutInventingHistory' -count=1 && go test ./internal/collect -run 'TestParseFeedPublicationIgnoresUpdated|TestParseFeedParity|TestStructuredJobsParity|TestHTMLLinksGetroParity' -count=1 && go test ./internal/source -run 'TestGreenhouse|TestMatchStructuredJob|TestSamePostingURL' -count=1 && go test ./internal/server -run TestMatchEnrichJob -count=1) && node --experimental-strip-types .local/collection1-review/dates-check.mjs && node .local/collection1-review/verify.mjs`
Exit: 1

```
ok  	github.com/w3con/sifty/internal/store	0.283s
ok  	github.com/w3con/sifty/internal/collect	0.178s
ok  	github.com/w3con/sifty/internal/source	0.197s
ok  	github.com/w3con/sifty/internal/server	0.252s
{
  "results": [
    {
      "name": "date-only keeps its stated calendar day",
      "pass": true,
      "detail": "2025-07-17T22:00:00.000Z"
    },
    {
      "name": "invalid calendar 2026-02-31 is rejected",
      "pass": true,
      "detail": ""
    },
    {
      "name": "future date is rejected",
      "pass": true,
      "detail": ""
    },
    {
      "name": "Web3 space/no-separator zone is accepted",
      "pass": true,
      "detail": ""
    },
    {
      "name": "Coinbag age uses completed units (4 years 6 months, not 5 years)",
      "pass": true,
      "detail": "4 years 6 months old"
    },
    {
      "name": "unknown source date is labelled honestly",
      "pass": false,
      "detail": "Publication date not reported"
    }
  ],
  "failures": [
    "unknown source date is labelled honestly: Publication date not reported"
  ]
}
```

## 2026-11-09 — F.1 correction — restore real copied rows and enrich basis

Mistake corrected: 990008/990009 are real source-collected StreamingFast jobs in the isolated copy, not APP-8 fixtures. Their only change from this session was published_basis='first_published'; both are restored to NULL and remain visibly legacy/unverified. No workflow/history/date/owner action was rewritten. Two properly labelled synthetic fixtures were inserted only in the isolated copy with unused IDs 999001/999002, example.invalid URLs and company 'Collection Fixture'; they carry first_published basis for the known-label check. EnrichJob now fills publication date and basis atomically for a genuinely missing date (or a missing basis on the confirmed same date), never a different legacy date, preserving workflow. Backend rebuilt and 8083 refreshed with the same argv/data/frontend; screenshots retaken.

Check: `(cd backend && go build ./... && go test ./internal/server -run TestEnrichFn -count=1) && node --experimental-strip-types .local/collection1-review/dates-check.mjs && node .local/collection1-review/verify.mjs`
Exit: 1

```
ok  	github.com/w3con/sifty/internal/server	0.316s
{
  "results": [
    {
      "name": "date-only keeps its stated calendar day",
      "pass": true,
      "detail": "2025-07-17T22:00:00.000Z"
    },
    {
      "name": "invalid calendar 2026-02-31 is rejected",
      "pass": true,
      "detail": ""
    },
    {
      "name": "future date is rejected",
      "pass": true,
      "detail": ""
    },
    {
      "name": "Web3 space/no-separator zone is accepted",
      "pass": true,
      "detail": ""
    },
    {
      "name": "Coinbag age uses completed units (4 years 6 months, not 5 years)",
      "pass": true,
      "detail": "4 years 6 months old"
    },
    {
      "name": "unknown source date is labelled honestly",
      "pass": false,
      "detail": "Publication date not reported"
    }
  ],
  "failures": [
    "unknown source date is labelled honestly: Publication date not reported"
  ]
}
```

## 2026-11-09 — note

2026-11-09T21:44Z final record: source d92112c04a3f40e24fd306a7b19d7c4a422625d5 technically accepted by coordinator. Preview http://127.0.0.1:8083 PID75373 serves the COLLECTION-1 binary/dist against the same isolated review data; source and data tested. No production 8080 change, no staging/Profile 8081/8082 writes, no merge/push/release. Next owner discussion is finishing updated Profile for one person with role-based profiles and separately keyed scores, not auth/logout; confirm the other Profile writer handback before any writes.
