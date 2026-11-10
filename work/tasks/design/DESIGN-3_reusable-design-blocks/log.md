# DESIGN-3 log

2026-11-09T17:26Z — owner «хорошо, делай одним прогоном» approves coordinator's corrected25–45min full appearance/documents/photo outcome. Previous115–190min estimate withdrawn. One executor and combined D.1, then one final review. Stop18:11Z. Baseline accepted SETTINGS-3 at7f5c0a3. Next: delegated isolated bootstrap/live brief, same executor implements D.1 in one pass. Actual live staging revision must be read fresh; never restore old snapshots over owner writes.

## 2026-11-09 — D.1 backend build and avatar critical test

Check: `cd backend && go build ./... && go test ./internal/profile -run '^TestAvatarReplacementFailurePreservesPrevious$' -count=1`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/profile	0.422s
```

## 2026-11-09 — D.1 locked frontend build

Check: `cd frontend && npm ci && npm run build`
Exit: 0

```

added 133 packages, and audited 134 packages in 2s

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
dist/assets/index-A64XKHKq.css                                 61.69 kB │ gzip:  12.63 kB
dist/assets/index-FSvbtpaC.js                                 894.43 kB │ gzip: 275.52 kB

(!) Some chunks are larger than 500 kB after minification. Consider:
- Using dynamic import() to code-split the application
- Use build.rollupOptions.output.manualChunks to improve chunking: https://rollupjs.org/configuration-options/#output-manualchunks
- Adjust chunk size limit for this warning via build.chunkSizeWarningLimit.
✓ built in 1.19s
```

## 2026-11-09 — D.1 real Chromium desktop/mobile, uploads, avatar and shared-style proof

Check: `node .local/design-3/review.cjs`
Exit: 0

```
{"checks":{"identity_rendered":true,"avatar_initial_fallback":true,"user_icon_nav":true,"desktop_two_columns":true,"desktop_53_47_ratio":true,"desktop_no_horizontal_scroll":true,"shared_block_at_least_two":true,"shared_style_reverted":true,"add_files_visible":true,"dropzone_visible":true,"multi_upload_success_kept":true,"multi_upload_individual_failure":true,"drop_upload_saved":true,"no_model_or_analyze_during_uploads":true,"no_unexpected_writes_during_uploads":true,"uploaded_document_not_analyzing":true,"avatar_uploaded":true,"no_model_or_analyze_during_avatar":true,"no_unexpected_writes_during_avatar":true,"avatar_persists_after_reload":true,"avatar_replaced":true,"server_rejects_fake_image":true,"failed_replace_preserves_photo":true,"avatar_deleted":true,"avatar_delete_persists":true,"profile_revision_unchanged":true,"profile_data_hash_unchanged":true,"avatar_upload_preserved_profile":true,"mobile_single_column":true,"mobile_full_width":true,"mobile_no_horizontal_scroll":true,"no_unexpected_page_errors":true},"failures":[],"ratio":0.529997723132969,"desktopCols":{"left":{"x":160,"y":309.359375,"w":581.9375},"right":{"x":763.9375,"y":309.359375,"w":516.0625},"scroll":1440,"client":1440},"mobile":{"left":{"x":12,"y":447.640625,"w":366},"right":{"x":12,"y":6621.640625,"w":366},"scroll":390,"client":390},"modelRequests":["search","search","search","search"],"unexpectedWrites":[],"consoleErrors":["Failed to load resource: the server responded with a status of 400 (Bad Request)"],"beforeRevision":14,"afterRevision":14}
```

## 2026-11-09 — D.1 read-only live staging browser verification

Check: `node .local/design-3/live-review.cjs`
Exit: 0

```
{"checks":{"identity_rendered":true,"user_icon_nav":true,"add_files_visible":true,"dropzone_visible":true,"avatar_fallback":true,"desktop_two_columns":true,"desktop_no_horizontal_scroll":true,"edit_cancel_only":true,"profile_revision_unchanged":true,"profile_hash_unchanged":true,"avatar_not_set_before_owner_use":true,"mobile_single_column":true,"mobile_no_horizontal_scroll":true,"no_page_errors":true},"failures":[],"blocked":["POST http://localhost:8081/api/search","POST http://localhost:8081/api/jobs/854/seen","POST http://localhost:8081/api/search","POST http://localhost:8081/api/jobs/854/seen"],"beforeRevision":14,"afterRevision":14,"desktop":{"left":{"x":160,"y":309.359375,"w":581.9375},"right":{"x":763.9375,"y":309.359375,"w":516.0625},"scroll":1440,"client":1440},"mobile":{"left":{"x":12,"y":447.640625,"w":366},"right":{"x":12,"y":5298.640625,"w":366},"scroll":390,"client":390},"pageErrors":[]}
```

## 2026-11-09 — D.1 staging data preservation

Check: `python3 .local/design-3/preservation_check.py /Volumes/SSD/Dev/tknff/sifty/.local/design-3-staging-snapshot-20261109T173957Z`
Exit: 0

```
PASS snapshot_recoverable
PASS profile_revision_preserved
PASS profile_data_hash_preserved
PASS profile_revisions_count_preserved
PASS documents_preserved
PASS facts_preserved
PASS prod_pid_unchanged
PASS prod_version_unchanged
PASS prod_health_200
PASS api_health_200
PASS api_served_generation
PASS api_served_source
PASS api_profile_revision_14
PASS vite_worktree_design3
PASS avatar_not_set_on_staging
SNAPSHOT /Volumes/SSD/Dev/tknff/sifty/.local/design-3-staging-snapshot-20261109T173957Z
listeners prod=79846 vite=64422 api=62782
staging profile revision 14 data_sha256 0a8e5b33d8cdc089aa6064af5a9dc7d043732d694e6a859eed865ea21c9fccb3 documents 8 facts 10
api generation g0001-7729a711a0 fingerprint 7729a711a07008dfedec5e1611a5cad5bdbe23dc0171c2eb9eb722266d840777
prod version {"version":"53c519aa8608-20261106T221830.347654Z"} health 200
RESULT PASS 15 / 15
```

## 2026-11-09 — note

D.1 implemented in one pass. Backend: additive avatar persistence under the existing data directory with server-side actual-image validation (PNG/JPEG/WebP) and configurable limit/types from config/runtime.yaml; GET meta, GET bytes, POST replace, DELETE routes; failed replacement keeps the previous file. Frontend: 53/47 two-column Profile composition on warm off-white with white cards, shared SectionShell/Tags/CheckGroup/Field blocks, user icon in navigation, profile header with persistent photo/initials, visible Add files plus dropzone using the existing upload path, per-file results, explicit Analyze preserved, and left/right section order from the approved plan. Isolated Chromium on a WAL-safe copy of current staging revision 14: 32/32 checks, including desktop/mobile columns, multi-upload with one invalid file kept success, drop upload, avatar upload/reload/replace/invalid-replace/delete, unchanged profile revision/hash, and shared-style change affecting multiple cards then reverted. Backend build + named avatar critical test pass; npm ci + frontend build pass. Staging snapshot .local/design-3-staging-snapshot-20261109T173957Z (revision 14, integrity ok, recoverable). Controlled retarget: 8082 API preview .local/preview-api-design-3-final generation g0001-7729a711a0 (worktree DESIGN-3), 8081 Vite from DESIGN-3; read-only live browser 14/14 and preservation 15/15. Production PID 79846/version 53c519aa8608-20261106T221830.347654Z unchanged. Finding for review/backlog: the pre-existing mobile global header places Profile and Settings in the same grid cell, so they overlap; verification opens Profile by DOM activation. Not changed because global navigation outside this card's approved boundary.

## 2026-11-09 — D.1 default test status

Check: `python3 bin/work-test`
Exit: 0

```
SKIPPED: automated tests are paused during rapid prototyping. Use bin/work-test --full to run them explicitly.
```

## 2026-11-09 — D.1 post-commit live staging browser verification

Check: `node .local/design-3/live-review.cjs`
Exit: 0

```
{"checks":{"identity_rendered":true,"user_icon_nav":true,"add_files_visible":true,"dropzone_visible":true,"avatar_fallback":true,"desktop_two_columns":true,"desktop_no_horizontal_scroll":true,"edit_cancel_only":true,"profile_revision_unchanged":true,"profile_hash_unchanged":true,"avatar_not_set_before_owner_use":true,"mobile_single_column":true,"mobile_no_horizontal_scroll":true,"no_page_errors":true},"failures":[],"blocked":["POST http://localhost:8081/api/search","POST http://localhost:8081/api/jobs/854/seen","POST http://localhost:8081/api/search","POST http://localhost:8081/api/jobs/854/seen"],"beforeRevision":14,"afterRevision":14,"desktop":{"left":{"x":160,"y":309.359375,"w":581.9375},"right":{"x":763.9375,"y":309.359375,"w":516.0625},"scroll":1440,"client":1440},"mobile":{"left":{"x":12,"y":447.640625,"w":366},"right":{"x":12,"y":3705.1875,"w":366},"scroll":390,"client":390},"pageErrors":[]}
```

## 2026-11-09 — D.1 post-commit staging preservation

Check: `python3 .local/design-3/preservation_check.py /Volumes/SSD/Dev/tknff/sifty/.local/design-3-staging-snapshot-20261109T173957Z`
Exit: 0

```
PASS snapshot_recoverable
PASS profile_revision_preserved
PASS profile_data_hash_preserved
PASS profile_revisions_count_preserved
PASS documents_preserved
PASS facts_preserved
PASS prod_pid_unchanged
PASS prod_version_unchanged
PASS prod_health_200
PASS api_health_200
PASS api_served_generation
PASS api_served_source
PASS api_profile_revision_14
PASS vite_worktree_design3
PASS avatar_not_set_on_staging
SNAPSHOT /Volumes/SSD/Dev/tknff/sifty/.local/design-3-staging-snapshot-20261109T173957Z
listeners prod=79846 vite=64422 api=72004
staging profile revision 14 data_sha256 0a8e5b33d8cdc089aa6064af5a9dc7d043732d694e6a859eed865ea21c9fccb3 documents 8 facts 10
api generation g0001-7729a711a0 fingerprint 7729a711a07008dfedec5e1611a5cad5bdbe23dc0171c2eb9eb722266d840777
prod version {"version":"53c519aa8608-20261106T221830.347654Z"} health 200
RESULT PASS 15 / 15
```

## 2026-11-09 — D.1 correction backend build and real-image critical test

Check: `cd backend && go build ./... && go test ./internal/profile -run '^TestAvatarReplacementFailurePreservesPrevious$' -count=1`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/profile	0.250s
```

