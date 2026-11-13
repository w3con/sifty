# TRI-1 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-04 — 1.1 — block_rules schema + user_version 11

Check: `cd sifty && go build ./...`
Exit: 0

```

```


## 2026-11-04 — 1.2 — AddBlockRule/ListBlockRules/RemoveBlockRule/blockMatch with unit tests

Check: `cd sifty && go test -race ./internal/store/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-05 — 1.3 — Blocked/BlockedBy populated on Get/List; go test -race ./... green

Check: `cd sifty && go test -race ./...`
Exit: 0

```
?   	github.com/w3con/sifty/cmd/sifty	[no test files]
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
?   	github.com/w3con/sifty/scripts/capture-fixtures	[no test files]
```


## 2026-11-05 — 2.1 — POST /api/jobs/{id}/block handler with company and source tests

Check: `cd sifty && go test -race ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-05 — 2.2 — GET /api/block-rules and DELETE /api/block-rules/{id} with handler tests

Check: `cd sifty && go test -race ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-05 — 3.1 — blocked/blockedBy on Job type, api helpers for block/list-rules/remove-rule

Check: `cd app && node_modules/.bin/tsc --noEmit`
Exit: 0

```

```


## 2026-11-05 — 3.2 — hide blocked from Triage filtered list and nav count

Check: `cd app && node_modules/.bin/tsc --noEmit`
Exit: 0

```

```


## 2026-11-05 — 3.3 — Block… control in Triage with company and source options

Check: `cd app && node_modules/.bin/tsc --noEmit`
Exit: 0

```

```


## 2026-11-05 — 4.1 — Dismissed & blocked Settings section with block rules and remove control

Check: `cd app && node_modules/.bin/tsc --noEmit`
Exit: 0

```

```


## 2026-11-05 — 4.2 — dismissed roles list in Settings with restore-to-Triage control

Check: `cd app && node_modules/.bin/tsc --noEmit`
Exit: 0

```

```


## 2026-11-05 — 5.1 — bin/work-test passes

Check: `bin/work-test`
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


## 2026-11-05 — 5.2 — web bundle and Go binary build; manual runtime test pending coordinator acceptance

Web bundle built in app/dist (285 kB JS, 26 kB CSS). Go binary built from sifty/cmd/sifty. All automated tests pass. Manual verification (block company in Triage, check Settings, remove rule, confirm roles return) is the coordinator's acceptance step — the gate is marked pending here and will be recorded verbatim by the coordinator on acceptance.


## 2026-11-05 — fix: block company uses company name

Check: `cd sifty && go test -race ./internal/store/... ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	(cached)
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-05 — fix: block company uses company name

Check: `cd app && npx tsc --noEmit`
Exit: 0

```

```


## 2026-11-05 — 5.2 — coordinator manual acceptance (real binary, copy of live data)

Built the fixed Go binary and web bundle from `work/TRI-1`, ran them against a copy of the live catalogue (store.sqlite, 2293 triage roles) on an isolated port; the live demo was not touched. The migration advanced `user_version` 10 → 11 on open.

Block a company — `POST /api/jobs/229/block {"dimension":"company"}` on a True Anomaly role:

```
rule returned: {"id":1,"kind":"company","value":"true anomaly","createdAt":...}
triage jobs: 2293 | blocked: 213 | distinct companies among blocked: 1 (True Anomaly)
```

213 is the exact count of True Anomaly postings in the catalogue; no other company was hidden. (Before the fix, the same action created `domain:job-boards.greenhouse.io` and hid 274 roles across 8 companies.)

Remove the rule — `DELETE /api/block-rules/1`:

```
{"removed":1}
job 229 blocked after unblock = False
/api/block-rules after delete = []
```

Source block — `POST /api/jobs/1/block {"dimension":"source"}` on a single-source (Remotive) role returned `{"kind":"source","value":"Remotive"}` and set that job `blocked=true, blockedBy="source:Remotive"`. Dismissed list — `GET /api/jobs?status=no` returned only dismissed roles. All paths behave as specified.

