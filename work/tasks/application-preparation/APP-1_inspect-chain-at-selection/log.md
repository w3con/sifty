# APP-1 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-03 — 1.1 — Build a preparation inspect path in the server

Check: `cd sifty && go test ./internal/background/... -run TestInspectOnlyOperationPopulatesDraft -v`
Exit: 0

```
=== RUN   TestInspectOnlyOperationPopulatesDraft
--- PASS: TestInspectOnlyOperationPopulatesDraft (0.07s)
PASS
ok  	github.com/w3con/sifty/internal/background	(cached)
```


## 2026-11-03 — 1.2 — Keep inspection off the inference lock

Check: `cd sifty && go test ./internal/background/... -run TestInspectCompletesWithInferenceLockHeld -v`
Exit: 0

```
=== RUN   TestInspectCompletesWithInferenceLockHeld
--- PASS: TestInspectCompletesWithInferenceLockHeld (0.07s)
PASS
ok  	github.com/w3con/sifty/internal/background	(cached)
```


## 2026-11-03 — 1.3 — Surface honest non-question outcomes

Check: `cd sifty && go test ./internal/background/... -run 'TestInspectNoFieldsReturnsHonestDone|TestInspectLoginRequiredReturnsHonestDone' -v`
Exit: 0

```
=== RUN   TestInspectNoFieldsReturnsHonestDone
--- PASS: TestInspectNoFieldsReturnsHonestDone (0.06s)
=== RUN   TestInspectLoginRequiredReturnsHonestDone
--- PASS: TestInspectLoginRequiredReturnsHonestDone (0.06s)
PASS
ok  	github.com/w3con/sifty/internal/background	(cached)
```


## 2026-11-03 — 2.1 — Enqueue inspect on pre-selection

Check: `cd sifty && go test ./internal/server/... -run 'TestPatchToPreselected' -v`
Exit: 0

```
=== RUN   TestPatchToPreselectedEnqueuesInspect
--- PASS: TestPatchToPreselectedEnqueuesInspect (0.01s)
=== RUN   TestPatchToPreselectedIdempotent
--- PASS: TestPatchToPreselectedIdempotent (0.01s)
=== RUN   TestPatchToPreselectedWorkerDrainsWithoutModelCall
--- PASS: TestPatchToPreselectedWorkerDrainsWithoutModelCall (0.06s)
PASS
ok  	github.com/w3con/sifty/internal/server	0.357s
```


## 2026-11-03 — 3.1 — Queue runtime fix with regression test

Check: `cd sifty && go test ./internal/server/... -run 'TestQueueRuntime' -v`
Exit: 0

```
=== RUN   TestQueueRuntimeNullWhenAbsent
--- PASS: TestQueueRuntimeNullWhenAbsent (0.01s)
=== RUN   TestQueueRuntimeDataIsVerbatimString
--- PASS: TestQueueRuntimeDataIsVerbatimString (0.01s)
PASS
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-03 — 3.2 — End-to-end manual verification

Step requires running the live server against the owner's real database with jobs 458 (Catena/Ashby ATS) and 758 (Aave/LinkedIn). Cannot be automated. To verify: cd sifty && go run ./cmd/sifty serve, pre-select jobs 458 and 758 via the UI, confirm Application tab shows inspected questions or honest no-questions/sign-in state without crashing the catalogue.


## 3.2 — coordinator end-to-end verification (2026-11-03)

Ran the APP-1 branch binary against the owner's real database (1309 jobs) on port 4317, with the Playwright worker (`sifty/workers/playwright`, deps installed) and the built frontend.

- `/api/queue` returns `runtime.data` as a string; catalogue does not crash. Queue fix confirmed live.
- Job 458 (Ashby, apply_url jobs.ashbyhq.com/...): inspect ran, found 0 readable fields (form is behind the ATS "Apply" button, which the worker does not click), operation ended `done` with "No readable questions found." — honest state works.
- Job 758 (apply_url unresolved): inspect fell back to the original listing URL, which returns a download; operation `failed` with `worker inspectForm: page.goto: Download is starting` — a raw failure, not an honest state.

Deferred to APP-4: click-through inspection for ATS "Apply" buttons, and honest handling of unresolved/non-HTML/download apply destinations.
