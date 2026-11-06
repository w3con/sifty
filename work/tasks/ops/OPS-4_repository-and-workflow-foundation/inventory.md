# OPS-4 — repository inventory and preservation baseline (step 1.1)

**Recorded:** 2026-11-06
**Worktree:** `/Volumes/SSD/Dev/tknff/sifty/.worktrees/OPS-4` on `work/OPS-4`
**Method:** `git worktree list`, `git status --short`, `git status --ignored --short`, `git submodule status`, directory listings and `du -sh`. Runtime data was inspected by **path name and size only**; no database, config, PDF, log or knowledge-base contents were read, copied or written into the repository. No credentials were opened.

This is a baseline for later relocation and cleanup steps (3.x, 7.2). It records what exists and what must survive; it does not authorize moving or deleting anything.

## 1. Worktrees and branches

| Worktree | Branch | HEAD | Working copy | Local ignored content | Preservation requirement |
| --- | --- | --- | --- | --- | --- |
| `/Volumes/SSD/Dev/tknff/sifty` | `main` | `c0181a1` | clean | runtime/build/dependency artifacts below | Base checkout. `main` is 66 commits ahead of `origin/main` and has not been pushed. Preserve; no push in this task. |
| `.worktrees/DESIGN-1` | `work/DESIGN-1` | `d56435f` | clean | none | Unfinished DESIGN-1 card (6 open steps). Preserve branch and worktree; do not relocate it with source folders. |
| `.worktrees/preview` | `assessment-panel` | `94f51d9` | clean | `app/node_modules/` (≈120 MB) | Preview/assessment-panel work. Preserve; do not delete as a leftover. |
| `.worktrees/OPS-4` | `work/OPS-4` | `c0181a1` | this stage's work | none | This task's worktree. Preserve until the coordinator finishes OPS-4. |

No worktree was added, removed or re-pointed during this stage. `git worktree list` reports exactly these four.

## 2. Task cards

43 card folders exist under `ai/tasks/`; 27 carry a terminal file (`done.md`). 16 are non-terminal, and none carries `cancelled.md`, `rejected.md` or `question.md`.

### Non-terminal cards (must remain reachable)

| Card | Open steps | State / note |
| --- | --- | --- |
| `application-preparation/APP-3_answers-from-kb-and-think-modal` | 1 | Blocked on SET-1's knowledge-base answer shape. |
| `application-preparation/APP-6_answer-memories-in-prepare` | 1 | Founded; real step. |
| `application-preparation/APP-7_draft-revision-check` | 1 | Founded; real step. |
| `assessment/ASM-1_rerun-failed-with-model` | 2 | Deferred by the owner 2026-11-04. |
| `collection/COL-2_learned-template-cache` | 2 | Reserved; plan is a placeholder. |
| `collection/COL-4_collection-progress-events` | 2 | Founded; plan is a placeholder. |
| `collection/COL-5_description-structure` | 2 | Founded; plan is a placeholder. |
| `design/DESIGN-1_design-token-layer` | 6 | Has the `work/DESIGN-1` worktree. |
| `go-migration/GO-5_consolidate-under-sifty` | 2 | Founded; plan is a placeholder. |
| `go-migration/GO-6_self-host-packaging` | 2 | Founded; plan is a placeholder. |
| `go-migration/GO-7_serve-background-timers` | 2 | Founded; short card with real steps. |
| `go-migration/GO-8_cli-command-parity` | 2 | Descoped 2026-11-06; kept as a record. |
| `go-migration/GO-9_mcp-server` | 2 | Founded; approach revised 2026-11-06; plan is a placeholder. |
| `ops/OPS-4_repository-and-workflow-foundation` | 23 | This card. |
| `settings/SET-1_first-time-answer-preparation` | 1 | Blocked on owner questions. |
| `settings/SET-4_status-bar-inline-models` | 3 | Founded; real steps. |

### Stale placeholder lines in completed cards

