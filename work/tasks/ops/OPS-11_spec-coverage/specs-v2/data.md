# Local data — specification

Status: owner decisions of 2026-11-13 applied (OPS-11); rule tags show the implementation state.

## 1. Purpose

The owner's data lives on the owner's machine, in one directory the owner can inspect, copy and restore without help from the application. All vacancies, assessments, profile evidence, decisions and application drafts are there. A release or a failed release never loses data the owner provided. The owner can tell which files they wrote and which the application produced. Only two classes of information leave the machine: requests to the job sources the owner configured, and prompts to the model providers the owner configured.

## 2. Scope

This specification covers the data directory location and structure, what the application writes and when, database schema migration, deletion and retention, what leaves the machine, and moving the data to another machine.

Neighboring specifications: [collection.md](collection.md) covers how vacancies are fetched from sources; [quick-match.md](quick-match.md) covers model provider configuration and prompt structure; [profiles.md](profiles.md) covers profile data and its planned separation into shared owner data and per-profile data (SETTINGS-5); [triage.md](triage.md) covers vacancy retention after dismissal and application. Release and deploy mechanics, rollback and staging data refresh are in [AGENTS.md](../../AGENTS.md) and not repeated here.

## 3. Rules

### Location and structure

1. All runtime state lives in one data directory, created with owner-only permissions on startup. One machine-wide model-call lock file is kept outside the data directory so that production and staging share it; it is created on demand. [implemented — `backend/internal/dirs/dirs.go:24-50,83-102`]

2. The application refuses to write any file whose resolved path falls outside the data directory root. [implemented — `backend/internal/dirs/dirs.go:74-81`]

3. Owner-provided files (configuration, documents, the knowledge base) and application-produced files (database, logs, backups, uploaded originals) are separated into identifiable locations within the data directory, so the owner can tell at a glance which files are theirs. [not implemented — OPS-8]

4. All filesystem paths used by the application can be overridden through configuration, so changing an environment or moving a directory requires editing configuration rather than code. [not implemented — OPS-7]

### What the application writes

5. The data directory contains: configuration files the owner can edit (`config/search.yaml`, `runtime.yaml`, `providers.yaml`, `candidate.yaml`; optionally `network.yaml` and `skills.yaml`), saved configuration backups (`config-backups/`), the SQLite database (`store.sqlite`) with its WAL sidecars, owner documents referenced from `candidate.yaml`, uploaded profile documents (`evidence/profile-documents/<id>/original.<ext>`), the profile photo (`profile-avatar/`), remembered answer files (`answers/`), owner feedback files (`feedback/`), and server and deploy logs. [implemented]

6. A configuration save writes the new file, then a timestamped backup of the previous file in `config-backups/`. A backup failure does not stop the save. Provider keys are never written to backup files; only the environment-variable reference is saved, and a save that contains a plain key is refused. Configuration backups are never pruned automatically. [implemented — `backend/internal/config/save.go:41-119`]

7. Answer and feedback records are written as human-readable markdown files first, then indexed in the database. Because the file is created with exclusive creation before the index row is written, a crash between the two steps leaves a file without an index row and never the reverse; the file is the authoritative record. An existing answer file is never overwritten. [implemented — `backend/internal/records/records.go:117-161`, `backend/internal/server/routes.go:857-864`]

8. Uploaded profile documents are stored under a stable name (`original.<ext>`) in a per-document folder inside `evidence/profile-documents/`; the original filename is kept as metadata only. Document text is extracted locally without calling a model or any network service. [implemented — `backend/internal/profile/documents.go:197-258`]

9. The application fails at startup with a read error when any required configuration file (`search.yaml`, `runtime.yaml`, `providers.yaml`, `candidate.yaml`) is absent. Optional files (`network.yaml`, `skills.yaml`) fall back to built-in defaults when absent. [implemented — `backend/internal/config/config.go:431-480`]

10. A new data directory is prepared from the repository template files in `config/` by copying them manually, following a documented procedure. No application command is needed. Owner, 2026-11-13: accepted the coordinator's recommendation for a documented manual procedure. [not implemented — no card; template files exist in the repository but no procedure is documented]

### Database schema migration

11. The database schema is migrated automatically every time the application opens the database, including an opened copy or a staging refresh copy. A database already at the target version is opened without any schema change. [implemented — `backend/internal/store/store.go:86`, `backend/internal/store/migrate.go:301-303`]

12. All migrations from version 15 onward are additive: new tables or new columns with defaults, no existing row changed. Version 14 is the one destructive step, migrating company block rules to stop words. Each migration step is written to be repeatable so an interrupted run can resume at the next step. [implemented — `backend/internal/store/migrate.go:305-501`]

13. A binary older than the stored schema version opens the database without a check or refusal. The protection against running an older binary on a newer database is a deploy rollback that restores the pre-deploy database snapshot together with the old binary, before any new migration has run. See [AGENTS.md](../../AGENTS.md) for the rollback mechanics. [implemented — `backend/internal/store/migrate.go:301-303`]