## 2026-11-09 — D.1 correction frontend build

Check: `cd frontend && npm run build`
Exit: 0

```

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
dist/assets/index-B02X5GVC.css                                 61.96 kB │ gzip:  12.69 kB
dist/assets/index-BQMWT4WL.js                                 894.58 kB │ gzip: 275.53 kB

(!) Some chunks are larger than 500 kB after minification. Consider:
- Using dynamic import() to code-split the application
- Use build.rollupOptions.output.manualChunks to improve chunking: https://rollupjs.org/configuration-options/#output-manualchunks
- Adjust chunk size limit for this warning via build.chunkSizeWarningLimit.
✓ built in 1.15s
```

## 2026-11-09 — D.1 correction targeted Chromium recheck

Check: `node .local/design-3/review.cjs`
Exit: 0

```
{"checks":{"identity_rendered":true,"avatar_initial_fallback":true,"user_icon_nav":true,"desktop_two_columns":true,"desktop_53_47_ratio":true,"desktop_no_horizontal_scroll":true,"shared_block_at_least_two":true,"shared_style_reverted":true,"identity_save_error_visible":true,"identity_draft_retained":true,"identity_cancel_restores":true,"add_files_visible":true,"dropzone_visible":true,"multi_upload_success_kept":true,"multi_upload_individual_failure":true,"drop_upload_saved":true,"no_analyze_during_uploads":true,"no_search_attempt_during_uploads":true,"no_unexpected_writes_during_uploads":true,"uploaded_document_not_analyzing":true,"avatar_uploaded":true,"no_analyze_during_avatar":true,"no_search_attempt_during_avatar":true,"no_unexpected_writes_during_avatar":true,"avatar_persists_after_reload":true,"avatar_replaced":true,"server_rejects_fake_image":true,"failed_replace_preserves_photo":true,"avatar_deleted":true,"avatar_delete_persists":true,"profile_revision_unchanged":true,"profile_data_hash_unchanged":true,"avatar_upload_preserved_profile":true,"mobile_single_column":true,"mobile_full_width":true,"mobile_no_horizontal_scroll":true,"mobile_settings_real_click":true,"no_unexpected_page_errors":true},"failures":[],"ratio":0.529997723132969,"desktopCols":{"left":{"x":160,"y":309.359375,"w":581.9375},"right":{"x":763.9375,"y":309.359375,"w":516.0625},"scroll":1440,"client":1440},"mobile":{"left":{"x":12,"y":447.640625,"w":366},"right":{"x":12,"y":7283.140625,"w":366},"scroll":390,"client":390},"unexpectedWrites":[],"consoleErrors":["Failed to load resource: the server responded with a status of 400 (Bad Request)","Failed to load resource: the server responded with a status of 400 (Bad Request)"],"analyzeAttempts":[],"blockedSearchAttempts":["POST http://127.0.0.1:18093/api/search","POST http://127.0.0.1:18093/api/search","POST http://127.0.0.1:18093/api/search","POST http://127.0.0.1:18093/api/search"],"blockedSearchAttemptsNote":"Attempted automatic App-mount search, intercepted and fulfilled by the review route; no search/model call executed.","beforeRevision":14,"afterRevision":14}
```

