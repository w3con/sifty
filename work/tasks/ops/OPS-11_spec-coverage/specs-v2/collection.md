# Collection of Vacancies — specification

Status: owner decisions of 2026-11-13 applied (OPS-11); rule tags show the implementation state.

## 1. Purpose

Collection brings vacancy listings from the owner's configured job-board sources into Sifty automatically, complete and readable, without storing duplicates or category pages, and without hammering any site. Every collected vacancy lands in the Triage stage as a "new" record, ready for the owner's review. Collection does not rank, score or assess; those steps run on stored records after the fact (see [quick-match.md](quick-match.md)).

## 2. Scope

This specification covers: source configuration; collection triggers and scheduling; fetching safety and pacing; description acquisition and text handling; deduplication and update of stored records; publication date; deleted-listing records; exclusion terms at collection. How the profile's search expressions filter which stored vacancies appear in the owner's Inbox is covered by [catalogue.md](catalogue.md). How the owner filters, triages and acts on vacancies is covered by [triage.md](triage.md). How the description, sections and publication date are shown in the vacancy view is covered by [vacancy-view.md](vacancy-view.md). How the collected text feeds Quick Match scoring is covered by [quick-match.md](quick-match.md).

## 3. Rules

### Sources

1. A source is an entry in `search.yaml` with a unique name, an http(s) URL, and an optional enabled flag that defaults to true. [implemented]
2. The owner adds, enables/disables and removes sources in Settings › Collection › Sources; the URL of an existing source cannot be changed in the application, only its enabled state can be toggled. [implemented — SETTINGS-6 will add URL editing]
3. Adding a source requires both a name and a URL. [deviates — the form labels the URL field as "URL (optional)" while the server requires a non-empty URL (`frontend/Collection.tsx:129`, `backend/internal/server/search_yaml.go:123–124`)]

### Adapters

4. Sources with a Greenhouse, Lever or Ashby board URL are collected from the ATS public API in one request with descriptions included; the company name is taken from the board slug or the organisation name returned by the API. [implemented]
5. For every other source, the owner-specified listing page is fetched and postings are found by structured data first (JSON-LD JobPosting), then a feed (RSS or Atom), then job-looking links in the HTML including Getro hydration data; one listing page is read per run, without pagination. [implemented]

### Triggers

6. Opening or reloading the application starts a collection when none is in progress and none completed within the collection interval. [implemented]
7. An unforced run is skipped when the most recent run of any kind started less than the collection interval ago (default 3 hours); the "Collect jobs" button forces a run that skips the interval but not the cross-process lock. Owner, 2026-11-13: accepted the coordinator's recommendation to keep this action. [implemented]
8. The collection interval is configurable by the owner in Settings › Sources. Owner, 2026-11-13: "This can be added to Settings, under Sources, as a frequency setting for how often to collect vacancies." [not implemented — no card; today only `collection_cooldown_ms` in `runtime.yaml`]
9. The server applies the same interval rule (rule 6) while the application page is closed: when the interval has elapsed since the last completed run, a new run starts on the next server health check, without requiring the application to be open. The separate daily scheduler at a fixed time is removed. Owner, 2026-11-13: accepted the coordinator's recommendation. [not implemented — no card; today the daily scheduler at `daily_hour`/`daily_minute` in `runtime.yaml` runs in its place]
10. Only one collection run can be in progress at a time; when `collection_disabled` is `true` in `runtime.yaml`, every trigger returns "skipped" and nothing is fetched. [implemented]
11. Settings › Sources has a collection on/off switch that disables all triggers. Owner, 2026-11-13: "There is no collection on/off toggle — it could be added to the Sources page in Settings." [not implemented — no card; today only `collection_disabled` in `runtime.yaml` via the Advanced editor]
12. After a successful run, Quick Match is recomputed locally for stored vacancies without a model call. [implemented]

### The run

13. A run reads each enabled source in `search.yaml` file order, one at a time; a source that fails (network error, block, unparsable response) is logged and skipped, and the run continues with the next source. [implemented]
14. A posting whose canonical URL was already seen in the same run from any source is skipped without a detail fetch. [implemented]
15. Exclusion terms are applied to the posting's title and company name before the detail page is fetched and again after resolving the description; a matching posting is not stored and its detail page is not fetched. [implemented]
16. The configuration snapshot taken when a run starts governs the whole run; source changes and exclusion-term changes made during a run apply to the next run. [implemented]

### Fetch safety and pacing

