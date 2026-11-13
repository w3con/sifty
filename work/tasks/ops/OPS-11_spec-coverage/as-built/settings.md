# Settings and profile behavior specification

Written from the code of main `d49de9f` on 2026-11-13 by card OPS-11; not yet reviewed by the owner.

## 1. Purpose

This document specifies what the Settings pages and the Profile screen do today, as implemented in the Go backend and the React frontend. It is a description of the running system, not a design document. Where the agreed design differs from what is implemented, those items are listed under "Agreed, not implemented". The multi-profile design agreed and documented in `docs/specs/profiles.md` is not implemented; this spec describes the single active profile that the system stores and uses today.

---

## 2. Settings navigation

The Settings overlay opens from the header "Settings" button (`frontend/main.tsx:142`). It replaces the main workspace view and is itself a two-column layout: a left navigation rail and a right content panel. Pressing Escape when no input or textarea is focused closes the overlay and returns to the workspace (`frontend/Settings.tsx:41`).

The navigation rail is organised into three labelled groups, each with one or more items (`frontend/Settings.tsx:24–28`):

- **Collection** — Sources, Search
- **AI** — Models, Assessment queue
- **System** — Advanced, Candidate context

The old URL segment `"settings"` is normalised to `"sources"` on entry (`frontend/Settings.tsx:30`).

---

## 3. Sources panel

**Panel id:** `sources`

Rendered by `<Collection section="sources" …/>` (`frontend/Settings.tsx:53`). The panel lists the job-board sources configured in `search.yaml`. Behaviour of this panel is documented in the Collection component (see `frontend/Collection.tsx`; not detailed here). Changes in the Sources panel call `onRulesChanged`, which increments the main epoch counter and triggers a job-list refresh (`frontend/main.tsx:144`).

---

## 4. Search panel

**Panel id:** `search`

Rendered by `<Collection section="search" …/>` (`frontend/Settings.tsx:53`). Behaviour of search expressions, exclusion terms and related collection settings is documented in the Collection component. Changes call `onRulesChanged` as above.

---

## 5. Models panel

**Panel id:** `models`

Loaded by `GET /api/providers` and `GET /api/stage-providers` on panel mount (`frontend/Settings.tsx:42`).

### 5.1 Stage models

Four processing stages, each assigned one configured provider (`frontend/Settings.tsx:58–59`):

| Stage | Purpose | Note |
|---|---|---|
| `assessment` | Detailed fit analysis of shortlisted roles | Required in providers.yaml |
| `preparation` | Drafts application-form answers | Optional (empty means unset) |
| `extraction` | Fallback listing parser when model extraction is on | Required in providers.yaml |
| `matching` | Requirement vs. profile comparison pass | Optional; defaults to extraction provider (`backend/internal/config/config.go:379–384`) |

The UI shows a `<select>` for each stage populated from `stageProviders.providers`. The selected value is the provider name. Changing the selection immediately calls `PUT /api/stage-providers` with `{stage, provider}` and updates the local state from the response (`frontend/Settings.tsx:59`).

`PUT /api/stage-providers` validates that the named stage is one of `assessment`, `preparation`, `extraction`, performs YAML node surgery to change only the stage scalar (preserving comments and field order), calls `config.Save`, then reloads the holder (`backend/internal/server/routes.go:757–802`). The matching stage is not shown in the stage UI; it is set only by editing providers.yaml in the Advanced panel (not verified in the frontend).

### 5.2 Provider list

`GET /api/providers` returns all configured providers with a live status string (`backend/internal/server/routes.go:643–655`). The panel renders each provider with:
- A coloured availability indicator: green (`ok`) when status does not contain "unavailable", "not running" or "error"; amber (`warn`) when it contains "not logged" or "warning"; red (`down`) otherwise (`frontend/Settings.tsx:31`).
- Provider name and model identifier.
- A pill label: Available / Needs attention / Unavailable.
- A "Check" button that calls `POST /api/providers/{name}/check` and displays the response time and raw result, or an error (`frontend/Settings.tsx:64`).

The footer note states that the check sends a small synthetic prompt, never owner data, and that no provider is substituted silently when one fails (`frontend/Settings.tsx:68`).

