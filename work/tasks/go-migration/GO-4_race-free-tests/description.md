# GO-4 — Race-free tests: make the background test stubs safe under the race detector and keep them that way

**Epic:** go-migration
**Created:** 2026-11-04
**Status:** founded

## Why this work exists

`go test -race ./...` in `sifty/` fails in `internal/background` with seven data-race warnings across five tests: `TestAssessmentWorkerDrainsQueue`, `TestAssessmentWorkerFailureRetries`, `TestPreparationWorkerDrainsQueue`, `TestPreparationWorkerFailure`, `TestPreparationWorkerNoDoubleRun`. The races are in the tests' own stub state, for example a slice of called job IDs appended from the worker goroutine and read by the test goroutine (`assessment_test.go` around lines 65, 110, 115), and concurrency counters read and written from several goroutines (`preparation_test.go` around lines 19, 178, 185, 189), with no synchronization. Every other package is clean under `-race`.

While the race detector is red, a real race introduced in production code would be lost in the noise, and the config hot-reload work already depends on the workers reading shared state safely. Nothing runs `-race` today: `scripts/test` and the CI workflow run plain `go test ./...`, which is how this went unnoticed.

## Must not touch

- Production code in `internal/background` and elsewhere: if a race turns out to involve a production variable rather than test-only state, stop and record it in `question.md` rather than changing production behaviour inside this card. What the tests assert.

## Notes

- Fix with a mutex or channels in the stubs, or by reading the stub's state only after the worker has stopped; avoid `time.Sleep` as synchronization.
- `sifty/scripts/test` and `sifty/.github/workflows/ci.yml` are the two places that run the suite; both should run it with `-race` afterwards.
- Code, comments and commit messages under `sifty/` follow the public repository's rules, which the hooks now enforce: no card identifiers, no step numbers, no tool-authorship trailers.
