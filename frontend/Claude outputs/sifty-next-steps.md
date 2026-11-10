# Sifty — what to do next in code

Last design commit: `71f949a6c0365673f591e55d30c2ba61f2d86133` on `work/DESIGN-3`
(after it, `main.tsx` also got the Sift → Inbox rename and stage labels; both are in the worktree).

**Wait:** the user merges DESIGN-3 with the Apply and freshness changes first. Start code only
after they say from which commit to build.

## 1. Overview tab (approved mockups: overview-a.png, overview-b.png)

- Tabs on a vacancy: **Overview** and **Application**. Application is disabled until the role is
  shortlisted. The "Fit assessment" tab goes away.
- **Left: listing text.**
  - Today (variant A): original text in original order, paragraphs and lists kept. Header
    "Listing text · original order · <descriptionStatus>" + "Refresh listing".
  - When the backend sends sections (variant B): toggle "By section | Original".
    Each section is its own panel (not collapsible): "What you'll do", "What they ask" (recognised),
    "What they offer" (one-line panel with "Not stated" when missing), "Other text" (grey panel,
    not classified, shown in full). Nothing is ever dropped.
- **Right rail.**
  - **Key facts:** Compensation, Work mode, Location, Employment type, Seniority, Posted
    (+ collected date), Expires, Source (site link only). Each fact shows where it came from;
    unknown values show an explicit grey "Not stated".
  - **Original listing** link only once, under the vacancy title (not in the rail).
  - **Quick match:** show the current `matchScore` (big number /100, bucket Strong 75+ /
    Partial 55–74 / Weak / "Not scored" when 0, gradient bar). Status "Explanation pending".
    **No invented per-term contributions** until the backend contract exists.
  - **AI analysis:** small panel: runs after shortlisting, shown in Application.
- LLM analysis lives in **Application**, for shortlisted roles only.

## 2. Backend contracts to agree (with the scoring module)

- Facts: `{value, source: 'listing data' | 'text' | …}` or explicit "not stated".
  JSON-LD already has `baseSalary`, `employmentType`, `responsibilities`, `qualifications`,
  `skills`, `experienceRequirements` — not extracted today (`collect/parse.go buildStructured`).
- Sections: keep HTML structure (today `collect/text.go cleanHTML` flattens it), return
  `[{kind, title, text}]` in original order, `kind = other` for unrecognised text.
- Quick match: `{status, score, explanations[], unknowns[]}`. Today `matchScore` is SQLite FTS
  ranking (`profile/ranking.go FastRankingQuery`), no per-term breakdown.

## 3. Open items handed to the coding agent

- Profiles as an app-level entity (own files, skills, roles, vacancies, scores); header switcher
  "+ Add new profile" and "Delete this profile" are disabled until then; profile name now lives
  in localStorage (`ProfileSwitcher.tsx`), move it server-side.
- `DELETE /api/profile/documents/:id` (UI done; drop open proposals, keep accepted facts).
- Add link / social profile as a document with `sourceUrl` ("+ Add link" disabled).
- `sendable` flag on documents; CV per profile.
- `geography.preferredRegions` (7 codes incl. `middle_east_africa`) — UI sends it.
- Language level "Native" missing in `LANGUAGE_LEVEL_ORDER`.
- Hide empty legacy fields (e.g. "Retained relocation notes").
- Filter fields from listings: salary, work mode, country/region, seniority.

## 4. Known UI bugs

- Collection → Search page is wider than the viewport (chips and Query line cut on the right).

## 5. How to write to the user's computer (lessons)

- Copy files into a **new** folder under `/mnt/user-data/outputs/design3/r<timestamp>/` before
  each `device_commit_files`; reusing a path can write a stale version. Verify with `md5sum` on
  the device afterwards.
- Never run the copy and the commit in parallel.
- Git in the worktree: `git --git-dir=$HOME/mnt/sifty/.git/worktrees/DESIGN-3 --work-tree=.`
  from `$HOME/mnt/sifty/.worktrees/DESIGN-3`; delete permission is needed for git lock files.
