---
estimate: 14
---
# GO-1 — plan

**Author:** coordinator
**Created:** 2026-11-01

## Statement of work

Create a fresh public MIT repository for Sifty with its leak-protection standing before the first commit, then rewrite the backend from TypeScript to a self-contained Go application inside it, reaching functional parity with the current Stages 1 and 2. Structured collection and fit assessment and application-answer preparation move to Go; browser automation (the Stage 2 form inspection and any dynamic-site collection) is isolated in a thin TypeScript worker running the official Playwright that the Go core invokes; the jobspy scraper is reused the same way, as a worker the Go core calls rather than a port. The application becomes self-contained: candidate data and remembered answers and feedback load from a configurable local data directory with file templates, with no writes to `kb/`, no resolution from the FCTO git root, no `FCTO_STATE_DIR` or `FCTO-Catalogue` naming, and no `tools/shared` dependence. The current TypeScript application stays working until an end-to-end parity run passes, after which the public Go repository becomes the product and the first push happens behind a full-tree leak sweep.

## Steps

### Phase 1 — Public repository and leak-protection

- [x] 1.1 — Create the fresh public repository (git init, MIT LICENSE, README) with a `.gitignore` that excludes every private and non-code path; check: repository initialises and `git status` tracks no private path.
- [x] 1.2 — Vendor the pilier-style leak scanner under `scripts/leakcheck/` with a census listing the repository's own origin and generic secret forms (cloud keys, personal access tokens, API keys, JSON Web Tokens), paths repointed; check: the scanner flags a seeded secret string and passes a clean one.
- [x] 1.3 — Install the leak-check hooks (pre-commit, commit-msg, pre-push range re-scan) via a relative `core.hooksPath`, plus a machine-setup script a fresh clone runs; check: a commit carrying a seeded secret is refused, a clean commit passes, and a `--no-verify` commit is caught by the pre-push re-scan.
- [x] 1.4 — Add a CI workflow that runs the leak scan and the build-and-test entry; check: the workflow file is present and the test entry runs locally.

### Phase 2 — Go skeleton, self-contained

- [x] 2.1 — Lay out the Go module and package structure with a single-binary `cmd` entry; check: `go build` produces a runnable binary.
- [x] 2.2 — Configuration and data paths: candidate data, remembered answers and feedback load from a configurable local data directory, with no `kb/` writes, no FCTO git-root resolution, no `FCTO_STATE_DIR` or `FCTO-Catalogue` naming, and no `tools/shared` dependence; check: the binary reads the data directory from config or environment and writes answers and feedback there.
- [x] 2.3 — Add answer and feedback file templates for the local data directory; check: a first write produces files matching the templates.
- [x] 2.4 — Define the Go-core to TypeScript-worker boundary (invocation protocol and input/output contract) used by both the Playwright worker and the jobspy worker; check: a stub worker round-trips a request from the Go core.

### Phase 3 — Port Stage 1 (collection and fit)

- [x] 3.1 — Structured collection in native Go (HTTP and parsing) plus the jobspy scraping worker integrated as a sidecar; check: a fixture collection run yields the same postings as the current TypeScript application.
- [x] 3.2 — Fit assessment with a provider-configurable model call matching current behaviour (full model separation is a later card); check: fit results on a fixture match the TypeScript application within tolerance.

### Phase 4 — Port Stage 2 (application-answer preparation)

- [x] 4.1 — Application-answer preparation in Go (the JOB2 flow); check: prepared answers for a fixture vacancy match the TypeScript application.
- [x] 4.2 — Form inspection through the TypeScript Playwright worker invoked by the Go core; check: form fields for a fixture form are returned via the worker, with no browser-automation code in Go.

### Phase 5 — Parity and cutover

- [x] 5.1 — End-to-end parity run against the current TypeScript application on a shared fixture, with any differences explained; check: a parity report is recorded in the log.
- [x] 5.2 — Full-tree leak sweep, then the first push of the public repository, with a cutover note that the public Go repository is the product going forward; check: the tree sweep is clean and the push succeeds.

## Acceptance criteria

- A fresh public MIT repository exists with a `.gitignore` excluding all private paths, a vendored leak scanner, leak-check hooks via a relative `core.hooksPath`, a machine-setup script, and CI; no private data is present anywhere in its history.
- The Go backend builds to a single binary and runs Stage 1 (collection and fit) and Stage 2 (application-answer preparation) producing output equivalent to the current TypeScript application on a shared fixture.
- All browser automation runs through a thin TypeScript worker on the official Playwright invoked by the Go core; there is no browser-automation code in Go.
- The application is self-contained: candidate data, answers and feedback load from a configurable local data directory with templates; there are no writes to `kb/`, no FCTO git-root resolution, no `FCTO_STATE_DIR` or `FCTO-Catalogue` naming, and no `tools/shared` dependence.
- The current TypeScript application under `app/` remained working until parity was proven; cutover happened only after the parity run passed.