`collection/COL-1_source-abstraction` and `triage/TRI-1_dismiss-and-block-rules` each have `done.md` and all plan steps closed, but their `description.md` still carries the template line `- [ ] To be defined in plan.md.`. These are not unfinished work. They are template residue for the task-state/navigation work (steps 4.1/4.3) to reconcile; do not treat them as open implementation steps and do not rewrite them during relocation.

## 3. Local artifacts (ignored or untracked in the main checkout)

| Path | Size | Nature | Preservation requirement |
| --- | --- | --- | --- |
| `data/` | 326 MB | Live production data: `store.sqlite` 131 MB, WAL 17 MB, two timestamped SQLite backups 87 MB each, `config/`, `kb/`, two PDFs, server/deploy logs | Never move, delete, commit or copy into tracked files. Take a consistent SQLite backup before any data-layout work (a separate task). Treat contents as personal. |
| `data-staging/` | 1.1 GB | Live staging data: `store.sqlite` 140 MB, WAL 14 MB, ten timestamped backups (~1 GB total), `config/`, `config-backups/` (6 files), PDFs, logs | Same as `data/`. The timestamped backups are cleanup candidates for a later, explicitly approved operation — not for relocation. |
| `.local/backups/claude-code-20261106-140723.tar.gz` | 3.0 MB | Local archive of the retired Claude Code configuration, SHA-256 `40e09c110fd62606f31cda800d320863cde7ecd5e41e4b0a394050b56d69a801` | Keep locally; never commit. |
| `.opencode/node_modules/` | 62 MB | Local OpenCode dependencies | Reproducible from `.opencode/package-lock.json`; not source. |
| `.opencode/package.json`, `.opencode/package-lock.json`, `.opencode/.gitignore` | ≈15 KB | Local OpenCode dependency manifest, ignored by `.opencode/.gitignore` | Leave local; the tracked executor profile is `.opencode/agents/executor.md`. |
| `.worktrees/` | 134 MB | The two worktrees above plus this one | Preserve as in §1. |
| `app/dist/` | 668 KB | Current production frontend bundle | Deployment artifact, not source. Preserve until its environment is redeployed and verified. |
| `app/dist-staging/` | 672 KB | Current staging frontend bundle | As above; staging and production bundles must stay distinct. |
| `app/node_modules/` | 120 MB | Local frontend dependencies | Reproducible from the lockfile; not source. |
| `sifty/web/node_modules/` | 70 MB | Local frontend dependencies | Reproducible from the lockfile; not source. |
| `sifty/workers/playwright/node_modules/` | 18 MB | Local worker dependencies | Reproducible from the lockfile; not source. |
| `bin/sifty`, `bin/sifty.prev` | 20 MB each | Production binary and its rollback copy | Preserve while the production process is live; replace only via the verified deploy path. |
| `bin/sifty-staging`, `bin/sifty-staging.prev` | 20 MB each | Staging binary and its rollback copy | Preserve while staging is live; replace only via the verified staging deploy path. |
| `.work-scratch/` | 0 B | Empty scratch directory | Ignored; no content to preserve. |
| `ai/incidents.md` | 6.3 KB | **Tracked** local incident record | Step 3.3 must untrack it while preserving the local file; do not rewrite history. |
| `.claude/settings.json` | 73 B | Tracked retired Claude Code project setting | Historical adapter; not to be extended. |
| `.DS_Store` files | small | macOS metadata | Ignored; disposable. |

Tracked configuration and runtime-adjacent paths that are **not** ignored: `config/` (providers, candidate, search, network, runtime), `prompts/`, `Makefile`, `bin/` scripts, `.githooks/`. `data/config/` and `data-staging/config/` are local copies and may contain provider settings or credentials; they were not opened.

## 4. Live runtime

| Environment | Port | Process | Binary | Data directory |
| --- | --- | --- | --- | --- |
| Production | 8080 | `sifty` (PID 47274 at inventory time) | `bin/sifty` | `data/` |
| Staging | 8081 | `sifty-staging` (PID 59911 at inventory time) | `bin/sifty-staging` | `data-staging/` |

