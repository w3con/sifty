# ASM-4 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-05 — 1.1 — FTS5 schema: jobs_fts table, match_score column, user_version 13, Go-level sync via refreshFTS and Delete

Check: `go test -race ./internal/store/... → ok (3.45s)`
Exit: 2

```
/bin/sh: -c: line 0: syntax error near unexpected token `('
/bin/sh: -c: line 0: `go test -race ./internal/store/... → ok (3.45s)'
```


## 2026-11-05 — 1.1 — FTS5 schema: jobs_fts table, match_score column, user_version 13, Go-level sync via refreshFTS and Delete

Check: `go test -race ./internal/store/... → ok (3.45s)`
Exit: 2

```
/bin/sh: -c: line 0: syntax error near unexpected token `('
/bin/sh: -c: line 0: `go test -race ./internal/store/... → ok (3.45s)'
```


## 2026-11-05 — 1.1 — FTS5 schema: jobs_fts table, match_score column, user_version 13, Go-level sync via refreshFTS and Delete

Check: `cd .worktrees/ASM-4/sifty && go test -race ./internal/store/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-05 — 1.2 — BuildFTSQuery helper with sanitiseFTS5 and unit tests

Check: `cd .worktrees/ASM-4/sifty && go test -race ./internal/store/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-05 — 1.3 — Rescore BM25 min-max scaler with unit tests

Check: `cd .worktrees/ASM-4/sifty && go test -race ./internal/store/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-05 — 2.1 — Rescore wired into collection and profile save

Check: `cd .worktrees/ASM-4/sifty && go test -race ./internal/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/assess	(cached)
ok  	github.com/w3con/sifty/internal/background	(cached)
ok  	github.com/w3con/sifty/internal/candidate	(cached)
ok  	github.com/w3con/sifty/internal/collect	(cached)
ok  	github.com/w3con/sifty/internal/config	(cached)
?   	github.com/w3con/sifty/internal/dirs	[no test files]
ok  	github.com/w3con/sifty/internal/fetch	(cached)
ok  	github.com/w3con/sifty/internal/inspect	(cached)
?   	github.com/w3con/sifty/internal/model	[no test files]
ok  	github.com/w3con/sifty/internal/prepare	(cached)
?   	github.com/w3con/sifty/internal/prompts	[no test files]
ok  	github.com/w3con/sifty/internal/records	(cached)
ok  	github.com/w3con/sifty/internal/server	(cached)
ok  	github.com/w3con/sifty/internal/source	(cached)
ok  	github.com/w3con/sifty/internal/store	(cached)
?   	github.com/w3con/sifty/internal/version	[no test files]
ok  	github.com/w3con/sifty/internal/worker	(cached)
```


## 2026-11-05 — 2.2 — EnqueueAll removed; EnqueueJob on preselect; collection test verifies queue stays empty

Check: `cd .worktrees/ASM-4/sifty && go test -race ./internal/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/assess	(cached)
ok  	github.com/w3con/sifty/internal/background	(cached)
ok  	github.com/w3con/sifty/internal/candidate	(cached)
ok  	github.com/w3con/sifty/internal/collect	(cached)
ok  	github.com/w3con/sifty/internal/config	(cached)
?   	github.com/w3con/sifty/internal/dirs	[no test files]
ok  	github.com/w3con/sifty/internal/fetch	(cached)
ok  	github.com/w3con/sifty/internal/inspect	(cached)
?   	github.com/w3con/sifty/internal/model	[no test files]
ok  	github.com/w3con/sifty/internal/prepare	(cached)
?   	github.com/w3con/sifty/internal/prompts	[no test files]
ok  	github.com/w3con/sifty/internal/records	(cached)
ok  	github.com/w3con/sifty/internal/server	(cached)
ok  	github.com/w3con/sifty/internal/source	(cached)
ok  	github.com/w3con/sifty/internal/store	(cached)
?   	github.com/w3con/sifty/internal/version	[no test files]
ok  	github.com/w3con/sifty/internal/worker	(cached)
```


## 2026-11-05 — 3.1+3.2 — matchScore on Job type, catalogue list, role header; sort and filter by matchScore; Assess now button

Check: `cd .worktrees/ASM-4/app && /Volumes/Documents/Work/FCTO/app/node_modules/.bin/tsc --noEmit`
Exit: 0

```

```


## 2026-11-05 — 4.1 — bin/work-test passes; go test -race ./... passes; go build and vite build both clean

Check: `cd .worktrees/ASM-4 && bin/work-test`
Exit: 0

```
ok   compile _work_lib.py
ok   compile work-new
ok   compile work-plan
ok   compile work-handover
ok   compile work-note
ok   compile work-commit
ok   compile work-resume
ok   compile work-accept
ok   compile work-finish
ok   template convention.md
ok   template description.md
ok   template short-description.md
ok   template plan.md
ok   template log.md
ok   template comments.md
ok   template attention.md
ok   template executor-brief.md
ok   AGENTS.md present
ok   AGENTS.md names the commands
ok   AGENTS.md has no @-import of the mechanism
ok   --help work-new
ok   --help work-plan
ok   --help work-handover
ok   --help work-note
ok   --help work-commit
ok   --help work-resume
ok   --help work-accept
ok   --help work-finish

PASS: all harness checks
```


## 2026-11-05 — 4.2 manual e2e on a copy of live data

Coordinator verification on a copy of the production data (/tmp/claude-501/sift-data, 2459 roles). Built the ASM-4 Go binary, ran it against the copy (migration added match_score + FTS and backfilled the index), triggered a rescore via POST /api/context {refreshRemote:false} (no network, no LLM). Result: total=2459, scored(match_score>0)=1633, min=0, max=100; top roles by score were Blockchain/Web3/Solidity at 95-100, matching the candidate profile. Scoring is instant and correct; the mass LLM enqueue on collection is gone (unit-tested in 2.2). Not deployed anywhere live; deployment is deferred to the OPS-1 pipeline per owner direction.

