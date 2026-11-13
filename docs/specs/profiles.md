# Candidate Profile — specification

Status: owner decisions of 2026-11-13 applied (OPS-11); rule tags show the implementation state.

## 1. Purpose

The candidate profile is the owner's self-description that every other feature compares vacancies with. It is the input to fit scoring (Fits you, You fit), vacancy collection (search expressions), and application preparation. The owner maintains it by filling in sections directly, uploading CV documents and accepting the facts the model extracts, and by marking requirements in the vacancy view. Every comparison runs on accepted profile facts only, never on a document.

## 2. Scope

This specification covers: the profile data model and its sections; CV document upload, analysis, and fact management; facts added from the vacancy view ("Confirmed by you"); the profile switcher; and configuration that governs the profile. It covers the agreed multi-profile design, tagged where not yet implemented.

Not in scope here: how Fits you and You fit are scored (see [Quick Match spec](quick-match.md) §6 and §14.12); how the vacancy view presents requirements and offers ✓/✕ actions (see vacancy-view spec); vacancy collection and search execution (collection spec); application preparation.

Out of scope for this feature: detailed assessment and application preparation per profile; more than one user or account; automatic selection of the best profile for a vacancy.

## 3. Rules

### Identity and avatar

1. The profile has a display name and a legal name. The legal name is used only where an application form asks for it. [implemented]

2. The profile can hold an avatar (photo). When a photo is uploaded, it is shown; when none is set, the owner's initials are shown, or a user icon when no name is set. The owner can remove the photo at any time. [implemented]

3. A profile headline is not proposed from documents and is not sent to the matcher; any stored headline value is preserved and not deleted. The headline may return as a per-profile field with application preparation; that is not decided yet. [implemented — SETTINGS-8 F.5]

### Skills

4. Each confirmed skill has a name and a level (Beginner, Proficient, Expert, or not set). Skills do not contribute to Fits you, which weighs goals only (quick-match.md §14.1); they decide requirement rows in the vacancy view and are an input of You fit (off by default, quick-match.md §14.14); levels are used in detailed assessment and preparation. A skill is matched against requirements by the skill vocabulary with aliases and whole words, so one skill answers differently worded requirements in every vacancy; that is the difference from a "Confirmed by you" fact (rule 41). [implemented — SETTINGS-3; the Fits you sentence corrected 2026-11-13]

4a. A skill's key keeps the characters that distinguish names: C, C++, C# and F# have different keys. Owner, 2026-11-13 (translated): "C# and C++ are completely different languages." Stored keys are recomputed when the profile is loaded. [not implemented — INBOX-2 (decided 2026-11-13); today `skillKeyOf` in `backend/internal/profile/service.go` drops + and #, so C, C++ and C# share the key "c"]

5. The profile keeps a separate "Skills you do not have" list (absent skills). A requirement for an absent skill counts as not met. Absent skills suppress those requirements from future offers. [implemented]

6. Absent skills are added from the vacancy view, not from the profile editor. Adding a skill removes it from the absent-skills list; naming a skill in the absent-skills list removes it from confirmed skills. [implemented]

### Goals

7. The profile states target roles (free-text tags) and target industries (free-text tags). Target roles contribute to Fits you; target industries are used in detailed assessment and preparation. [implemented]

### Search (per profile)

8. Each profile owns its own search: titles, keywords, and expressions. The search editor lives on the profile page, not in Settings; sources, stop words, providers and schedule stay in Settings. [not implemented — SETTINGS-5 S.2; today a single search.yaml holds all expressions, shared by the one profile]

9. A collection run fetches each source once and applies every active profile's search expressions. The vacancy catalogue is shared; a vacancy appears in the Inbox of the profile whose search found it; other profiles reach it through search only. [not implemented — SETTINGS-5 S.2]

10. Vacancy extraction — the model reading requirements from the vacancy text — depends only on the vacancy text and is shared by all profiles. A second profile does not cause extraction to run again. [not implemented — SETTINGS-5 S.2]

11. The Skills section offers "Copy to search": adding one skill, a selection, or all skill names to the active profile's search keywords, skipping duplicates. The skill remains a confirmed fact; removing the word from search does not touch the skill. [not implemented — SETTINGS-5 S.5]

### Compensation

12. The profile states minimum and desired cash amounts, currency, tax basis (gross/net), period (month/year), payment method (crypto accepted/only/not accepted), and an equity-interest flag. A day-rate sub-section covers contracts. [implemented — SETTINGS-3]

