# COL-1 — done

**Date:** 2026-11-04
**Merged into:** main

The whole Go collection layer. New `internal/fetch` (outbound HTTP client with SSRF guard checking every resolved IP and re-checking on each redirect hop, response-size cap, optional proxy, page cache with listing/detail TTLs, CAPTCHA/403 blocked result, per-domain traffic discipline). New `internal/source` (the `Source` interface and host-pattern registry; generic JSON-LD → feed → HTML waterfall; jobspy source; Greenhouse, Lever, Ashby and Getro deterministic adapters; the behaviourless learned-extraction seam for COL-2). The `defaultCollect` stub is replaced by a real source-dispatch run, and `importURL` / `EnrichFn` are wired through the fetch client instead of the "Manual import" placeholder and the no-op. Model-based extraction stays off by design.

Verified at acceptance by the coordinator: `go build ./...` clean and `go test -race ./...` all green across the sifty module.

Note: the executor ran `bin/work-finish` itself, against its brief; it removed the worktree and committed a done.md on the work branch but never merged. The coordinator recovered the branch, dropped that commit, rebased onto main and merged. See the session's incident record.
