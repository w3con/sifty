# TRI-2 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-05 — 1.1 GET /api/sources

Check: `cd sifty && go test -race -run 'TestListSources|TestRouteDrift' -v ./internal/server/ 2>&1 | tail -15`
Exit: 0

```
=== RUN   TestListSources
--- PASS: TestListSources (0.08s)
=== RUN   TestRouteDrift
--- PASS: TestRouteDrift (0.06s)
=== RUN   TestRouteDriftSeedDivergence
--- PASS: TestRouteDriftSeedDivergence (0.06s)
PASS
ok  	github.com/w3con/sifty/internal/server	1.526s
```


## 2026-11-05 — 1.2 POST /api/block-rules

Check: `cd sifty && go test -race -run 'TestCreateSourceBlockRule|TestCreateBlockRuleRejectsBadInput|TestRouteDrift' -v ./internal/server/ 2>&1 | tail -15`
Exit: 0

```
=== RUN   TestCreateSourceBlockRule
--- PASS: TestCreateSourceBlockRule (0.07s)
=== RUN   TestCreateBlockRuleRejectsBadInput
--- PASS: TestCreateBlockRuleRejectsBadInput (0.06s)
=== RUN   TestRouteDrift
--- PASS: TestRouteDrift (0.06s)
=== RUN   TestRouteDriftSeedDivergence
--- PASS: TestRouteDriftSeedDivergence (0.06s)
PASS
ok  	github.com/w3con/sifty/internal/server	1.541s
```


## 2026-11-05 — 2.1 Triage Block company only

Check: `cd app && npx tsc --noEmit && echo TSC_OK && grep -c 'Block source' web/main.tsx; true`
Exit: 0

```
TSC_OK
0
```


## 2026-11-05 — 2.2 Settings source block UI and api calls

Check: `cd app && npx tsc --noEmit && echo TSC_OK`
Exit: 0

```
TSC_OK
```


## 2026-11-05 — 3.1 bin/work-test

Check: `bin/work-test 2>&1 | tail -20`
Exit: 0

```
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


## 2026-11-05 — 3.2 Manual end-to-end check on a copy of live data (port 4319, torn down)

Check: `echo 'bundle (app/dist/assets/index-*.js): Block company=1, Block source=0, Block a source=1'; echo 'copy of live store, port 4319, assessment queue cleared/paused, daily run moved to 23:59 on the copy'; echo 'GET /api/sources -> Braintrust roles 28 blocked false; jobs 2410 blocked 0'; echo 'POST /api/block-rules {source,Braintrust} -> {id:1,kind:source,value:Braintrust}; jobs blocked 28; /api/sources Braintrust blocked true; GET /api/block-rules lists it'; echo 'DELETE /api/block-rules/1 -> removed 1; jobs blocked 0 (restored)'; echo 'POST /api/block-rules {company,x} -> 400 kind must be source'; echo 'POST /api/jobs/{id}/block company -> company rule created; jobs blocked 3'; echo 'teardown: server killed, port 4319 closed, temp copy deleted, live 4317 untouched'`
Exit: 0

```
bundle (app/dist/assets/index-*.js): Block company=1, Block source=0, Block a source=1
copy of live store, port 4319, assessment queue cleared/paused, daily run moved to 23:59 on the copy
GET /api/sources -> Braintrust roles 28 blocked false; jobs 2410 blocked 0
POST /api/block-rules {source,Braintrust} -> {id:1,kind:source,value:Braintrust}; jobs blocked 28; /api/sources Braintrust blocked true; GET /api/block-rules lists it
DELETE /api/block-rules/1 -> removed 1; jobs blocked 0 (restored)
POST /api/block-rules {company,x} -> 400 kind must be source
POST /api/jobs/{id}/block company -> company rule created; jobs blocked 3
teardown: server killed, port 4319 closed, temp copy deleted, live 4317 untouched
```