13. How the compensation section feeds into Fits you and You fit: see [Quick Match spec](quick-match.md) §6 and §14.12.

### Engagement, hours, and work modes

14. The profile states engagement types (employment, contract), working time (full-time, or part-time with optional desired hours per week), and work modes (remote, hybrid, on-site). [implemented — SETTINGS-3]

### Geography and work authorization

15. The profile states a current location (structured country and optional city). When a structured location has not yet been set, a legacy free-text location is retained until the owner replaces it. [implemented]

16. When the structured location fields are empty, the edit form shows "not set" even when a legacy location text exists; the legacy text is not pre-filled in the editor. Owner, 2026-11-13: accepted the coordinator's recommendation to absorb this fix into SETTINGS-5 S.1 — the fix and the structured-location migration belong to the same code. [deviates — SETTINGS-9; to be fixed within SETTINGS-5 S.1]

17. The profile lists preferred regions, preferred work locations (structured country and optional city), and work authorizations by country (country + status: eligible to work / needs visa sponsorship). [implemented — SETTINGS-3]

18. Work authorization establishes eligibility to work in a country. Current residence does not establish work eligibility. [implemented — ASSESSMENT-8; Quick Match spec §4.1 §6.2]

19. The profile states willingness to relocate (tri-state), preferred timezones (IANA IDs), and desired overlap hours. [implemented — SETTINGS-3]

20. How geography and work authorization feed into Fits you: see [Quick Match spec](quick-match.md) §6 and §14.1 (geography group applicability, restricted remote).

### Languages

21. Each language has a name and a proficiency level: Pre-A1, A1–A2–B1–B2–C1–C2 (CEFR labels), or Native. Languages are used in detailed assessment and preparation; not in Fits you. [implemented — SETTINGS-3]

### Availability

22. The profile states a notice period (number and unit: days, weeks, or months) and an available-from date (DD/MM/YYYY). Preferred timezones are owned by the Geography section, not here. [implemented]

### Experience

23. The profile holds an ordered list of experience entries. Each entry has company name, role title, start (month/year), end (month/year), a current-role flag, and a rich-text summary. Experience is used in detailed assessment and preparation; not in Fits you. [implemented — SETTINGS-3]

### Other requirements

24. The profile has a free-text "Other requirements" section, which also carries dealbreakers. It is used in detailed assessment; Quick ranking does not interpret its text. [implemented]

### Demographics

25. The profile has a demographics section (gender, race, ethnicity). These answers are never used in matching, Fits you, You fit, fit assessment, preparation, or document extraction; nothing is sent automatically. The section is for filling application forms where required. [implemented]

### CV documents

26. The owner uploads CV documents (PDF, DOCX, MD, TXT by default). The body size limit is set in configuration (default 2 MiB). [implemented]

27. Deleting a document removes the stored file, its extracted text, and all proposals that have not been accepted. Facts the owner already accepted stay in the profile; they show that their source was deleted. [implemented — SETTINGS-7]

28. CV analysis (Analyze) starts an extraction job for the document. The extraction uses the provider configured for the `extraction` stage. The job transitions from idle to analyzing and back to idle (with facts stored) or error on completion. Cancel and retry are available for in-progress or failed analyses. [implemented]

29. A document that exceeds the model's context budget is split into chunks; proposals from all chunks are merged and de-duplicated before storage. [implemented — SETTINGS-8 F.6]

30. The number of facts stored from one document analysis is bounded by configuration (default 100). [implemented]

### CV analysis proposals (facts)

31. Each extracted fact has one of five states: proposed (not yet reviewed), profile_match (extracted value already equals the confirmed profile — not shown as a pending decision), accepted (the owner confirmed it, written into the profile), rejected (dismissed), or conflict (a different confirmed value already exists for a single-value field). [implemented — SETTINGS-8 F.4]

32. An extracted value equal to the confirmed profile value (trimmed; case-insensitive for names) is recorded as profile_match and not presented for review. For list fields (skills, languages, target roles), a value already in the list is profile_match. For experience entries, the same company name and either the same title or an overlapping date span is profile_match. [implemented — SETTINGS-8 F.4]

33. Each fact carries an exact verbatim quote from the source document. Facts about work authorization are extracted from statements of citizenship, residency, work permit, or visa. The current location is extracted from address lines. Neither is extracted without a quote. [implemented — SETTINGS-8 F.2]

