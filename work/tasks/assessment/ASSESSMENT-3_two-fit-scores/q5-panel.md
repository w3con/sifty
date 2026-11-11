# Q.5 — panel and catalogue: what was checked

Data: a fresh copy of `data-staging` in `.local/assessment-3/data` (taken with `sqlite3 .backup`, because a preview server on port 8082 is reading the staging store; the previous copy was kept as `data-q4`). Server on port 8094 only, binary and `frontend/dist` from this branch, provider `qwen3.5-9b`. Changes made to the copy and nothing else: `runtime.yaml` with the `quick_match` block (`daily_limit` 4 to 9 while testing, `daily_hour` 23:59), the vacancy text of Parity (84) put back from the previous copy (staging holds 28 characters, its detail fetch failed with HTTP 429), `first_seen` of 84, 22, 2739 and 2508 set to now so that the worker takes them first, and profile revisions 22 to 25 saved through the API (explained below).

## Before and after re-matching with the new profile

The copy's profile is revision 21 with 11 experience entries. To show what re-matching does, the experience section was emptied on the copy (revision 24), the three vacancies were analysed and photographed, the experience was restored (revision 25) and only the matching pass ran again. The extraction rows were byte for byte unchanged (hash `729e1152f966` before and after) and no extraction ran for the profile save.

| Vacancy | Without experience | With experience |
|---|---|---|
| Parity, Tech Lead (84) | Fits you 82 · You fit 69 (2 match, 0 partial, 0 no, 3 unclear) | Fits you 82 · You fit 88 (4 match, 0 partial, 0 no, 1 unclear) |
| Ethena Labs, Engineering Manager / Tech Lead (22) | 58 · 55 (1 match, 10 unclear) | 58 · 57 (2 match, 9 unclear) |
| Solana, Senior Security Engineer (2508), a page with no requirements | 45 · No requirements stated | the same: it does not depend on the profile |

Parity: the three requirements about leading teams, being a lead or manager and having founded a company moved from "Missing from your profile" to "Matches", citing the computed facts ("9 years in leadership roles", "Founded or co-founded a company"); only "Excellent asynchronous communication skills" stays missing (a soft skill the profile does not state).

## Files (`q5-screenshots/`, panel crop `panel-<id>-<phase>.png`, whole window `window-<id>-<phase>.png`)

- `before-experience`: 84, 22. `after-experience`: 84, 22, 2508.
- `old-panel`: 84, 22, the panel as it was before Q.5 on the same data (single score, coverage percentage, group arithmetic, provenance).
- `state-limit` (1035): a vacancy waiting for analysis, worker idle.
- `state-model-down` (1035): the same vacancy with the extraction provider pointed at an unreachable address on the copy: the retry-kind line "Requirements could not be analysed — the local model is not running." and "Not analysed yet" instead of a number.
- `19-before-experience`: a vacancy whose description could not be fetched: no numbers, "Full job description is unavailable", Refresh.
- The first `before-experience` set and the first `after-experience` set were retaken after the list cap and the shortened evidence were added, so that all images show the final panel; `19` and the two `state-*` sets were taken one build earlier (the list cap and the shortened evidence do not appear in those states).

## Checks

- `npm run build` (tsc and vite): clean. `node --test frontend/quickmatch.test.mjs`: 7 of 7 pass (numbers and buckets, states without a number, model-not-running wording free of mechanics, catalogue order including pending, stale, unavailable and no-result rows, Fit filter, line wording and shortening, profile section for a missing requirement).
- CSS added in this step contains no literal colour except two new tokens, `--fit-strong-bg` and `--fit-strong-ink` (the design system has no green for the Strong bucket; the old panel used the same two values as literals). Everything else is `var(--…)` of existing tokens and the message system (`msg retry`, `msg broken`, `msg info`).
- Data preservation on the copy, hashes before and after the whole session: assessment queue, assessments, workflow history, application drafts and revisions, profile revisions up to 21 and queue state identical. Changed, all on the copy: `user_profiles` (the four revisions above), and `jobs` (`first_seen` of the four vacancies, `seen_at` of the roles opened in the browser, and the effects of the collection run below). New rows only in `match_vacancy_extractions` (6), `match_requirement_matches` (4) and `match_fit_results` (2 781).

## A side effect that was not intended

Opening the app in a browser makes it start one unforced collection run (`POST /api/search`, the existing "fire one collection on mount" behaviour, throttled on the server). The headless browser used for the screenshots did that on the copy: `collection_runs` row 23 on the copy, one source (`web3.careers`) timed out, one new vacancy was collected (Kalshi, id 2791, then analysed by the worker as a new vacancy) and `last_seen` of about a hundred vacancies moved. Requests therefore went to the job boards from this machine under the staging source configuration. `data-staging` and `data` were not touched. The server was stopped before the run could finish, so row 23 stays "running" in the copy. A browser session against a copy needs `/api/search` blocked; noted for Q.6.

## Q.5b — small panel fixes (2026-11-11)

Screenshots: `q5-screenshots/panel-84-q5b.png`, `panel-22-q5b.png` (panel crops) and `window-84-q5b.png`, `window-22-q5b.png` (whole window), taken on the copy in `.local/assessment-3/data` with the new build.

1. **Counts under You fit.** The counts line is now part of the You fit column. It breaks only between pairs ("4 match · 0 partial ·" / "0 no · 1 unclear"), never inside one. Goal lines in Matches (Role, Contract type) come first and a thin rule separates them from the requirement lines. I chose the rule over a second heading because it adds no text to the panel; on Parity the header reads "Matches 6", the counts line reads 4 match, and the rule shows that two of the six are goals.
2. **Alternatives.** `FitLine.alternatives` (`matchcontract/fit.go`, filled in `quickmatch/fit.go`, omitted when empty) and shown after the requirement: Parity, "Web3, blockchain infrastructure, ... (Rust · Substrate · Go · C++)"; Ethena, "Senior path from Go, Kotlin or Java (Go · Kotlin · Java)". The stored results of the copy were written before this field existed. All 2 781 rows of `match_fit_results` carry a `computed_at` of the minute the new server started, so they were evaluated again at start (no model call, the extractions are reused); that is why the old results show members. Which startup step does this I did not trace.
3. **The "before" image.** `panel-22-before-experience.png` was a byte copy of `panel-22-after-experience.png` (same md5 `b72deb43…`). I deleted it instead of retaking it: a retake needs the profile's experience emptied and restored again on the copy (two more profile revisions), and the whole-window image `window-22-before-experience.png` is a different file and stays. The Ethena before/after comparison therefore rests on the window images and on the table above.
4. **`POST /api/search` blocked.** This session ran the server on port 8094 behind a small reverse proxy on 8095 that answers 403 to `POST /api/search` and forwards everything else (the server was started with `--public-port 8095`). The proxy refused 6 calls from the page, so no collection run started and no job board was contacted. The proxy lives in the session scratchpad, not in the repository; Q.6 needs the same guard (or an equivalent) for every browser session.

Checks: `npm run build` clean; `node --test frontend/quickmatch.test.mjs` 8 of 8; `go build ./... && go vet` clean on the touched packages; `go test -count=1 ./internal/matchcontract ./internal/quickmatch` pass (new: `TestRequirementLineCarriesAlternatives`).