---

## 6. Assessment queue panel

**Panel id:** `queue`

Data comes from the `queue` prop passed from the main App, which polls `GET /api/queue` every 2.5 seconds (`frontend/main.tsx:69`).

### 6.1 Assessment toggle

A toggle switch labelled "Assess vacancies" calls `PUT /api/queue/assessment` with `{enabled: !assessmentOn}` (`frontend/main.tsx:73`). The server stores the flag in the database via `Store.SetQueueEnabled` and returns the current queue state (`backend/internal/server/routes.go:532–548`). The toggle is disabled while a save is in progress. The status caption reads:
- "Saving…" while saving.
- "Off · current inference finishing" when off and `counts.running > 0`.
- "Off" when off and no running items.
- "On" when on (`frontend/Settings.tsx:76`).

### 6.2 Queue counts

Five counters displayed in a row (`frontend/Settings.tsx:78`): running, pending, failed (highlighted when non-zero), missing-description (highlighted when non-zero), done (current cycle).

### 6.3 Queue table

All rows from the assessment queue are shown (`frontend/Settings.tsx:82`): role id (and error if any), status tag, provider, elapsed time, and retry count. When the queue is empty, a placeholder message is shown. The footer note says a role can be retried via "Assess now" in its Application tab.

### 6.4 Runtime details

An expandable `<details>` section shows the raw queue runtime JSON, parsed and pretty-printed (`frontend/Settings.tsx:85`).

### 6.5 Stage summary in queue panel

A footnote in the queue panel shows the current assessment and preparation stage labels, with a link to the Models panel to change them (`frontend/Settings.tsx:79`).

---

## 7. Advanced panel

**Panel id:** `advanced`

This panel is the raw YAML editor for all configuration files. It is loaded from `GET /api/config/{name}` on mount and whenever the selected file tab changes (`frontend/Settings.tsx:43`).

### 7.1 File tabs

A segmented control selects among: `search`, `providers`, `network`, `candidate`, `runtime` (`frontend/Settings.tsx:91`). The displayed label below the tabs reads `config/{name}.yaml` (`frontend/Settings.tsx:92`).

### 7.2 YAML textarea

The raw YAML content is editable in a full-width textarea. The `dirty` flag is set on any edit and cleared on successful save (`frontend/Settings.tsx:93–94`).

### 7.3 Save

Clicking "Save configuration" calls `PUT /api/config/{name}` with `{content}` (`frontend/Settings.tsx:94`). On success:
- The notice "Valid YAML · saved, backup in .local/" is shown (note: the actual backup directory is `config-backups/` relative to the data root, per `backend/internal/config/save.go:107–110`; the displayed message names `.local/` which does not match the implementation — see Contradictions).
- The dirty indicator clears.

The save handler on the server (`backend/internal/server/routes.go:688–726`) does the following:
1. Calls `config.Save`, which parses YAML, validates schema, checks for inline secrets, writes a timestamped backup, and writes the new file (`backend/internal/config/save.go:41–119`).
2. For the `runtime` file: detects a port change before reload, reloads the holder, recalculates Quick Match locally, and may set `restartRequired` or `quickMatchWarning` in the response.
3. For all other files: reloads the holder silently.

### 7.4 Validation

`config.Save` validates the incoming YAML against the target file's schema before writing:
- `search`: version must be 1, at least one site required, names must be unique, URLs must be HTTP(S) (`backend/internal/config/save.go:122–143`).
- `runtime`: version must be 1, port 1024–65535, `quick_match` block validated when present (`backend/internal/config/save.go:145–196`).
- `providers`: version must be 1, provider type in `{opencode, claude, ollama, compatible}`, model non-empty, stage providers named and known (`backend/internal/config/save.go:198–233`).
- `candidate`: version must be 1, `cache_hours` positive (`backend/internal/config/save.go:235–243`).
- `network`: no required fields; only parsing is checked (`backend/internal/config/save.go:94–97`).

Inline secrets are rejected: strings matching `sk-[A-Za-z0-9]{12}`, `Bearer <token>`, or URLs containing embedded credentials in the user:password@ form cause a save error (`backend/internal/config/save.go:22–25,100–102`).