PIDs are transient. Both servers were listening on `127.0.0.1` at inventory time. No deploy, restart or smoke check was run in this stage.

## 5. Repository wiring: submodule and symlinks

- `.gitmodules` declares `ai/shared` = `git@github.com:w3con/ai.git`, pinned at gitlink `aebbcda835ab708be1b465cfcb566b1994f09b19`.
- `ai/bin`, `ai/harness` and `ai/templates` are tracked symlinks into `ai/shared/`.
- The **main checkout** has `ai/shared` initialized and all three symlinks resolve.
- **All three linked worktrees** (`DESIGN-1`, `preview`, `OPS-4`) report `git submodule status` with a leading `-` and have the three symlinks **dangling**, because the submodule is not initialized in a linked worktree. `bin/work-handover` creates a worktree without initializing submodules; `bin/work-setup` only prints the `git submodule update --init ai/shared` instruction.
- Consequence for this task: do not remove the symlinks or replace them with copies. Any check that depends on `ai/bin`, `ai/harness` or `ai/templates` fails inside a linked worktree until the submodule is initialized. Fresh-clone and hook-verification work (steps 2.3 and 3.2) must account for this.
- `core.hooksPath` is **unset** in the main checkout and in this worktree, so the tracked `.githooks/` are not active as Git hooks by default. `bin/work-setup` sets `core.hooksPath` to the relative `.githooks`.

## 6. Preservation requirements (normative for later steps)

1. **Runtime data is untouchable by relocation.** `data/`, `data-staging/` and their contents stay in place; any later data work needs a consistent backup and a separately approved task. Do not commit, copy into tracked files or expose their contents.
2. **Personal artifacts never enter Git.** That includes the two PDFs, config, knowledge base, logs, `answers/`, `evidence/`, `feedback/` (empty at inventory time), and the `.local` archive.
3. **Unfinished work survives.** Keep `work/DESIGN-1`, `assessment-panel`, all non-terminal cards and their evidence reachable. Do not delete worktrees as "old folders"; only OPS-4's own worktree is removed by the coordinator's finish.
4. **History is preserved.** `main` is 66 commits ahead of `origin/main`; do not push or rewrite. Task identifiers and `done.md` evidence stay attributable.
5. **Deployment artifacts stay paired per environment.** `bin/sifty` + `app/dist` belong to production; `bin/sifty-staging` + `app/dist-staging` belong to staging. Never let a staging deploy overwrite production artifacts, and keep `.prev` rollback copies until the replacement is verified.
6. **Submodule and symlink wiring is preserved.** Do not replace tracked symlinks with copied directories. Initializing `ai/shared` in a linked worktree is the supported way to make them resolve.
7. **Local ignored material is not "leftover code".** Before step 3.5 removes any directory, audit ignored/untracked contents and keep runtime data, dependencies needed for a live server, backups and retained legacy code.
8. **Cleanup is staged.** This stage performs no move, rename, delete, untrack, merge, push or deploy.

## 7. Baseline checks

Recorded verbatim in `log.md` by `bin/work-note` on 2026-11-06:

| Check | Result |
| --- | --- |
| `git status --short` | exit 0, no output (worktree clean at start of stage) |
| `git worktree list` | exit 0, four worktrees as in §1 |
| `python3 bin/work-test` | exit 0, `PASS: all harness checks` |
| `python3 bin/work-plan OPS-4` | exit 0, `plan for OPS-4 is fit: 23 step(s), acceptance criteria present.` |
| `git diff --check` | exit 0, no output (recorded after the inventory edit) |

`bin/work-test` covers compilation, template presence, AGENTS.md shape and command `--help`; it does not exercise lifecycle safety or application behavior. `bin/work-plan` validates plan shape, not owner approval.

## 8. Findings requiring review (not fixed in this stage)