17. Only public http(s) addresses are fetched; private, loopback, link-local, multicast and unspecified addresses are refused before every request and at every redirect hop (at most six hops); response bodies over 8 MB are refused; the fetcher identifies itself as "Sifty/1.0 (personal read-only job discovery)". [implemented]
18. A 403 response or a page whose body matches human-verification markers (Cloudflare, "verify you are human", "just a moment…") is treated as a blocked result, not as content; the posting is kept with its listing fields and no description. [implemented]
19. Consecutive requests to the same domain are paced (at least 3 s plus up to 3 s jitter); a request failure triggers an exponential backoff (starting at 30 s, doubling up to 1 h; a success resets it). [implemented]
20. A site that blocked Sifty is not requested again for 24 hours. [deviates — the 24-hour block cooldown resets at the end of every run because a new HTTP client is created per run; a blocked site is requested again at the next run (`backend/internal/background/collect.go:22`)]
21. Every request has a time limit set by `request_timeout_ms` in `runtime.yaml`. Owner, 2026-11-13: accepted the coordinator's recommendation to wire this key. [not implemented — no card; today `request_timeout_ms` is read but not applied; a hanging server can hold a run open indefinitely (`backend/internal/background/collect.go:39`)]

### Description acquisition

22. For Greenhouse, Lever and Ashby sources, the description is taken from the listing API response; no detail page is fetched. [implemented]
23. For other sources, descriptions from JSON-LD or feed items in the listing are used directly; a detail page is fetched only for postings found as plain HTML links that carry no description. [implemented]
24. When a detail page is fetched and contains JSON-LD, exactly one posting is matched by URL or by unique title and company; an ambiguous page is never guessed. [implemented]
25. When no structured data exists on a detail page, the cleaned text of the whole page body is used as the description, which may include navigation and other page chrome not part of the listing. [implemented — INBOX-2 fixes specific sources (Remote3, We Work Remotely, Web3 Careers) to use the correct content block]
26. Text cleaning removes script, style and navigation elements; headings, list markers and numbering are lost. [implemented — INBOX-2 replaces this with a structured form that preserves them]
27. A detail page that is blocked or unreachable leaves the posting with its listing fields and no description; the posting is not discarded. [implemented]
28. Category pages (a page whose title is a known category name and company is "Unknown") are not stored as vacancies. [not implemented — INBOX-2 fixes the Remote3 adapter; COLLECTION-2 adds this guard for Remotive and a general filter for every source]
29. The original fetched HTML or feed item of every newly collected vacancy is stored compressed as received, so the vacancy can be re-parsed without a new network request when the parser improves. [not implemented — INBOX-2]
30. A sanitised structured form of the description is stored alongside the plain-text description; the plain text is derived from the structured form by one function. [not implemented — INBOX-2]
31. Each source has stored parsing rules (which content block is the description, how headings look); the parser measures its own quality per source; when a source falls below a threshold, stored originals are re-parsed using updated rules without a new network fetch. [not implemented — COL-2]

### Deduplication and updates

32. A new posting is matched to an existing vacancy by canonical URL, by apply URL, or by normalised identity (lower-cased company plus lower-cased title); the identity match is skipped when the company is "Unknown" or when the apply URLs conflict. [implemented]
33. A posting with no match is inserted as a new vacancy with status "new"; deleted and merged vacancy ids are never reused. [implemented]
34. A matched vacancy is updated; when multiple existing vacancies match, all are merged into the oldest, with associated drafts, assessments, sightings and history moved to the survivor; workflow state (dismissed, applied, shortlisted) is never lost on update. [implemented]
35. The stored description is replaced when the incoming description is fresh (at least 100 characters with a status that does not contain "metadata", "pending", "missing" or "unavailable") or at least as long as the stored description, unless the stored description is in a final state. [deviates — the freshness check uses a character-set comparison against a concatenated word list, so nearly every status including "available structured description" and "available cleaned html" is treated as non-final; in practice a longer incoming description always replaces the stored one (`backend/internal/store/jobs.go:593–603`)]

### Publication date

36. Only a date declared by the source as a publication date is stored; modification dates (Greenhouse `updated_at`, Atom `updated`) are excluded; an established date is never overwritten by a different date from a later collection. [implemented]
37. A source date is accepted in ISO, date-only, RFC and feed formats; a date in the future is rejected; accepted dates are normalised to UTC. [implemented]
38. A vacancy whose `validThrough` date has passed is flagged "Listing expired" but is not hidden or removed. [implemented]
39. No vacancy is hidden, penalised in ranking or removed because of its age. [implemented]

How publication dates and the Newest ordering are displayed is covered in [catalogue.md](catalogue.md) and [vacancy-view.md](vacancy-view.md).

### Deleted listings