### 7.5 Configuration backups

Before each successful save, the existing file content is written to `<data-root>/config-backups/<name>-<timestamp>.yaml` (UTC, `20060102-150405` format) (`backend/internal/config/save.go:104–112`). The backup directory is created with mode 0700 if absent. If the file cannot be read (first-time save), no backup is written and the save continues.

### 7.6 Configuration files

| File | Required | Purpose |
|---|---|---|
| `search.yaml` | Yes | Job-board sources, search titles, keywords, expressions, preferences |
| `runtime.yaml` | Yes | Port, collection limits, assessment limits, model extraction flag, Quick Match config, profile upload limits |
| `providers.yaml` | Yes | Provider definitions and stage assignments |
| `candidate.yaml` | Yes | Candidate evidence sources for the legacy context pipeline |
| `network.yaml` | Optional | Per-source network profiles and proxy settings |

Templates are in `config/*.yaml` in the repository. Configuration files are stored in the data directory (`./data/config/` for production, `./data-staging/config/` for staging).

### 7.7 Embedded defaults

Certain keys have embedded defaults that are used when the key is absent (zero value) (`backend/internal/config/config.go`):

- `runtime.profile_max_document_bytes`: 2 MiB (`config.go:278–283`)
- `runtime.profile_text_timeout_ms`: 30 000 ms (`config.go:292–297`)
- `runtime.profile_extraction_timeout_ms`: 120 000 ms (`config.go:300–305`)
- `runtime.profile_max_facts`: 100 (`config.go:308–313`)
- `runtime.profile_document_types`: `[pdf, docx, md, txt]` (`config.go:316–321`)
- `runtime.profile_avatar_max_bytes`: 2 MiB (`config.go:324–329`)
- `runtime.profile_avatar_types`: `[png, jpg, webp]` (`config.go:338–343`)
- Quick Match defaults when `quick_match.version == 0`: weights role=0.25, compensation=0.20, geography=0.15, conditions=0.10; required_weight=2, preferred_weight=1; Wellfound location `needs_corroboration` (`config.go:250–265`)
- Quick Match model pass defaults: max_requirements=30, max_attempts=3, temperature=0.1, evidence_passages=3 (`config.go:189–196`)

Port changes in `runtime.yaml` require a server restart; the response includes `restartRequired` when the port changed (`backend/internal/server/routes.go:707–710`).

### 7.8 Live reload

After a successful `PUT /api/config/{name}`, the server reloads the configuration holder atomically via `Holder.Reload` (`backend/internal/config/holder.go:53–60`). The new configuration is used for all subsequent requests in the same process. The browser does not need to reload; changes to rules or provider selection take effect in the next run or analysis tick.

---

## 8. Candidate context panel

**Panel id:** `context`

Loaded by `GET /api/context` on panel mount (`frontend/Settings.tsx:42`).

### 8.1 Display

When a context record exists, the panel shows (`frontend/Settings.tsx:102–105`):
- Version and creation timestamp.
- Current preferences as a JSON block (derived from the active profile when one exists, otherwise from `search.yaml` preferences).
- A list of contradictions and missing facts to review.
- Collapsible source snapshots for each evidence source, showing reference, retrieval time, category, scope, error (if any), SHA-256 hash, and the raw source text.

When no context is built, a placeholder is shown.

### 8.2 Refresh

The "Refresh sources" button calls `POST /api/context` with `{refreshRemote: true}` (`frontend/Settings.tsx:100`). This triggers `candidate.RefreshContextForProfile`, which re-fetches remote evidence sources configured in `candidate.yaml`, updates the stored context, and recomputes Quick Match locally (`backend/internal/server/routes.go:596–636`).

---

## 9. Configuration and model selection via API

The API endpoints that the Settings panels use directly:

| Endpoint | Panel | Action |
|---|---|---|
| `GET /api/config/{name}` | Advanced | Read raw YAML |
| `PUT /api/config/{name}` | Advanced | Validate, backup, write, reload |
| `GET /api/providers` | Models | List providers with status |
| `POST /api/providers/{name}/check` | Models | Smoke-check one provider |
| `GET /api/stage-providers` | Models, Queue | Current stage assignments |
| `PUT /api/stage-providers` | Models | Change one stage's provider |
| `GET /api/queue` | Queue | Queue state, counts, rows, runtime |
| `PUT /api/queue/assessment` | Queue | Enable/disable assessment |
| `GET /api/context` | Candidate context | Read context record |
| `POST /api/context` | Candidate context | Rebuild context from sources |
| `GET /api/quick-match/order` | (App load) | Read background worker order |
| `PUT /api/quick-match/order` | (Sort toggle in Inbox) | Save order to runtime.yaml |

---

## 10. Profile screen

The Profile screen is opened from the header "Profile" icon (`frontend/main.tsx:122, 142`) and from Quick Match "unknown" links in the vacancy detail panel. It replaces the workspace view. Opening the profile screen does not start any network request until the screen mounts.

The profile screen has two top-level containers: one for the owner's uploaded files (column 1), and one for the confirmed profile data (column 2), and one for the AI-proposed facts (column 3) (`frontend/profile/ProfileView.tsx:355–386`). The three-column layout collapses to a narrower two-column layout when no facts are proposed or in conflict.

### 10.1 Profile identity and avatar

The profile header shows (`frontend/profile/ProfileView.tsx:205–251`):
- An avatar button: shows the uploaded photo, or the owner's initials from the display name, or a user icon when no name is set.
- Display name and legal name, editable inline. Legal name is used only where an application form asks for it.
- A derived subtitle: current-role title and company from experience, plus current location from geography.
- Photo status note.
- A "Remove photo" button when a photo is set.
- A "Close" button.

**Avatar upload**: clicking the avatar button opens a file picker. The accepted file types come from `GET /api/profile/capabilities` (`avatarTypes`). The file is base64-encoded and sent via `POST /api/profile/avatar` (`backend/internal/server/profile_api.go:193–230`). The server checks the body size limit (`EffectiveProfileAvatarUploadBodyBytes`), decodes the base64, and stores the avatar file. The avatar is served at `GET /api/profile/avatar` with `Cache-Control: no-store`.

**Avatar delete**: clicking "Remove photo" calls `DELETE /api/profile/avatar` (`backend/internal/server/profile_api.go:232–239`).

Profile name displayed in the tab / switcher is derived from the first non-empty target role in the profile, falling back to "Profile 1" (`frontend/profile/ProfileView.tsx:158–160`).

### 10.2 Derived state indicator

A banner below the header shows the state of the Quick ranking derivation (`frontend/profile/ProfileView.tsx:324–340`):
- "Quick ranking up to date" with a green dot when `profile.fastRankingCurrent` is true.
- A "Refresh scoring" button otherwise, which calls `POST /api/profile/derived/refresh`. The refresh re-runs local Quick Match derivation from the saved profile without fetching remote sources or calling a model (`backend/internal/server/profile_api.go:140–153`).
- A separate warning when `profile.contextCurrent` is false (candidate context is out of date, independent of quick ranking).

### 10.3 Profile sections

The profile has ten editable sections. Each section has a read view and an edit view controlled by an Edit/Save/Cancel header button (`frontend/profile/ProfileView.tsx:176–199`). Section saves call `PUT /api/profile/sections/{section}` with `{value: <section JSON>}`. The server merges the section into the current stored profile inside one transaction (`backend/internal/server/profile_api.go:71–89`). Section ids, labels, and their role in matching:

| Section id | Label | Used in quick ranking | Used in assessment and preparation |
|---|---|---|---|
| `goals` | Target roles and industries | Yes (target roles are ranking search signals) | Yes |
| `skills` | Skills | Yes (skill names are ranking search signals) | Yes |
| `compensation` | Compensation | No | Yes |
| `work` | Engagement, hours and work modes | No | Yes |
| `geography` | Geography and authorization | No | Yes |
| `languages` | Languages | No | Yes |
| `availability` | Availability | No | Yes |
| `experience` | Experience | No | Yes |
| `other` | Other requirements | No | Yes |
| `demographics` | Application form answers | Never | Never (not used in matching, assessment, preparation or document extraction) |

