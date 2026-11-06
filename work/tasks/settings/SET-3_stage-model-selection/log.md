# SET-3 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-04 — 1.1 — Add Preparation to config.Providers.Stages and include it in both validation loops

Check: `go test ./sifty/internal/config/... -v -run TestPreparationStage|TestSavePreparationStage`
Exit: 127

```
/bin/sh: TestSavePreparationStage: command not found
# ./sifty/internal/config/...
pattern ./sifty/internal/config/...: directory prefix sifty/internal/config does not contain main module or its selected dependencies
```


## 2026-11-04 — 1.1 — Add Preparation to config.Providers.Stages and include it in both validation loops

Check: `go test ./sifty/internal/config/... -run TestPreparationStage`
Exit: 1

```
# ./sifty/internal/config/...
pattern ./sifty/internal/config/...: directory prefix sifty/internal/config does not contain main module or its selected dependencies
FAIL	./sifty/internal/config/... [setup failed]
FAIL
```


## 2026-11-04 — 1.1

Check: `go test`
Exit: 1

```
go: cannot find main module, but found .git/config in /Volumes/Documents/Work/FCTO
	to create a module there, run:
	cd ../.. && go mod init
```


## 2026-11-04 — 1.1 — Add Preparation to config.Providers.Stages and include it in both validation loops

Check: `cd sifty && go test ./internal/config/... -run TestPreparationStage`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/config	(cached)
```


## 2026-11-04 — 1.2 — Change defaultPrepareJob to default to stages.preparation with request override

Check: `cd sifty && go test ./internal/background/... -run TestDefaultPrepareJobProviderResolution`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/background	0.270s
```


## 2026-11-04 — 2.1 — Add GET /api/stage-providers endpoint returning stage selections and provider list

Check: `cd sifty && go test ./internal/server/... -run TestGetStageProviders`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	0.281s
```


## 2026-11-04 — 2.2 — Add PUT /api/stage-providers endpoint that sets one stage provider, writing providers.yaml and reloading

Check: `cd sifty && go test ./internal/server/... -run TestPutStageProviders`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	0.377s
```


## 2026-11-04 — 3.1 — Add Stage models section to Configuration tab with labelled selects per stage

Check: `cd app && node_modules/.bin/tsc --noEmit && grep -c 'stage-models' web/Settings.tsx`
Exit: 0

```
1
```


## 2026-11-04 — 3.2 — Rebuild frontend bundle so Go-served app carries the Stage models section

Check: `cd app && grep -c 'Stage models' dist/assets/*.js`
Exit: 0

```
1
```


## 2026-11-04 — 4.1 — Live verification: set preparation to luna, confirmed provider used; set assessment to openai, confirmed change with no restart

Started own server on port 4319 against a copy of the data dir. GET /api/stage-providers showed preparation empty, assessment=luna. PUT /api/stage-providers {stage:preparation,provider:luna} updated providers.yaml and live config. Queued preparation for job 339 with no request provider; operation failed with content error (no unanswered readable questions) rather than a provider error, confirming provider was resolved from stages.preparation=luna. PUT /api/stage-providers {stage:assessment,provider:openai} changed assessment default to openai; GET confirmed assessment=openai, preparation=luna, with no restart.


## 2026-11-04 — 2.2 fix: preserve providers.yaml on stage set

Check: `go test ./internal/server/... ./internal/config/... ./internal/background/...`
Exit: 1

```
# ./internal/server/...
pattern ./internal/server/...: directory prefix internal/server does not contain main module or its selected dependencies
FAIL	./internal/server/... [setup failed]
# ./internal/config/...
pattern ./internal/config/...: directory prefix internal/config does not contain main module or its selected dependencies
FAIL	./internal/config/... [setup failed]
# ./internal/background/...
pattern ./internal/background/...: directory prefix internal/background does not contain main module or its selected dependencies
FAIL	./internal/background/... [setup failed]
FAIL
```


## 2026-11-04 — 2.2 fix: preserve providers.yaml on stage set

Check: `cd sifty && go test ./internal/server/... ./internal/config/... ./internal/background/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
ok  	github.com/w3con/sifty/internal/config	(cached)
ok  	github.com/w3con/sifty/internal/background	(cached)
```

