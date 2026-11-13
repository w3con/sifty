# OPS-11 — combined questions for the owner (2026-11-13)

Collected from the seven corrected feature specifications in `specs-v2/`. Each question carries the coordinator's recommendation; the recommendation is the coordinator's, not a decision. Already answered by the owner on 2026-11-13: collection run feedback, search expressions select the profile's Inbox, collection on/off switch, failed run status, collection interval (see `log.md`).

## Collection (`collection.md`)

1. Keep the "Collect jobs" button (starts a run at once, skipping the interval)? Recommendation: keep — useful right after adding a source.
2. The daily 09:00 scheduler runs even when the page is closed. Keep it, or replace it by the interval rule? Recommendation: replace it by the interval rule and let the server apply the same rule while the page is closed, so there is one rule.
3. Eight `runtime.yaml` keys are read but do nothing (`max_pages`, `max_jobs_per_source`, `max_reviewed_per_source`, `max_detail_requests`, `detail_concurrency`, `max_query_groups`, `request_timeout_ms`, `model_extraction`). Recommendation: wire `request_timeout_ms` (a hung site can hold a run open forever today), remove the other seven.

## Catalogue and search (`catalogue.md`)

4. The `/` search looks only in the loaded list. Keep it, or search the database? Recommendation: keep — the whole catalogue is loaded anyway.

## Triage and decisions (`triage.md`)

5. Screen names: Inbox / Shortlist / Archive (in production since DESIGN-3) or Triage / Preparation / Archive (MAP and old cards)? Recommendation: Inbox / Shortlist / Archive; update MAP and cards.
6. Detailed model assessment: every vacancy every 30 seconds (today, while the queue is on) or only vacancies the owner selected? Recommendation: selected only.
7. Applied vacancies are promised to be deleted after 30 days (the deletion is never called), which would erase the record of past applications that TRIAGE-1 needs. Options: (a) keep a short application record before deleting, (b) never delete automatically, remove the promise, (c) delete and accept the loss. Recommendation: (b).
8. "Block company" hides a company in Inbox only and the 15 blocks cannot be seen or removed; NAVIGATION-2/3 designed one exclusion list (hides everywhere, stops collection). Recommendation: one exclusion list, visible and editable in Settings; blocking the placeholder company "Unknown" refused. This also settles domain blocks (remove the unused kind).

## Vacancy view (`vacancy-view.md`)

9. The INBOX-2 branch adds a "?" money state (the owner's minimum falls inside the range) and a day rate × 260 to the agreed ✓/✕ rule. Accept both? Recommendation: accept.
10. The Location tile shows no verdict; both prerequisites (work authorizations in the profile, region comparison) are released. Should it show the geography verdict? Recommendation: yes, small separate card after INBOX-2.

## Candidate profile (`profiles.md`)

11. Fix the geography editor's "not set" (SETTINGS-9) inside the first stage of multiple profiles? Recommendation: yes, same code.
12. Website and social profile links as sources (SETTINGS-4): still wanted? Recommendation: keep in backlog after multiple profiles.

## Local data (`data.md`)

13. Fresh start or moving to another machine: a documented manual procedure, or an application command? Recommendation: documented procedure.
14. Should the snapshot taken at every release also cover the owner's files (settings, documents, answers), so a rollback restores everything? Recommendation: yes.

## Application preparation (`application.md`) — for a later design session

The feature is not designed (owner, 2026-11-13). Nine design questions (answers store, what is prepared first, who writes the first answer, matching, sensitive answers, when drafting starts, the "think" dialogue, shortlist re-check on a stronger model) stay in the spec for a dedicated design session. Recommendation: do not answer now.

## Findings without a card (for the owner to decide whether to found cards)

- Search expressions edited in Settings have no visible effect in production; the caption "apply to the next collection run" is wrong (SETTINGS-5 S.2 builds the real behaviour; the caption can be fixed now).
- Keyboard shortcuts N and A depend on the keyboard layout (Y was fixed by TRIAGE-2).
- "Not suitable" does not ask why (owner's remark in APP-8 comments).
- The owner-decided collection items (feedback, on/off switch, failed status, interval setting) — one collection card after INBOX-2 and COLLECTION-2.