**Goals section** (`profile-goals`): target roles (free-text tags) and target industries (free-text tags) (`frontend/profile/ProfileView.tsx:496–509`).

**Skills section** (`profile-skills`): a list of skills, each with a name and a level (from capabilities `skill_levels`). Skill names are quick-ranking search signals; levels are used only in detailed assessment and preparation. The section also displays "Skills you do not have", which are marked from vacancy requirements using the cross button in the vacancy view, and suppress those requirements from future offers (`frontend/profile/ProfileView.tsx:438–466`).

**Compensation section** (`profile-compensation`): minimum and desired cash amounts, currency, tax basis (gross/net), period (month/year), payment method (crypto accepted/only/not accepted), equity interest flag, and a day-rate sub-section for contracts (`frontend/profile/ProfileView.tsx:511–559`). Compensation is not part of the quick score.

**Work section** (`profile-work`): engagement types (employment, contract, and legacy employee/contractor retained for compatibility), working time mode (full-time, part-time) with optional weekly hours for part-time, and work modes (remote, hybrid, on-site) (`frontend/profile/ProfileView.tsx:562–602`).

**Geography section** (`profile-geography`): current location (structured country and city), preferred regions (checkbox group), preferred work locations (structured country/city list), work authorization by country (country + status), willingness to relocate (tri-state), timezones (IANA ids), desired overlap hours (`frontend/profile/ProfileView.tsx:608–671`). Legacy free-text fields are retained but not active once structured fields are present.

**Languages section** (`profile-languages`): a list of languages, each with a name and a proficiency level (CEFR-style from capabilities `language_levels`; "Pre-A1" is an allowed beginner label). Autocomplete for language names is provided via a datalist (`frontend/profile/ProfileView.tsx:686–705`). Used in detailed assessment and preparation; not in quick ranking.

**Availability section** (`profile-availability`): notice period as a numeric value plus a unit (days/weeks/months from capabilities `notice_units`), and a start date in DD/MM/YYYY format. Timezones are owned by the Geography section, not here. When a structured notice value and unit are set, any legacy `noticePeriod` wording is retained but not shown as the active notice period. When the structured notice is deliberately cleared (the `noticeSet` flag), the legacy wording is also not reactivated (`frontend/profile/ProfileView.tsx:718–740`). Used in detailed assessment and preparation; not in quick ranking.

**Experience section** (`profile-experience`): ordered list of roles with company, title, start (month/year), end (month/year), current-role flag, and rich-text summary. Used in detailed assessment and preparation; not in quick ranking (`frontend/profile/ProfileView.tsx:468–494`).

**Other requirements section** (`profile-other`): a rich-text field for other job requirements. Previously stored dealbreakers and work mode limits appear here until replaced. Used in detailed assessment and preparation; quick ranking does not interpret this text. The section id is `other-requirements` (`frontend/profile/ProfileView.tsx:742–755`).

**Demographics section** (`profile-demographics`): gender (with self-describe option), race (checkboxes with prefer-not-to-say mutual exclusion), and ethnicity. The section description states explicitly that these answers are never used in matching, quick ranking, fit assessment, preparation or document extraction, and nothing is sent automatically (`frontend/profile/ProfileView.tsx:398–436`).

### 10.4 Absent skills

Skills the owner does not have are added from the vacancy detail view (via `POST /api/profile/absent-skills`), not from the profile editor. They appear in the Skills section read view with a remove button. Removing one calls `PUT /api/profile/sections/absent-skills` with the filtered list (the skill to be removed is excluded) — the same section endpoint used for all sections (`frontend/profile/useProfile.ts:199–212`, `frontend/profile/api.ts:18–19`). A requirement for an absent skill counts as not met and is not offered again.

### 10.5 Documents (column 1)

`GET /api/profile/documents` returns the list of uploaded documents. The panel shows each document with its filename, state, and analysis summary.