34. A condition stated as the candidate's requirement is extracted only when a quote from the document supports it as the candidate's own requirement (not a description of a past team or role). [implemented — SETTINGS-8 F.3]

35. The headline (identity.headline) is never proposed from document analysis. [implemented — SETTINGS-8 F.5]

36. Accepting a fact merges its value into the confirmed profile. The fact stays visible in section read views with a provenance note. [implemented]

37. Rejecting a fact marks it as rejected; it is not shown again. [implemented]

38. A conflict (different confirmed value for a single-value field) requires the owner to choose which value to keep. [implemented]

### "Confirmed by you" (from the vacancy view)

"Confirmed by you" is My data: it is shared across all profiles (profiles.md §2, agreed 2026-11-12).

39. When the owner marks ✓ on a non-skill requirement in the vacancy view, the fact "You have: <requirement text>" is added to the "Confirmed by you" list in the profile. When the owner marks ✕, "You do not have: <requirement text>" is added to the same list. [not implemented — INBOX-2 L.5r6, implemented on branch work/INBOX-2]

40. Classification rule (owner, 2026-11-13, translated: "Maybe everything the user has not EXPLICITLY declared as a Skill should be kept as Facts?"): a requirement goes to Skills (+) or "Skills you do not have" (−) only when its name is in the skill vocabulary; a list of such names is split into separate skills, and alternatives ("Go or Rust", "C++ or C") are offered to choose from on + and recorded separately on −. Anything else (activities, phrases, experience statements) goes to "Confirmed by you" as "You have: …" or "You do not have: …". This replaces the earlier rule of the same day that treated any name of at most three words without an -ing form as a skill; it made skills of "Hands-on observability", "background jobs" and "communication". The action buttons on an undecided requirement row are + ("I have this") and − ("I don't have this"); there is no Dismiss for now (ASSESSMENT-4). [not implemented — INBOX-2 (decided 2026-11-13)]

41. The same normalised requirement text (case and spacing ignored) in other vacancies is decided ✓/✕ without asking the owner again. Limitation: vacancies word the same requirement differently, so this exact-text reuse rarely fires; owner, 2026-11-13, translated: "this seems useless, because every vacancy uses its own wording." Matching facts by meaning belongs to the model's matching pass (You fit) when it is on; matching by shared words without a model is a possible later card, not planned. [not implemented — INBOX-2 L.5r6, implemented on branch work/INBOX-2]

42. The "Confirmed by you" list appears in About me next to "Skills you do not have"; each entry is removable. Each fact has "Move to skills"; each skill and each "Skills you do not have" entry has "Move to facts". The owner decides the kind; a moved name that is not in the vocabulary is matched as a skill by whole words. Owner, 2026-11-13, translated: "each fact should have a 'move to skill' button ... the user moves facts to skills and skills to facts himself." Russian originals: INBOX-4 and INBOX-2 card records. [not implemented — INBOX-2; list and removal on branch work/INBOX-2, move buttons decided 2026-11-13]

43. A change to the confirmed-facts list increments the profile revision, which triggers Fits you recalculation. [not implemented — INBOX-2 L.5r6, implemented on branch work/INBOX-2]

### My data and profiles (the agreed multi-profile design)

44. The following are My data — shared across every profile and stored once: identity (name, contacts, photo); confirmed skills and levels; "Skills you do not have"; "Confirmed by you" facts; experience entries; computed facts derived from experience (total years of experience, years in leadership roles, founder flag, current-role flag); languages; current location and work authorization; library of documents and accepted facts; answers for application forms; history of submitted applications. [not implemented — SETTINGS-5 S.1; today all data lives in one profile record]

45. A profile defines one kind of work the owner is looking for. It holds its own name, target roles and industries, compensation, engagement, hours, work modes, geography preferences, dealbreakers, search expressions, and Inbox/Shortlist/rejection decisions. It refers to the shared My data facts; it never holds its own copy. [not implemented — SETTINGS-5 S.1]

46. Within a profile the owner may select which of the shared skills and experience entries apply to that profile; the confirmed fact and its level exist only once. [not implemented — SETTINGS-5 S.1]

47. The active profile sets the context for the whole application. Background work always carries an explicit profile ID and never reads the active profile implicitly; a scoring task started for one profile does not switch to another on a profile change. [not implemented — SETTINGS-5]

48. A change to a profile makes only that profile's results stale. A change to a shared My data fact makes the affected results stale in every profile. [not implemented — SETTINGS-5 S.3]

