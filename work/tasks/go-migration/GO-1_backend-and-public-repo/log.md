# GO-1 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-01 — 1.1 — public repo skeleton (LICENSE, README, .gitignore)

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-1.1.sh`
Exit: 0

```
== tracked/staged files ==
A  .gitignore
A  LICENSE
A  README.md
== assert no private path staged ==
PASS: no private path is tracked
== git check-ignore confirms exclusion ==
ai/secret.md
kb/note.md
.env
cv.pdf
profile.pdf
data/store.sqlite
.local
```


## 2026-11-01 — 1.2 — vendor pilier-style leak scanner under scripts/leakcheck with census

Check: `python3 sifty/scripts/leakcheck/leakcheck-selftest`
Exit: 0

```
ok  flag  [cloud access key]
ok  flag  [personal access token]
ok  flag  [cloud API key]
ok  flag  [JSON Web Token]
ok  flag  [private key (PEM block)]
ok  flag  [secret assigned a real value]
ok  flag  [.env file content]
ok  pass  'fitScore := assess(candidate, posting)'
ok  pass  'api_key = ${SIFTY_API_KEY}'
ok  pass  'export API_KEY from the environment'
ok  pass  'token: your-token-here'
ok  pass  'password = <changeme>'
ok  pass  '// read the key from a local file the .gitignore excludes'

PASS: 7 seeded secrets flagged, 6 clean lines passed
```


## 2026-11-01 — 1.3 — install leak-check hooks via relative core.hooksPath and a setup script

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-1.3.sh`
Exit: 0

```
setup: core.hooksPath = .githooks
setup: leak-check hooks installed (pre-commit, commit-msg, pre-push)
hooksPath=.githooks
== Test A: commit carrying a seeded secret is refused ==
PASS-A: refused (cloud access key named)
== Test B: a clean commit passes ==
PASS-B: clean commit accepted
== Test C: a --no-verify secret commit is caught by the pre-push re-scan ==
committed with --no-verify: d9766f8
PASS-C: pre-push caught the --no-verify'd secret
ALL PASS
```


## 2026-11-01 — 1.4 — CI workflow runs leak scan and build-and-test entry

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-1.4.sh`
Exit: 0

```
== workflow present ==
PASS: .github/workflows/ci.yml present
== test entry runs locally (in a fresh repo root) ==
== leak-scanner self-test ==
ok  flag  [cloud access key]
ok  flag  [personal access token]
ok  flag  [cloud API key]
ok  flag  [JSON Web Token]
ok  flag  [private key (PEM block)]
ok  flag  [secret assigned a real value]
ok  flag  [.env file content]
ok  pass  'fitScore := assess(candidate, posting)'
ok  pass  'api_key = ${SIFTY_API_KEY}'
ok  pass  'export API_KEY from the environment'
ok  pass  'token: your-token-here'
ok  pass  'password = <changeme>'
ok  pass  '// read the key from a local file the .gitignore excludes'

PASS: 7 seeded secrets flagged, 6 clean lines passed
== go: no go.mod yet, skipping build and test ==
test: OK
```


## 2026-11-01 — 2.1 — Go module and package layout with single-binary cmd entry

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-2.1.sh`
Exit: 0

```
built: /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.JMIIhaZbDT/sifty
== runs: version ==
0.1.0-dev
== runs: help ==
help ok
```


## 2026-11-01 — 2.2 — configuration and data paths from a configurable data directory

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-2.2.sh`
Exit: 0

```
== prohibitions absent from Go code ==
PASS: none of the forbidden forms present
== environment-based data directory ==
data directory: /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.iow1IgzLtQ (from SIFTY_DATA_DIR)
config: none present (skipped)
wrote answer: /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.iow1IgzLtQ/answers/2026-11-01-selfcheck-answer.md
wrote feedback: /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.iow1IgzLtQ/feedback/2026-11-01-selfcheck-feedback.md
read back: 1 answer(s), 1 feedback record(s)
PASS-env: answer and feedback written under $SIFTY_DATA_DIR
== flag-based data directory ==
data directory: /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.8PLikn9Kwz (from --data-dir)
config: none present (skipped)
wrote answer: /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.8PLikn9Kwz/answers/2026-11-01-selfcheck-answer.md
wrote feedback: /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.8PLikn9Kwz/feedback/2026-11-01-selfcheck-feedback.md
read back: 1 answer(s), 1 feedback record(s)
PASS-flag: answer and feedback written under --data-dir
== config read from the data directory ==
ALL PASS
```


## 2026-11-01 — 2.3 — answer and feedback file templates

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-2.3.sh`
Exit: 0

