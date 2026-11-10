# ASM-6 — technical acceptance evidence (Q.5 historical, J.1 current)

> **Integration status,2026-11-10:** source is in main (`58af432`); this file remains the historical Q.5 and J.1 evidence record. Card is review, not done; production is not updated.

**Status:** Q.5 historical evidence, followed by the bounded J.1 acceptance described at the end of this document. Q.5 was a corrective technical handback; J.1 is the current proven-adoption evidence. Q.1–Q.4 were accepted earlier. Q.5 benchmark, real-profile calculation, browser checks and the three bounded corrections (B legacy profile status, C ambiguity classification, D profile-edit scope) are implemented and verified; B/C/D were closed after the corrections recorded below. This is **not** owner acceptance, not merged to `main`, not released to production and not `done`. J.1 now demonstrates normal persisted scored adoption on a fresh isolated read-only snapshot of data-staging with the actual confirmed profile `local@20`.

- Branch/worktree: `work/ASM-6` in `.worktrees/ASM-6`
- Q.5 source revision built: `a4fe608df4ff3fd84992e8d2b3b0eab1f770bcfc`; accepted branch head is `8677836`. J.1 built the merged Design delta on top of that head and reviewed it at PID `22058` (see the J.1 section).
- Runtime: PID `29765` (restarted from `54157` after the corrective build), `http://127.0.0.1:8083`, own binary `.local/collection1-review/sifty`, own data `/var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/opencode/app8-iso/data`, `frontend/dist`, `backend/workers`
- No `8080`, `8081` or `8082` process was changed.

## 1. Runtime reuse, migration and preservation

The pre-existing 8083 process (`PID 39932`) was verified first: cwd was this worktree, binary under `.local/collection1-review`, data under the isolated `app8-iso` directory. It was stopped; only that verified PID was stopped. A consistent copy of the isolated database was preserved with SQLite `.backup` at `.local/collection1-review/backup-pre-q5.sqlite` (200,810,496 bytes). The new binary replaced the old one and was restarted with the same argv, port and data.

Schema migrated **18 → 20** on the isolated review database only (`match_job_inputs`, `match_job_facts`, `match_job_results` created). Startup recomputation produced 2,769 retained inputs, 2,769 fact rows and 2,769 result rows.

Before/after SHA-256 over ordered rows (`.local/qm-bench/before.txt`, `.local/qm-bench/after.txt`):

| Data | Rows | Before | After |
|---|---|---|---|
| `user_profiles` | 1 | `5a61333a…85ef` | identical |
| `profile_revisions` | 1 | `ea484231…95e8` | identical |
| `workflow_history` (incl. Apply reason) | 345 | `e905a1c8…6627` | identical |
| `assessments` | 1,493 | `9e31e0fd…ad11` | identical |
| `assessment_queue` (one `pending` row) | 1 | `1974e5b2…159d` | identical |
| `collection_runs` | 15 | `64e65042…2764` | identical |
| `jobs` (title/company/location/publication basis/status/Apply fields) | 2,769 | `b71bcd1f…f1c7` | identical |
| `profile_documents` | 8 | `b617db65…1c1` | identical |

Queue remained disabled/preserved: `assessment_enabled=false`, `config.enabled=false`, one `pending` row, no new assessment or collection run. No model, search, collection or data-refresh call was made.

## 2. Benchmark — 300 saved jobs, no network/models

Runner: temporary `backend/internal/qmbench` (removed before the handback commit) on a **copy** of the migrated review database. Extraction, pure evaluation and the full application effect were timed separately. Input selection: 300 deterministically spread saved roles with text, 1,584,529 bytes of retained title/company/description. Hardware/runtime: darwin/arm64, 14 CPUs, Go 1.26, existing `runtime.yaml`, rules `quickmatch.rules/1`, source policy `source-policy/1+2f62927a…abc2d2`.

**The review runtime's persisted profile** is `local@1` and typed but empty — an old test copy. It is not the owner's current profile. Extraction is still real; every persisted result in that review database is honestly `unscored`.

| Phase | Total | p50 | p95 | max | Result |
|---|---|---|---|---|---|
| Extraction | 92.24 ms | 0.286 ms | 0.438 ms | 0.781 ms | 300 facts |
| Pure evaluation | 2.80 ms | 0.003 ms | 0.034 ms | 0.389 ms | 0 scored / 300 unscored |
| Full fresh (read→extract→save facts→evaluate→save) | 300.93 ms | 0.852 ms | 1.257 ms | 20.766 ms | 300 unscored |
| Full warm (reuse facts) | 176.42 ms | 0.507 ms | 0.710 ms | 17.702 ms | 300 unscored |
| GET jobs decoration (2,273 roles) | 490.51 ms | — | — | — | 2,273 decorated, 2,273 unscored |

**Clearly labelled synthetic fixture profile** (revision 2, on a separate copy; never written to the review database, never claimed as real-profile adoption): 0.35–0.40 ms/job extraction, 0.008–0.016 ms/job evaluation, **353.24 ms fresh** and **160.10 ms warm** for 300 scored jobs (p95 1.34/0.74 ms), all 300 scored (`avg_score=15.49`). This is well inside ≤0.5 s/job; the 300-job batch is far below the 30 s target.

