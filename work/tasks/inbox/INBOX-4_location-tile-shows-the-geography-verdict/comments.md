# INBOX-4 — comments

Captured from the owner's report on 2026-11-13. No plan or work has started.

## 2026-11-13 — G.1 — Unrestricted remote is a match

Check: `cd backend && go test ./internal/quickmatch/ ./internal/matchsvc/ ./internal/matchcontract/`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/quickmatch	(cached)
ok  	github.com/w3con/sifty/internal/matchsvc	(cached)
ok  	github.com/w3con/sifty/internal/matchcontract	(cached)
```

## 2026-11-13 — G.2 — Check on a production copy

Check: `cd backend && SIFTY_PROBE_DATA_DIR=scratch/data go test ./internal/matchsvc/ -run TestLiveUnrestrictedRemote -count=1 -v`
Exit: 1

```
=== RUN   TestLiveUnrestrictedRemote
    live_restriction_test.go:823: read /Volumes/SSD/Dev/tknff/sifty/.worktrees/INBOX-4/backend/internal/matchsvc/scratch/data/config/search.yaml: open /Volumes/SSD/Dev/tknff/sifty/.worktrees/INBOX-4/backend/internal/matchsvc/scratch/data/config/search.yaml: no such file or directory
--- FAIL: TestLiveUnrestrictedRemote (0.00s)
FAIL
FAIL	github.com/w3con/sifty/internal/matchsvc	0.231s
FAIL
```

## 2026-11-13 — G.2 — Check on a production copy (absolute path)

Check: `cd /Volumes/SSD/Dev/tknff/sifty/.worktrees/INBOX-4/backend && SIFTY_PROBE_DATA_DIR=/Volumes/SSD/Dev/tknff/sifty/.worktrees/INBOX-4/scratch/data go test ./internal/matchsvc/ -run TestLiveUnrestrictedRemote -count=1`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/matchsvc	2.551s
```

## 2026-11-13 — G.3

G.3 complete. regions.default.yaml: | separator, telecommute alias, 229 ISO alpha-2 codes. 11 tests pass (4 new G.3 cases). Dump: main 375 unknown → branch 333 unknown + 57 unrestricted (G.3: 42 unknown→unrestricted, all TELECOMMUTE-label vacancies; ISO fix: 0 verdict changes — 250-code vacancies have needs_corroboration reliability). Jobs 1136+17 confirmed not_covered. 69 no-hiring-locations unknowns: LLM-extracted city/vague location text (IDs 16=Austin TX, 25=URL-reference, 138=anywhere in the world). Root cause separate from TELECOMMUTE/ISO, not addressed. 333 unknowns remain.

## 2026-11-13 — G.3-correction

G.3 correction: removed | from separators (coordinator's error — Anywhere and TELECOMMUTE are two separate hiringLocations entries, not a compound label). Fixed 4 G.3 test cases to use separate labels ([]string{Anywhere, TELECOMMUTE} etc.). Counts unchanged: 57 unrestricted / 333 unknown / 2 covered / 12 not_covered.
