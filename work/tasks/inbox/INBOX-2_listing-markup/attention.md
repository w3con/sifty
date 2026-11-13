
## Exchange-rate defaults in config/vacancy-view.yaml are placeholders (coordinator, 2026-11-13)

`config/vacancy-view.yaml` ships with placeholder rates (USD 0.92, GBP 1.17, CAD 0.68, AUD 0.60, CHF 1.08, JPY 0.0062). These are not live market data. The owner must edit this file and fill in current rates before the release for the Money tile and Fits you money comparison to be meaningful. Owner: «пока мне это не важно, и это мелочь. пусть пока так и будет.» — acknowledged, no action needed before review. Rates are hand-edited; no network fetch is planned.


**Resolved (coordinator, 2026-11-13):** owner: «пока мне это не важно, и это мелочь. пусть пока так и будет.» The placeholder rates stay for the release; replacing them is a later task for the owner.
---

## Production artifacts were rebuilt from this branch (coordinator, 2026-11-12)

`.local/deploy/prod/state.json` records a production build-only at 2026-11-12T10:04Z from commit 64b7f8c8 of `work/INBOX-2` (dirty), and `build-recovery` holds another at 2026-11-11T23:15Z from 625d8ec8. `bin/sifty` and `frontend/dist` in the main checkout (the production artifacts) are therefore unreleased INBOX-2 code. Production was not running at the time, so nothing served it, but starting `bin/sifty` directly would run unreleased migrations and the backfill on production data. Never run `make build` / `bin/sift-deploy --env prod` from this worktree; build into a temporary directory for review servers. The owner restarts production with `make run` from the main checkout, which rebuilds from main first. Recorded for OPS-9: sift-deploy run inside a worktree writes the main checkout's production artifacts.

**Resolved (coordinator, 2026-11-12, after stage A):** `.local/deploy/prod/state.json` now records a production build from main commit 881a9a8f (clean, 10:50Z); production runs on 8080 from `./bin/sifty` started 11:50 local, and the live bases are at schema 22 (`data`) and 18 (`data-staging`), so no INBOX-2 migration has touched them. The warning above no longer applies; the rule "never run make build or sift-deploy from this worktree" still does.
