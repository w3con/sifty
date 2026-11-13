# COL-6 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-05 — 1.1

Check: `cd sifty && go build ./... && go test ./internal/store/ -run Schema`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-05 — 1.2

Check: `cd sifty && go test -race ./internal/store/ -run 'StopWord'`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	1.406s
```


## 2026-11-05 — 1.3

Check: `cd sifty && go test -race ./...`
Exit: 0

```
?   	github.com/w3con/sifty/cmd/sifty	[no test files]
ok  	github.com/w3con/sifty/internal/assess	(cached)
ok  	github.com/w3con/sifty/internal/background	5.996s
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
ok  	github.com/w3con/sifty/internal/store	3.256s
?   	github.com/w3con/sifty/internal/version	[no test files]
ok  	github.com/w3con/sifty/internal/worker	(cached)
?   	github.com/w3con/sifty/scripts/capture-fixtures	[no test files]
```


## 2026-11-05 — 1.4

Check: `cd sifty && go test -race ./internal/store/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-05 — 2.1

Check: `cd sifty && go test -race ./internal/background/ -run 'Collect'`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/background	1.788s
```


## 2026-11-05 — 2.2

Check: `cd sifty && go test -race ./internal/...`
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


## 2026-11-05 — 3.1

Check: `cd app && npx tsc --noEmit`
Exit: 0

```

```


## 2026-11-05 — 3.2

Check: `cd app && npx tsc --noEmit`
Exit: 0

```

```


## 2026-11-05 — 3.3

Check: `cd app && npx tsc --noEmit`
Exit: 0

```

```


## 2026-11-05 — 4.1

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


## 2026-11-05 — 4.2

Check: `echo -n 'bundle strings (Stop-words, Would hide, Add stop-word): '; grep -o 'Stop-words\|Would hide\|Add stop-word' /private/tmp/claude-501/-Volumes-Documents-Work-FCTO-app/81af53fb-fe6d-4a0c-9c1b-cb0464f96b06/scratchpad/dist/assets/index-*.js | sort | uniq -c | tr '\n' ' '; echo; (cd sifty && go test -race ./internal/background/ -run TestCollectSkipsStopWordPostings -v 2>&1 | grep -E '^(---|ok|PASS|FAIL)'); echo '--- isolated end-to-end on a copy of live data (port 4319; assessment queue disabled+paused, daily run at 23:59 on the copy; collection not triggered, so the collection skip is proven by the test above)'; /private/tmp/claude-501/-Volumes-Documents-Work-FCTO-app/81af53fb-fe6d-4a0c-9c1b-cb0464f96b06/scratchpad/e2e.sh`
Exit: 0

```
-n bundle strings (Stop-words, Would hide, Add stop-word): 
   1 Add stop-word    1 Stop-words    1 Would hide 
--- PASS: TestCollectSkipsStopWordPostings (0.08s)
PASS
ok  	github.com/w3con/sifty/internal/background	1.453s
--- isolated end-to-end on a copy of live data (port 4319; assessment queue disabled+paused, daily run at 23:59 on the copy; collection not triggered, so the collection skip is proven by the test above)
port: 4319 daily_hour: 23 daily_minute: 59 
copy store user_version before: 11
  list status=triage: 2335
  list status=preselected: 11
  list status=applied: 4
  list status=no: 60
user_version after migration: 12
--- baseline
total 2410 {'applied': 4, 'no': 60, 'preselected': 11, 'triage': 2335} stopped_flag_in_list 0
--- GET /api/stop-words
[]

--- preview engineer
{"count":1249,"word":"engineer"}

--- stored roles with engineer in title (sqlite): 1249
--- POST add
{"id":1,"word":"engineer","createdAt":"2026-11-05T02:33:44.538Z"}

--- after add
total 1161 {'applied': 1, 'no': 42, 'preselected': 2, 'triage': 1116} stopped_flag_in_list 0
  list status=triage: 1116
  list status=preselected: 2
  list status=applied: 1
  list status=no: 42
--- sqlite rows still present: 2410
--- one stopped role via GET /api/jobs/{id}
[Hiring] Head of Engineering @Lemon.io stopped= True stoppedBy= engineer
--- preview after add (already hidden -> 0)
{"count":0,"word":"engineer"}

--- empty add -> 400
400
--- DELETE
{"removed":1}

--- after remove
total 2410 {'applied': 4, 'no': 60, 'preselected': 11, 'triage': 2335} stopped_flag_in_list 0
--- block rules untouched: [{"id":4,"kind":"company","value":"ilant health","createdAt":"2026-11-05T02:03:41.378Z"},{"id":3,"kind":"company","value":"mindcare solutions","createdAt":"2026-11-05T02:03:38.309Z"},{"id":2,"kind":"company","value":"innovecs","createdAt":"2026-11-05T02:00:01.119Z"},{"id":1,"kind":"company","value":"chainlink labs","createdAt":"2026-11-05T01:59:37.649Z"}]
teardown: server pid 10704 stopped; port 4319 listening? 0 ; live 4317 listening? 1 (untouched)
temp copy removed: yes
```

