# Catalogue and Search — specification

Status: owner decisions of 2026-11-13 applied (OPS-11); rule tags show the implementation state.

## 1. Purpose

The owner collects vacancies from job boards and then decides which ones deserve attention. The catalogue is the list panel where that decision begins: it shows all collected vacancies with their fit scores, lets the owner narrow the view with filters and a text search, and advances each vacancy through the triage workflow. Search expressions define what the owner is looking for; exclusion terms hide unwanted vacancies from the displayed catalogue; filters and sort orders surface the most relevant ones first.

## 2. Scope

This specification covers: search expressions (OR and AND terms) and their current and intended effect; exclusion terms that hide stored vacancies from every view; what each catalogue row shows; the in-list text search; filters; sort orders; how the list stays current; and list resize.

Out of scope, with links:
- Fits you and You fit scoring formula → [Quick Match](quick-match.md);
- vacancy detail pane content → [Vacancy view](vacancy-view.md);
- stage decisions (Dismiss, Select, Apply, Undo), keyboard shortcuts, stage counts, Archive tabs, company block rules → [Triage](triage.md);
- exclusion terms at collection time (vacancies not stored) → [Collection](collection.md); this spec covers their display-time effect only;
- profile structure and My data → [Profiles](profiles.md).

## 3. Rules

### 3.1 Search expressions

The owner configures search terms in two sets: OR terms (any must appear) and AND terms (all must appear). Owner, 2026-11-13: "Fetch all vacancies from the source, then search based on the search queries in the user profile."

1. The search editor shows three columns: OR terms, AND terms and exclusion terms. Every add or remove saves immediately without a Save button or unsaved-changes state. [implemented — `frontend/Collection.tsx`]

2. Each term is matched as a phrase after sanitisation (special characters stripped, whitespace normalised), so a multi-word term matches the exact phrase, not each word independently. Duplicate terms within one column are silently dropped. [implemented — `backend/internal/store/fts_query.go:72–90`]

3. Collection stores every posting of a source (stop words excepted); the profile's search expressions then determine which stored vacancies appear in that profile's Inbox: a vacancy appears in the Inbox of the profile whose expressions matched it and is not shown in other profiles' Inboxes (other profiles reach it through search). Multiple expression objects combine with AND at the top level; within each object OR terms join with OR and AND terms join with AND. [not implemented — SETTINGS-5 S.2; see [Profiles spec §3](profiles.md)]

4. Today, expressions build a combined BM25 ranking query together with the profile's target roles and skills. The BM25 ranking path updates a legacy score field on stored vacancies; it is called by the testable collection function but is not called by the production collection run. The displayed catalogue is sorted by Quick Match (Fits you). The UI caption stating that expressions "apply to the next collection run" is inaccurate for the current implementation. [deviates — `backend/internal/background/collect.go:21`; `backend/internal/store/rescore.go:8–10`]

5. Expressions are persisted in `search.yaml`. When expressions are present the legacy `titles` and `keywords` keys are cleared; when the expressions list is emptied they are restored. [implemented — `backend/internal/server/search_yaml.go:46–77`]

### 3.2 Exclusion terms (display-time effect)

The collection-time behavior of exclusion terms (vacancies blocked before storage) is in the [Collection spec](collection.md). This section covers their effect on already-stored vacancies.

6. A stored vacancy whose title or company name contains any exclusion term as a case-insensitive literal substring is hidden from every view including the Archive and from every navigation count. A multi-word term matches as a continuous substring with no tokenisation. [implemented — `backend/internal/store/stopwords.go:95–117`, `backend/internal/store/jobs.go:331–333`]

7. Before adding an exclusion term the owner can preview how many currently visible vacancies it would newly suppress, excluding vacancies already matched by an existing term. [implemented]

8. Removing an exclusion term makes stored vacancies visible again on the next poll. Vacancies that were never stored because the term matched them at collection time remain absent until a new collection run. [implemented]

### 3.3 Catalogue row content

9. Each row shows the company name (with a Web3 mark when flagged), the role title, a location summary and a publication date. In the applied view the row also shows the confirmed application date. When the vacancy has not yet been opened it is marked unseen; opening it marks it seen without a full list refresh. [implemented — `frontend/main.tsx:154–158`]

10. Each row carries a Fits you / You fit score pair. Fits you shows the current score (0–100), or a dash when unscored or stale; a stale score remains visible with a stale indicator and ranks after current results. You fit is hidden while the model's matching pass is switched off. [implemented — `frontend/quickmatch.ts:220–237`]

11. When no two-score result has been computed for a vacancy yet (`fit === null`), the row shows one placeholder (not two). [not implemented — INBOX-2 L.4c; built on the INBOX-2 branch, not yet in main]

