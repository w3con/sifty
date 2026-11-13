# Local data behavior specification

Status: Written from the code of main `d49de9f` on 2026-11-13 by card OPS-11; not yet reviewed by the owner.

## 1. Purpose

The owner's outcome (requirement R-DATA in `docs/MAP.md`): everything Sifty knows about the owner, the vacancies and the owner's decisions lives on the owner's own machine, in one directory the owner can see, copy and restore. The owner can tell which files are theirs (documents, configuration, remembered statements) and which the application produced (database, logs, backups). A release or a failed release never loses that data, and the owner knows exactly which text leaves the machine and when. This specification describes what the code does today, what is only on a branch, what is only agreed, and where the documents disagree with the code. Release tooling (`bin/sift-deploy`, `bin/sift-promote`, `bin/sift-data-refresh`) is described in [AGENTS.md](../../AGENTS.md) and [the environment runtime page](../architecture/preview-and-environments.md); this page only describes what those tools do to the owner's data.

## 2. Where the data lives

One data directory holds all runtime state. The application resolves it from the `--data-dir` flag, then the `SIFTY_DATA_DIR` environment variable, then the default `./data`, and makes the path absolute; it never looks for a git root. `backend/internal/dirs/dirs.go:24-40`. The usage text printed by `sifty help` also mentions a "config" step between the environment variable and the default, which the resolver does not have (see Contradictions). `backend/cmd/sifty/main.go:30-33`.

On start the server creates the data directory, `config/`, `answers/`, `feedback/` and `evidence/` with private permissions (owner-only). `backend/internal/dirs/dirs.go:43-50`, `backend/cmd/sifty/serve.go:52`. Every path the application writes beneath the root goes through a helper that refuses a path escaping the root. `backend/internal/dirs/dirs.go:74-81`. The single exception is the machine-wide model lock file, kept outside the data directory on purpose so that production, staging and review servers share one lock; its place is `SIFTY_MODEL_LOCK`, else `sifty/model.lock` under the user cache directory, else under the system temporary directory. `backend/internal/dirs/dirs.go:83-102`.

What is found in the directory (checked against the production directory on 2026-11-13, listing only):

| Path under the data directory | What it is | Who writes it |
|---|---|---|
| `config/search.yaml`, `runtime.yaml`, `providers.yaml`, `candidate.yaml`, `network.yaml` (optional), `skills.yaml` (optional) | Owner configuration: sources, search terms, port and limits, model providers and stage choices, evidence references | Owner, or the configuration routes of the application |
| `config-backups/` | One timestamped copy of a configuration file per save | The application, on every save |
| `store.sqlite`, `store.sqlite-wal`, `store.sqlite-shm` | The database (vacancies, assessments, profile, drafts, collection history) in write-ahead-log mode | The application |
| `cv.pdf`, `profile.pdf`, `kb/…` and any other file named in `candidate.yaml` | Owner documents, referenced by path relative to the data directory root, not to `evidence/` | Owner; the application only reads them |
| `evidence/profile-documents/<id>/original.<ext>` | Originals of documents uploaded through the profile page | The application, on upload |
| `profile-avatar/<uuid>.<ext>` and `profile-avatar.json` | The profile photo and its metadata | The application, on upload |
| `answers/`, `feedback/` | One markdown file per remembered answer or owner feedback statement | The application, on the owner's action |
| `prod-server.log`, `staging-server.log`, `deploy.log` | Server output and a one-line-per-deploy log | The deploy tooling |
| `store.sqlite.backup.<timestamp>` | Older database snapshots | Not verified which tool wrote them: the current deploy code writes its snapshots elsewhere (section 6); the files on disk are dated 2026-11-05 and 2026-11-06 |

Production is the main checkout's `data/` (port 8080); staging is `data-staging/` (port 8081). Both are ignored by git. `AGENTS.md:122-125`, `.gitignore`. Neither directory is ever inside a task worktree: relative destinations resolve against the main checkout. `AGENTS.md:120`. The binaries, front-end bundles and worker trees are not in the data directory.

At the time of writing production holds an empty `answers/` and `feedback/`, 49 configuration backups, two old database snapshots, a 300 MB database at schema version 22 with 39 application tables plus the full-text index tables (read-only inspection). The staging directory is a copy produced by a data refresh (section 6) and has drifted since.

## 3. What the application writes, and when