## 2026-11-09 — note

Final acceptance corrections: (1) ProfileHeader now renders the identity draft error with role=alert beside the editor; isolated Chromium failed-save check confirms the draft and visible error remain and Cancel restores the stored name. (2) Header actions are wrapped in .header-actions with non-overlapping mobile grid placement; real Playwright clicks now open Profile and Settings on mobile, and there is no horizontal overflow. (3) Profile header and document accept lists come only from capabilities; photo selection stays disabled until avatar capabilities load, with no duplicated type fallback. (4) Avatar validation now fully decodes PNG/JPEG/WebP (golang.org/x/image/webp) so signature-only or truncated data cannot replace a valid photo; the existing named critical test additionally covers signature-only/truncated PNG/JPEG and a real metadata-write failure, with the previous photo preserved. (5) Isolated evidence now records analyzeAttempts=[] and blockedSearchAttempts as attempted automatic App-mount searches intercepted by the review route; it does not claim zero attempts. No live owner photo/profile mutation; staging revision remains 14.

## 2026-11-09 — D.1 correction live staging browser recheck

Check: `node .local/design-3/live-review.cjs`
Exit: 0

```
{"checks":{"identity_rendered":true,"user_icon_nav":true,"add_files_visible":true,"dropzone_visible":true,"avatar_fallback":true,"desktop_two_columns":true,"desktop_no_horizontal_scroll":true,"edit_cancel_only":true,"profile_revision_unchanged":true,"profile_hash_unchanged":true,"avatar_not_set_before_owner_use":true,"mobile_single_column":true,"mobile_no_horizontal_scroll":true,"mobile_settings_real_click":true,"no_page_errors":true},"failures":[],"blocked":["POST http://localhost:8081/api/search","POST http://localhost:8081/api/jobs/854/seen","POST http://localhost:8081/api/search","POST http://localhost:8081/api/jobs/854/seen"],"blockedNote":"Non-GET requests blocked during read-only live review; automatic App-mount search was intercepted and fulfilled, not executed.","beforeRevision":14,"afterRevision":14,"desktop":{"left":{"x":160,"y":309.359375,"w":581.9375},"right":{"x":763.9375,"y":309.359375,"w":516.0625},"scroll":1440,"client":1440},"mobile":{"left":{"x":12,"y":447.640625,"w":366},"right":{"x":12,"y":3705.1875,"w":366},"scroll":390,"client":390},"pageErrors":[]}
```

