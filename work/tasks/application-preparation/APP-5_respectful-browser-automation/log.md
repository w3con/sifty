# APP-5 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-04 — 1.1 realistic fingerprint

Check: `cd sifty/workers/playwright && node --test --test-name-pattern '1\.1' automation.test.mjs`
Exit: 0

```
TAP version 13
# Subtest: 1.1 realistic fingerprint: UA and navigator.webdriver
ok 1 - 1.1 realistic fingerprint: UA and navigator.webdriver
  ---
  duration_ms: 159.53525
  type: 'test'
  ...
1..1
# tests 1
# suites 0
# pass 1
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 333.424459
```


## 2026-11-04 — 1.2 host pacing interval

Check: `cd sifty/workers/playwright && node --test --test-name-pattern '1\.2' automation.test.mjs`
Exit: 0

```
TAP version 13
# Subtest: 1.2 host pacing: sequential navigations are spaced by the configured interval
ok 1 - 1.2 host pacing: sequential navigations are spaced by the configured interval
  ---
  duration_ms: 313.277542
  type: 'test'
  ...
1..1
# tests 1
# suites 0
# pass 1
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 504.292167
```


## 2026-11-04 — 1.3 bounded concurrency

Check: `cd sifty/workers/playwright && node --test --test-name-pattern '1\.3' automation.test.mjs`
Exit: 0

```
TAP version 13
# Subtest: 1.3 concurrency: at most the configured number of navigations run at once
ok 1 - 1.3 concurrency: at most the configured number of navigations run at once
  ---
  duration_ms: 326.281084
  type: 'test'
  ...
1..1
# tests 1
# suites 0
# pass 1
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 562.971875
```


## 2026-11-04 — 2.1 block detection

Check: `cd sifty/workers/playwright && node --test --test-name-pattern '2\.1' automation.test.mjs`
Exit: 0

```
TAP version 13
# Subtest: 2.1 block detection: 429 response ends in backed-off state with no retry
ok 1 - 2.1 block detection: 429 response ends in backed-off state with no retry
  ---
  duration_ms: 150.900084
  type: 'test'
  ...
1..1
# tests 1
# suites 0
# pass 1
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 328.273167
```


## 2026-11-04 — 2.2 cooldown deferral

Check: `cd sifty/workers/playwright && node --test --test-name-pattern '2\.2' automation.test.mjs`
Exit: 0

```
TAP version 13
# Subtest: 2.2 cooldown: second inspect to blocked host is deferred; proceeds after cooldown
ok 1 - 2.2 cooldown: second inspect to blocked host is deferred; proceeds after cooldown
  ---
  duration_ms: 376.8075
  type: 'test'
  ...
1..1
# tests 1
# suites 0
# pass 1
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 553.116
```


## 2026-11-04 — 3.1 config from runtime.yaml

Check: `cd sifty/workers/playwright && node --test --test-name-pattern '3\.1' automation.test.mjs`
Exit: 0

```
TAP version 13
# Subtest: 3.1 config from runtime.yaml: interval override is observed; absent key uses default
ok 1 - 3.1 config from runtime.yaml: interval override is observed; absent key uses default
  ---
  duration_ms: 262.439875
  type: 'test'
  ...
1..1
# tests 1
# suites 0
# pass 1
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 462.789334
```


## 2026-11-04 — move browser discipline to Go

Check: `go test -race ./... (sifty/) and node --test automation.test.mjs`
Exit: 2

```
/bin/sh: -c: line 0: syntax error near unexpected token `('
/bin/sh: -c: line 0: `go test -race ./... (sifty/) and node --test automation.test.mjs'
```


## 2026-11-04 — move browser discipline to Go: go tests

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
ok  	github.com/w3con/sifty/internal/inspect	(cached)
?   	github.com/w3con/sifty/internal/model	[no test files]
ok  	github.com/w3con/sifty/internal/prepare	(cached)
?   	github.com/w3con/sifty/internal/prompts	[no test files]
ok  	github.com/w3con/sifty/internal/records	(cached)
ok  	github.com/w3con/sifty/internal/server	(cached)
ok  	github.com/w3con/sifty/internal/store	(cached)
?   	github.com/w3con/sifty/internal/version	[no test files]
ok  	github.com/w3con/sifty/internal/worker	(cached)
?   	github.com/w3con/sifty/scripts/capture-fixtures	[no test files]
```


## 2026-11-04 — move browser discipline to Go: node tests

Check: `cd sifty/workers/playwright && node --test automation.test.mjs`
Exit: 0

```
TAP version 13
# Subtest: 1.1 realistic fingerprint: UA and navigator.webdriver
ok 1 - 1.1 realistic fingerprint: UA and navigator.webdriver
  ---
  duration_ms: 157.26825
  type: 'test'
  ...
# Subtest: 1.2 host pacing: sequential navigations are spaced by the configured interval
ok 2 - 1.2 host pacing: sequential navigations are spaced by the configured interval
  ---
  duration_ms: 307.010125
  type: 'test'
  ...
# Subtest: 1.3 concurrency: at most the configured number of navigations run at once
ok 3 - 1.3 concurrency: at most the configured number of navigations run at once
  ---
  duration_ms: 327.76125
  type: 'test'
  ...
# Subtest: 2.1 block detection: 429 response ends in backed-off state with no retry
ok 4 - 2.1 block detection: 429 response ends in backed-off state with no retry
  ---
  duration_ms: 163.817375
  type: 'test'
  ...
# Subtest: 2.2 cooldown: second inspect to blocked host is deferred; proceeds after cooldown
ok 5 - 2.2 cooldown: second inspect to blocked host is deferred; proceeds after cooldown
  ---
  duration_ms: 367.95425
  type: 'test'
  ...
1..5
# tests 5
# suites 0
# pass 5
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 1500.730625
```

