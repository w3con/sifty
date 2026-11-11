# Q.6 run record

## Review server (step 1)

Address for the owner: http://localhost:8094

The server runs directly on port 8094 against the copy `.local/assessment-3/data` (git-ignored). Vacancy collection is blocked by a configuration switch added in commit a5157eb: `collection_disabled: true` in the copy's `runtime.yaml` (spec section 14.10). It stops the daily scheduler as well as `POST /api/search`, which answers `{"status":"skipped"}`. A first version used a reverse proxy in front of the server; it only covered the browser request, so it was replaced by the switch and the script was deleted.

Copy configuration changed for the run in `.local/assessment-3/data/config/runtime.yaml`: `collection_disabled: true` (new) and `quick_match.model_passes.daily_limit` 5 → 0 (no limit). The original file is kept in `.local/assessment-3/runtime.yaml.before-q6`.

Start (from the worktree root `.worktrees/ASSESSMENT-3`):

```
L=$(pwd)/.local/assessment-3
(cd backend && go build -o ../.local/assessment-3/sifty-q6 ./cmd/sifty) && npm --prefix frontend run build
nohup $L/sifty-q6 serve --data-dir $L/data --dist $(pwd)/frontend/dist --workers-dir $(pwd)/backend/workers --port 8094 > $L/q6-server.log 2>&1 & echo $! > $L/q6-server.pid
```

Stop:

```
kill $(cat .local/assessment-3/q6-server.pid)
```

Baseline hashes taken before the first start: `q6-hashes-baseline.txt`, produced by `q6-hashes.sh <store.sqlite>`; the same command is run after the run for step 5.

The machine is held awake for 8 hours by `caffeinate` started at 03:22 on 2026-11-11.

Timeline: first start 03:22 (an earlier build with the database-row lock and a guard proxy); restart 03:27 on the build with the machine-wide model lock and the collection switch. Results of vacancies 84 (Parity) and 22 (Ethena Labs) were deleted from the three result tables (`match_vacancy_extractions`, `match_requirement_matches`, `match_fit_results`, all new tables) at about 03:30 so that the run re-reads them and they are timed.

## Step 0 — model work follows the Inbox toggle (live check)

Final build: commit 589a03b. The review server was restarted on it at about 03:50 (the third start: 03:22 first build, 03:27 build with the lock and the switch, 03:43 build with the toggle, 03:50 final). The copy's `runtime.yaml` now carries `order: fit` under `quick_match.model_passes`; the leftover `new_window_hours` line was deleted from the copy (an old key is ignored by the loader anyway).

Live check (`.local/assessment-3/q6-toggle-check.sh`, output below): the order was flipped through `PUT /api/quick-match/order` and the vacancies the worker took next were compared with the head of the expected list.

- Expected for Newest: pending vacancies with a basis-backed date, newest first: 2696, 2697, 2698, 2699 (all 2026-11-07).
- After `PUT order=newest` at 02:43:51Z the worker took 90 (already in progress under Best fit, saved 5 s after the switch), then 2696, then 2697. The switch took effect at the next vacancy.
- Expected for Best fit: pending vacancies by Fits you: 162, 163, 181, 192 (all 0.45 before analysis).
- After `PUT order=fit` at 02:45:03Z the worker took 162, 163, 181.

Persisted by the PUT: `order:` in the copy's `runtime.yaml`; each PUT also left a timestamped backup in `.local/assessment-3/data/config-backups/`.

Unit tests: `TestPlanWorkOrder` (both orders and unset), `TestModelWorkFollowsTheConfiguredOrder` (real worker, three orders including a switch between two vacancies), `TestNewestKeyMatchesTheInbox` and `frontend/dates.test.mjs` (one shared fixture), `TestSetModelPassOrder`, `TestQuickMatchOrderEndpoint`.
