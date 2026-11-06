---
estimate: 14
---
# GO-11 — plan

**Author:** coordinator
**Created:** 2026-11-06

## Statement of work

Move the React frontend from `app/web` to `sifty/web` with its own build, and repoint the build and deploy wiring, without touching the retired `app/src` backend. Relocate the frontend's four shared dependencies out of `app/src` into frontend-owned files under `sifty/web`: copy `core.ts`, `workflow.ts` and `preparation-error.ts` (each depends only on the `yaml`/`zod` npm packages), and create a small types file holding just `ApplicationDraft` and `AnswerField` so the backend `store.ts` is never dragged in; then rewrite every `../src/...` import in the frontend to the new local paths. Give the frontend its own `package.json` (frontend dependencies only), `vite.config.ts` and `tsconfig.json` (scoped to the frontend tree), building to a `sifty`-owned dist. Update `Makefile`, `bin/sift-deploy` and `bin/sift-accept` so build, serve and acceptance use the new location, keeping the two-environment contract (staging 8081/data-staging, prod 8080/data; staging never touches prod artifacts). The legacy `app/src`, `app/test` and `app/ops` stay in place and keep building; their copies of the shared modules are left untouched (temporary duplication, removed when `app/src` is deleted in a later card). Verify with `go test -race ./...`, the frontend `tsc --noEmit` and `vite build`; the coordinator then deploys staging and checks the app serves from the new path.

## Steps

### Phase 1 — Move the frontend and its shared modules

*Gate: no `../src` import remains in `sifty/web`; `rg "\.\./src/" sifty/web` is empty.*

- [ ] 1.1 — `git mv app/web sifty/web`; copy `app/src/{core,workflow,preparation-error}.ts` into `sifty/web` (e.g. `sifty/web/model/`) and add a frontend-owned types file with `ApplicationDraft` and `AnswerField`; leave `app/src` untouched.
- [ ] 1.2 — Rewrite every frontend import of `../src/{core,workflow,application-store,preparation-error}` to the new local paths; confirm no `../src` reference remains in `sifty/web`.

### Phase 2 — Frontend-owned build config

*Gate: `npm --prefix sifty/web ci && npm --prefix sifty/web run build` (`tsc --noEmit && vite build`) passes.*

- [ ] 2.1 — Add `sifty/web/package.json` with only the frontend dependencies (react, react-dom, vite, typescript, the `@types/*`, `@fontsource/*`, `yaml`, `zod`) and a `build` script `tsc --noEmit && vite build`; add `sifty/web/vite.config.ts` and a `tsconfig.json` scoped to the frontend only.
- [ ] 2.2 — Build to a `sifty`-owned dist (prod default and a `BUILD_OUTDIR` staging path); run the build and record output.

### Phase 3 — Repoint build and deploy wiring

*Gate: `make build` succeeds and produces the frontend bundle at the new path.*

- [ ] 3.1 — Update `Makefile`: the `build` target builds the frontend at `sifty/web`; the `run`/`run-staging` targets pass the new `--dist` path.
- [ ] 3.2 — Update `bin/sift-deploy` (npm path, `DIST_DIR` defaults, `BUILD_OUTDIR`) and `bin/sift-accept` (npm ci + tsc + vite at `sifty/web`); keep the "app-touching" path check matching `sifty/`.

### Phase 4 — Verify

*Gate: `go test -race ./...`, frontend `tsc --noEmit` and `vite build` all pass.*

- [ ] 4.1 — Run `go test -race ./...` (from `sifty/`), and the frontend `tsc --noEmit` and `vite build`; record output.
- [ ] 4.2 — (Coordinator) Deploy staging from the branch and confirm the app serves and the UI loads from the new `sifty/web` build.

## Acceptance criteria

- The frontend lives in `sifty/web` and builds with its own `package.json`, `vite.config.ts` and `tsconfig.json`; no import from `app/src` remains in the frontend (its shared modules are frontend-owned copies).
- `app/src`, `app/test` and `app/ops` are unchanged and still build; nothing in the legacy backend is broken by the move.
- `make build`, `bin/sift-deploy` (both environments) and `bin/sift-accept` build, serve and accept the frontend from `sifty/web`; the two-environment contract is intact and staging serves the app.
- `go test -race ./...`, `tsc --noEmit` and `vite build` pass.
