# COLLECTION-1 — technical acceptance

> **Integration status,2026-11-10:** source published to main in commit `3ee6260` (final application source `58af432`). Card is review, not done; production is not updated. The report below remains the technical acceptance record at its original date.

2026-11-09T21:43Z, coordinator ses_e4a7d8341ffe1Xqrv4Rmx52jqw. Source d92112c04a3f40e24fd306a7b19d7c4a422625d5 technically accepted after corrections. Owner authorized «Делай.» at20–35min. Owner acceptance and preserving main/production integration remain separate.

## Delivered / actual use

Review http://127.0.0.1:8083, same isolated review data as APP-8. Catalogue/detail show date and completed age separately from Collected. Newest uses three explicit tiers: source-declared publication newest-first; legacy unverified source dates newest-first; missing/invalid/future dates last. Coinbag2022-04-25 now visibly reads4years6months, not a recently collected new vacancy. The old raw date remains available without pretending its original field provenance is known. No age score deductions, automatic expiry/hiding, model calls or changed workflow.

Source fields datePosted/pubDate/published/first_published/ATS publication fields supply a publishedBasis. Greenhouse updated_at and Atom updated no longer masquerade as publication. Date-only, actual Web3 space+offset, ISO and feed formats are handled. Upsert/enrich fill missing genuine dates and retain established dates/workflow; a different new date cannot lend provenance to an old date. Related JSON-LD items require unique matching, preserving identity query parameters; no first-item guess.

## Boundaries and schema

model/date.go and frontend/dates.ts own date semantics/presentation helpers; adapters/parsers own source field meaning; jobs store and enrichment own persistence; API/container passes data, list/detail render it. Existing configuration/path/provider mechanisms retained. One nullable jobs.published_basis column, targetVersion18; existing raw legacy dates are not mass-rewritten. No settings/provenance framework or Profile frontend changes.

## Evidence

- Backend/frontend builds passed. Narrow date, source/feed/parser, matcher/no-wrong-job, fill/preserve/basis association and enrichment checks passed. Schema fresh/existing, outcome-stale preservation and migration regression checks passed; no broad/full suites. Defaultwork-test once: SKIPPED. A preserved TestMigrationV14 was reported failing on its sparse fixture and excluded; not counted passed or repaired as part of this task.
- Real Chromium screenshots/checks in .local/collection1-review/{list-newest,detail-coinbag}.png and log.md. Known synthetic source publications are explicitly labelled Collection Fixture/example.invalid with unused IDs999001/999002, old Coinbag/Web3 are actual copied records. Browser review blocked App-mount search/nonessential writes, no real source re-fetch/model call.
- Coordinator reviewed actual diffs/screenshots and live read-only API: real copied990008/990009 have no invented basis; labelled fixtures999001/999002 have known basis; Coinbag1028 retains its raw legacy date. Executor model metadata independently verified123assistant messages across resumes, only opencode-go/deepseek-v4.1-flash.
-8083PID75373 uses COLLECTION-1 binary/dist and the same isolated data. Production8080, staging/Profile8081/8082, dirty main, APP-8 and the other DESIGN-3 writer preserved. Recovery argv/binary retained under own .local/collection1-review.

## Failures, corrections and limitation

First review found missing actual Web3 no-space-offset format, impossible calendar dates accepted by JavaScript, rounded-up ages and unsafe related-posting/query-ID matching; corrected with narrow checks. Blanket unverified labels could not distinguish future genuine publication from unknown legacy origins; added the minimal basis field, explicit source labels and separate ordering tiers. Enrichment initially failed to save that basis alongside a newly filled date; corrected and checked.

Executor mistook two real source-collected rows in the isolated review copy for APP-8 fixtures and temporarily set invented first_published basis. Coordinator caught the misleading evidence. Only that added basis was restored to NULL; original dates/workflow/history unchanged; new fixtures use genuinely unused IDs and explicit fixture names. This incident did not touch production/staging. A first launch also used a wrong build path, recovered to the intended worktree path. Both go in the executor's incident record; no extra harness repair.

Legacy coverage limitation is explicit: without original field provenance, old dates cannot be certified as publication. They remain visible as unverified source dates; no claim that source update proves publication/closure, no bulk re-fetch or retrospective invented origin.

## Time and next action

Start21:10Z, implementation/corrections to21:42Z, acceptance21:43Z: approximately33min, within20–35min. Existing preview/data reused. Main time went into date/source semantics and correcting insufficient initial checks; no repeated environment setup. Stop further implementation here. Owner views freshness on8083; next discussion is finishing updated Profile for one person with role-based profiles and separately keyed scores, not auth/logout. Confirm other Profile writer handback before any writes there.
