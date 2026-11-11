# Q.4 end-to-end check on the copy (real server, real local model)

Data: `.local/assessment-3/data` (git-ignored), a copy of the staging store; the staging and production directories were not opened for writing. Binary built from this branch, run as `sifty serve --data-dir .local/assessment-3/data --port 8094` (not 8080, not 8081), provider `qwen3.5-9b` (Ollama, `qwen3.5:9b`), the assessment queue left disabled as it is in the copy (`assessment_enabled=false`, 2 319 rows pending, 373 done, 4 failed).

Changes made to the copy for this check, and nothing else: `runtime.yaml` got the `quick_match` block with `daily_limit: 8` (so that only a few of the 2 327 vacancies with text are analysed) and `daily_hour: 23` (so that the daily collection cannot start during the check); job 84 (Parity) got `first_seen` = now, standing for a fresh collection (its description is the one patched in Q.3).

## What the worker did

- Start: `GET /api/quick-match/analysis` → state `idle`, 2 327 vacancies with usable text, 0 extracted, 2 327 owed extraction. The vacancies without text (metadata only, pending, errors: 453 of 2 780 in the copy) are never queued.
- It took vacancies one at a time, about one every 20 seconds (extraction 9–17 s, matching 4–9 s), newly collected first. Counts after 15 s steps: extracted 1/matched 1, 2/2, 3/2, 3/3, 4/3, 5/4 … then state `limit` ("Today's limit of 8 vacancies is reached") at 6 extracted and 6 matched, 2 321 still owed.
- Parity (84) as served by `GET /api/jobs/84` under `fit`: Fits you 82 (scored), You fit 69 (2 match, 0 partial, 0 no, 3 unclear). Matches: role title aligned ("tech lead"), engagement compatible (contract), and two requirement lines with the profile evidence they cite (`English (C2)` for asynchronous communication; `Golang (Proficient)` for Web3 / systems). Missing from your profile (unclear): "Software Engineer, Tech Lead, or Engineering Manager", "Experience managing or leading engineering teams", "Founder, co-founder, or early-stage startup lead". To clarify: pay (the vacancy states none) and work mode. No condition conflicts.

## Profile save runs only the matching pass

`PUT /api/profile` on the copy (revision 20 to 21: the level of the skill `graphql` from Beginner to Proficient): at once 6 matchings owed and 0 extractions, in the next 50 s the six matchings ran (matched 0 → 6, state `working` then back to `limit`), and the six extraction rows are byte-for-byte the same (`created_at` unchanged). Extraction was reused; the daily limit, which counts extractions, did not hold matching back.

## Data preservation (copy before the run, taken before the server started, against after)

Hashes of the full rows of each table, before and after:

```
    bd42acd2951a  select * from assessment_queue order by job_id
    f1ab01dac7ff  select * from assessments order by 1
    6a76ed917b72  select * from workflow_history order by 1
    71dceb7ee42d  select * from application_drafts order by 1
    801a565372a6  select * from application_revisions order by 1
    c2410d66cc7a  select * from profile_revisions where revision<=20 order by 
    1188b399bdd5  select * from user_profiles order by 1
    7d6aae995b01  select id,identity,title,company,original_url,apply_url,desc
    2b00ca5e6a8d  select * from queue_state order by 1
```

Identical after the run for every table except `user_profiles`, which changed only because this check saved a profile revision (21) on the copy. Profile revisions up to 20, the jobs table (description, status, assessment columns, first/last seen), the assessment queue, assessments, workflow history, application drafts and revisions and the queue state are identical. New data went only into `match_vacancy_extractions`, `match_requirement_matches` and `match_fit_results`, plus the single-score tables `match_job_*` that the existing recomputation always writes.

## Injected end-to-end test (temp database, model stub)

`backend/internal/background/fitworker_test.go`: scoring of both passes, the assessment queue, assessments and profile revisions hash-identical before and after, new-first order, profile change running only matching, changed description re-extracted while a new sighting of the same text is not, provider down (backoff, work kept, recovers), unusable answer (set aside), pause, paid provider refused, daily limit, inference lock held elsewhere. `backend/internal/matchsvc/fit_test.go`: rescoring wakes the worker, stale marking, sort order.