| Event | What is written | Where | Reference |
|---|---|---|---|
| Server start | Opens or creates the database, runs the schema migration (section 4), recomputes local match scores from stored facts without any fetch or model call | `store.sqlite` | `backend/internal/store/store.go:65-118`, `backend/cmd/sifty/serve.go:80,142-150` |
| Owner saves a configuration file (editor, source list, search terms, stage choice) | Validated new file, plus a copy of the previous file | `config/`, `config-backups/` | `backend/internal/config/save.go:41-119`, `backend/internal/server/routes.go:688-720` |
| Collection (daily schedule or manual) | Vacancies, sightings, raw captures, run history and per-source health | `store.sqlite` | `backend/internal/background/collect.go:78,184`, `backend/internal/store/migrate.go:102-132` |
| Assessment, model passes, application preparation | Assessments, queue state, extractions, verdicts, drafts, revisions | `store.sqlite` | `docs/architecture/data-directory.md` provenance table; `backend/internal/store/migrate.go:478-521` |
| Owner uploads a profile document | Original file, then its database row, then locally extracted text | `evidence/profile-documents/<id>/`, `store.sqlite` | `backend/internal/profile/documents.go:200-258` |
| Owner uploads a profile photo | Photo and a metadata file, written through a temporary file and a rename | `profile-avatar/`, `profile-avatar.json` | `backend/internal/profile/avatar.go:199-228` |
| Owner saves feedback on a vacancy | A markdown file first, then an index row in the database, then the vacancy is queued for reassessment | `feedback/`, `store.sqlite` | `backend/internal/server/routes.go:857-867` |
| Owner remembers an application answer | A markdown file only; no database row is written | `answers/` | `backend/internal/server/routes.go:986-994` |

Answer and feedback files are the authoritative record. Each file is readable prose and carries a base64 JSON copy of the same record in an HTML comment marker (`SIFT_ANSWER`, `SIFT_FEEDBACK`). A file is created with exclusive creation, so an existing file is never overwritten, and with owner-only permissions. `backend/internal/records/records.go:117-161,169-220`. The application reads them back by scanning the directory and decoding the markers; a file without a marker is skipped, a file with a broken marker makes the read fail. `backend/internal/records/records.go:231-263`. There is no transaction across the file and the database row: the file is written first, so a failed index write leaves a file without an index row and never the reverse. `backend/internal/server/routes.go:857-864`.

Uploaded profile documents are stored under a fixed name (`original.<ext>`) in a per-document folder; the client's filename is metadata only. Size and type limits come from `runtime.yaml`. `backend/internal/profile/documents.go:197-217`. Text extraction from the stored file is local and uses no model. `backend/internal/profile/documents.go:254-256`. Files named in `candidate.yaml` are imported as database records pointing at the file, the file itself is not copied, and an absolute or escaping path is recorded as missing rather than read. `backend/internal/profile/documents.go:585-650`.

Configuration is never created for the owner. Startup fails with a read error when any of `search.yaml`, `runtime.yaml`, `providers.yaml` or `candidate.yaml` is missing; `network.yaml` and `skills.yaml` are optional (a built-in skill vocabulary is the fallback). `backend/internal/config/config.go:431-480`, `backend/internal/config/skills.go:60-80`. Template files for a new data directory exist in the repository's `config/` folder, but no code copies them; how a new installation obtains them is a manual step (not verified whether any document describes it).

## 4. Schema migrations and their guarantees

The database schema version is stored in SQLite's `user_version`. Main targets version 22. `backend/internal/store/migrate.go:8`. The comment at the top of `store.go` also says 22; two other places say otherwise (see Contradictions).

Rules, as implemented:

- Migration runs automatically every time the server opens the database, including a copied one. `backend/internal/store/store.go:86`. A database already at the target version is opened without any schema change. `backend/internal/store/migrate.go:301-303`.
- Steps are applied in order, each only when the stored version is below it, and the version is raised after each step. `backend/internal/store/migrate.go:305-501`. A fresh database is brought to the target in one pass. `backend/internal/store/migrate.go:306-309`.
- Versions 15 to 22 are additive: new tables or new columns with defaults, no existing row touched. Version 16 records the profile revision behind each score; 17 adds the outcome reason; 18 adds a publication basis that stays empty for old rows instead of inventing one; 19 to 22 add the match input, result, requirement-extraction and two-score tables. `backend/internal/store/migrate.go:404-501`.
- Version 14 is the one destructive step: company block rules are copied into stop words and all source block rules are deleted, because source switching moved to the `enabled` flag in `search.yaml`. `backend/internal/store/migrate.go:385-402`.
- Every step is written to be repeatable (`CREATE … IF NOT EXISTS`, add a column only if it is missing), so a start interrupted between steps picks up again. The steps are not wrapped in one transaction; what a crash in the middle of one step leaves is not verified. `backend/internal/store/migrate.go:575-599`.
- Nothing refuses a database that is newer than the binary: if the stored version is at or above the target, the binary opens it and goes on. `backend/internal/store/migrate.go:301-303`. There is no downgrade step. The protection against running an older binary on a newer database is the deploy rollback, which restores the pre-deploy database snapshot together with the old binary (section 6).
- Deleting a vacancy relies on foreign keys (enabled on every connection) to remove its match rows, and removes the rest explicitly. `backend/internal/store/store.go:66-70`, `backend/internal/store/jobs.go:917-927`.

