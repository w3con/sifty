# Profiles behavior specification

Status: **agreed with the owner on 2026-11-12, not implemented.** Card: [SETTINGS-5](../../work/tasks/settings/SETTINGS-5_user-data-and-role-profiles/plan.md) (design sections 1–8, re-plan section 10). Each stage moves its part from "agreed" to "implemented on the branch" and, after release, to "released". Related: [Quick Match](quick-match.md) §14 (two scores), §15 (profile as a fact store, vacancy pipeline).

## 1. Purpose

One person looks for more than one kind of work (for example blockchain roles and general engineering roles). Each kind is a **profile**: its own goals, its own search, its own scores and decisions. The facts about the person exist once and are shared by every profile.

## 2. My data and profiles

| My data (shared, one per person) | Profile (one per kind of work) |
|---|---|
| Identity, contacts, photo | Name (for example "Blockchain", "General engineering") |
| Skills and levels; "Skills you do not have" (✓ / ✕) | Target roles and industries |
| Experience; computed facts (years, leadership, founder, current role) | Compensation, engagement, hours, work modes, wanted places, dealbreakers |
| Languages; current location and work authorization | Search: titles, keywords, expressions |
| Library of documents (CV files, links) and the facts accepted from them | Fits you, Triage, Shortlist, rejections and their reasons |
| Answers for application forms; history of submitted applications | |

- My data is a fact store (Quick Match §15.1): a model runs only when a file or link is added, to propose facts; accepted facts are the only input of comparisons.
- A profile refers to shared facts; it never holds its own copy of a skill or an experience entry.
- The active profile is the context of the whole application (header switcher). Background work never reads "the active profile" implicitly: every scoring task carries the profile it is for.

- No headline for now (owner, 2026-11-12): role matching uses target roles and experience; a CV title may return per profile with application preparation.

**How facts are proposed from documents (implemented SETTINGS-8, 2026-11-12)**

One fact record is created per extracted item. Skills and languages each produce one fact per item (not a combined list fact). Work authorizations produce one fact per citizenship or residency statement. Each fact carries an exact verbatim quote from the source document.

A fact goes through four possible states:
- `proposed` — a new value the user has not yet seen; shown as a pending decision.
- `profile_match` — the extracted value is already present in the confirmed profile (same field, same canonical value); shown as "already in your profile" and not a pending decision.
- `accepted` — the user confirmed the value; it has been written into the profile.
- `rejected` — the user dismissed the proposal; the profile is unchanged.
- `conflict` — a different confirmed value already exists for a single-value field; the user must choose.

Before storing facts, the extraction pipeline compares each proposed value against the current confirmed profile:
- For single-value fields (name, location, etc.): trimmed-equal values → `profile_match`.
- For list fields (skills, languages, target roles): value already in the list → `profile_match`.
- For experience entries: same company name and either same title or overlapping date span → `profile_match`.

A document that exceeds the model's context budget is split into chunks; proposals from all chunks are merged and de-duplicated before storage.

## 3. Search belongs to the profile

- Each profile has its own search (titles, keywords, expressions). The search editor lives on the profile page, not in Settings.
- Sources (the sites list) are shared by all profiles and stay in Settings, with stop words, providers and the schedule.
- A collection run fetches each source once and applies the search of every profile; it records which profile's search found each vacancy.
- **Inbox scope:** a vacancy appears in the Inbox of the profile whose search found it; other profiles do not show it there and reach it through search. The catalogue of vacancies is one; nothing is copied per profile.
- **Copy to search:** the Skills section offers "Copy to search" for one skill, a selection or all; it adds them to the active profile's search keywords, skipping duplicates. The skill stays a fact; removing the word from search does not touch the skill.

## 4. Scores and decisions per profile

- **Fits you** is computed per profile, from that profile's goals and the shared facts (Quick Match §14.1). Switching profiles reads stored results and never calls a model.
- **Vacancy extraction** (requirements and conditions read by the model, Quick Match §14.5) depends only on the vacancy text and is shared by all profiles; a second profile does not double the model's work.
- **You fit**, when it returns (ASSESSMENT-5), is computed per profile.
- A change of a profile makes only that profile's results stale; a change of shared facts makes the affected results of every profile stale.
- The background worker serves every profile; its order follows the active profile's Inbox toggle (Best fit / Newest).
- **Triage, Shortlist, rejections** and their history are per profile: a vacancy rejected for one profile may be shortlisted in another.
- **A submitted application is a fact about the person:** it is recorded once with the profile used, and it is visible in every profile.

## 5. Profile page layout

The centre column has two groups with their own headings:

1. **What I am looking for** (this profile): Target roles, then **Search** (one explanatory line: "Words Sifty looks for in listings"; titles, keywords and expressions as chips), then compensation and working conditions.
2. **About me** (shared by all profiles): Skills (with "Copy to search"), Experience, Languages, documents.

The difference between "Rust" as a skill and "Rust" as a search word is made by the grouping and order, not by colour (colours carry state meaning in the design system).

## 6. Managing profiles (interface)

- Create: the existing "Add new profile" in the header switcher.
- Switch and rename: in the switcher; the name is stored with the profile, not in the browser.
- No copy and no archive in the interface for now; the server keeps an archive operation for later. Archiving a profile never removes shared facts, documents or the application history.

## 7. Migration

The current single profile becomes My data plus the first profile (named by the owner; today's blockchain focus). Shared facts move to My data; goals and search stay with the profile; ids, revisions, quotes and history are preserved. Existing Triage, Shortlist and rejection states belong to the first profile. A second profile "General engineering" is seeded from the search backup of 2026-11-10 at the owner review.

## 8. Out of scope

Detailed assessment (AI analysis) and application preparation per profile; several users or accounts; automatic choice of the "best" profile for a vacancy.
