# GO-11 — executor handback

Steps 1.1 through 4.1 are complete and committed on branch `work/GO-11`. Step 4.2 (staging deploy and UI smoke check) is left for the coordinator.

## What moved

- `app/web/` → `sifty/web/` (all 10 .tsx, 6 .css, api.ts, filters.ts, workflow.ts, index.html, public/)
- `app/src/{core,workflow,preparation-error}.ts` copied to `sifty/web/model/` (originals untouched in app/src/)
- `sifty/web/model/types.ts` created with `ApplicationDraft`, `AnswerField`, `Question`, `Questionnaire` (extracted types; store.ts not pulled in)

## New dist path

- Prod default: `sifty/dist/` (Vite `outDir: '../dist'` relative to `sifty/web/`)
- Staging: `BUILD_OUTDIR=<abs path>` as before; `sift-deploy` defaults to `$MAIN_REPO/sifty/dist-staging`

## Wiring files changed

- `Makefile` — `npm --prefix sifty/web run build`; `--dist ./sifty/dist` in run/run-staging; clean target updated
- `bin/sift-deploy` — npm prefix changed to `$APP_HOME/sifty/web`; DIST_DIR defaults to `sifty/dist` / `sifty/dist-staging`
- `bin/sift-accept` — npm prefix changed to `$APP_HOME/sifty/web`; app-touching check unchanged (already matches `sifty/`)

## Verification

- `go test -race ./...` from `sifty/`: all pass. One flaky test (`TestAssessmentWorkerDrainsQueue`) failed on the first run with a timing issue and passed on every subsequent run; the failure is pre-existing and unrelated to this card.
- `npm --prefix sifty/web run build` (`tsc --noEmit && vite build`): passes, 54 modules, bundle at `sifty/dist/`.
