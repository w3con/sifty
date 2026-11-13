---
estimate: 3
---
# GO-4 — plan

**Author:** coordinator
**Created:** 2026-11-04

## Statement of work

`go test -race ./...` in `sifty/` reports seven data races in five `internal/background` tests, all in unsynchronized test-stub state (called-ID slices and concurrency counters shared between the worker goroutine and the test goroutine); every other package is clean, and nothing runs the suite with `-race`, which is why it went unnoticed. This card synchronizes the stubs in `assessment_test.go` and `preparation_test.go` without changing what the tests assert or touching production code, and makes `scripts/test` and the CI workflow run the suite with `-race` so a future race fails the build.

## Steps

### Phase 1 — Synchronize the stubs

- [x] 1.1 — Make the assessment worker test stubs race-free (mutex, channels, or reading state only after the worker stops; no sleeps as synchronization), keeping every assertion; check: `go test -race -run 'TestAssessmentWorker' -count=3 ./internal/background/` passes with no race warning.
- [x] 1.2 — Make the preparation worker test stubs race-free the same way, keeping every assertion including the no-double-run guarantee; check: `go test -race -run 'TestPreparationWorker' -count=3 ./internal/background/` passes with no race warning.

### Phase 2 — Keep it that way

- [x] 2.1 — Run the suite with `-race` in `scripts/test` and in `.github/workflows/ci.yml`; check: `scripts/test` passes locally and `go test -race ./...` over the whole module passes with no race warning.

## Acceptance criteria

- `go test -race ./...` in `sifty/` passes with no data-race warning, with no production code changed and no test assertion weakened.
- `scripts/test` and the CI workflow run the suite with `-race`.