```
=== RUN   TestAnswerFirstWriteMatchesTemplate
--- PASS: TestAnswerFirstWriteMatchesTemplate (0.00s)
=== RUN   TestFeedbackFirstWriteMatchesTemplate
--- PASS: TestFeedbackFirstWriteMatchesTemplate (0.00s)
=== RUN   TestWriteIsExclusive
--- PASS: TestWriteIsExclusive (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/records	0.297s
```


## 2026-11-01 — 2.4 — Go-core to TypeScript-worker boundary with stub round-trip

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-2.4.sh`
Exit: 0

```
=== RUN   TestStubWorkerRoundTrip
--- PASS: TestStubWorkerRoundTrip (0.04s)
=== RUN   TestWorkerReportsHandledError
--- PASS: TestWorkerReportsHandledError (0.03s)
=== RUN   TestWorkerTimeout
--- PASS: TestWorkerTimeout (0.30s)
PASS
ok  	github.com/w3con/sifty/internal/worker	0.567s
```


## 2026-11-01 — 3.1 — native Go structured collection with jobspy worker sidecar

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-3.1.sh`
Exit: 0

```
=== RUN   TestStructuredJobsParity
--- PASS: TestStructuredJobsParity (0.00s)
=== RUN   TestParseFeedParity
--- PASS: TestParseFeedParity (0.00s)
=== RUN   TestHTMLLinksGetroParity
--- PASS: TestHTMLLinksGetroParity (0.00s)
=== RUN   TestHTMLLinksAnchorsParity
--- PASS: TestHTMLLinksAnchorsParity (0.00s)
=== RUN   TestJobspySidecar
--- PASS: TestJobspySidecar (0.05s)
PASS
ok  	github.com/w3con/sifty/internal/collect	(cached)
```