Other state moves with the schema only through code, not through the schema version: the full-text index is maintained by explicit code calls, not by database triggers, and is rebuilt on migration 13 from stored vacancies. `backend/internal/store/migrate.go:222-232,364-383`.

## 5. Deletion and what is kept

- **Dismissing or rejecting a vacancy** moves it between stages; the row, its history and its reason stay. `backend/internal/store/jobs.go:800-808` (workflow history is written with every move).
- **Permanent delete** is available only outside Triage, only with an explicit confirmation in the request, and is refused while collection or an assessment on that vacancy is running. It removes the vacancy, its aliases, sightings, captures, assessments, queue entry, feedback index rows, history, application drafts, revisions, questionnaire snapshots and preparation operations, drops the vacancy from the discovery cache, and never reuses its id. `backend/internal/server/routes.go:183-206`, `backend/internal/store/jobs.go:827-991`.
- **Tombstones.** Permanent delete first writes one hash per canonical listing address (original and apply address, plus every sighting address) into `deleted_listings`. A later collection of the same listing is skipped silently; the tombstone is lifted only by an upsert called with the restore flag, and no caller in main passes it. `backend/internal/store/jobs.go:862-915`, `backend/internal/store/jobs.go:381-433`, `backend/internal/background/collect.go:78,184`.
- **Merged duplicates.** When collection finds two rows for one vacancy it moves the history, drafts and feedback onto one row, deletes the other and records the old id as an alias. `backend/internal/store/jobs.go:509-545`.
- **Block rules and stop words** are rows the owner can delete one by one. `backend/internal/server/routes.go:330,385`.
- **Profile documents.** Deleting one removes its extracted text, its unaccepted proposals and, if the application stored the original, the whole per-document folder. Facts the owner already accepted stay in the profile and are reported as coming from a deleted file. A file the owner keeps elsewhere in the data directory and referenced from `candidate.yaml` is forgotten but never deleted. `backend/internal/profile/documents.go:418-467`. Profile revisions are an append-only history; no code deletes them. `backend/internal/store/migrate.go:281-291`.
- **Profile photo.** Replacing or removing it deletes the previous file. `backend/internal/profile/avatar.go:225-251`.
- **Answers and feedback files.** No code path deletes or edits them. Deleting a vacancy deletes the feedback index rows but leaves the feedback files, which keep carrying the vacancy id. `backend/internal/store/jobs.go:919-924`.
- **Retention.** The only timed retention in the code is a purge of applied vacancies 30 days after the application date. The function exists, but nothing calls it, so nothing is purged automatically. `backend/internal/store/core.go:54-55`, `backend/internal/store/jobs.go:1087-1129`. No other table (captures, collection events, configuration backups, profile revisions, discovery cache) has any size or age limit in the code that was read.

## 6. Backups, snapshots and restore

The application itself writes exactly one kind of backup: the copy of the previous configuration file made on each save, named `<file>-<UTC date>-<UTC time>.yaml` in `config-backups/` with owner-only permissions. The copy is made with errors ignored, so a failed backup does not stop the save; two saves in the same second share one name and the later copy replaces the earlier one; the new file is written in place, not through a temporary file; the folder is never pruned. `backend/internal/config/save.go:104-115`. A configuration save also rejects inline secrets, so provider keys remain environment references and are never in a backup. `backend/internal/config/save.go:99-102`. Restoring a configuration is manual: the owner copies a backup over the file in `config/`, then saves or restarts so that the running server reloads it (not verified that the interface offers it; the search of the front-end sources for the backups folder found nothing).

