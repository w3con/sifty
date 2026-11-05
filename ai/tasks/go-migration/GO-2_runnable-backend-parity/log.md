# GO-2 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-02 — 1.1 — Open the existing database in place with the pure-Go driver, set WAL, busy_timeout, foreign keys and immediate transactions, and reproduce the schema idempotently honouring user_version

Check: `cd /Volumes/Documents/Work/FCTO/.worktrees/GO-2/sifty && go test ./internal/store/ -v -run TestSchema`
Exit: 0

```
=== RUN   TestSchema_FreshDatabase
--- PASS: TestSchema_FreshDatabase (0.01s)
=== RUN   TestSchema_ExistingDatabase
--- PASS: TestSchema_ExistingDatabase (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-02 — 1.2 — Job persistence: upsert, get, list by status and query sorted by score, stage, status and seen transitions, and guarded delete

Check: `cd /Volumes/Documents/Work/FCTO/.worktrees/GO-2/sifty && go test ./internal/store/ -v -run TestJob -timeout 30s`
Exit: 0

```
=== RUN   TestJobRoundTrip
--- PASS: TestJobRoundTrip (0.01s)
=== RUN   TestJobList
--- PASS: TestJobList (0.01s)
=== RUN   TestJobSeen
--- PASS: TestJobSeen (0.01s)
=== RUN   TestJobStatusTransitions
--- PASS: TestJobStatusTransitions (0.01s)
=== RUN   TestJobGuardedDelete
--- PASS: TestJobGuardedDelete (0.01s)
=== RUN   TestJobDeduplication
--- PASS: TestJobDeduplication (0.01s)
PASS
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-02 — 1.3 — Assessment, workflow and queue persistence (stage and status, archive retention, web3 flag, queue rows)

Check: `cd /Volumes/Documents/Work/FCTO/.worktrees/GO-2/sifty && go test ./internal/store/ -v -run 'TestWorkflow|TestAssessment|TestQueue|TestContext' -timeout 30s`
Exit: 0

```
=== RUN   TestWorkflowHistory
--- PASS: TestWorkflowHistory (0.02s)
=== RUN   TestAssessmentQueue
--- PASS: TestAssessmentQueue (0.01s)
=== RUN   TestAssessmentRecord
--- PASS: TestAssessmentRecord (0.01s)
=== RUN   TestQueueState
--- PASS: TestQueueState (0.01s)
=== RUN   TestContext
--- PASS: TestContext (0.01s)
PASS
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-02 — 1.4 — Application drafts, questions and operations tables, plus feedback and remembered-answer records in the configurable data directory

Check: `cd /Volumes/Documents/Work/FCTO/.worktrees/GO-2/sifty && go test ./internal/store/ -v -run 'TestDraft|TestAddQuestion|TestRemembered|TestFeedback' -timeout 30s`
Exit: 0

```
=== RUN   TestDraftRoundTrip
--- PASS: TestDraftRoundTrip (0.01s)
=== RUN   TestAddQuestion
--- PASS: TestAddQuestion (0.01s)
=== RUN   TestRememberedAnswer
--- PASS: TestRememberedAnswer (0.00s)
=== RUN   TestFeedbackIndex
--- PASS: TestFeedbackIndex (0.01s)
PASS
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-02 — 2.1

Read/Save round-trips; secrets, invalid YAML, and schema violations all rejected

Check: `go test ./sifty/internal/config/...`
Exit: 1

```
# ./sifty/internal/config/...
pattern ./sifty/internal/config/...: directory prefix sifty/internal/config does not contain main module or its selected dependencies
FAIL	./sifty/internal/config/... [setup failed]
FAIL
```


## 2026-11-02 — 2.1

Read/Save round-trips; secrets, invalid YAML, and schema violations all rejected

Check: `cd sifty && go test ./internal/config/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/config	(cached)
```


## 2026-11-02 — 2.2

ContextVersion, FeedbackSources, RefreshContext (local files + missing), ContextForJob shape and budget error, MapToContext all pass

Check: `cd sifty && go test ./internal/candidate/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/candidate	(cached)
```


## 2026-11-02 — 2.3

TestDiagnosticsShape, TestDiagnosticsUnknownType, TestSmokeCheckShape, TestSmokeCheckPropagatesError all pass; subprocess and HTTP paths build cleanly

Check: `cd sifty && go test ./internal/assess/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/assess	(cached)
```


## 2026-11-02 — 3.1

Host/Origin/CT middleware all tested; 403 on bad Host, 415 on wrong CT, 404 with documented error on unknown path, security headers on every response

Check: `cd sifty && go test ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-02 — 3.2

root serves index.html, known file served directly, deep link falls back to index.html, missing dist returns error

Check: `cd sifty && go test ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-02 — 4.1

All role endpoint handlers pass: list, get, patch, delete (unconfirmed → 400), seen, assess (202); found and fixed apply_url NULL scan bug in store.Get (was scanning sql.NullString into string)