**Upload**: a file picker sends the file as `{filename, contentBase64}` or `{filename, text}` to `POST /api/profile/documents`. The server enforces the body size limit (`EffectiveProfileUploadBodyBytes`). Accepted types come from `GET /api/profile/capabilities` (`documentTypes`: pdf, docx, md, txt by default) (`backend/internal/server/profile_api.go:273–312`).

**Delete**: `DELETE /api/profile/documents/{id}` removes the file, its text, and unaccepted proposals. Accepted facts remain and show that their source was deleted (`backend/internal/server/profile_api.go:349–358`).

**CV analysis (Analyze)**: `POST /api/profile/documents/{id}/analyze` starts an extraction job for the document. The extraction uses the configured extraction provider (or falls back when model extraction is off — not verified). The document state transitions from `idle` to `analyzing` and back to `idle` (with facts stored) or `error` on completion (`backend/internal/server/profile_api.go:360–367`).

**Cancel and retry**: `POST /api/profile/documents/{id}/cancel` and `POST /api/profile/documents/{id}/retry` are available for in-progress or failed analyses (`backend/internal/server/profile_api.go:369–385`).

Document states are defined by `profile.DocumentStates()` (not verified in detail here).

### 10.6 CV analysis proposals and accept/reject (column 3)

`GET /api/profile/facts` returns all extracted proposals (facts) for the active documents. `GET /api/profile/documents/{id}/facts` scopes to one document.

Each fact has a state: `proposed`, `accepted`, `rejected`, or `conflict`. The column 3 panel shows proposed and conflicted facts. For each fact (`backend/internal/server/profile_api.go:390–457`):

- **Accept**: `POST /api/profile/facts/{id}/accept` with `{expectedRevision}`. The fact's value is merged into the confirmed profile. Returns the updated profile and fact.
- **Reject**: `POST /api/profile/facts/{id}/reject`. The fact is marked rejected and not shown again.
- **Resolve**: `POST /api/profile/facts/{id}/resolve` with `{expectedRevision, choice}`. Used when a proposed fact conflicts with an existing profile value; the choice selects which value to keep.

Accepted facts remain visible in the section read views with a provenance note (`frontend/profile/ProfileView.tsx:115–122`).

The maximum number of facts stored from one document analysis is bounded by `EffectiveProfileMaxFacts()` (default 100) (`backend/internal/config/config.go:308–314`).

---

## 11. Profile switcher and the status bar

### 11.1 Profile switcher

A `<ProfileSwitcher>` component is rendered in the app header next to the Settings button (`frontend/main.tsx:142`). Clicking it opens a dropdown menu with:
- The current profile name (custom name from `localStorage['sifty.profile.name']`, or the first target role title-cased, or "Profile 1").
- An inline rename form.
- A disabled "Add new profile" button with tooltip "Several profiles need backend support".

Clicking the profile name in the dropdown opens the profile screen (`frontend/profile/ProfileSwitcher.tsx:63`). The first profile colour from `PROFILE_COLORS` (`#3db328`) is always applied since only one profile exists.

### 11.2 Status bar

The CSS for a bottom status bar (classes `queue-status`, `queue-bar`, `assessment-control`, `switch-track`) is imported from `frontend/queue-control.css` and `frontend/style.css` and exists in the stylesheet. However, no React component in the current `frontend/` source renders these classes. The status bar was present in the legacy TypeScript application (`legacy/app/`) and its CSS was carried forward, but the React port does not render it. Card SET-4 ("Taller status bar with inline model selection", status: founded) describes the planned addition of a bottom bar showing live assessment and preparation model names with clickable in-place model selection. This feature is agreed but not yet implemented in the React frontend.

The app header contains the main workflow navigation tabs (Inbox, Shortlist, Archive with counts) and the Brand logo but these are not part of the Settings or Profile scope.

---

## 12. Configuration keys edited in the interface versus by hand

