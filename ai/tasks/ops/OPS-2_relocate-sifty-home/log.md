# OPS-2 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-05 — 1.1 — verify ASM-4 merged

Check: `git -C /Volumes/Documents/Work/FCTO log --oneline -3`
Exit: 0

```
0846645 chore(work): finish ASM-4
c0f1331 chore(tasks): close ASM-4 4.2 (manual e2e verified on live-data copy)
b7c95d9 chore(tasks): record steps 2.1–4.1 in log
```


## 2026-11-05 — 2.1 — new repo and app files

Check: `git -C /Volumes/SSD/Dev/tknff/sifty log --oneline -1`
Exit: 0

```
5d8b361 feat(sifty): initial snapshot — app, frontend, config, prompts, site
```


## 2026-11-05 — 2.1 — new repo and app files (go build)

Check: `cd /Volumes/SSD/Dev/tknff/sifty/sifty && go build ./...`
Exit: 0

```

```


## 2026-11-05 — 2.2 — dev harness

Check: `cd /Volumes/SSD/Dev/tknff/sifty && python3 bin/work-test`
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


## 2026-11-05 — 2.3 — Makefile; go test and build pass

Check: `cd /Volumes/SSD/Dev/tknff/sifty/sifty && go test -race ./... 2>&1 | tail -5`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/source	(cached)
ok  	github.com/w3con/sifty/internal/store	(cached)
?   	github.com/w3con/sifty/internal/version	[no test files]
ok  	github.com/w3con/sifty/internal/worker	(cached)
?   	github.com/w3con/sifty/scripts/capture-fixtures	[no test files]
```


## 2026-11-05 — 3.1 — data/config/answers/evidence/feedback copied

Check: `ls /Volumes/SSD/Dev/tknff/sifty/data/ && cat /Volumes/SSD/Dev/tknff/sifty/data/config/runtime.yaml | grep port`
Exit: 0

```
answers
config
evidence
feedback
store.sqlite
port: 8080
# Measured discovery bounds, independent from imported matches and model calls.
```


## 2026-11-05 — 3.2 — app running on port 8080

Check: `curl -s -o /dev/null -w '%{http_code}' http://127.0.0.1:8080/`
Exit: 0

```
200
```


## 2026-11-05 — 5.1 — end-to-end: build, run on 8080, data git-ignored, scores populated

Check: `sqlite3 /Volumes/SSD/Dev/tknff/sifty/data/store.sqlite 'SELECT COUNT(*) FROM jobs WHERE match_score>0;'`
Exit: 0

```
1642
```


## 2026-11-05 — 5.1 — build command check

Check: `cd /Volumes/SSD/Dev/tknff/sifty && rm -f bin/sifty && rm -rf app/dist && make build 2>&1 | tail -3`
Exit: 0

```
../dist/assets/index-LzYizq43.css                                 26.52 kB │ gzip:  6.49 kB
../dist/assets/index-CN5R6hXf.js                                 288.46 kB │ gzip: 89.48 kB
✓ built in 412ms
```


## 2026-11-05 — note

Open item for coordinator: data/config/candidate.yaml references kb/Strategy/positioning.md and kb/faq.md relative to the data directory. These files remain in the FCTO repo. After cleanup (4.1/4.2), these references will be broken. Options: (a) copy those two KB files to data/kb/, (b) update candidate.yaml to remove them, or (c) make absolute paths. The cv.pdf and profile.pdf were already copied to data/. Scoring still works (1642 jobs scored) because match_score comes from the fast keyword ranking (ASM-4), not the AI context.


## 2026-11-05 — 4.1 — remove app trees and Sifty product epics from FCTO

Check: `git -C /Volumes/Documents/Work/FCTO/.worktrees/OPS-2 diff --cached --stat | tail -5`
Exit: 0

```
 site/profile.pdf                                   | 4883 --------------------
 site/resume-fr.html                                |  101 -
 site/tikonoff-qr.png                               |  Bin 1613 -> 0 bytes
 site/tikonoff-qr.svg                               |    2 -
 518 files changed, 7 insertions(+), 49378 deletions(-)
```


## 2026-11-05 — 4.2 — update FCTO CLAUDE.md and README.md

CLAUDE.md: removed app/ reference from opening sentence, added Sifty home pointer, removed job-search-assistant.md paragraph. README.md: added 'Sifty application' section with pointer to new home.


## 2026-11-05 — note

Correction: site/ was wrongly removed from FCTO and added to the new home. site/ is the CV/portfolio website (cv.html, profile.html, resume-fr.html, Validité assets, fonts, QR codes), a job-search deliverable maintained by the cv/website task cards that stayed in FCTO. Restored to FCTO via git checkout main -- site/; removed from new home via git rm. Confirmed no reference to site/ path in Makefile, .gitignore, Go cmd, or config in the new home. FCTO diff main..work/OPS-2 shows no site/ entries.