### 3.4 Catalogue text search

12. The in-list text search filters the already-loaded list by case-insensitive substring across title, company, tags and description. It does not query the FTS index and does not affect collection or storage. [implemented — `frontend/workflow.ts:25`]

13. A vacancy absent from the loaded list cannot be found through the text search even if its text would match; card INBOX-1 proposed linking it to the FTS index and was not approved. Owner, 2026-11-13: accepted the coordinator's recommendation to keep client-side-only text search. [implemented]

### 3.5 Filters

14. The owner can narrow the list by: Fits you bucket (Strong 75+, Partial 55–74, Weak below 55, Not scored), discovery age (7 or 30 days), remote, Web3, leadership title, source, AI assessment status, and unread (Inbox and Shortlist only). [implemented — `frontend/workflow.ts:24`, `frontend/CatalogueFilters.tsx`]

15. The unread filter keeps the currently selected vacancy visible until the owner moves to another selection, so the owner is not forced out of a vacancy they just opened. [implemented — `frontend/main.tsx:47,83,151`]

16. Filter choices survive a reload. The Reset action clears all filters, the text search query, the unread toggle and resets the sort order to Best fit. [implemented — `frontend/main.tsx:78–79`]

### 3.6 Sort orders and server ordering

17. Two sort orders apply in Inbox and Shortlist: Best fit ranks vacancies by Fits you score descending (scored before unscored and stale, You fit as a tiebreaker); Newest ranks by publication date descending in three tiers (verified source date, unverified, absent). [implemented — `frontend/quickmatch.ts:294–299`, `frontend/dates.ts:39–42`]

18. The active sort order survives a reload and is also written to the server, which uses it to order the background analysis worker's queue; changing the toggle immediately reorders the queue without triggering a rescore. [implemented — `frontend/main.tsx:49,78`]

19. The server returns the job list ordered by LLM assessment score; the client immediately overrides this with Quick Match (Fits you) sort. The server-side ordering by LLM score has no visible effect in the production frontend. [deviates — `backend/internal/store/jobs.go:List`; `frontend/main.tsx`]

### 3.7 How the list stays current

20. On load the client fires a throttled collection attempt (the server skips it when one ran recently) and starts a periodic poll. The poll refreshes the full job list, the assessment queue status and the background worker status; a new poll cycle does not start while the previous one is running. [implemented — known performance issue: the full job list is approximately 35 MB on a 2 780-vacancy database]

21. A snapshot from a poll that started before an in-flight write (stage change, seen mark) is discarded; the client does not let a stale poll response overwrite a change the owner just made. [implemented — `frontend/main.tsx:64`]

22. After any change to exclusion terms or search expressions the displayed list refreshes immediately without waiting for the next poll tick. After a stage transition the client re-fetches only the affected vacancy, not the full list. [implemented]

23. The periodic poll reads only queue and worker status; the job list is fetched on load, on the owner's actions and when a server-reported change stamp changes, not on every tick. [not implemented — INBOX-2 L.4e; built on the INBOX-2 branch, not yet in main]

### 3.8 List resize

24. The catalogue panel width is adjustable by the owner via a drag handle; the width is clamped to a usable range, resets to a default on a double-click and survives a reload. [implemented — `frontend/ListResizer.tsx`]

## 4. Configuration

| What the owner can set | Where |
|---|---|
| OR and AND search terms | Settings → Search (after SETTINGS-5: the profile page, Search section) → `search.yaml` |
| Exclusion terms | Settings → Search (stays there after SETTINGS-5) → database `stop_words` table |
| Company block rules | vacancy detail pane, "Block company" action → database `block_rules` table; also see [triage.md](triage.md) |
| Fit bucket, freshness, remote, Web3, leadership, hide-expired filters | filter popover in the catalogue; choices persist across reloads |
| Sort order | sort toggle in the Inbox; also written to `runtime.yaml` on the server |
| List panel width | drag handle; persists across reloads |

## 5. Questions for the owner

All questions from this draft were answered by the owner on 2026-11-13 (see log.md). There are no open questions for this feature.

## 6. Open cards

- SETTINGS-5 — search expressions move to the profile page; exclusion terms, sources, providers and the schedule stay in Settings; starts after INBOX-2.
- INBOX-2 L.4c — catalogue row shows one placeholder (not two) when no two-score result exists; on the INBOX-2 branch, not yet in main.
- INBOX-2 L.4e — periodic poll reads only queue/worker status; job list fetched on change stamp; on the INBOX-2 branch, not yet in main.
- INBOX-1 — database-backed text search over all collected vacancies; not approved.
- GO-7 — wires the archive retention timer (applied vacancies deleted 30 days after confirmed application date); archive behavior is in [triage.md](triage.md).
