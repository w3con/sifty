# Documentation

Current entry points after the 2026-11 repository relocation:

- [docs/MAP.md](MAP.md) — the current requirement-to-code navigation map. Start here when a change needs its affected frontend, backend, workers, storage, checks, decisions and task cards.
- `product/` — historical product framing, including the archived 2026-11-01 map. Some notes retain pre-relocation paths and decisions.
- `architecture/` — the Go backend architecture, HTTP contract, runtime data directory and provenance, worker protocol, the parity audit and the development-setup research notes.
- `../work/` — canonical task cards, decisions, backlog and the build roadmap (`../work/ROADMAP.md`); decisions are in `../work/decisions/`. `../work/incidents.md` is a local, ignored incident record and is not published.

The active navigation documents (`docs/MAP.md`, this page, `../work/WORKFLOW.md` and `../work/ROADMAP.md`) are link-checked by `../bin/map-check`. Archived historical prose is deliberately outside that scope.

## Current and historical material

Archived documents written before 2026-11-06 keep their original wording for attribution, including references to the earlier private repository, an MIT licensing plan, `ai/` paths and the retired `app/`/`sifty/` source layout. Those statements are historical, not current instructions. The current license is **Elastic License 2.0** (`../LICENSE`): source-available and self-hostable, not "open source", and not MIT. Current source lives in `frontend/`, `backend/` and the retained `legacy/app/`; current records live under `work/`.

Each archived document states its own date and status where it matters. When a historical statement and this page disagree, this page and `LICENSE` are current.