### Deletion and retention

14. Dismissing or rejecting a vacancy keeps its row, its history and the reason; only a permanent delete removes it. A permanent delete is available only outside Triage, requires explicit confirmation in the request, and is refused while collection or assessment is running on that vacancy. [implemented — `backend/internal/server/routes.go:183-206`, `backend/internal/store/jobs.go:827-991`]

15. A permanently deleted vacancy leaves a tombstone so that collection never re-collects it from the same listing address. No code path in the current release lifts a tombstone; a deleted vacancy stays deleted. [implemented — `backend/internal/store/jobs.go:862-915`]

16. Answer and feedback files are never deleted or edited by the application. Deleting a vacancy removes its feedback index rows but leaves the feedback files on disk; those files still carry the vacancy id and are readable. [implemented — `backend/internal/store/jobs.go:919-924`]

17. Automatic retention limits on applied vacancies are covered in [triage.md](triage.md).

### Backup and recovery

18. Each deploy takes a consistent SQLite snapshot of the production database after the server is verifiably stopped and before the new binary starts; this snapshot is used for automatic rollback on a failed deploy. See [AGENTS.md](../../AGENTS.md) for the deploy and rollback mechanics. [implemented]

19. The deploy snapshot covers the database and all owner files (`config/`, `answers/`, `feedback/`, `evidence/`, the knowledge base, the profile photo), so a deploy rollback restores the complete state before the deploy. Owner, 2026-11-13: accepted the coordinator's recommendation. [not implemented — no card; today the snapshot covers only the database; a rollback does not restore changes to owner files made after the previous deploy]

20. Staging holds a copy of production data only after an explicit data refresh; an ordinary deploy never copies production data to staging. A data refresh copies the production database and owner files, including `config/`, `evidence/`, `answers/`, `feedback/`, the knowledge base and the profile photo, so staging holds the owner's personal documents once refreshed. See [AGENTS.md](../../AGENTS.md) for the refresh command. [implemented]

21. The owner has no application command or supported procedure to back up the whole data directory (database plus owner files together), export it or restore from a whole-directory backup. [not implemented — OPS-8]

### What leaves the machine

22. The application server accepts connections only from the local machine; it binds to `127.0.0.1`. There is no telemetry and no upload of the data directory contents. [implemented — `backend/cmd/sifty/serve.go:170`]

23. Collection fetches go to the job-source addresses listed in `search.yaml`; they carry the search query or URL, not owner data. [implemented — `backend/internal/fetch/client.go:170`, `backend/internal/background/collect.go`]

24. Model prompts for assessment and application preparation include the vacancy text and assembled owner evidence (profile facts, goals, preferences). For `opencode` and `claude` providers, these prompts reach the provider's remote service under the owner's subscription. For `ollama` and `compatible` providers, the application refuses any endpoint address that is not loopback. [implemented — `backend/internal/assess/infer.go:55-65,106-116,153-162,183-197,251-257`]

25. The database file, configuration files, answer and feedback files, original uploaded documents and the profile photo never leave the machine through any application code path. Owner text can reach a remote model provider only as part of a model prompt under rule 24. [implemented]

### Portability

26. Copying the whole data directory to another machine while the server is stopped, then starting the application with `--data-dir` pointing at the copy, is the supported procedure for moving to another machine. Paths to documents and uploaded files are stored relative to the data directory root. The machine-wide model lock and provider credentials are not part of the data directory and must be configured on the new machine separately. Owner, 2026-11-13: accepted the coordinator's recommendation for a documented manual procedure. [not implemented — no card; the procedure is not documented and has not been tested end to end]

27. The owner can run a single command to get the complete application running on another machine with its own data directory, using a container image and docker compose setup. [not implemented — GO-6]

## 4. Configuration

| Setting | Resolution order and default |
|---|---|
| Data directory root | `--data-dir` flag, then `SIFTY_DATA_DIR` environment variable, then `./data` relative to the working directory; always made absolute |
| Model-call lock file | `SIFTY_MODEL_LOCK` environment variable, then `sifty/model.lock` under the user cache directory, then `sifty/model.lock` under the system temporary directory; created on demand |
| Required configuration files | `config/search.yaml`, `config/runtime.yaml`, `config/providers.yaml`, `config/candidate.yaml`; edited directly or through the application settings screens |
| Optional configuration files | `config/network.yaml` for network profiles and proxy addresses; `config/skills.yaml` for a custom skill vocabulary; both fall back to built-in defaults when absent |

No other filesystem paths used by the application are configurable; they are code constants (OPS-7 is the card for changing this).

## 5. Questions for the owner

All questions from this draft were answered by the owner on 2026-11-13 (see log.md). There are no open questions for this feature.

## 6. Open cards

- OPS-7 — centralizes all filesystem paths in configuration rather than code constants.
- OPS-8 — audits live data references, defines four-class ownership, designs consistent backup and restore with a tested rollback, and migrates live data only after owner approval.
- GO-6 — container image and docker compose setup for self-hosting, so a single command starts the complete application on any machine.
