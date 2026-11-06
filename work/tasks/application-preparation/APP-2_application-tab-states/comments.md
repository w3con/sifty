# APP-2 — comments

Running discussion. Newest at the bottom.

## 2026-11-04 — 1.1

Check: `npx tsx test/application-tab-states.ts 2>&1 | tail -6`
Exit: 0

```
Step 1.3 backed-off passed
Step 1.3 failed-with-retry passed
Step 1.3 not-pre-selected passed
Step 1.3 passed: all five outcomes show distinct states, none shows generic banner
Step 1.4 passed: stage default sends no provider, explicit sends provider, no-stage shows config prompt
{"passed":true,"steps":["1.1","1.2","1.3","1.4"]}
```


## 2026-11-04 — 1.2

Check: `npx tsx test/application-tab-states.ts 2>&1 | grep -E '1.2|passed'`
Exit: 0

```
Step 1.1 passed
Step 1.2 passed
Step 1.3 sign-in-required passed
Step 1.3 no-resolved-form passed
Step 1.3 backed-off passed
Step 1.3 failed-with-retry passed
Step 1.3 not-pre-selected passed
Step 1.3 passed: all five outcomes show distinct states, none shows generic banner
Step 1.4 passed: stage default sends no provider, explicit sends provider, no-stage shows config prompt
{"passed":true,"steps":["1.1","1.2","1.3","1.4"]}
```


## 2026-11-04 — 1.3

Check: `npx tsx test/application-tab-states.ts 2>&1 | grep -E '1.3|passed'`
Exit: 0

```
Step 1.1 passed
Step 1.2 passed
Step 1.3 sign-in-required passed
Step 1.3 no-resolved-form passed
Step 1.3 backed-off passed
Step 1.3 failed-with-retry passed
Step 1.3 not-pre-selected passed
Step 1.3 passed: all five outcomes show distinct states, none shows generic banner
Step 1.4 passed: stage default sends no provider, explicit sends provider, no-stage shows config prompt
{"passed":true,"steps":["1.1","1.2","1.3","1.4"]}
```


## 2026-11-04 — 1.4

Check: `npx tsx test/application-tab-states.ts 2>&1 | grep -E '1.4|passed'`
Exit: 0

```
Catalogue: http://127.0.0.1:4319
Step 1.1 passed
Step 1.2 passed
Step 1.3 sign-in-required passed
Step 1.3 no-resolved-form passed
Step 1.3 backed-off passed
Step 1.3 failed-with-retry passed
Step 1.3 not-pre-selected passed
Step 1.3 passed: all five outcomes show distinct states, none shows generic banner
Step 1.4 passed: stage default sends no provider, explicit sends provider, no-stage shows config prompt
{"passed":true,"steps":["1.1","1.2","1.3","1.4"]}
```


## 2026-11-04 — 2.1

Check: `npx tsx test/step21-live-verify.ts 2>&1`
Exit: 0

```
O3 passed: not-pre-selected shows Application tab disabled and selection prompt
O2 passed: no-resolved-form shows Apply via the original listing link
O1 passed: form with questions shows each question with "No answer yet" per-field state
O4 passed: prepare request sends no provider field when selector is at default stage
{"passed":true,"observations":["O1","O2","O3","O4"]}
```