There is no backup, export or restore command in the application: the Go binary offers `version`, `serve`, `help` and `selfcheck` only, and the older TypeScript `backup` command (a consistent database copy) was not ported; the owner confirmed he does not use a command line. `backend/cmd/sifty/main.go:36-64`, `work/tasks/go-migration/GO-8_cli-command-parity/description.md`.

What exists comes from the deploy tooling (details in [AGENTS.md](../../AGENTS.md), "Rollback" and "Staging data"):

| Snapshot | When it is taken | What it contains | Where it is | Restore |
|---|---|---|---|---|
| Deploy recovery point | Every deploy of an environment: artifacts before the stop, then the database after the server is verifiably stopped and before the new files are published | Previous binary, front-end bundle, worker tree and a consistent database copy made with SQLite's backup mechanism (so the write-ahead log is included) | `.local/deploy/<env>/recovery/<id>/` in the main checkout, outside the data directory and git-ignored | Automatic on a failed deploy; by hand `bin/sift-deploy --recover DIR`; for a release, the recovery request of `bin/sift-promote` (`backend`-independent tooling) |
| Staging refresh backup | Every data refresh of staging | The whole previous staging data (database and every non-runtime top-level entry) | `data-staging/.refresh-backup/<timestamp>/` | Automatic on a failed refresh; otherwise manual copy |
| Old database snapshots | Not verified | Whole database copies | `data/store.sqlite.backup.<timestamp>` | Manual copy over `store.sqlite` after removing the `-wal` and `-shm` files |
| Configuration backups | Every configuration save | One file each | `config-backups/` | Manual copy |

References: database copy `bin/_deploy_lib.py:512-530`; restore removes the old sidecar files and refuses to treat a missing snapshot as "no database" `bin/_deploy_lib.py:542-556`; artifact copy `bin/_deploy_lib.py:1203-1217`; order of stop, snapshot, publish, start `bin/_deploy_lib.py:1587-1660`; recovery directory path `bin/_deploy_lib.py:1529`; staging refresh backup `bin/_deploy_lib.py:1296-1306,1334-1411`.

Properties the owner can rely on: the database snapshot of a deploy is consistent, because it is taken from a stopped server by SQLite's own backup call, and it precedes the first start of a new binary, so a schema migration caused by the new binary is covered. A copy of the files while the server runs is not a safe backup. `docs/architecture/data-directory.md` (store section).

Properties the owner cannot rely on: the deploy snapshot copies only the database, not `config/`, `kb/`, `answers/`, `feedback/`, `evidence/` or the profile photo, so a deploy rollback does not restore owner files; no code limits the number of recovery points (production holds ten, about 3 GB, at the time of writing); the application does not check for a restorable backup before migrating.

**Staging copy of production data.** A staging data refresh is the only way production data reaches staging. It stops the recorded staging server, backs up the old staging data, takes a consistent copy of the production database, then replaces every top-level entry of the staging directory with the production entry of the same name, except the database sidecars, logs, old snapshots and refresh folders. That includes `config/`, so staging then runs with production's configuration, and it includes the owner's CV, profile documents, knowledge base and photo; staging therefore holds the owner's personal documents as soon as it is refreshed. `bin/_deploy_lib.py:1271-1286,1373-1400`. A normal deploy never copies data. `AGENTS.md:150`.

## 7. What leaves the machine, and what never does

The server listens on `127.0.0.1` only. `backend/cmd/sifty/serve.go:170`. There is no account, no telemetry call and no upload of the data directory in the code that was read (not verified exhaustively for the front-end and the worker scripts).

Leaves the machine:

- **Collection fetches.** Requests to the job sources named in `search.yaml` (page and feed fetches through the fetch client, browser-driven fetches through the worker, the scraping library for the job-board source). They carry the search address, not owner data. `backend/internal/fetch/client.go:170`, `backend/internal/background/collect.go`. Per-source network profiles come from `network.yaml` and proxy addresses come from environment variables. `config/network.yaml`.
- **Profile and remote-profile fetches.** The context refresh fetches the profile addresses listed in `candidate.yaml` over HTTP. `backend/internal/candidate/context.go:217-218`. Authenticated scraping is not performed. `backend/internal/profile/documents.go:290-293`.
- **Model calls.** The providers are chosen per stage in `providers.yaml`. Provider type `opencode` and `claude` start the matching command-line tool as a subprocess with a restricted environment and send the prompt through it, so the prompt text goes wherever that tool's own login sends it (a remote provider, under the owner's subscription). Provider type `ollama` and `compatible` call an HTTP endpoint and the code refuses any endpoint that is not loopback. A key for such an endpoint is read from the environment variable the configuration names. `backend/internal/assess/infer.go:55-65,106-116,153-162,183-197,251-257`. The prompt for each stage contains vacancy text together with the owner's evidence: assessment and application preparation use the assembled candidate context, profile fact extraction sends the document text, and the two model passes send the vacancy requirements and selected profile passages. `backend/internal/background/assessment.go:188`, `backend/internal/profile/analysis.go:400`, `backend/internal/vacancyai/vacancyai.go:4`. The exact content of each prompt is not verified here; the prompt templates are in `config/prompts` and embedded in the binary. A model call is serialised by a machine-wide lock. `backend/internal/dirs/dirs.go:83-102`. [Quick Match](quick-match.md) section 14.7 describes the guard for paid providers.