## 2026-11-09 — D.1 correction staging preservation and final generation

Check: `python3 .local/design-3/preservation_check.py /Volumes/SSD/Dev/tknff/sifty/.local/design-3-staging-snapshot-20261109T173957Z`
Exit: 0

```
PASS snapshot_recoverable
PASS profile_revision_preserved
PASS profile_data_hash_preserved
PASS profile_revisions_count_preserved
PASS documents_preserved
PASS facts_preserved
PASS prod_pid_unchanged
PASS prod_version_unchanged
PASS prod_health_200
PASS api_health_200
PASS api_served_generation
PASS api_served_source
PASS api_profile_revision_14
PASS vite_worktree_design3
PASS avatar_not_set_on_staging
SNAPSHOT /Volumes/SSD/Dev/tknff/sifty/.local/design-3-staging-snapshot-20261109T173957Z
listeners prod=79846 vite=64422 api=12715
staging profile revision 14 data_sha256 0a8e5b33d8cdc089aa6064af5a9dc7d043732d694e6a859eed865ea21c9fccb3 documents 8 facts 10
api generation g0001-b5431a7ac3 fingerprint b5431a7ac3e0439e9b8d6adf643c2cc9020aad8aaf40fa7c1ef53fde5eb06171
prod version {"version":"53c519aa8608-20261106T221830.347654Z"} health 200
RESULT PASS 15 / 15
```

## 2026-11-09 — note

Runtime transition incident during correction verification: restarting the existing API preview state after the clean correction commit hit the known preview generation-reuse failure (backend reported g0006-b5431a7ac3 while the new build expected g0001-b5431a7ac3; supervisor state=failed though the old generation still answered). Fix: stopped only the verified supervisor and started the same DESIGN-3 worktree/data/port with a fresh state directory .local/preview-api-design-3-corrections. Final served generation g0001-b5431a7ac3, commit cae126e, dirty=false, health ok; data-staging revision 14 unchanged and production untouched. This is a harness transition issue, not an application or data defect; no refresh or retarget of owner data was performed.

## 2026-11-09 — note

Final resume: D.1 closed and technically accepted through application commit cae126e; acceptance.md records the coordinator's review, 38/38 isolated and 15/15 live checks, served generation g0001-b5431a7ac3, preserved revision 14 data and unchanged production. Owner review on staging is the next step. No live executor remains after this handback and no further implementation is authorized. Integration, release, merge, push and done.md are not performed. Actual elapsed from 17:26Z to 17:53Z is about 27 minutes, within the corrected 25–45 minute estimate. The known preview generation-reuse incident and fresh-state correction are recorded in the worktree's ignored work/incidents.md.
