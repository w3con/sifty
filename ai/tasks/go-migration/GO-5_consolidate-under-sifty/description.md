# GO-5 — Consolidate the whole application under sifty

**Epic:** go-migration
**Created:** 2026-11-04
**Status:** founded

## Why this work exists

The application is meant to be self-hostable: someone clones one repository and runs a complete product. Today it is split — the Go backend and the Node workers live under `sifty/`, but the only frontend lives in `app/web` (React/TypeScript, built to `app/dist` and served by the Go backend through `--dist`), and the original TypeScript backend still sits in `app/src`. So `sifty/` is backend-only, and there are two backends at once. The decision, made by the owner: `sifty/` must hold the whole application — backend, frontend, database schema, workers and documentation — and the old TypeScript backend is not wanted and is retired. Go becomes the single backend.

## What this card delivers

- The frontend moves from `app/web` into `sifty/` (for example `sifty/web`) together with its own build configuration (the Vite config, `tsconfig`, and the frontend's `package.json`), building to a `sifty`-owned `dist`, which the Go backend serves by default without needing an external `--dist` path.
- Application documentation lives under `sifty/docs`.
- The leak-scan and public-subtree rules extend to the moved frontend and documentation, since they now enter the public export: no card identifiers, tool-authorship trailers, absolute local paths or Russian prose in that code, documentation or in commit messages touching them.
- The old TypeScript backend `app/src` is removed, as the final step and only once the parity gate below is satisfied; whatever remains in `app/` after the frontend moves out is removed too.

## Parity gate — do not delete the old backend before this holds

The Go backend is not yet at parity. Two gaps are already known: the daily collection run in Go is a stub (`internal/background/scheduler.go` `defaultCollect` only re-enqueues; real source collection is unimplemented), and the model-extraction fallback is not wired in Go. Deleting `app/src` before Go can collect postings would ship a self-host build that cannot gather any vacancies. Therefore the removal of `app/src` is gated on: the Go collection run working (COL-1), and a parity audit confirming nothing else depends only on the TypeScript backend. Run the audit before the deletion step and record its result in this card.

## Must not touch

- The behaviour of the running application: the move is structural, the product that builds and runs must be identical afterwards.
- The assessment, preparation, collection and browser-discipline logic (this card relocates and deletes; it does not rewrite behaviour).
- The deletion of `app/src` until the parity gate is satisfied.

## Notes

- The frontend currently builds in `app/` with `npm run build` (Vite) to `app/dist`; the demo serves it via `--dist /…/app/dist`. After the move the default served path is the `sifty`-owned build output.
- `app/node_modules` must never be committed or symlinked; install with `npm ci` wherever the frontend builds.
- Depends on COL-1 for the collection-run half of the parity gate. The self-host packaging, documentation set and license are GO-6.

## Steps

- [ ] To be defined in plan.md.

## Acceptance criteria

- `sifty/` contains backend, frontend, workers, database schema and documentation; the Go backend serves its own frontend build by default.
- The public-subtree rules and leak scan cover the moved frontend and documentation.
- `app/src` and the rest of the old `app/` tree are removed, only after the parity gate (COL-1 plus a recorded parity audit) is satisfied.
- The built-and-run product is unchanged in behaviour.
