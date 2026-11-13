# Search and catalogue behavior specification

Written from the code of main `d49de9f` on 2026-11-13 by card OPS-11; not yet reviewed by the owner.

## 1. Purpose

This specification covers how roles are filtered, ranked, displayed and navigated in the catalogue — the left panel of the application — and how the search expressions and exclusion rules that govern collection are configured. It does not cover the scoring formula itself (see [Quick Match specification](quick-match.md)), application preparation, or collection mechanics beyond their search-filter contract.

Related specifications: [Quick Match](quick-match.md) · [Profiles](profiles.md) · [Vacancy view](vacancy-view.md).

---

## 2. Search expression configuration

### 2.1 Three groups

The owner configures search terms in Settings → Search in three groups. `frontend/Collection.tsx:138–160`

| Group | UI label | Meaning |
|---|---|---|
| OR | "Any of these" | At least one term must appear in the indexed vacancy text. |
| AND | "All of these" | Every term must appear anywhere in the indexed vacancy text. |
| Exclusions | "None of these" | A role whose title or company contains any exclusion term is hidden. |

Changes to any group are saved immediately (no draft state) and apply on the next collection run. `frontend/Collection.tsx:62–69`

### 2.2 FTS query construction

When the owner saves OR and/or AND terms the backend builds an FTS5 MATCH expression from them. `backend/internal/store/fts_query.go:28–70`

