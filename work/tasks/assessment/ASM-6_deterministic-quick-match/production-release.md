# ASM-6 production release and selective profile transfer — 2026-11-10

## Approved source and live runtime

- Approved pinned source: `157fe9f6d0f86e91b9101150087ea2aa19fecaf4` (full main SHA; main working tree remains dirty only with unrelated documentation).
- Release command: `python3 bin/sift-promote --source 157fe9f6d0f86e91b9101150087ea2aa19fecaf4 --approve 157fe9f6d0f86e91b9101150087ea2aa19fecaf4` (normal adapter, no `--adopt-legacy`).
- Live URL: <http://localhost:8080>; recorded PID `71548`, binary `<main>/bin/sifty`, data `<main>/data`, dist `<main>/frontend/dist`, workers `<main>/backend/workers-prod`, port `8080`.
- Build/version: `157fe9f6d0f8-20261110T151309.703877Z`; `GET /api/version` matches; `GET /api/health` returns HTTP 200.
- Recovery points: `.local/deploy/prod/recovery/prod-20261110T151150.318061Z-157fe9f6` and `.local/deploy/prod/recovery/prod-20261110T151309.*` (adapter-owned complete artifact/DB backups). An extra consistent pre-import DB snapshot is at ignored `.local/ops/ASM-6-release-20261110/prod-pre-profile-import.sqlite`.
- Release receipt and deploy state both record source `157fe9f...`, result `OK`.

## Profile transfer scope

Read-only consistent snapshot of `data-staging/store.sqlite` (SQLite backup API; staging left untouched, 8081/8082 untouched). Imported only profile `local` and its provenance:

- `user_profiles`: 1 row, revision 20.
- `profile_revisions`: 20 rows.
- `profile_documents`: 8 rows (2 stored PDFs, 4 remote-profile records, 2 missing-source Markdown entries) exactly as staging.
- `profile_facts`: 22 rows, including fact states/supersedes.
- Referenced stored files: `cv.pdf` and `profile.pdf`; production already held byte-identical copies (`4e7711...`, `143b48...`), so no overwrite occurred.
- Avatar: `profile-avatar/e7b59621-...jpg` (76,859 bytes) and `profile-avatar.json` copied into production-owned `data/`; no collision existed.

The import ran in one target transaction with foreign-key check; it would have refused non-empty independent profile edits and any referenced-file hash collision. No whole staging database, queue, config, catalogue, collection history or model assessment was copied. Provenance `source_reference` values reference documents/versions, not staging filesystem paths, so no path rewrite was needed.

## Preservation evidence

Compared against the pre-release adapter recovery snapshot:

- Jobs: same 2777 rows; all columns except the approved derived `match_score`/added `match_profile_revision` are byte-identical by ordered hash (`490df205...` both).
- `assessments`: 1493 rows unchanged (`9e31e0fd...`).
- `workflow_history`: 336 rows unchanged on all original columns (`30eb1348...`); schema added a nullable `reason` column only.
- `application_revisions` 117, `application_drafts` 109 unchanged.
- `assessment_queue` 2497 unchanged; `queue_state` unchanged: `assessment_enabled=false`, collector config unchanged. No model, search or collection request executed.
- `config/*.yaml` unchanged. `cv.pdf`/`profile.pdf` hashes match staging.

Live edits during the window: none observed. The only writes were the approved profile import and derived Quick Match rows; the browser's automatic `POST /api/jobs/{id}/seen` was intercepted in the page and never reached the server.

## Quick Match result

Startup recalculation ran locally over all 2777 saved jobs after the restart: 2777 persisted results, all `scored`, average 10.38, minimum 0, maximum 100. Distribution: 1898 zero, 837 in 1–49, 38 in 50–69, 3 in 70–89, 1 in 90–100. All rows carry profile revision 20. No assessment deletion was needed.

Old BM25 `match_score` distribution for the same 2777 IDs (captured before overwrite): average 7.72; 1073 zero, 1687 in 1–49, 14 in 50–69, 2 in 70–89, 1 in 90–100. The distributions differ as expected between corpus-relative keyword matching and profile-fact scoring; this is **not** a paired quality benchmark and proves no ranking superiority.

## Browser and API checks

Existing Playwright stack; `POST /api/search` intercepted and fulfilled without reaching collection; all non-GET API calls intercepted and recorded (none essential). Results:

- Brand `Sifty`, title `Sifty · Job triage`, 1909 visible rows in Inbox; no console errors.
- Job 84: numeric `100/100`, state `Current`, evidence coverage 29%.
- Job 977: numeric `0/100`, state `Current`, evidence coverage 29% — a real zero, not "not scored".
- Profile: `local` revision 20, 8 documents visible including `profile.pdf`, `cv.pdf`, `positioning.md`, `faq.md`.
- `POST /api/search` was never delivered (1 interception); only auto-seen writes were intercepted (2 in the first pass, 1 in the zero pass). No model/assessment call.
- Screenshots: `evidence/production-quickmatch.png`, `evidence/production-quickmatch-zero.png`; scrubbed API counts: `evidence/production-api-counts.json`.

Unscored UI state could not be exercised with actual data because production currently reports 0 unscored rows; that distinction was covered earlier by focused tests and is not claimed as observed here.

## Timing (engineering observation)

Operational start 15:11:49Z. First pinned release completed 15:12:01Z (~12 s, warm build cache). Profile snapshot/validation/atomic import completed 15:13:05Z (~5 s after validation corrections). Restart through the same adapter completed 15:13:18Z (~9 s). Startup recalculation had persisted all 2777 results by 15:13:22Z (≤4 s observed after restart; overall ~13 s after process start). Browser pass and records followed. Total implementation/handback within the coordinator's corrected 5–10 minute operation bound; owner waiting excluded.

## Limitations and issues

- No ranking-quality validation: old and new scores use different inputs and semantics; distributions are descriptive only.
- Unscored-state rendering was not observed on this production data because no job is unscored.
- Browser checks intentionally intercepted auto-seen writes, so read state was not modified by verification.
- The first release and the restart both minted build IDs from the same pinned source; the live build is the `151309` one.
- `docs/MAP.md` retains an unrelated pre-existing dirty paragraph; it was excluded from staging by index-blob construction, not leaked or reverted.
