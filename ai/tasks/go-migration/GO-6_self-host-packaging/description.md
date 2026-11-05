# GO-6 — Self-host packaging, docs and source-available license

**Epic:** go-migration
**Created:** 2026-11-04
**Status:** founded

## Why this work exists

The project is source-available and self-hostable. The owner sells a hosted offering for non-technical users; technical users may download, self-host and modify it for their own use, but not offer it to others as a competing commercial service. For that to work the public repository must be easy to install, well documented, and approachable for a programmer who wants to change it. The multi-runtime stack makes a bare manual install painful: a Go backend, Node workers (`.mjs`), the Python jobspy scraping library, and optionally a local model runtime (ollama). This card establishes the self-host story, the documentation set, and the license.

## Decisions already made

- License: **Elastic License 2.0** (source-available). It permits download, modification and self-hosting, including a business self-hosting for its own use, and forbids providing the software to third parties as a hosted or managed service. The project is described as source-available, not open source. Relicensing applies to future versions; code already published under MIT stays MIT for those who received it.
- Install path: **Docker Compose** as the primary self-host story — one image bundling the Go binary, Node and Python with jobspy, with ollama added optionally by the compose file, so a self-hoster runs one command. A documented manual setup remains as the developer path.

## What this card delivers

- `sifty/LICENSE` replaced with Elastic License 2.0, the copyright line kept, and the source-available terms referenced from the README; any per-file or packaging license notices updated to match.
- A container image and a `docker compose` setup that builds the frontend, runs the Go backend serving it, and starts the workers, creating the application's own data directory; ollama is an optional compose service. `docker compose up` yields a running application on a documented port.
- A documentation set under `sifty/docs`: a quickstart from clone to running, an architecture overview (backend, frontend, workers, the assessment and preparation pipelines, the data store), a configuration reference (the YAML files, credentials by environment reference, the stage models), and a short contributor guide naming the extension points (notably the source abstraction from COL-1).
- A developer setup path documented for running without containers (toolchain prerequisites, building the frontend, running the backend and workers).

## Must not touch

- Application behaviour; this card packages and documents, it does not change features.
- The credentials model: secrets stay as environment references, never baked into an image or committed.
- The `sifty/` public-subtree rules in all added files and commit messages.

## Notes

- Dependencies to cover in the image and docs: Go toolchain (build), Node (workers and frontend build), Python with the jobspy library (scraping), optional ollama (local models). The Playwright worker needs its browser dependencies in the image.
- Best sequenced after GO-5 (so the frontend and docs already live under `sifty/`). The license swap itself is independent and can land first.
- Confirm the final license wording for the owner's situation with a lawyer; this card applies the chosen license text, it is not legal advice.

## Steps

- [ ] To be defined in plan.md.

## Acceptance criteria

- `sifty/LICENSE` is Elastic License 2.0 and the README describes the project as source-available with the self-host and no-managed-service terms.
- `docker compose up` builds and runs the complete application (backend serving the frontend, workers up, own data directory), with ollama as an optional service.
- `sifty/docs` holds a quickstart, an architecture overview, a configuration reference and a contributor guide; a non-container developer setup is documented.
- No application behaviour changed; credentials remain environment references; the public-subtree rules hold in the added files.