Raw output: `.local/qm-bench/bench-actual.txt`, `.local/qm-bench/bench-fixture.txt`.

### 2b. Real current profile — read-only, no staging write

The owner's current profile was read with a read-only SQLite connection directly from `data-staging/store.sqlite` (`mode=ro`; **not** `store.Open`, which would migrate), decoded into the existing typed profile, and evaluated with the pure packages against 300 already stored isolated-review facts. Nothing was written to staging or to the review profile, and no raw profile JSON is committed.

Aggregate: profile `local`, revision **20**, 3 target roles, 11 skills, cash minimum 90000 EUR. Evaluation of 300 saved jobs: **300 scored, 0 unscored, average score 9.01, average coverage 29.4%**; all 300 scored results had coverage in the 25–50% bucket. This proves the actual-profile calculation path works, but only as a read-only calculation — it is not the review runtime's persisted catalogue.

Samples (vacancy titles are job data, not candidate data; title comparison is described generically):

| Job ID | Vacancy title | Score | Coverage | Role-title comparison |
|---|---|---|---|---|
| 1 | [Hiring] Senior Back-end Developer @Lemon.io | 25 | 29.4% | title_match |
| 10 | AI Engineer | 0 | 29.4% | title_match |
| 21 | Senior/Staff Software Engineer | 33 | 29.4% | title_match |

Raw aggregate output: `.local/qm-bench/real-profile-eval.txt`. A single computed result was also used as a clearly labelled browser response overlay, not persisted and not claimed as normal adoption.

## 3. Actual Chromium review (existing Playwright installation)

Script: `.local/qm-bench/review-q5.mjs` against the live 8083 runtime; non-GET app-mount/selection requests were answered with harmless 200 JSON stubs (`POST /api/search`, `POST /api/jobs/*/seen`) so no write occurred and no failed-fetch banner appeared. Result JSON: `.local/qm-bench/browser-q5.json`. Screenshots are in the card at `evidence/`.

Actual catalogue/profile (real data):

- 1,900 visible Inbox rows, all rendered as unscored (`—`), none faked as a score.
- Overview shows **Not scored**, the explicit "not a zero" explanation, 0% evidence coverage and the original listing text (2,922 characters on the sampled role).
- Profile navigation opens the existing profile; its Skills section is present.
- Screenshots: `evidence/q5-actual-catalogue-unscored.png`, `evidence/q5-actual-profile-edit.png`.

Labelled response fixture (real evaluator output from the synthetic profile, attached only to the browser response; **not** stored in the review database and not claimed as a real score):

- Scored row and panel show **67/100, Current, Partial fit, 29% evidence coverage**.
- Criterion comparisons show six rows, including profile value, vacancy value, withheld/unknown values, source field/quote and utility.
- Six actionable unknowns; **Edit Skills** opened the existing `#profile-skills` section.
- A genuine evaluated **0** renders as `0`, distinct from the unscored `—`.
- A stale result renders as the previous score (class `previous`), not as current.
- Screenshots: `evidence/q5-fixture-overview-scored.png`, `evidence/q5-fixture-profile-edit.png`, `evidence/q5-fixture-zero-and-list.png`.
- Real-profile overlay (clearly labelled, not persisted): job 21 renders its real computed **33/100** with 29% coverage and criterion comparisons; screenshot `evidence/q5-real-profile-overview.png`.

Q.5 ran 12 browser checks; the only console message was the expected `net::ERR_FAILED` from deliberately blocked POSTs. The J.1 merged-design rerun ran 14 checks with **0 failures and 0 console errors**; non-GET requests were fulfilled with inert 200 stubs rather than aborted.

## 4. Boundaries and code touched

Only documentation/evidence was changed in the historical Q.5 pass; J.1 then merged the finished Design frontend delta and re-ran real-browser acceptance. The implemented modules remain separated: `matchcontract` (domain contract), `jobextract` (extractor + source policy, contract-only), `quickmatch` (pure evaluator, contract-only), `matchsvc` (profile projection, configuration, effects/CAS persistence), existing store/API and the presentational `QuickMatchPanel`. No hardcoded profile, mockup data, machine path, port or weight was added to product code.

## 5. Failures and corrections during Q.5

- First fixture run omitted `-fixture`; it therefore repeated the actual-profile measurement. Re-run with the flag.
- Fixture profile used `hours.mode="full-time"`; the typed contract requires `full_time`. Corrected and re-run.
- Browser walk initially used a non-existent `.profile-screen` assumption for a header menu; and the Edit Skills assertion read before the profile finished loading. Added the menu item and `#profile-skills` wait; all checks then passed.
- The fixture route stayed in flight while closing the context; the report was already complete, and the close error was handled so the evidence run exits 0.
- No speculative harness or product fix was added.

## 6. Remaining limitations and findings