Never leaves the machine by any code path read: the database file, the configuration files, the answer and feedback files, the original uploaded documents and the photo as files; the application has no export or upload route. Text extracted from them can leave only as part of a model prompt through a `claude` or `opencode` provider. Local startup rescoring and local text extraction call no model and no network. `backend/cmd/sifty/serve.go:142-150`, `backend/internal/profile/documents.go:254-256`.

## 8. Moving data to another machine

Not supported as a feature: the application has no export, import or machine-transfer function and no document describes a procedure. What the code allows is that the data directory is self-contained, with paths inside it recorded relative to its root (documents, photo, answer and feedback files), so copying the whole directory with the server stopped, then starting a binary with `--data-dir` pointing at the copy, is expected to work; the database copy must come from a stopped server or from SQLite's backup call, with the `-wal` and `-shm` files not copied separately. This procedure is inferred from the code and has not been run. `backend/internal/profile/documents.go:224-246,589-595`, `backend/internal/profile/avatar.go:199-211`. Two things are not portable: the machine-wide model lock lives outside the directory (it is recreated on demand), and any provider tool login or environment variable (keys, proxy addresses) belongs to the machine, not to the data. The packaging card for self-hosting (section 10) does not change this.

## 9. Implemented on a branch, not released

| Item | Card | What the branch does to data |
|---|---|---|
| Migrations 23 to 27 (the card text names 24 to 27; the branch also adds 23): a column with the structured vacancy description, a table keeping the raw page or feed item of each vacancy compressed as received (never returned by the interface), a re-fetch mark, a table of description sections, and a counter seeded from the highest vacancy id. All additive. | INBOX-2 (`.worktrees/INBOX-2/work/tasks/inbox/INBOX-2_listing-markup/plan.md`, `backend/internal/store/migrate.go` in that worktree, target version 27) | The database grows with the stored originals; a local backfill builds the structured form from stored text without touching the existing plain description; a release step removes 40 category-page rows from each database and requires the standard pre-release backup to exist first |

Section 4's statement that nothing refuses a newer database matters for this branch: after its release the database is at version 27 and an older binary would open it unchecked. The deploy rollback restores the snapshot taken before the release (section 6).

## 10. Agreed, not implemented

