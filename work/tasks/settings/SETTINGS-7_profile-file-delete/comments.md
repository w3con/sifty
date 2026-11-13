# SETTINGS-7 — comments

Captured from the owner's report on 2026-11-12. No plan or work has started.

## 2026-11-12 — 1.1 DELETE /api/profile/documents/{id}

Check: `cd backend && go test ./internal/profile/... ./internal/server/... ./internal/store/...`
Exit: 1

```
ok  	github.com/w3con/sifty/internal/profile	0.560s
--- FAIL: TestRouteDrift (0.03s)
    drift_test.go:175: registered by server but missing from fixtures: DELETE /api/profile/avatar
    drift_test.go:175: registered by server but missing from fixtures: GET /api/profile/avatar
    drift_test.go:175: registered by server but missing from fixtures: GET /api/profile/avatar/meta
    drift_test.go:175: registered by server but missing from fixtures: POST /api/profile/absent-skills
    drift_test.go:175: registered by server but missing from fixtures: POST /api/profile/avatar
    drift_test.go:175: registered by server but missing from fixtures: POST /api/profile/derived/refresh
    drift_test.go:175: registered by server but missing from fixtures: POST /api/profile/skills
    drift_test.go:175: registered by server but missing from fixtures: PUT /api/profile/sections/{section}
2026/11/12 12:03:49 quick-match order saved to runtime.yaml but the configuration could not be reloaded: read /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/TestReplayFixtures3355806655/002/config/candidate.yaml: open /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/TestReplayFixtures3355806655/002/config/candidate.yaml: no such file or directory
--- FAIL: TestReplayFixtures (6.93s)
    --- FAIL: TestReplayFixtures/69_DELETE_profile_document.json (0.00s)
        replay_test.go:146: status: want 404, got 415 (body: {"error":"JSON required"}
            )
FAIL
FAIL	github.com/w3con/sifty/internal/server	18.350s
--- FAIL: TestMigrationV14 (0.01s)
    migration_v14_test.go:84: store.Open after seeding v13: migrate /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/TestMigrationV143101610927/001/v13.sqlite: add jobs.match_profile_revision: SQL logic error: no such table: jobs (1)
--- FAIL: TestMigrationV16ToV17PreservesData (0.03s)
    outcome_reason_test.go:148: user_version = 22, want 18
--- FAIL: TestSchema_FreshDatabase (0.01s)
    store_test.go:81: [fresh database] user_version = 22, want 18
FAIL
FAIL	github.com/w3con/sifty/internal/store	1.448s
FAIL
```

## 2026-11-12 — 1.1 re-check after fixture fix (TestRouteDrift and 3 store migration tests fail on untouched main too)