- **Profile distinction (historical Q.5 vs current J.1).** In the historical Q.5 runtime the persisted profile was an old empty test copy (`local@1`); Q.5 scored only via a synthetic fixture and a read-only real-profile overlay, so normal persisted adoption was unproven then. J.1 supersedes that limitation: a fresh read-only snapshot of `data-staging` preserving the actual confirmed profile `local@20` (3 target roles, 11 skills, complete cash compensation) was served normally, migrated 18→20, and persisted 2,271 visible Quick Match results against revision 20 (1,539 evaluated zeros, 732 nonzero, none stale). The current source therefore does contain proven real-profile adoption; the Q.5 fixture remains historical evidence only.
- **Corrected item B (verified).** `profile.Service.decorate` no longer returns before the ranking block: context status is computed independently and `fastRankingCurrent` always reflects `jobs.match_profile_revision` (the existing compatibility revision mirrored by the Quick Match CAS). `refreshDerived` collects the context error and still runs Quick Match recalculation. `ProfileView` renders “Quick ranking up to date” plus a separate candidate-context warning instead of calling Quick ranking stale. Live check: “Quick ranking up to date | Candidate context is not current: stored context predates the confirmed profile; refresh context. This is separate from Quick ranking.”
- **Corrected item C (verified).** `matchcontract.Ambiguity` now carries `kind`; the extractor emits exactly two kinds (`missing_structure` for no retained heading, `source_conflict` for board/text disagreement) and the extractor version is `jobextract/3`. The evaluator appends a Conflict only for `source_conflict`; every ambiguity remains an Unknown with the generic reason “Source evidence is incomplete or ambiguous.” Live check on an actual role: 0 conflicts, 1 unknown, no “Sources disagree”.
- **Corrected item D (verified).** Unknowns receive `profileEdit` only for skill-not-evidenced, compound skill requirement, authorization-profile-missing and authorization-status-missing. Source-fact gaps and unsupported comparisons expose no profile edit button. Live check: the sampled role has 0 edit buttons. Years/level copy now says the comparison is unsupported, not that the owner lacks evidence.
- v1 limitations in the spec: narrow explicit headings/labels, no inferred years/semantic match/broad aliases, no currency/tax inference, unsupported country names/regions stay unknown. Calibration, multiple profiles and model-queue redesign remain excluded; the queue is known unresolved and is not claimed to be selected-only.
- Broad suites remain skipped by default; default `bin/work-test` is reported separately.

## 7. Elapsed and next action

Q.5 initial pass actual elapsed: approximately 11 minutes (12:58Z–13:09Z), correcting the earlier 17-minute claim. This bounded corrective pass started 13:11Z; the real-profile read-only evaluation and the 200-stub browser rerun were completed by about 13:15Z. The coordinator subsequently accepted the B/C/D corrections in `8677836`; they are not open. Q.5 remains subject only to the normal owner/acceptance boundary, not to an undisclosed code item.

## J.1 — merged Design source and normal persisted adoption

J.1 merged the finished DESIGN-3 delta `a46b9fb..2f6c026` into this branch (14 files, three reconciled overlaps: RoleContent fact strip moved to the tabs row while keeping the real Quick Match panel; main.tsx Design header/Settings plus our Quick Match/refresh/sorting; theme.css both blocks). Backend build and frontend `tsc --noEmit && vite build` completed with exit 0.

A consistent SQLite `.backup` of `/Volumes/SSD/Dev/tknff/sifty/data-staging/store.sqlite` was taken through a read-only connection into the ignored review directory `.local/asm6-j1-review/data` (snapshot `c0f741de…7491`, 251,564,032 bytes). Source file/config SHA-256 values were identical immediately before and after the snapshot and after the review: `store.sqlite 24e5fb07…c1e8`, `store.sqlite-wal 1078a23a…4122`, and all five config files unchanged. Only the isolated copy was opened by the server; the source was never opened for writing.

On normal startup the isolated copy migrated schema 18→20 and produced `match_job_inputs`/`match_job_facts`/`match_job_results` = 2,774 rows each. The API returned 2,271 visible jobs, **all with a numeric persisted result against `profile_revision=20`**, none stale: 1,539 evaluated zeros and 732 nonzero scores; 0 null/unscored and 0 missing envelopes in this catalogue. This replaces the earlier unproven adoption evidence, which used the old empty `app8-iso` profile.

Logical before/after hashes of owner/revisions/workflow/queue/collection/sightings tables were identical. The only `jobs` differences were derived `match_score` (2,058 rows) and one `seen_at` value that existed in the snapshot at backup time but is absent from the live source now; source bytes were unchanged across the snapshot/review window, so this is a live-source/preview artifact, not a write by this review server. Queue remained `assessment_enabled=false`; no model, collection or search request reached the server because browser non-GET calls were inert 200 stubs (`POST /api/search`, `POST /api/jobs/84/seen`).

**Next action:** coordinator review/acceptance of the corrected Q.5 evidence; then the normal release decision. Do not merge, release, tick the card `done` or claim owner acceptance from this document.