## 2026-11-01 — 3.2 — fit assessment with provider-configurable model call, parity within tolerance

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-3.2.sh`
Exit: 0

```
=== RUN   TestAssessParity
--- PASS: TestAssessParity (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/assess	(cached)
```


## 2026-11-01 — 4.1 — Application-answer preparation in Go (the JOB2 flow)

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-4.1.sh`
Exit: 0

```
=== RUN   TestPrepareParity
--- PASS: TestPrepareParity (0.00s)
=== RUN   TestPrepareGuards
=== RUN   TestPrepareGuards/no_unanswered_fields
=== RUN   TestPrepareGuards/unknown_question
=== RUN   TestPrepareGuards/duplicate_question
=== RUN   TestPrepareGuards/unsupported_evidence
=== RUN   TestPrepareGuards/type_mismatch_list_for_text
=== RUN   TestPrepareGuards/type_mismatch_scalar_for_checkbox
=== RUN   TestPrepareGuards/invented_option
=== RUN   TestPrepareGuards/exceeded_maxLength
=== RUN   TestPrepareGuards/valid_answer
--- PASS: TestPrepareGuards (0.00s)
    --- PASS: TestPrepareGuards/no_unanswered_fields (0.00s)
    --- PASS: TestPrepareGuards/unknown_question (0.00s)
    --- PASS: TestPrepareGuards/duplicate_question (0.00s)
    --- PASS: TestPrepareGuards/unsupported_evidence (0.00s)
    --- PASS: TestPrepareGuards/type_mismatch_list_for_text (0.00s)
    --- PASS: TestPrepareGuards/type_mismatch_scalar_for_checkbox (0.00s)
    --- PASS: TestPrepareGuards/invented_option (0.00s)
    --- PASS: TestPrepareGuards/exceeded_maxLength (0.00s)
    --- PASS: TestPrepareGuards/valid_answer (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/prepare	(cached)
```


## 2026-11-01 — 4.2 — Form inspection through the TypeScript Playwright worker invoked by the Go core

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-4.2.sh`
Exit: 0

```
== Playwright worker present ==
PASS: workers/playwright/index.mjs present
== no browser-automation dependency or code in Go ==
PASS: no browser-automation dependency or import in Go
== form fields returned via the worker ==
=== RUN   TestInspectFormViaWorker
--- PASS: TestInspectFormViaWorker (0.33s)
PASS
ok  	github.com/w3con/sifty/internal/inspect	(cached)
```


## 2026-11-02 — 5.1 — end-to-end parity run against the current TypeScript application on a shared fixture

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-5.1.sh`
Exit: 0

```
==============================================================
 GO-1 5.1 — END-TO-END PARITY REPORT (Go vs current TypeScript)
==============================================================

Shared scenario: CTO vacancy at Acme; candidate with Go/Rust backing.
Each stage diffs the Go output against the TypeScript golden for that
stage (golden = output of app/src, generated by checks/tsgolden/gen-*.mjs).

== Build: single binary ==
PASS: single binary builds and runs

== Stage 1a — structured collection (parsers) ==
ok  	github.com/w3con/sifty/internal/collect	0.195s
PASS: collection parsers match the TypeScript goldens (jsonld, feed, getro, anchors)

== Stage 1b — fit assessment ==
ok  	github.com/w3con/sifty/internal/assess	0.204s
PASS: grounded assessment matches the TypeScript golden within tolerance

== Stage 2a — application-answer preparation (JOB2) ==
ok  	github.com/w3con/sifty/internal/prepare	0.257s
PASS: prepared answers match the TypeScript golden

== Stage 2b — form inspection via the Playwright worker ==
=== RUN   TestInspectFormViaWorker
--- PASS: TestInspectFormViaWorker (0.35s)
PASS
ok  	github.com/w3con/sifty/internal/inspect	0.512s
PASS: form fields returned through the Go core via the TypeScript Playwright worker

== Golden provenance against the current TypeScript ==
live-TS regeneration skipped: app/node_modules not installed.
Goldens were generated from the current TypeScript app/src on 2026-11-01 by
checks/tsgolden/gen-*.mjs; app/ is unchanged since (the card forbids touching it),
so the committed goldens are the current TypeScript application's output.

== Explained difference ==
The model inference call is injected identically on the TypeScript golden side and
the Go side (full model separation is a later card). Parity therefore covers all
deterministic logic and holds the single non-deterministic step constant on both sides.

== Result ==
PARITY: Go matches the current TypeScript application across Stages 1 and 2.
```


## 2026-11-02 — 5.2 — full-tree leak sweep of the public repository (NOT clean; push withheld)

Check: `bash ai/tasks/go-migration/GO-1_backend-and-public-repo/checks/check-5.2.sh`
Exit: 1

```
== publishable tree ==
tracked files: 65
node_modules tracked: 0
origin: github.com/w3con/sifty

== full-tree leak sweep ==
tree-check: refused — /private/var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/tmp.2NVn4Te0HF carries 10 form(s) this public repository must not:
go.mod:6: [untidy] reference to a private repository
    github.com/PuerkitoBio/goquery v1.13.0
    replace with: name the kind of information without naming its address
go.mod:11: [untidy] reference to a private repository
    github.com/andybalholm/cascadia v1.3.4 // indirect
    replace with: name the kind of information without naming its address
go.sum:1: [untidy] reference to a private repository
    github.com/PuerkitoBio/goquery v1.13.0 h1:mqHbjD7Jmnul4DTR24LKTjo1uUmHUh072kteGV+xpFM=
    replace with: name the kind of information without naming its address
go.sum:2: [untidy] reference to a private repository
    github.com/PuerkitoBio/goquery v1.13.0/go.mod h1:Hip5mdBL8K2wEGKJdr27sRaNwIdDajmCwB/ExUPwW+g=
    replace with: name the kind of information without naming its address
go.sum:3: [untidy] reference to a private repository
    github.com/andybalholm/cascadia v1.3.4 h1:vM2lgh0Vru9Vwyfm4cQqWP2HHMW0u0+2PAW7Q38Qufg=
    replace with: name the kind of information without naming its address
go.sum:4: [untidy] reference to a private repository
    github.com/andybalholm/cascadia v1.3.4/go.mod h1:BLRmbRjpEtNKieZOCCvYj4RqN+KRA41GBe/5O+G93kM=
    replace with: name the kind of information without naming its address
internal/assess/assess.go:110: [untidy] Russian-language text
    absenceRe       = regexp.MustCompile(`(?i)\black|does not (?:know|have)|no .{0,30}experience|not know|не зна|нет опыта`)
    replace with: write the same sentence in English
internal/assess/assess.go:111: [untidy] Russian-language text
    feedbackQuoteRe = regexp.MustCompile(`(?i)not known|no experience|don't know|do not know|does not know|не знаю|нет опыта`)
    replace with: write the same sentence in English
internal/collect/parse.go:12: [untidy] reference to a private repository
    "github.com/PuerkitoBio/goquery"
    replace with: name the kind of information without naming its address
internal/collect/text.go:8: [untidy] reference to a private repository
    "github.com/PuerkitoBio/goquery"
    replace with: name the kind of information without naming its address
```

