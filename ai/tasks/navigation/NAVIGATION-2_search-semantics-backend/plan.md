---
estimate: 15
---
# NAVIGATION-2 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

Extend the search-term model in `config/search.yaml` from two flat OR lists into OR/AND expressions, with a documented schema and a version handled so the existing flat `titles`/`keywords` config still parses. Update `store.BuildFTSQuery` and both its callers — collection (`internal/background/collect.go`) and the ASM-4 rescore path (`internal/server/routes.go`) — so collection and ranking honour AND, and confirm fit scores are still produced across the catalogue. Unify exclusions: generalise stop-word matching so one term is tested against the role's title OR its company name; migrate existing company block rules into exclusion terms through a schema migration; and remove the separate source-block, since source control is now the enabled flag on a site in `search.yaml`. Expose reading and writing of the search expressions and the exclusion terms through the HTTP API (reusing the config-save path where it fits, or a dedicated endpoint). Keep `go test -race ./...` green, with tests for the OR/AND combinator and for the migration. No frontend work.

## Steps

### Phase 1 — Term model

*Gate: `go test -race ./internal/config/... ./internal/store/...` passes.*

- [ ] 1.1 — Define the OR/AND expression schema on `config.Search`, parsing the existing flat `titles`/`keywords` as a backward-compatible OR group.
- [ ] 1.2 — Update `store.BuildFTSQuery` to compile OR/AND expressions into an FTS query; add combinator unit tests.
- [ ] 1.3 — Point `collect.go` and the `routes.go` rescore at the new query; add a test that AND narrows and that ASM-4 scores are still produced.

### Phase 2 — Unified exclusions

*Gate: `go test -race ./internal/store/... ./internal/background/...` passes.*

- [ ] 2.1 — Generalise stop-word matching to test title OR company (e.g. `RoleMatchesExclusions`), keeping the collection-time skip.
- [ ] 2.2 — Add a schema migration (bump user_version) that copies company block rules into exclusion terms and drops source-block rules.
- [ ] 2.3 — Remove the source-block code path; leave source enable/disable to the `search.yaml` sites list.

### Phase 3 — Endpoints and full test

*Gate: `go test -race ./...` passes.*

- [ ] 3.1 — Provide API read/write for the search expressions and for the exclusion terms.
- [ ] 3.2 — Add tests for the migration and the endpoints; run `go test -race ./...` and record output.

## Acceptance criteria

- `config/search.yaml` accepts OR/AND expressions, and the previous flat `titles`/`keywords` config still parses and behaves as an OR group.
- Collection and ASM-4 ranking honour AND; fit scores are still computed across the catalogue (verified by a test, not only by inspection).
- A single exclusion list hides roles by title or by company; pre-existing company block rules are migrated into it; source-block rules are removed and source control is the `search.yaml` enabled flag.
- The HTTP API can read and write both the search expressions and the exclusion terms.
- `go test -race ./...` passes with the new tests; no frontend file is changed.