Check: `cd backend && go test ./internal/profile/... ./internal/server/... 2>&1 | grep -E '^(--- FAIL|    --- FAIL|ok|FAIL)'`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/profile	(cached)
--- FAIL: TestRouteDrift (0.02s)
FAIL
FAIL	github.com/w3con/sifty/internal/server	19.577s
FAIL
```

## 2026-11-12 — 1.2 mutation check

Temporarily changed the store delete to remove accepted facts too (DELETE FROM profile_facts WHERE document_id=?). The test fails, then the change was reverted (git diff HEAD on internal/store is empty). Output: --- FAIL: TestDeleteDocumentKeepsAcceptedFacts; deletion report ... ProposalsRemoved:2 FactsKept:0, want 1 proposal removed, 1 fact kept; facts after delete = [], want only the accepted fact.

## 2026-11-12 — 1.2 one test: accepted facts survive file delete

Check: `cd backend && go test -count=1 -run TestDeleteDocumentKeepsAcceptedFacts -v ./internal/profile`
Exit: 0

```
=== RUN   TestDeleteDocumentKeepsAcceptedFacts
--- PASS: TestDeleteDocumentKeepsAcceptedFacts (0.02s)
PASS
ok  	github.com/w3con/sifty/internal/profile	0.350s
```

## 2026-11-12 — 1.3 delete confirmation label: cause = .profile-document .profile-document-actions button{background:#fff!important} (specificity 0,2,1) beat .profile-doc-confirm-delete{background:#b23a22!important} (0,1,0) while label stayed #fff!important; no danger-* tokens existed anywhere, four added to theme.css :root

Check: `grep -n 'profile-doc-confirm-delete' frontend/profile/profile.css; (cd frontend && npx tsc --noEmit && echo tsc-ok); cd /private/tmp/claude-501/-Volumes-SSD-Dev-tknff-sifty/5b7b8e5b-3dc7-461f-81be-b3e7064d8408/scratchpad/settings7-pw && node confirm.mjs`
Exit: 0

```
378:.profile-document .profile-document-actions button:not(.profile-doc-confirm-delete){background:#fff!important}
392:.profile-document-actions .profile-doc-confirm-delete{background:var(--danger-solid);color:var(--danger-on-solid)}
393:.profile-document-actions .profile-doc-confirm-delete:hover:not(:disabled){background:var(--danger-solid-hover)}
tsc-ok
light {"normal":{"color":"rgb(255, 255, 255)","bg":"rgb(178, 58, 34)"},"hover":{"color":"rgb(255, 255, 255)","bg":"rgb(150, 46, 25)"}}
dark {"normal":{"color":"rgb(255, 255, 255)","bg":"rgb(178, 58, 34)"},"hover":{"color":"rgb(255, 255, 255)","bg":"rgb(150, 46, 25)"}}
```

## 2026-11-12 — 1.4 browser flow on a data COPY (Playwright, throwaway binary on port 8091, models disabled in the copy)

Run on a copy of data/ with the copy's providers pointed at a dead endpoint and collection disabled; no model call, server log clean. Output of the flow script:
== profile.pdf (configured file, 4 accepted facts, no proposals): before facts {accepted:4}, file on disk true. UI notice: 'profile.pdf deleted. 4 accepted facts stay in your profile.' Tile gone, API documents no longer lists it, facts after {accepted+sourceDeleted:4}, text endpoint 404, owner's configured file still on disk (kept on purpose), tile still gone after reload (not re-imported).
== settings7-note.md (upload, seeded 1 accepted + 1 proposed fact in the copy): before facts {accepted:1, proposed:1}, file dir on disk true. Notice: '1 accepted fact stays in your profile.' Tile gone, facts after {accepted+sourceDeleted:1} (proposal removed), stored directory removed from disk, text 404, still gone after reload.
Screenshots (scratch, large): /private/tmp/claude-501/-Volumes-SSD-Dev-tknff-sifty/5b7b8e5b-3dc7-461f-81be-b3e7064d8408/scratchpad/settings7-pw/flow-profile.pdf-1-confirm.png, flow-profile.pdf-2-after.png, flow-settings7-note.md-1-confirm.png, flow-settings7-note.md-2-after.png. Small confirmation-button screenshots in this folder: delete-confirm-light.png, delete-confirm-dark.png (emulated light and dark colour scheme; the app has no dark variant of the profile screen, the button carries its own solid background and label so it reads the same: rgb(255,255,255) on rgb(178,58,34), contrast about 6:1; hover rgb(150,46,25)). Limit: the suggestions list shows only open and rejected facts, so the '(file deleted)' label on an accepted fact is carried by the API (documentDeleted) and rendered by the fact row, but accepted facts are not listed in that card today.

## 2026-11-12 — 1.4 cleanup: throwaway server stopped, data copy and binary deleted; original data/ and data-staging/ unchanged

Check: `S=/private/tmp/claude-501/-Volumes-SSD-Dev-tknff-sifty/5b7b8e5b-3dc7-461f-81be-b3e7064d8408/scratchpad; ls $S; (lsof -nP -iTCP:8091 -sTCP:LISTEN || echo 'port 8091: nothing listening'); cd /Volumes/SSD/Dev/tknff/sifty && find data data-staging -type f | sort | xargs shasum | shasum; find data data-staging -type f -print0 | sort -z | xargs -0 shasum | shasum`
Exit: 0

```
documents.go.bak
mutation.txt
settings7-pw
storage.go.bak
port 8091: nothing listening
e937d3bab09ce8f454777f70f8528703ee26e3f1  -
a6849aa0707db186aba1563f3eaefbf63ad2da5e  -
```

## 2026-11-12 — 1.4 follow-up: configured-file handling found in the browser run (files named in candidate.yaml would have been imported again after delete); re-run Go suites

Check: `cd backend && go vet ./internal/profile ./internal/store ./internal/server && go test -count=1 ./internal/profile/... ./internal/server/... 2>&1 | grep -E '^(--- FAIL|    --- FAIL|ok|FAIL)'`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/profile	0.372s
--- FAIL: TestRouteDrift (0.02s)
FAIL
FAIL	github.com/w3con/sifty/internal/server	17.664s
FAIL
```

## 2026-11-12 — 1.4 data checksum finding

Checksum of the original data/ and data-staging/ (106 files) differs between start (25b0b5d5... plain / 906e9b57... null-separated) and end (e937d3ba... / a6849aa0...). Cause is not this run: the owner's live production server (port 8080) and staging preview (port 8081) were running and writing. File times show only data/store.sqlite, data/store.sqlite-wal (12:11) and data-staging/store.sqlite-shm (12:11) changed after the run started; no other file in either directory was modified after 12:00. The run read the originals only (one cp -R) and worked on the copy.