- Each term is sanitised before use: FTS5 special characters (`"`, `'`, `*`, `^`, `(`, `)`, `+`, `\`) are stripped; hyphens are converted to a space (treating them as word separators); runs of whitespace are collapsed to one space; leading and trailing whitespace is trimmed. `backend/internal/store/fts_query.go:127–159`
- Sanitised terms are wrapped in double quotes, making each one a phrase match (the multi-word phrase "Fractional CTO" matches as a phrase, not as independent tokens). `backend/internal/store/fts_query.go:72–90`
- Duplicate terms (after case-folding) within a group are silently dropped. `backend/internal/store/fts_query.go:75–85`
- An OR group whose terms survive sanitisation becomes `("term1" OR "term2" OR …)`. `backend/internal/store/fts_query.go:33–41`
- An AND group whose terms survive sanitisation becomes `("term1" AND "term2" AND …)`. `backend/internal/store/fts_query.go:43–51`
- Multiple non-empty groups are combined with AND at the top level: a role must satisfy every group. `backend/internal/store/fts_query.go:56–70`
- Empty groups (all terms collapse to empty after sanitisation) are silently skipped — an empty group imposes no constraint. `backend/internal/store/fts_query.go:24,36,44`
- When no terms survive at all, the return value is the empty string. The caller treats an empty string as "no FTS constraint" — all stored roles may match. `backend/internal/store/fts_query.go:28,106`

The resulting string is used at collection time, not at display time; it filters what the backend fetches and stores, not what the frontend shows from an already-collected list.

### 2.3 Search expression persistence

Expressions are stored in `search.yaml` under the `search.expressions` key as a list of maps with `or` and/or `and` keys. When expressions are present, the legacy `titles` and `keywords` keys are cleared (the expressions path is authoritative). When the expressions list is empty, the `expressions` key is removed and the backward-compatible `titles`/`keywords` path is restored. `backend/internal/server/search_yaml.go:46–77` `backend/internal/server/routes.go:1052–1079`

### 2.4 Exclusion preview

Before adding an exclusion term the owner can preview its effect: a debounced call to `GET /api/stop-words/preview?word=…` returns how many currently visible roles the word would newly hide, without creating any rule. `frontend/Collection.tsx:57–58` `backend/internal/server/routes.go:356–365`

The preview shows the count with the text "Would hide N role(s)" next to the input field. `frontend/Collection.tsx:157`

The count excludes roles already hidden by an existing exclusion rule (so adding an overlapping term reports only net-new hidden roles). `backend/internal/store/stopwords.go:129–162`

---

## 3. Stop words (exclusion terms)

### 3.1 Storage and normalisation

Stop words are stored in the `stop_words` table, normalised to lowercase on write (trimmed and lowercased). The UNIQUE constraint on the `word` column is therefore case-insensitive in effect. `backend/internal/store/stopwords.go:19–43`

Adding a word that is already present returns the existing row without error (idempotent). `backend/internal/store/stopwords.go:30`

### 3.2 Match rule

A role is excluded when its title OR its company name contains any stop word as a case-insensitive literal substring. The title is checked first; if it does not match, the company name is checked. `backend/internal/store/stopwords.go:95–117`

Multi-word stop words match as a continuous substring (e.g. the stop word "junior developer" matches a title containing "Senior Junior Developer"). No tokenisation is applied during the match — it is a raw substring check on the lowercased field.

### 3.3 Two application points

Stop words are applied at two points:
1. **Collection time**: the collection loop checks each incoming role against the current stop-word list and skips collecting it when it matches, so matched roles do not reach the database. `backend/internal/store/stopwords.go:87–90` (exported for the collection loop)
2. **Display time**: `GET /api/jobs` omits roles where the `stopped` flag is true; the client additionally filters `j.stopped` from its local copy to handle stale states. `backend/internal/store/jobs.go:331–333` `frontend/main.tsx:81`

Stop-word matches suppress the role from every view and every count, including the Archive.

### 3.4 List and removal

`GET /api/stop-words` returns all stop words ordered by creation time descending. `backend/internal/server/routes.go:346–352` `backend/internal/store/stopwords.go:47–70`

`DELETE /api/stop-words/{id}` removes a stop word by its integer ID. Removal does not un-hide roles that were blocked at collection time (they were never stored); roles that are present in the database but hidden by the stop word become visible again on the next poll. `backend/internal/server/routes.go:385–395`

---

## 4. Block rules (company blocks)

### 4.1 Kinds

Block rules are stored in the `block_rules` table with a `kind` and a `value`. Two kinds are defined in the matcher: `company` and `domain`. `backend/internal/store/blocks.go:109–136`

| Kind | Created by | Match field |
|---|---|---|
| `company` | "Block company" button in the detail pane | `normalizeText(job.Company)` exact match |
| `domain` | Not created by any current UI action (code exists, not wired) | `url.Parse(OriginalURL).Hostname()` exact match |

Source block rules of kind `source` were retired in migration v14; the source enable/disable toggle in search.yaml is now the only mechanism for disabling a source. Creating a source block rule returns an error. `backend/internal/store/blocks.go:84–103` `backend/internal/server/routes.go:315–319`

### 4.2 Match behaviour

A matched role has its `blocked` flag set to true and the matching rule stored as `blockedBy` ("kind:value"). `backend/internal/store/blocks.go:109–136`

Blocked roles are still returned by `GET /api/jobs`. The frontend hides blocked roles in the Triage view but shows them in other views. `frontend/main.tsx:83`

### 4.3 Creating a company block

Clicking "Block company" in the detail pane calls `POST /api/jobs/{id}/block` with `{"dimension":"company"}`. The server derives the block value as `normalizeText(job.Company)` and inserts the rule. `backend/internal/server/routes.go:208–225` `backend/internal/store/blocks.go:87–103`

Insertion is idempotent: a duplicate (kind, value) pair is silently ignored and the existing row is returned. `backend/internal/store/blocks.go:21–37`

---

## 5. What each catalogue row shows

Each row is a `<button>` element in the `job-list` div with CSS class `job-row`. `frontend/main.tsx:154–158`

| Element | Content | Notes |
|---|---|---|
| Company line | ⬡ mark (if isWeb3) + company name | ⬡ has aria-label "Web3" |
| Icon group | Expired indicator ◷ (if expired) + shortlist checkmark ✓ (in Triage view only, if stage=preselected) + score pair | Score pair from `ScorePair` component |
| Title | Role title | Rendered as `<strong>` |
| Location line | Summarised location text, or "Location unstated" | Compressed form shown in tooltip when text is truncated |
| Date line | Publication date/age text | "Published MMM D, YYYY · N days old" (verified) or "Source date … (unverified)" or "Publication date not reported" |

In the applied archive view the location line also shows "· Applied MMM D, YYYY" for the confirmed application date. `frontend/main.tsx:156`

### 5.1 Score pair in a row

The `ScorePair` component (`frontend/QuickMatchPanel.tsx`, called via `frontend/main.tsx:155`) renders two `ScoreCell` values derived from `fitsYouCell(j.fit)` and `youFitCell(j.fit)`. `frontend/quickmatch.ts:220–231,255–276`

- Fits you shows a numeric score (0–100) when scored and not stale, a dash otherwise.
- You fit is hidden entirely while the model's matching pass is switched off (`matchingOff === true` or status `off`). When off, only the Fits you cell is drawn. `frontend/quickmatch.ts:235–237`
- A stale result (after a profile save before rescoring completes) is shown with a visual indicator (the `previous` flag) and ranks after all current results.

### 5.2 Seen / unseen state

A row receives CSS class `seen` or `unseen` based on `j.seen`. When the owner selects a role that has not been seen, the client calls `POST /api/jobs/{id}/seen` to mark it, and the row transitions to `seen` without a full list refresh. `frontend/main.tsx:92–94`

---

## 6. The catalogue text search

A search input (shortcut `/`) sits at the top of the catalogue. `frontend/main.tsx:146`

The input runs a client-side substring filter via `quickMatch()` on the already-loaded list. The haystack is `${j.title} ${j.company} ${(j.tags??[]).join(' ')} ${j.description}`, lowercased; the query is lowercased; matching is by `.includes()`. `frontend/workflow.ts:25`

This filter is independent of the FTS search expressions used at collection time. The catalogue text search runs against the already-collected list and is not an FTS query. It does not affect what gets collected.

---

## 7. Filters

The filter popover is opened by the "Filters" button in the list toolbar. The active filter count badge shows the number of non-default active filters. `frontend/main.tsx:137,146–150`

### 7.1 Quick filters

| Filter | Key | Behaviour |
|---|---|---|
| Fits you buckets | `quick.fit` | Multi-select: Strong (75+), Partial (55–74), Weak (below 55), Not scored. Applies to the Fits you score only. `frontend/workflow.ts:24` `frontend/quickmatch.ts:302–306` |
| Collected within | `quick.fresh` | Toggles: Any time, 7 days, 30 days. Compares `j.firstSeen` (discovery date) against now. `frontend/workflow.ts:24` |
| Hide expired | `quick.hideExpired` | Hides roles where `j.expired` is true. `frontend/workflow.ts:24` |
| Remote | `quick.remote` | Shows only roles where location text matches `/remote|anywhere|telecommute/i`. `frontend/workflow.ts:24` |
| Web3 / blockchain | `quick.web3` | Shows only roles where `j.isWeb3` is true. `frontend/workflow.ts:24` |
| Leadership title | `quick.management` | Shows only roles where the title matches CTO, Chief Technology Officer, Head, Director, Manager, or Lead (excluding "Lead generation"). `frontend/filters.ts:10` |

Quick filter state is persisted in localStorage (`sift.quick`). `frontend/main.tsx:78`

### 7.2 Source filter

Shows a checkbox per distinct source name from the loaded configuration (`filter-options`). Each checkbox shows the count of loaded jobs with that source. When no sources are selected, the filter is inactive (all sources shown). `frontend/CatalogueFilters.tsx:9–13`

### 7.3 AI assessment filter

Shows a checkbox per assessment status value: assessed, stale, pending, running, failed, missing-description. The status of each job is derived from the queue rows and the job's own assessment field. `frontend/filters.ts:15` `frontend/CatalogueFilters.tsx:11`

### 7.4 Unread filter

The "Unread" toggle (Triage and Shortlist views only) restricts the list to roles where `j.seen` is false. When a role is selected while the filter is active, it is held visible until the owner selects a different role. `frontend/main.tsx:47,83,151`

### 7.5 Reset

The "Reset filters" button clears all quick filters, the selection-based filters (Sources, AI assessment), the text search query, the Unread toggle, and resets the sort order to Best fit. `frontend/main.tsx:79`

---

## 8. Sort orders

Two sort orders apply to Triage and Shortlist views. The Archive always sorts by confirmed application date descending, regardless of the toggle. `frontend/main.tsx:84`

### 8.1 Best fit

Ordered by `compareByFit(a.fit, b.fit)`: `frontend/quickmatch.ts:294–299`

1. Roles with a current (non-stale) Fits you score sort before roles without one.
2. Within the current group, higher Fits you raw score first.
3. Within ties, higher You fit raw score first; a pending or no_requirements You fit counts as 0.5 (the midpoint).
4. Final tie-break: lower job id first.

A result is current only when `fit.stale` is false and `fitsYou.status === 'scored'`. Stale results and roles without a Fits you number sort after all current results.

### 8.2 Newest

Ordered by `newestKey(j.publishedAt, j.publishedBasis)`: `frontend/dates.ts:39–42` `frontend/filters.ts:14`

1. Rank 0: roles with a recorded publication-date basis (a verified source date).
2. Rank 1: roles with a publication date but no basis (unverified source date).
3. Rank 2: roles with no usable publication date.
4. Within ranks 0 and 1, later date first; tie-break by lower job id first.
5. Rank 2 has no date to compare; tie-break by lower job id first.

### 8.3 Persistence and server synchronisation

The active sort order is stored in localStorage (`sift.sort`) and is also saved to the server via `PUT /api/quick-match/order` (which writes to `runtime.yaml`). `frontend/main.tsx:49,78`

The server-side value determines the order in which the background analysis worker processes vacancies, so changing the toggle immediately reorders the analysis queue. Changing the sort order does not trigger a rescore.

The server's value takes precedence on load: on mount the app reads `GET /api/quick-match/order` and sets the toggle to match. `frontend/main.tsx:50`

---

## 9. How the list refreshes

### 9.1 On mount

The app fires one throttled collection when it loads: `POST /api/search` with `{"force":false}`. The server honours a cooldown; if one was recently run, it returns `{"status":"skipped"}`. `frontend/main.tsx:68`

### 9.2 Polling

A 2.5-second interval poll calls, in order: `GET /api/queue` (assessment queue status), `GET /api/quick-match/analysis` (background worker status), then fetches the full job list via `GET /api/jobs`. `frontend/main.tsx:69`

The poll is serialised: a new poll does not start while the previous one is still running. `frontend/main.tsx:69`

A snapshot from a poll that started before an in-flight write (stage change, seen mark, etc.) is discarded: the `epoch` counter advances on every write start and every write end; a snapshot that was started at an older epoch is ignored. `frontend/main.tsx:64`

**Known performance issue**: the full job list (`GET /api/jobs`) is fetched on every poll tick (approximately 35 MB, 2.7 s server time per call on a database of 2 780 vacancies, measured in OPS-10). Card INBOX-2 step L.4e (on the INBOX-2 branch, not yet in main) is planned to replace this with a revision-stamp check, fetching the full list only when the stamp changes. `work/tasks/inbox/INBOX-2_listing-markup/plan.md:101`

### 9.3 After rules changes

When search expressions or stop words change, `onRulesChanged()` fires, which increments the epoch and triggers an immediate refresh of the job list. `frontend/main.tsx:144` `frontend/Collection.tsx:60,94`

### 9.4 After workflow transitions

After a stage transition (Dismiss, Select, Apply, Undo), the app re-fetches the individual job via `GET /api/jobs/{id}` to restore the Quick Match envelope. `frontend/main.tsx:97–104`

---

## 10. List resize

The vacancy list panel has a drag handle on its right edge. `frontend/ListResizer.tsx:24–45`

- Dragging the handle changes the list width in real time.
- Arrow keys while the handle is focused: ArrowLeft/ArrowRight adjust by 16 px; with Shift, by 48 px.
- Double-clicking the handle resets the width to 380 px (the default).
- Width is clamped to 260–720 px. `frontend/ListResizer.tsx:5–8`
- The current width is stored in localStorage (`sift.listWidth`) and restored on next load. `frontend/ListResizer.tsx:11–19`

---

## 11. Selection and keyboard navigation

| Key | Action |
|---|---|
| `J` or `↓` | Move to the next role in the filtered list. |
| `K` or `↑` | Move to the previous role. |
| `/` | Focus the search input. |
| `Y` or `KeyY` (physical key) | Select (shortlist) the current role (Triage, stage=triage only). |
| `N` | Dismiss the current role (Triage, stage=triage only). |
| `A` | Apply (opens the employer URL in a new tab, Shortlist view only). |
| `Escape` | Close the filter popover; blur the focused element. |

Keyboard shortcuts are suppressed when focus is inside an input, textarea, select, contenteditable element, textbox role, or switch role. `frontend/main.tsx:126–135` `frontend/workflow.ts:28`

Shortcuts are also suppressed when the profile screen, settings panel, or any `<dialog>` is open. `frontend/main.tsx:126`

Clicking a row selects it; if the selection changes, the held-unread state is cleared. `frontend/main.tsx:154–158`

After a Dismiss or Select action, focus advances to the next undecided role in the list (the first role after the current one that is still in `triage` stage). `frontend/main.tsx:97–104` `frontend/workflow.ts:27`

---

## 12. Counts and empty states

### 12.1 Stage counts in the navigation

| Stage | Count formula |
|---|---|
| Inbox | `live.filter(j.stage==='triage' && !j.blocked).length` |
| Shortlist | `live.filter(j.stage==='preselected').length` |
| Archive | `live.filter(j.stage==='applied').length` |

`live` is all loaded jobs with `j.stopped === false`. `frontend/main.tsx:81,140`

Stopped roles (matched by an exclusion term) are excluded from every count.

### 12.2 Filter summary

The filter popover footer shows "Showing X / Y roles" where X is the count after all filters and Y is the total unfiltered count. `frontend/CatalogueFilters.tsx:18`

### 12.3 Empty state messages

| Condition | Message |
|---|---|
| Filters active, no results | "No matching roles. Adjust your filters." with a "Reset filters" button. |
| No filters, Triage is empty | "Inbox zero. Collect more jobs in Settings → Sources." |
| No filters, Shortlist is empty | "Nothing shortlisted yet — press Y in Inbox." |
| No filters, Archive is empty | "No applications yet." |
| Session-dismissed roles visible in Triage | Note: "N dismissed this session — they disappear next time." |

`frontend/main.tsx:157–158`

---

## 13. Archive

The Archive view (stage = "applied") has two tabs: Applied and Dismissed.

- **Applied** tab: shows roles where `stage === "applied"`, sorted by `appliedAt` descending. Shows a retention note: "Applications are permanently deleted 30 days after the confirmed application date. Entries without a valid date are kept." `frontend/main.tsx:153`
- **Dismissed** tab: fetches roles via `GET /api/jobs?status=no` on demand; sorted as returned by the server (by assessment score descending). `frontend/main.tsx:77,155`

A dismissed role in the Dismissed tab can be restored to Inbox via "Restore to Inbox" (which calls `PATCH /api/jobs/{id}` with `{"stage":"triage"}`). `frontend/main.tsx:116`

---

## Implemented on a branch, not released

- **L.4c (INBOX-2 branch)**: When the two-score result is absent entirely (`fit === null`), the catalogue row now draws one Fits you placeholder instead of two. Also: after a collection run, the Quick Match two-score result is computed locally (without a model call) for newly collected vacancies immediately, so Fits you appears without waiting for the background worker. `work/tasks/inbox/INBOX-2_listing-markup/plan.md:43`
- **L.4e (INBOX-2 branch)**: The jobs list is fetched on load, on the owner's actions and on a server-reported change stamp, not on every 2.5-second tick. The 2.5-second poll reads only the queue and worker status. `work/tasks/inbox/INBOX-2_listing-markup/plan.md:101`

---

## Agreed, not implemented

- **SETTINGS-5 (Profiles spec §2)**: Search expressions move from Settings into each role profile; the profile can have its own search terms, goals and triage rules. Card SETTINGS-5 starts after INBOX-2. Owner, 2026-11-12: «поиск переходит в профиль». [Profiles spec](profiles.md)
- **INBOX-1 (backlog, not approved)**: A combined search/scoring/overview card proposing to link the UI search bar to the FTS engine (currently the UI search is client-side substring only). Not approved; see `work/tasks/inbox/INBOX-1_search-catalogue-overview-quick-match/plan.md`.

---

## Known limits and defects

- **UI search is client-side only**: the catalogue text search input (shortcut `/`) does not query the FTS index; it filters the already-loaded list by substring on title + company + tags + description. A role not in the loaded list cannot be found through it even if its text would match.
- **Full-list polling**: the app fetches all jobs every 2.5 seconds (not verified, but measured in OPS-10 on staging at approximately 35 MB per call). This keeps the server busy and is listed as a known performance defect to be fixed by INBOX-2 L.4e.
- **Stop words do not un-hide roles already in the database**: removing a stop word makes roles that exist in the database visible again, but roles that were blocked from ever being collected (stop-word applied at collection time) remain absent from the database; a new collection run is required to bring them in.
- **Domain block rules cannot be created from the UI**: the `domain` kind matcher exists in the code but no UI action creates it; block rules can only be of kind `company`.
- **Archive retention timer not wired**: the archive-retention deletion method exists but no background ticker calls it. Card GO-7 owns the wiring. `docs/MAP.md:R-TRI`

---

## Contradictions

- MAP.md (R-SEARCH) states: "FTS/BM25 remains query matching; fit ordering is determined separately by Quick Match (R-ASM), not by the BM25 score." The code confirms this for display ordering. However, `store/jobs.go:List` sorts the returned list by `assessment.score` (the LLM assessment score, not the Quick Match Fits you score) when no `sort` parameter is passed. The frontend does not pass a `sort` parameter to `GET /api/jobs` and applies `sortCatalogue` client-side — so the server-side `sortJobs` order is overridden immediately by the client. The server-side sort by assessment score therefore has no visible effect in the current production frontend, but the code is not dead: a direct API call without a client would receive that order.
- MAP.md (R-SEARCH) says "COL-4 is the separate collection-detail gap, not a search gap"; the code at `backend/internal/server/routes.go:426` confirms `GET /api/collection` works for the top-level run summary, but the detail rows are empty per COL-4.
- `frontend/Collection.tsx:139` reads "Changes save immediately and apply to the next collection" (about expressions). This is accurate — the expressions are saved to `search.yaml` on every add/remove, but they only filter during collection, not at display time. The UI does not say this clearly: the user might expect a saved expression to immediately change the visible list.

---

## Related specifications

- [Quick Match specification](quick-match.md) — Fits you and You fit scoring, the fact extractor and the two-score result that the catalogue displays.
- [Profiles specification](profiles.md) — Agreed design for per-profile search, goals and triage; search moves into profiles in SETTINGS-5.
- [Vacancy view specification](vacancy-view.md) — What the detail pane shows when a role is selected.
