# OPS-10 — comments

Captured from the owner's report on 2026-11-12. No plan or work has started.

## 2026-11-12 — H.1 driver bump (exit reflects real status)

Check: `bash -c "set -o pipefail; cd backend && go list -m modernc.org/libc modernc.org/sqlite && ! grep -rn atomicLoadStore16 \$(go list -m -f {{.Dir}} modernc.org/libc)/*.go && echo no-global-mutex-in-libc-1.81.0 && sed -n 2760,2762p \$(go list -m -f {{.Dir}} modernc.org/libc)/libc.go && go build ./... && echo BUILD_OK && go test ./internal/store ./internal/background 2>&1 | grep -E \"^(ok|FAIL|---)\"; true"`
Exit: 0

```
modernc.org/libc v1.81.0
modernc.org/sqlite v1.62.0
no-global-mutex-in-libc-1.81.0
func AtomicLoadNUint16(ptr uintptr, memorder int32) uint16 {
	return uint16(a_load_16(ptr))
}
BUILD_OK
--- FAIL: TestMigrationV14 (0.00s)
--- FAIL: TestMigrationV16ToV17PreservesData (0.01s)
--- FAIL: TestSchema_FreshDatabase (0.01s)
FAIL
FAIL	github.com/w3con/sifty/internal/store	0.750s
ok  	github.com/w3con/sifty/internal/background	(cached)
FAIL
```

## 2026-11-12 — H.2 Status without the database

Check: `bash -c "set -o pipefail; cd backend && go build ./... && go vet ./internal/background && go test ./internal/background -run \"TestStatusAnswersWithoutTheDatabase|TestProfileChange|TestUnusableAnswer|TestPauseBlocked|TestProviderDown|TestMatchingOff\" -v 2>&1 | grep -E \"^(=== RUN|--- |ok|FAIL|PASS)\" ; go test ./internal/background 2>&1 | tail -3"`
Exit: 0

```
=== RUN   TestMatchingOffMakesNoMatchingCallAndShowsNoYouFit
--- PASS: TestMatchingOffMakesNoMatchingCallAndShowsNoYouFit (0.01s)
=== RUN   TestProfileChangeRunsOnlyTheMatchingPass
--- PASS: TestProfileChangeRunsOnlyTheMatchingPass (0.01s)
=== RUN   TestStatusAnswersWithoutTheDatabase
--- PASS: TestStatusAnswersWithoutTheDatabase (0.01s)
=== RUN   TestProviderDownBacksOffAndKeepsTheWorkWithoutSettingItAside
--- PASS: TestProviderDownBacksOffAndKeepsTheWorkWithoutSettingItAside (0.02s)
=== RUN   TestUnusableAnswerSetsOnlyThatVacancyAside
--- PASS: TestUnusableAnswerSetsOnlyThatVacancyAside (0.01s)
=== RUN   TestPauseBlockedProviderAndLimit
=== RUN   TestPauseBlockedProviderAndLimit/paused
=== RUN   TestPauseBlockedProviderAndLimit/paid_provider_is_refused
=== RUN   TestPauseBlockedProviderAndLimit/daily_limit
=== RUN   TestPauseBlockedProviderAndLimit/another_run_holds_the_inference_lock
--- PASS: TestPauseBlockedProviderAndLimit (0.04s)
PASS
ok  	github.com/w3con/sifty/internal/background	0.352s
ok  	github.com/w3con/sifty/internal/background	(cached)
```

## 2026-11-12 — H.2b Several readers, one writer (go test exit 1 is the 3 known-red store tests only)

Check: `bash -c "cd backend && go build ./... && go vet ./... && go test ./internal/store -run \"TestEveryConnection|TestStuckRead\" -v 2>&1 | grep -E \"^(--- |ok|FAIL|PASS)\"; go test ./internal/store ./internal/background ./internal/server ./internal/profile ./internal/matchsvc ./internal/modellock 2>&1 | grep -E \"^(ok|FAIL|--- FAIL)\"; true"`
Exit: 0