40. When the owner permanently deletes a vacancy, every URL the vacancy was ever seen under is recorded as a URL hash; a posting whose canonical URL or apply URL matches a recorded hash is silently dropped at collection. [implemented]
41. A posting whose URL matches a deleted-listing record costs no detail fetch. [deviates — the detail page is fetched before the check and the posting is counted in the run log before being dropped (`backend/internal/background/collect.go:62–82`)]
42. Permanent delete requires the owner's explicit confirmation, is refused while a collection or inference run is in progress, and cannot be undone. [implemented]

### Exclusion terms

43. The owner manages exclusion terms (the "None of these" list) in Settings › Collection › Search; a term is a plain string matched as a case-insensitive literal substring of a posting's title or company name; a matching posting is not stored and its detail page is not fetched when the title is known from the listing. [implemented]
44. Stored vacancies are hidden by the same terms at read time; removing a term makes them visible again without a new collection run. [implemented]
45. The interface shows how many stored vacancies a new term would hide before the owner adds it. [implemented]
46. Collection fetches every posting of every enabled source; the profile's search expressions then decide which stored vacancies appear in that profile's Inbox. Owner, 2026-11-13: "Fetch all vacancies from the source, then search based on the search queries in the user profile (currently still in Settings, but a move to the profile is planned)." [not implemented — SETTINGS-5 S.2; today expressions affect only ranking]

### Owner visibility

47. The owner can see that a collection is in progress. [implemented]
48. Per-source detail (which sources ran, counts added/updated/duplicate per source, event timeline) is not shown because the collection run never writes per-source records. [not implemented — COL-4]
49. A source that failed or returned no vacancies is shown to the owner as a warning notice in Inbox and Shortlist, and as a coloured mark next to that source in Settings › Sources. Owner, 2026-11-13: "An informational notice (probably striped orange?) in Inbox/Shortlist and also on the Settings page — in the sources list — with some symbol and colour." [not implemented — no card]
50. A run in which every source failed or returned nothing is recorded as "failed" and the owner receives a message. Owner, 2026-11-13: "Yes, and show a message." [not implemented — no card]
51. New vacancies appear in Inbox with status "new" after a completed run. [implemented]

## 4. Configuration

| Key | File | Default | Purpose |
|---|---|---|---|
| `sites` | `search.yaml` | (none; at least one required) | The list of job-board sources: each has a unique name, an http(s) URL, and an optional `enabled` flag. |
| `collection_cooldown_ms` | `runtime.yaml` | 10 800 000 (3 hours) | Minimum gap between unforced collection runs. Not listed in the `runtime.yaml` template. |
| `daily_hour`, `daily_minute` | `runtime.yaml` | 9, 0 | Local time for the daily scheduled run. See rule 9: to be replaced by the server-side interval rule. |
| `request_timeout_ms` | `runtime.yaml` | (none; no timeout today) | Per-request time limit for collection HTTP requests. See rule 21: to be wired. |
| `collection_disabled` | `runtime.yaml` | false | When true, all collection triggers return "skipped" and nothing is fetched. |
| Exclusion terms ("None of these") | Settings › Collection › Search | (empty) | Substrings that prevent a matching posting from being stored. Stored in the database, not in a config file. |

The following keys in `runtime.yaml` are parsed into configuration but have no effect on collection and are to be removed: `max_pages`, `max_jobs_per_source`, `max_reviewed_per_source`, `max_detail_requests`, `detail_concurrency`, `max_query_groups`, `model_extraction`. Owner, 2026-11-13: accepted the coordinator's recommendation. The proxy mapping in `network.yaml` is also not used by collection.

## 5. Questions for the owner

All questions from this draft were answered by the owner on 2026-11-13 (see log.md). There are no open questions for this feature.

## 6. Open cards

- COL-2 — per-source parsing rules, their quality measurement, and repair from stored originals; depends on INBOX-2 storing originals.
- COL-4 — write per-source run records and events so the progress indicator shows real per-source detail.
- COL-7 — operational instruction to run a production collection and assessment (backlog; coordinator to decide whether still wanted).
- COLLECTION-1 — publication date display and age accounting; released to production on 2026-11-10; card in review pending OPS-9 review closure.
- COLLECTION-2 — remove Remotive category pages and add a collection guard against them; depends on INBOX-2 being released first.
- COLLECTION-3 — sources that require a login (LinkedIn and others); no behaviour agreed yet.
- SETTINGS-6 — allow editing a source URL in Settings › Sources.
- INBOX-2 — structured description form, stored originals, section splitter, source-specific collection fixes (Remote3, We Work Remotely, Web3 Careers); implemented on branch `work/INBOX-2`, not yet in main.
