---
estimate: 3
---
# COLLECTION-1 — publication date, age and genuine Newest

**Approval:** owner «Делай.» approves the complete20–35min publication/age/Newest plan shown in chat. Start2026-11-09T21:10Z, absolute stop21:45Z including preparation/review. One combined F.1. Coordinator ses_e4a7d8341ffe1Xqrv4Rmx52jqw. Next instruction «Как закончишь со свежестью вакансий - надо будет завершить с профилем. там дизайн уже поменялся» is recorded as the subsequent priority; do not edit the other agent's Profile during this task.

## Observable outcome and use

The owner sees when a vacancy was published and its age in the list/detail while deciding whether to keep it. Newest uses publication rather than discovery, so an old vacancy collected today does not look fresh. Demonstrate using current copied jobs, including the old Coinbag/Web3 examples, known recent postings and missing dates. This delivers manual review value immediately; no new feed, taxonomy or scoring engine.

## Scope and boundaries

Reuse existing publishedAt/firstSeen fields, adapters/parsers, job store/API, catalogue row/detail and sorter. Normalize supported date-only/ISO/timezone/feed dates consistently; missing/invalid/future or uncertain update-only dates remain honestly labelled/unknown instead of a false publication age. Preserve original data when it is insufficient to establish publication; do not manufacture provenance.

Fill a newly obtained valid publication date on an existing job and preserve it across re-collection; existing enrichment must not discard a genuine date. Keep discovery/re-discovery distinct. Source updatedAt, Atom updated or page fetch time must not be passed off as publishedAt. Correct those existing adapter/parser paths and represent uncertain legacy dates honestly; no universal website-date extractor or bulk source re-fetch/backfill.

List/detail show absolute date and relative age with an explicit source/unknown label as applicable. Collected date remains separate. Newest places known valid publication dates newest-first, unknown dates last with stable ordering; a source update must not rejuvenate publication age. Preserve Best fit/local BM25 and existing workflow/filtering, App-mount collection/cooldown. No automatic hiding/deletion/expiry, age score deductions, model calls or detailed-assessment/scoring changes in this first manual-review step. Duplicate/prior-application recognition remains TRIAGE-1; free-text rejection feedback remains APP-8 follow-up.

Data/configuration: stored vacancy/source fields are authoritative, domain/date helpers own parsing/age/effective-date semantics, store/adapters own persistence and source meaning, API/container owns effects, views render supplied values/callbacks and labels. Reuse current path/config/provider adapters; no owner/example data, machine paths, runtime addresses, model choices or adjustable magic limits in application logic/views. No new settings framework.

Use a new COLLECTION-1 worktree from accepted APP-8 d2db849; preserve dirty main and active DESIGN-3 Profile writer/8081preview. Reuse the already configured independent8083review loop and isolated data at final handover rather than repeat environment setup; preserve that review data and verify ownership before any process transition. No production8080 write/release, staging refresh, history reconciliation or Profile frontend edits.

## Complete estimate and steps

**Complexity3:** existing source date fields and UI are reusable; parser semantics, stable ordering and persistence are coupled. AI preparation3–5min, implementation12–20min, acceptance5–10min; **total20–35min** excluding owner wait/external source delays, superseding provisional80–150min for the broader intake. One executor/one combined pass. Subscription OpenCode Go, attributable monetary cost unknown; no new paid service/runtime inference. Existing preview/data reused. Upper35min is a stop/report boundary from preparation/dispatch start; the previous modal overrun is not permission to extend silently.

| Part of F.1 | Result and necessity | AI elapsed | External cost |
|---|---|---|---|
| Dates/source handling | Valid source publication survives collection/enrich; update/discovery timestamps do not masquerade as publication |10–15min including shared preparation|No paid service; no bulk requests|
| Catalogue/detail/sort | Owner sees date/age and Newest orders by publication, with honest unknowns |5–10min|None|
| Acceptance/review handover | Builds, few hidden date/order/persistence invariants and one short real-browser demonstration using existing preview |5–10min|None|

All parts mandatory; total20–35min, no optional harness/component-library/test-infrastructure work.

## Checkpoints

- [x] F.1 — Preserve and display publication/age, correct Newest ordering, verify source semantics and reuse existing review preview.

## Acceptance criteria

- Old publication with recent firstSeen shows its actual old age and does not precede a recent known publication in Newest. Missing/invalid/uncertain publication is clearly labelled and placed after known dates.
- Absolute date/relative age agree for supported formats/timezones and date-only values. Source update and re-discovery cannot silently become publication or make a posting young; no claim that age proves vacancy closure.
- Re-collection/enrichment of an existing job can fill a valid previously missing date while preserving workflow/application history and an existing established publication date. No owner data reset, mass rewrite or inferred source facts.
- Few focused hidden-failure checks cover date semantics/ordering and preservation on existing-job updates. Visible list/detail are reviewed in real Chromium with actual copied jobs; no broad suite or exhaustive format/source matrix.
- Build affected app, report defaultwork-test once as SKIPPED; reuse existing preview/data/configuration and avoid a new environment/test framework. During review block collection/model/nonessential writes; no new runtime inference.
- Report actual elapsed, code/data/config/view boundaries, actual preview source/version, evidence and corrected failures. Owner review is separate from main integration/production release; no whole cumulative Profile branch released as a dates-only change.