- Dangling `ai/bin`, `ai/harness`, `ai/templates` in every linked worktree; `work-handover` does not initialize the submodule. Relevant to steps 2.3 and 3.2.
- `core.hooksPath` is unset, so tracked project Git hooks are inactive until `bin/work-setup` is run; step 2.3 must verify effective installation rather than printed success.
- Two completed cards carry a stale template open-step line (COL-1, TRI-1); relevant to task-state cleanup (4.1/4.3), not to relocation.
- `ai/incidents.md` is still tracked; step 3.3 must untrack it without deleting the local file or rewriting history.
- `data-staging/` is dominated by ten timestamped SQLite backups (~1 GB). They are not obsolete source directories; if cleanup is ever wanted it needs its own explicit decision.
- The main checkout is 66 commits ahead of `origin/main`. Publication steps must not assume the remote is current.
- `bin/work-note` appends each entry with a blank line at EOF, so `git diff --check` fails on the card log it just wrote (`new blank line at EOF`, exit 2). The log was normalized (trailing blank lines stripped, evidence lines untouched) before the passing check above. This is a harness defect for the coordinator to schedule; it was not fixed in this stage because changing `bin/work-note` is outside the assigned scope.

## 9. Step 3.3 retention requirements for the main checkout (2026-11-06)

The executor's worktree preserves the local files, but a merge into `main` will
remove the tracked paths listed here. **The coordinator must perform these
copies in the main checkout before `bin/work-finish` / the merge.** This section
records requirements only; the executor did not write to the main checkout.

1. **Incident record.** Source: `/Volumes/SSD/Dev/tknff/sifty/ai/incidents.md`
   (tracked at `main` `c0181a1`, SHA-256
   `ea43e12946752a06a49d83d0495d30d99d12d5b6eb7599a2510470add213b2ae`).
   Destination: `/Volumes/SSD/Dev/tknff/sifty/work/incidents.md` (ignored).

   ```sh
   mkdir -p /Volumes/SSD/Dev/tknff/sifty/work
   cp -p /Volumes/SSD/Dev/tknff/sifty/ai/incidents.md \
      /Volumes/SSD/Dev/tknff/sifty/work/incidents.md
   shasum -a 256 /Volumes/SSD/Dev/tknff/sifty/work/incidents.md
   ```

   The destination hash must equal the value above before merging.

2. **Pre-relocation staging bundle.** Source:
   `/Volumes/SSD/Dev/tknff/sifty/sifty/dist-staging/` (20 tracked files,
   manifest SHA-256 `ef6fb24288551ba3420bdc31b2ce6936d903bdee894d16b734d61955e35f3824`,
   computed as `find … -type f | sort | xargs shasum -a 256 | shasum -a 256`).
   Destination: `/Volumes/SSD/Dev/tknff/sifty/.local/retained/sifty-dist-staging-20261106/`
   (ignored local material).

   ```sh
   mkdir -p /Volumes/SSD/Dev/tknff/sifty/.local/retained
   cp -a /Volumes/SSD/Dev/tknff/sifty/sifty/dist-staging \
      /Volumes/SSD/Dev/tknff/sifty/.local/retained/sifty-dist-staging-20261106
   find /Volumes/SSD/Dev/tknff/sifty/.local/retained/sifty-dist-staging-20261106 \
      -type f -print0 | sort -z | xargs -0 shasum -a 256 | shasum -a 256
   ```

   The destination manifest must equal the value above before merging. The
   bundle is generated and rebuildable, but the owner asked that its local bytes
   not be discarded.

3. **No history rewrite.** Both paths stop being tracked in the OPS-4 commit;
   earlier commits retain them. Never delete the local copies or force-remove
   the main checkout's material. `bin/work-finish` also refuses to delete a
   worktree that contains ignored local material, so the worktree copy of
   `work/incidents.md` stays available if the coordinator needs to recover it.

The worktree hashes recorded in the step 3.3 log entry are the same as the main
checkout values above, so the copies are verifiable byte for byte.
