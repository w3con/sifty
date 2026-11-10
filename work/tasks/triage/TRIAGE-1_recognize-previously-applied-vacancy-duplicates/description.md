# TRIAGE-1 — Recognize previously applied vacancy duplicates

**Epic:** triage
**Created:** 2026-11-09
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
А насчет того, что попадает в триаж - еще надо определять дупликаты - потому что сегодня я опять увидел вакансии, на которые уже подавался на прошлой неделе. 
```

## Preparation

### Provisional estimate

- **Complexity:**5 — matching source/ATS identities and preserving submitted-application/history state are coupled; false merges can conflate distinct openings. Owner's concrete repeated pair is not yet identified.
- **AI preparation:**5–10min; **AI implementation:**15–25min; **AI acceptance:**5–10min; **total25–45min**, excluding owner wait/source latency. Existing subscription, attributable monetary price unknown; no paid service or runtime LLM needed. Provisional capture estimate, not execution approval.

### Use and initial inspection

The owner should recognize a previously applied opening during manual review, with the prior application date/history available, rather than reconsider it as new. Demonstrate the reported pair on a safe data copy; preserve genuine separate openings. Exact duplicate identification should reuse existing canonical URL/apply URL and source identities; ambiguous similarity must not automatically destroy/merge records. Do not build a semantic/LLM dedup engine.

Read-only findings: store.Upsert already matches canonical sighting URL, shared apply URL or guarded company/title identity; it preserves applied when a match is actually found. Same-host listings, Unknown company and conflicting apply URLs intentionally restrict title-only matches. Existing same-title Kraken rows have different ATS opening IDs, so they are not proof of a duplicate; another same-source-ID candidate is explicitly marked zzz-wlsn-probe-2 synthetic data and is not evidence of the owner's live failure. No confirmed owner-reported pair yet. Inspect workflow_history as well as current status: manual stage changes can clear applied_at, while history records earlier transitions. No live merge/backfill/deletion has been made.

This is the duplicate/history outcome within the owner's minimal manual workflow, linked to INBOX-1 and APP-8. Profile frontend belongs to another agent. Next: obtain one actual repeated listing/previous application example, then prepare the complete bounded plan and revised estimate before implementation.

### Supplied companies and actual evidence —2026-11-09T18:52Z

Owner: «любая вакансия от chainsync, coinbag, uniswap». Readonly inspection of both databases found a concrete application-history divergence, not a confirmed second canonical record for the same application:

- Coinbag Software Engineer id1028 has the same Wellfound listing URL in both. Staging records Applied2026-11-06T16:34:07.028Z; production has no such Applied event and currently rejected following Nov9 manual transitions. Other Coinbag listings have distinct role titles/Wellfound IDs.
- Uniswap Senior Smart Contract Engineer id339 has the same Ashby opening ID in both. Staging records Applied2026-11-06T10:58:48.836Z; production records Applied2026-11-09T12:37:09.596Z. This is consistent with repeating an application across independent environment histories, not proof of a failed URL deduplication. Other Uniswap roles have distinct ATS IDs.
- Literal chainsync/chain sync was not found among company/original URLs in either database. Do not silently substitute Chainlink/ChainSecurity/Chainalysis.

Ask which environment is authoritative for real applications (8080production or8081staging). Do not merge/refresh either database or declare the scraper bug fixed. A possible history reconciliation is new explicit data-changing scope requiring a preservation plan and approval; later decisions do not erase the fact of an earlier submission. Preserve original events and do not infer a submission for every role at the same company. Duplicate detection remains worth checking, but the reported examples first expose separated owner histories.

**Owner answered:** «да я иногда жал что подался, а иногда возвращал в триаж и удалял оттуда. не важно. все делал на 8080». Production8080 is the real-use environment. Stop the two-environment forensic detour; do not treat it as the established root cause or propose silent history reconciliation. Later recognition must examine prior submission/history despite returning to Triage/rejecting/removing a listing; current applied status alone is insufficient. Existing transitions can clear applied_at; permanent deletion/history retention and tombstones require a bounded inspection in this card. No dedup/history implementation authorized by APP-8.

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.