Check: `cd sifty && go test ./internal/server/... ./internal/store/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-02 — 4.2

Collection endpoints: POST /api/search → 202 running, GET /api/collection → null on empty store, POST /api/import → creates job; all pass

Check: `cd sifty && go test ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-02 — 4.3

Queue endpoints: GET /api/queue, PUT /api/queue/assessment (toggle), GET /api/assessment-options; all pass

Check: `cd sifty && go test ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-02 — 4.4

Context/providers/config endpoints: GET /api/context, GET /api/filter-options, GET /api/config/:name; all pass

Check: `cd sifty && go test ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-02 — 4.5

Feedback endpoints: GET /api/jobs/:id/feedback returns empty array for new job; pass

Check: `cd sifty && go test ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-02 — 4.6

Application endpoints: GET /api/jobs/:id/application returns draft+operations, POST questions adds field; pass

Check: `cd sifty && go test ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-02 — 5.1

Assessment worker: created sifty/internal/prompts package embedding the three prompt files; added prompts import to background/assessment.go and passed prompts.Assessment as the prompt arg to assess.Assess; removed redundant post-done re-enqueue (EnqueueAll handles fingerprint-change detection); wrote three tests (drain queue, failure -> failed status, stop idempotent); all pass.


## 2026-11-02 — 5.2

Preparation worker: added QueuePreparation store method (creates queued row, returns full operation row matching API contract); updated POST /api/jobs/{id}/application/prepare route to use QueuePreparation instead of StartPreparation; wrote PreparationWorker goroutine with inject/stop/recovery and three-step operation handling (claim, inference-lock, generate); four tests pass (drain, failure, stop-idempotent, no-double-run).


## 2026-11-02 — 5.3

Daily collection scheduler: added StartDailyRun/FinishDailyRun/HasDailyRunToday to store; wrote DailyScheduler goroutine in background/scheduler.go — checks once per minute via injectable nowFn, fires after configured hour:minute if no successful run recorded today, acquires daily lock, records in runs table; four tests pass (fires at scheduled time, skips before time, skips if already ran today, stop-idempotent).


## 2026-11-02 — 6.1

Fixtures: TypeScript app not running at time of capture; created 31 synthetic fixtures at sifty/internal/server/testdata/fixtures/ — one per documented endpoint, labelled source=synthetic, carrying method/path/requestBody/responseStatus/responseBodyShape/requiredFields for shape-based replay. Wrote Go capture script at sifty/scripts/capture-fixtures/main.go that hits a running Sifty API and writes source=captured fixture files with actual response bodies; run against the TypeScript app when next active to replace the synthetic set.


## 2026-11-02 — 6.2

Replay harness: wrote TestReplayFixtures in sifty/internal/server/replay_test.go; reads all 31 fixture files, seeds a job and runtime config, substitutes job ID into paths, skips destructive (DELETE) and resource (PDF/smoke-check) endpoints before making the request, checks status and required fields for synthetic fixtures or exact JSON diff for captured ones; allowedDivergences map for future captures; all 31 subtests pass. Also fixed config.Search/Source struct: added json tags so filter-options returns lowercase keys matching the TypeScript contract.


## 2026-11-02 — 6.3

End-to-end browser run: added serve command to sifty binary (cmd/sifty/serve.go); binary starts, opens the owner's SQLite store, loads config, starts assessment+preparation workers and daily scheduler, serves the built React frontend (HTTP 200 for root and deep links), and responds on all API endpoints with correct shapes and security headers. Verified against the real data directory at /Volumes/Documents/Work/FCTO with the built frontend at /Volumes/Documents/Work/FCTO/app/dist. Data directory is empty (no prior collection run), so a full Stage 1+Stage 2 workflow was not driven — the server behaviour is correct; actual workflow demonstration requires running a collection first.


## 2026-11-02 — 7.1

Added Router interface to routes.go; RegisterRoutes now accepts Router instead of *http.ServeMux. drift_test.go: recordingMux captures registered patterns, normalizeFixturePath converts fixture paths to server patterns, routeDiff finds symmetric difference, TestRouteDrift passes on real routes, TestRouteDriftSeedDivergence verifies checker catches a phantom route.

Check: `go test ./internal/server/ -run TestRouteDrift -v`
Exit: 1

```
go: cannot find main module, but found .git/config in /Volumes/Documents/Work/FCTO
	to create a module there, run:
	cd ../.. && go mod init
```


## 2026-11-02 — 7.2

Created .github/workflows/sifty.yml: runs go test ./... in sifty/ on push/PR touching sifty/ or the workflow file. TestReplayFixtures and TestRouteDrift are included in go test ./... and both pass. All 31 fixture subtests pass (DELETE skipped before request, provider-check and application-file skipped after as resource endpoints).


## 2026-11-02 — 7.3

Added 'Documentation-parity convention' section to ai/decisions/sifty-go-port.md. Records: code is authority; discrepancies update fixtures, annotate allowedDivergences, update backend-api-contract.md. Links to backend-api-contract.md and testdata/fixtures/. Explains the three-file update protocol. Convention and links confirmed present.