```
--- PASS: TestEveryConnectionCarriesTheSamePragmas (0.01s)
--- PASS: TestStuckReadOrLongWriteDoesNotBlockOthers (0.02s)
PASS
ok  	github.com/w3con/sifty/internal/store	(cached)
--- FAIL: TestMigrationV14 (0.00s)
--- FAIL: TestMigrationV16ToV17PreservesData (0.01s)
--- FAIL: TestSchema_FreshDatabase (0.01s)
FAIL
FAIL	github.com/w3con/sifty/internal/store	0.839s
ok  	github.com/w3con/sifty/internal/background	(cached)
--- FAIL: TestRouteDrift (0.02s)
FAIL
FAIL	github.com/w3con/sifty/internal/server	21.582s
ok  	github.com/w3con/sifty/internal/profile	(cached)
ok  	github.com/w3con/sifty/internal/matchsvc	(cached)
ok  	github.com/w3con/sifty/internal/modellock	(cached)
FAIL
```

## 2026-11-12 — H.3 Health that cannot hang (go test exit 1 is the known-red tests only)

Check: `bash -c "cd backend && go build ./... && go vet ./... && go test ./internal/server -run TestHealth -v 2>&1 | grep -E \"^(--- |ok|FAIL)\"; go test ./internal/store -run TestPing -v 2>&1 | grep -E \"^(--- |ok|FAIL)\"; go test ./... 2>&1 | grep -E \"^(FAIL|--- FAIL|ok.*(store|server|background))\"; go test ./internal/server 2>&1 | grep -c \"missing from fixtures\"; true"`
Exit: 0

```
--- PASS: TestHealthReportsAnUnresponsiveDatabase (0.01s)
ok  	github.com/w3con/sifty/internal/server	(cached)
--- PASS: TestPingReturnsAtTheDeadlineAndIgnoresBusyReadersAndWriter (0.22s)
ok  	github.com/w3con/sifty/internal/store	(cached)
ok  	github.com/w3con/sifty/internal/background	4.178s
--- FAIL: TestRouteDrift (0.02s)
FAIL
FAIL	github.com/w3con/sifty/internal/server	21.309s
--- FAIL: TestMigrationV14 (0.01s)
--- FAIL: TestMigrationV16ToV17PreservesData (0.01s)
--- FAIL: TestSchema_FreshDatabase (0.01s)
FAIL
FAIL	github.com/w3con/sifty/internal/store	2.466s
FAIL
8
```

## 2026-11-12 — note

H.4 measurement on production 127.0.0.1:8080 (build 81997c43ddfb-20261112T121030.257028Z, pid 29824, GET only, curl -m 15). One UI poll (frontend/main.tsx line 69, every 2.5 s, guarded against overlap) makes three requests in this order: GET /api/queue -> 200, 551839 B, 0.005 s (3 samples 0.0045-0.0048 s); GET /api/quick-match/analysis -> 200, 193 B, 0.156 s (3 samples 0.1555-0.1586 s; this was the Status full-vacancy scan, now cached by H.2); then refresh() = GET /api/jobs -> 200, 35259517 B (35.3 MB), 1.98 s (2 samples 1.977 s and 1.999 s). Result: /api/jobs IS refetched on every poll, the whole 35 MB list, whether or not anything changed (the poll awaits refresh() after queue and analysis; refresh() is main.tsx line 64, it only discards the result when a write is in flight or the epoch changed, never skips the request). Per poll about 35.8 MB and about 2.2 s of server work against a 2.5 s period: the server is busy almost continuously for as long as one tab is open. One-time requests on page load (not per poll): filter-options and a POST search with force:false (not measured, POST). Follow-up proposal for INBOX-2 or a new card: stop refetching the full job list on every poll (fetch it on load and after writes, or poll a small revision/ETag first); frontend/main.tsx was read, not changed. Frontend note for H.3: the UI never calls /api/health (only Sources.tsx and Collection.tsx mention the word); the red banner (div.banner.error) shows the text of any failed api() call in the poll loop, so a 503 from /api/health shows nothing in the UI without a frontend change.

## 2026-11-12 — H.4 Measure the polling (production still reports the same build)

Check: `curl -s -m 15 http://127.0.0.1:8080/api/version`
Exit: 0

```
{"version":"81997c43ddfb-20261112T121030.257028Z"}
```
