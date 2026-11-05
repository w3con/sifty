---
estimate: 4
---
# GO-10 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

Make the project's source-available license real. The roadmap states the license is the Elastic License 2.0, but the repository has no `LICENSE` file, so the statement is not backed by anything a reader or a self-hoster can rely on. This card adds the Elastic License 2.0 as the actual license: the canonical licence text in a `LICENSE` file at the repository root with the copyright line, a short License section in the README and `AGENTS.md` stating the project is source-available under the Elastic License 2.0 (download, modification and self-hosting for one's own use are allowed; offering it to third parties as a hosted or managed service is not), and the `license` field in `app/package.json` set accordingly. This card does only the license; the Docker/compose packaging, the documentation set and the developer-setup path stay in GO-6 and are not touched here.

## Coordinator's implementation decisions (settled)

- The licence text is the canonical Elastic License 2.0, written verbatim; fetch it from its canonical source (`https://www.elastic.co/licensing/elastic-license/`, or the raw text at `https://raw.githubusercontent.com/elastic/elasticsearch/main/licenses/ELASTIC-LICENSE-2.0.txt`) through `bin/websearch` or a web fetch, and do not paraphrase it.
- Copyright line: `Copyright (c) 2026 Alex Tikonoff` (the owner; the coordinator will confirm the holder with the owner before merge).
- `app/package.json` `license` field set to `"LicenseRef-Elastic-2.0"` (SPDX custom-reference form) with the authoritative terms in `LICENSE`.

## Must not touch

The application's behaviour. The GO-6 packaging, documentation and developer-setup scope. The coordinator-gated public-repo push (GO-1). Do not relicense or remove any pre-existing third-party notices.

## Acceptance criteria

- A `LICENSE` file at the repository root holds the Elastic License 2.0 verbatim (contains the headings "Elastic License 2.0", "Acceptance", "Limitations", "Patents") and the copyright line.
- The README and `AGENTS.md` each carry a short License section naming the Elastic License 2.0 and the source-available terms, pointing at `LICENSE`.
- `app/package.json` has its `license` field set to `LicenseRef-Elastic-2.0`.
- `GO-6`'s description records that the licence was delivered in GO-10, leaving packaging and docs as its remaining scope.
- `go test -race ./...`, `tsc --noEmit` and `vite build` pass (no application code is changed).

## Checkpoints

**1 — The license**

*Gate: the files exist and carry the canonical text and references.*

- [x] 1.1 — Fetch the canonical Elastic License 2.0 text and write it verbatim to `LICENSE` at the repo root with the line `Copyright (c) 2026 Alex Tikonoff`; a check greps for the "Elastic License 2.0" and "Limitations" headings.
- [x] 1.2 — Add a short License section to the README and `AGENTS.md` (source-available under Elastic License 2.0; self-host and modify for own use; not as a service to third parties; see `LICENSE`), and set `app/package.json` `license` to `LicenseRef-Elastic-2.0`; a check greps all three.
- [x] 1.3 — Update `ai/tasks/go-migration/GO-6_self-host-packaging/description.md` to note the licence was delivered in GO-10 and that GO-6 now covers packaging and docs only; a check greps the note.

## Gate evidence

Дословный вывод каждой прогнанной проверки записывается сюда, без сокращений и пересказа.
