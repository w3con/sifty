# GO-11 — Move the frontend into sifty/web

**Epic:** go-migration
**Created:** 2026-11-06
**Status:** founded

## Why this work exists

The Go migration moved the backend into `sifty/` but left the frontend in `app/web`, so `sifty/` is backend-only and `app/` still holds both the live React frontend and the retired TypeScript backend (`app/src`, `app/test`, `app/ops`). The owner chose to move the frontend into `sifty/` first, before the rest of the consolidation and before any design work, so that the design pass lands on the final location. This card moves the frontend to `sifty/web` with its own build, updates the build and deploy wiring to the new path, and leaves the legacy `app/src` backend in place for a later card to delete. It is deliberately scoped to the move only: no CLI, no MCP, no deletion of `app/src`.

## Must not touch

The Go backend code in `sifty/` beyond the serve `--dist` default and build wiring. The ASM-4 ranking, the workers, the HTTP routes and the data directories. The retired TypeScript backend `app/src`, `app/test` and `app/ops` — they stay exactly as they are and keep building; their deletion is a later card. Production data and the two-environment deploy contract (staging on 8081 / data-staging, prod on 8080 / data; staging never touches prod artifacts; prod advances only via `bin/sift-promote`).

## Notes

Owner decision (2026-11-06): move the frontend first, standalone; keep the rest of the migration (CLI descoped, MCP later) separate.

Grounding read by the coordinator (current state on main):
- The frontend is `app/web` (10 `.tsx`, 6 `.css`, `api.ts`, `filters.ts`, `workflow.ts`, `index.html`, `public/`). Build config is `app/vite.config.ts` (`root: 'web'`, `outDir` from `BUILD_OUTDIR` or `../dist`), `app/tsconfig.json` (includes `src`, `web`, `test` together) and `app/package.json` (whose `build` script is `tsc --noEmit && vite build`; its `start`/`cli`/`mcp`/`test` scripts belong to the legacy backend, not the frontend).
- The frontend imports four modules from the legacy `app/src`: `core` (9×, imports only `yaml` and `zod`), `workflow` (3×, no imports), `preparation-error` (1×, no imports), and `application-store` (1×, **type-only** — `import type {ApplicationDraft, AnswerField}`). `application-store.ts` itself imports the backend `store.ts`, but because the frontend's use is type-only, `store.ts` is not pulled into the frontend bundle. There is also a separate `app/web/workflow.ts` (frontend) distinct from `app/src/workflow.ts` (shared); keep both, they are different files.
- Build/deploy wiring that references `app/`: `Makefile` (`npm --prefix app run build`, `--dist ./app/dist`), `bin/sift-deploy` (`$APP_HOME/app` for npm; `DIST_DIR` defaults `app/dist` and `app/dist-staging`; `BUILD_OUTDIR=../dist-staging`), `bin/sift-accept` (npm ci + tsc + vite in `app`; "app-touching" path check matches `sifty/` or `app/`), and the Go serve default `--dist ./app/dist` in the Makefile run targets.
- `app/.claude/settings.json` pins the subagent model; the repo-root `.claude/settings.json` has the same pin, so the frontend's move does not lose it.

Coordinator decision on the shared modules: the frontend takes its own copies of `core.ts`, `workflow.ts` and `preparation-error.ts` into `sifty/web`, and a small frontend-owned types file holding just `ApplicationDraft` and `AnswerField` (so the backend `store.ts` is not dragged in). The legacy `app/src` keeps its own copies untouched and still builds; the duplication is temporary and is removed when `app/src` is deleted in the later consolidation card. This keeps the move low-risk: nothing in `app/src` breaks.