49. The background worker serves every profile; its model work order follows the active profile's Inbox toggle (Best fit / Newest). [not implemented — SETTINGS-5 S.3]

50. Archiving a profile never removes shared facts, documents or the application history. [not implemented — SETTINGS-5 S.5]

### Scores and decisions per profile

51. Fits you is computed per profile from that profile's goals and the shared facts. Switching profiles reads stored results without calling a model. [not implemented — SETTINGS-5 S.3; today one Fits you score per vacancy]

52. You fit, when it returns (ASSESSMENT-5), is computed per profile. [not implemented — SETTINGS-5 S.3]

53. Inbox, Shortlist, and rejections are per profile: a vacancy rejected in one profile may be shortlisted in another. [not implemented — SETTINGS-5 S.4; today one set of Inbox/Shortlist states]

54. A submitted application is recorded once (at the My data level) with the profile used; it is visible in every profile. [not implemented — SETTINGS-5 S.4]

### Profile switcher

55. The header switcher shows the current profile name. The name is stored in the profile record on the server; a custom name overrides the derived name (the first target role, title-cased, or "Profile 1" when no target role is set). [not implemented — SETTINGS-5 S.5; today the derived name is implemented and a custom name is kept in browser localStorage]

56. The header switcher allows creating, switching, and renaming profiles. No copy or archive in the interface; the server retains an archive operation for later. [not implemented — SETTINGS-5 S.5]

57. Each profile has its own colour. [not implemented — SETTINGS-5 S.1]

### Profile page layout

58. The profile page has two groups in the centre column: "What I am looking for" (profile-specific, in this order: target roles, then a Search section labelled "Words Sifty looks for in listings" with titles/keywords/expressions shown as chips, then compensation and working conditions) and "About me" (shared My data: skills, experience, languages, documents). The difference between groups is made by grouping and order, not by colour (colours carry state meaning in the design system). [not implemented — SETTINGS-5 S.5]

### Migration

59. When SETTINGS-5 is built, the current single profile becomes My data plus the first profile (named by the owner). Shared facts move to My data; goals and search stay with the profile. IDs, revisions, quotes, and history are preserved. Existing triage, shortlist, and rejection states belong to the first profile. A second profile "General engineering" is seeded from the 2026-11-10 search backup at the owner review. [not implemented — SETTINGS-5]

### Derived state and Quick ranking

60. The profile screen shows whether the Quick ranking is up to date (fastRankingCurrent). "Refresh scoring" re-runs local Quick Match derivation from the saved profile without fetching remote sources or calling a model. A separate flag (contextCurrent) shows whether the candidate context is out of date, independent of the ranking state. [implemented]

## 4. Configuration

All profile configuration lives in `runtime.yaml` under the keys listed below. Defaults are applied when a key is absent.

| Key | Default | Purpose |
|---|---|---|
| `profile_max_document_bytes` | 2 MiB | Maximum body size for a document upload request |
| `profile_text_timeout_ms` | 30 000 ms | Timeout for text extraction from a document |
| `profile_extraction_timeout_ms` | 120 000 ms | Timeout for one fact-extraction model call |
| `profile_max_facts` | 100 | Maximum facts stored from one document analysis |
| `profile_document_types` | pdf, docx, md, txt | Accepted document file types |
| `profile_avatar_max_bytes` | 2 MiB | Maximum body size for an avatar upload request |
| `profile_avatar_types` | png, jpg, webp | Accepted avatar file types |

The model stage used for CV analysis is `providers.stages.extraction` in `providers.yaml`. The matching stage used for requirement matching is `providers.stages.matching` (defaults to the extraction provider when not set).

## 5. Questions for the owner

All questions from this draft were answered by the owner on 2026-11-13 (see log.md). There are no open questions for this feature.

## 6. Open cards

- SETTINGS-5 — multi-profile support: My data / Profile split, per-profile search, per-profile scores and decisions, interface switcher, migration; start after INBOX-2 is merged.
- SETTINGS-9 — geography editor shows "not set" for legacy location text (backlog bug).
- INBOX-2 — "Confirmed by you" (L.5r6/L.5r7) implemented on branch work/INBOX-2; merges into main when that card is released.
- SETTINGS-4 — website and social profile URL sources; execution not yet approved; depends on SETTINGS-2 (done). Owner, 2026-11-13: accepted the coordinator's recommendation to keep in backlog and start after SETTINGS-5 is complete.