| Key / group | Editable in UI | Location |
|---|---|---|
| Search expressions | Yes (Search panel via Collection) | `search.yaml` |
| Sources (sites) | Yes (Sources panel via Collection) | `search.yaml` |
| Stage providers (assessment, preparation, extraction) | Yes (Models panel) | `providers.yaml` |
| Assessment on/off | Yes (Queue panel toggle) | Database (queue state) |
| Background model order | Yes (Inbox sort toggle, reflected in runtime.yaml) | `runtime.yaml` via `PUT /api/quick-match/order` |
| All config files | Yes (Advanced panel raw YAML editor) | respective `.yaml` file |
| Provider API keys | No — must use env variable references; inline secrets refused | `providers.yaml` via Advanced or by hand |
| Port | No effective change — requires restart after edit | `runtime.yaml` |
| Quick Match weights | No (by hand only via Advanced) | `runtime.yaml` |
| network.yaml profiles | No (by hand only via Advanced) | `network.yaml` |

---

## 13. Implemented on a branch, not released

The INBOX-2 plan (`work/tasks/inbox/INBOX-2_listing-markup/plan.md` on the `.worktrees/INBOX-2` branch) adds a `config/vacancy-view.yaml` file for vacancy display configuration. This file and the "Confirmed by you" vacancy field are on that branch and are not in main. They are not described in this spec.

---

## 14. Agreed, not implemented

**Multi-profile support** (docs/specs/profiles.md): the agreed design stores multiple profile entities, each with its own name, colour, files, and scoring. Today the system stores exactly one profile per data directory. The profile-delete button in the footer is disabled with the label "Your only profile can't be deleted" (`frontend/profile/ProfileView.tsx:381–384`), confirming that the multi-profile delete path is structurally anticipated but not activated.

The `ProfileSwitcher` component (`frontend/profile/ProfileSwitcher.tsx`) confirms today's single-profile state: "Add new profile" is disabled with tooltip "Several profiles need backend support". The profile is named after its first target role (derived from `GET /api/profile`), or "Profile 1" when no target role is set. A custom name is stored in `localStorage` under key `sifty.profile.name` and takes precedence over the derived name. The header button opens a dropdown menu with the current profile name, a rename inline form, and the disabled add button (`frontend/profile/ProfileSwitcher.tsx:25–74`).

**Status bar with inline model selection** (card SET-4, status: founded): a bottom bar showing the live assessment and preparation model names, with clickable in-place model selection without opening Settings. The extraction stage would remain in Settings only. The CSS for this bar is already in the stylesheet (see section 11.2).

**Other agreed but unread cards**: SETTINGS-4, SETTINGS-5, SETTINGS-6, SETTINGS-9 (not yet read; any behaviour they record may be missing from this spec — see Known limits).

---

## 15. Known limits and defects

- Document states (`DocumentStates()`) and fact states (`FactStates()`) returned by the capabilities endpoint were not read from the profile package source; their full set is not listed here.
- Cards SETTINGS-4, SETTINGS-5, SETTINGS-6, SETTINGS-9, SET-1, OPS-7, SETTINGS-7, SETTINGS-8 were not read in detail; any agreed or completed behaviour recorded there may be missing from this spec.
- The `config/vacancy-view.yaml` template (INBOX-2 branch) was not read per the brief instruction.
- The actual main HEAD at the time of reading was `6529ff7` (three commits ahead of the `d49de9f` reference in the status line, which is what the brief specified as the reading baseline).

---

## 16. Contradictions

**Backup directory name**: the Advanced panel save notice reads "Valid YAML · saved, backup in .local/" (`frontend/Settings.tsx:94`). The actual backup directory is `<data-root>/config-backups/` per `backend/internal/config/save.go:107–110`. The displayed path `.local/` does not exist as a backup directory.

**Matching stage in UI**: the providers.yaml `stages.matching` key is documented in `backend/internal/config/config.go:369–376` and is a supported optional field, but the Models panel's stage dropdown only shows `assessment`, `preparation`, and `extraction` (`frontend/Settings.tsx:58`). The matching stage can only be set by editing providers.yaml directly in the Advanced panel.

---

## 17. Related specifications

- `docs/specs/profiles.md` — agreed multi-profile design (not implemented)
- `docs/specs/quick-match.md` — Quick Match scoring rules, model passes, and configuration
- `docs/specs/vacancy-view.md` — vacancy display configuration (references config/vacancy-view.yaml on INBOX-2 branch)