| Item | Card | Owner's words or agreement |
|---|---|---|
| Separate the data into four classes (owner-provided evidence, owner-derived context and memory, vacancies and assessments, applications) with a dedicated evidence root; audit of every live reference, file-versus-database consistency check, backup and restore tested on scratch data, and owner approval before any live data is moved | OPS-8 (backlog); the target is described, marked proposed, in `docs/architecture/data-directory.md` | The card records the plan only; it contains no quoted owner words about the layout. It states that no live data is moved or split before the owner approves the migration plan |
| All filesystem paths set in configuration instead of in code, with defaults for a public clone, worktrees and both environments | OPS-7 (backlog) | «все должно задаваться где-то в конфигах», «все пути» |
| Single profile becomes "my data" plus a first profile; shared facts move to my data, goals and search stay with the profile; ids, revisions, quotes and history are preserved; the second profile is seeded from the 2026-11-10 search backup | SETTINGS-5 (`docs/specs/profiles.md` section 7) | Agreed on 2026-11-12 per the specification; the card's re-plan records the owner's «да» to seeding from the search backup |
| Self-host packaging: container image and compose file that creates the application's own data directory; documented quick start and configuration reference; optional local model service | GO-6 (founded, no steps yet) | Decisions recorded in the card; the card quotes no owner words |
| Remove the dead server record that blocks a release, build from a worktree must not write production artifacts, production started by hand must be recorded | OPS-9 (backlog) | «Заводи.» (the owner's reply to the list of defects) |
| History cleanup of the repository | OPS-6 (backlog) | «отдельная задача - надо посмотреть на git log и видимо как-то compact его - там жопа какая-то» (concerns repository history, not user data) |
| A command-line backup | GO-8 (descoped) | The owner does not operate the backend from a command line or cron, so none is planned |

## 11. Known limits and defects

- No application-level backup of the whole data directory, and no export, import or restore for the owner; deploy snapshots cover the database only. No card owns a whole-directory backup other than the design steps of OPS-8 (steps 1.3 and 1.4), which prepare a plan and do not build it.
- Recovery points grow without limit (about 3 GB for ten production deploys), configuration backups and staging refresh backups likewise. No card.
- Configuration backup failures are ignored, same-second saves collide, and the new configuration file is written in place rather than atomically. No card.
- The 30-day purge of applied vacancies exists but is not called. No card (decision needed: wire it or remove it).
- Migrations are not run in one transaction and the binary does not refuse a newer database. No card.
- Answer and feedback files are never deleted or edited, are not removed when their vacancy is deleted, and an answer has no database index row. The map's description of the file-first records as "backed up" is true only of the staging refresh and manual copying. No card (OPS-8 step 1.2 audits their consistency).
- Four tests on main fail independently of any card, among them the migration and schema tests, because their version constants are stale. OPS-9 item 5.
- The old `store.sqlite.backup.*` files in production and staging are untracked leftovers from earlier tooling; nothing cleans or restores from them. No card.

## 12. Contradictions

Left for the coordinator to decide; none is resolved here.

- `docs/architecture/data-directory.md` says the store target is version 14, the code says 22 (`backend/internal/store/migrate.go:8`); the comment above the base schema says the schema runs "through user_version 16" (`backend/internal/store/migrate.go:10-11`).
- The same page says `evidence/` is empty and lists no photo files; the production directory has `evidence/profile-documents/` (written by `backend/internal/profile/documents.go:197`) and `profile-avatar/` with `profile-avatar.json` (`backend/internal/profile/avatar.go:40,199`), neither in the layout listing. The page also attributes `store.sqlite.backup.<timestamp>` snapshots to the deploy scripts; the current deploy code writes snapshots to `.local/deploy/<env>/recovery/` (`bin/_deploy_lib.py:1529,1618`).
- The same page lists the answer route as the writer of an answer index; the route writes only the file (`backend/internal/server/routes.go:986-994`), and the `feedback` table is the only index.
- `backend/README.md:27` says Sifty writes answer and feedback files from templates on first run; the code writes them only on an owner action (and the `selfcheck` command), never at start (`backend/internal/records/records.go:130,169`, `backend/cmd/sifty/serve.go`).
- `backend/cmd/sifty/main.go:30-33` says the data directory comes from the flag, then the variable, then "config", then `./data`; `backend/internal/dirs/dirs.go:24-40` has no configuration step.
- `AGENTS.md:152` says the live processes still use the old layout and the first production transition still needs `--adopt-legacy`; the adoption record in `.local/deploy/prod/adoption.json` shows the adoption completed on 2026-11-06 and the production state file shows later normal deploys, the latest on 2026-11-13. `docs/MAP.md` R-REL status line says the same as `AGENTS.md`.
- `docs/MAP.md` R-DATA status says "limited; provenance and recovery are documented"; restore of the owner's files after a failed deploy is not covered by any snapshot (section 6).
- The INBOX-2 card text names migrations 24 to 27; the branch also contains 23.

## 13. Related specifications

- [Profiles](profiles.md): the owner's profile data, its migration into "my data" plus profiles, document-derived facts.
- [Quick Match](quick-match.md): stored match results, the two model passes, staleness and the model lock.
- [Vacancy view](vacancy-view.md): the stored description and where the displayed values come from.
- [Data directory and provenance](../architecture/data-directory.md): current layout page, to be corrected per the contradictions above.
- [Environment and preview runtime](../architecture/preview-and-environments.md) and [AGENTS.md](../../AGENTS.md): deploy, rollback and staging refresh tooling.
- Cards: [OPS-7](../../work/tasks/ops/OPS-7_configurable-paths/description.md), [OPS-8](../../work/tasks/ops/OPS-8_data-layout-migration/description.md), [OPS-9](../../work/tasks/ops/OPS-9_finish-process-defects/description.md), [GO-6](../../work/tasks/go-migration/GO-6_self-host-packaging/description.md).
