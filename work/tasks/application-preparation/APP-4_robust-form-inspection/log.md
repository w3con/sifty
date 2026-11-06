# APP-4 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-03 — 1.1 — ATS click-through on Ashby fixture

Check: `cd sifty && go test ./internal/inspect/ -run TestInspectAshbyClickThrough -v -timeout 120s`
Exit: 0

```
=== RUN   TestInspectAshbyClickThrough
--- PASS: TestInspectAshbyClickThrough (0.42s)
PASS
ok  	github.com/w3con/sifty/internal/inspect	(cached)
```


## 2026-11-03 — 1.2 — Generic inline-form fixture reads fields unchanged

Check: `cd sifty && go test ./internal/inspect/ -run TestInspectGenericInlineFormUnaffected -v -timeout 120s`
Exit: 0

```
=== RUN   TestInspectGenericInlineFormUnaffected
--- PASS: TestInspectGenericInlineFormUnaffected (0.36s)
PASS
ok  	github.com/w3con/sifty/internal/inspect	(cached)
```


## 2026-11-03 — 2.1 — Honest done states for download destination and no resolved apply URL

Check: `cd sifty && go test ./internal/background/ -run 'TestInspectDownloadDestinationIsDone|TestInspectEmptyApplyURLIsDone' -v -timeout 30s`
Exit: 0

```
=== RUN   TestInspectDownloadDestinationIsDone
--- PASS: TestInspectDownloadDestinationIsDone (0.07s)
=== RUN   TestInspectEmptyApplyURLIsDone
--- PASS: TestInspectEmptyApplyURLIsDone (0.07s)
PASS
ok  	github.com/w3con/sifty/internal/background	(cached)
```


## 2026-11-03 — 3.1 — Playwright worker dependencies installed via setup.sh

Check: `sifty/workers/playwright/setup.sh && echo '{"method":"inspectForm","params":{"html":"<html lang=\"en\"><body><form><label for=\"n\">Name</label><input id=\"n\" name=\"name\"></form></body></html>"}}' | node sifty/workers/playwright/index.mjs`
Exit: 0

```
Installing Playwright npm dependencies...

up to date, audited 3 packages in 346ms

found 0 vulnerabilities
Installing Chromium browser binary...
Playwright worker ready.
{"ok":true,"result":{"url":"about:blank","capturedAt":"2026-11-03T22:14:31.226Z","language":"en","status":"partial","gaps":["Visible accessible controls only; hidden conditional and gated later steps have not been inspected. No fields were filled or submitted."],"fields":[{"id":"54c232934245c80c4baf05d8d5b7844fd8547f5e918cc1ba020a752870bc84fa","label":"Name","instructions":"","labelStatus":"resolved","type":"text","options":[],"required":false,"maxLength":null,"accept":"","sourceUrl":"about:blank","origin":"site","language":"en","signature":"c8bfc4076f4af8472175105475c2b22df33fdc3329a52defc54a5a952e439e20","controlName":"name"}],"family":"Generic rendered HTML/accessibility controls"}}
```


## 2026-11-03 — 3.2 — End-to-end: job 458 Ashby form questions, job 758 honest done state

Check: `curl -s 'http://127.0.0.1:4317/api/jobs/458/application' | python3 -c "import sys,json; d=json.load(sys.stdin); dr=d.get('draft',{}); ops=d.get('operations',[]); op20=next((o for o in ops if o.get('id')==20),None); print('458 fields:', len(dr.get('fields',[]))); print('458 op:', op20.get('status') if op20 else 'n/a', '|', op20.get('progress','')[:60] if op20 else ''); q=dr.get('questionnaire',{}); print('458 Q-status:', q.get('status') if isinstance(q,dict) else '?'); [print(' ',f.get('label','?')[:50], f.get('type','?')) for f in dr.get('fields',[])[:6]]"; echo; curl -s 'http://127.0.0.1:4317/api/jobs/758/application' | python3 -c "import sys,json; d=json.load(sys.stdin); ops=d.get('operations',[]); op21=next((o for o in ops if o.get('id')==21),None); print('758 op:', op21.get('status') if op21 else 'n/a', '|', op21.get('progress','')[:60] if op21 else '')"`
Exit: 0

```
458 fields: 6
458 op: done | Questionnaire ready for review; nothing submitted
458 Q-status: partial
  Question wording unavailable — review the source file
  Name text
  Email text
  Resume file
  Start typing... select
  GitHub text

758 op: done | no resolved application form; apply via the original listing
```

