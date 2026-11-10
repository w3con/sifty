# ASM-6 J.2 — main publication evidence

Published 2026-11-10T13:45Z after coordinator J.1 technical acceptance of `63f7d47`.
This is integration evidence only; it does not claim card completion or production release.
Actual J.2 mechanics elapsed: 13:43Z–13:46Z (approximately 3 minutes), excluding coordinator waiting; no production, deploy, push or broad test run.

## Result

- Main before: `4156d24a6c9ac1b50d7cf2556f2e03aa598f2b9c`
- Main after: `58af43263dc97b574ec2d4bc147cc50c990197d3`
- Seven Conventional Commits, one per completed card, no checkpoint replay and no separate DESIGN-3 initial commit:

| Card | Commit | Subject | Files |
|---|---|---|---|
| SETTINGS-1 | `564f334` | `feat(SETTINGS-1): add user profile workflow and fields` | 80 |
| SETTINGS-2 | `7f80bbf` | `feat(SETTINGS-2): add profile workflow and document fields` | 23 |
| SETTINGS-3 | `c827cee` | `feat(SETTINGS-3): add rich profile fields and reference data` | 23 |
| APP-8 | `a3ead8b` | `feat(APP-8): record expired application outcome` | 10 |
| COLLECTION-1 | `3ee6260` | `feat(COLLECTION-1): retain publication date and age` | 31 |
| DESIGN-3 | `5bedae3` | `feat(DESIGN-3): add reusable design system and finished layout` | 35 |
| ASM-6 | `58af432` | `feat(ASM-6): add deterministic Quick Match` | 50 |

DESIGN-3 is one completed-card commit: the initial avatar/backend slice (`7f5c0a3..fe31cc3`) and the finished frontend slice (`2b8a1d5..2f6c026`) were both staged before that commit. APP-8 and COLLECTION-1 were applied against SETTINGS-3 with three-way reconciliation, before the completed DESIGN-3 tree, so the final tree still includes all four.

Exact touched path sets per card are in `main-integration-inventory.md`; `git show --name-only <SHA>` on each commit reproduces the published scope.

## Method

- Prepared branch `work/ASM-6-main-integration` in a temporary worktree from `4156d24`.
- Applied exact per-card deltas from the recorded source commits over `backend/`, `config/` and `frontend/`; ASM-6 also added `docs/specs/quick-match.md`.
- Excluded `work/**`, `AGENTS.md`, `work/WORKFLOW.md`, `work/ROADMAP.md`, `docs/MAP.md` and other process/navigation docs from application commits. Root copies remain local for J.3 reconciliation.
- Fast-forwarded the real main checkout only after the temporary commit tree had exact application parity with the accepted source. No ref-only publication, stash, reset, stage-all or source-tree replacement was used.

## Final parity

`git diff --quiet 63f7d47 -- backend frontend config docs/specs` in the main checkout: **OK**.
Main `HEAD` and main disk `backend/`, `frontend/`, `config/` and `docs/specs/` therefore equal the accepted J.1 source, apart from intentionally deferred navigation/task-card text.

## Root work preservation

Recovery directory (ignored, retained): `.local/integration-recovery/20261110T134309Z/`.

- Archived all 150 dirty/untracked paths from 53 status entries, plus the index and a status/hash manifest.
- 139 unrelated paths remain byte-for-byte unchanged and still local; hashes verified against the manifest.
- 11 overlapping paths were logically superseded by accepted newer application files. Originals are both in the recovery copy and under `superseded/`; root hashes are compared with accepted source in `supersession.tsv`.
- Root `frontend/main.tsx` was an older pre-Profile copy, not an independent newer feature edit; the accepted source adds Profile/Design/Quick Match and supersedes it. Other overlapping app files matched their earlier accepted SETTINGS-1/Design snapshots.
- `docs/specs/quick-match.md` already matched the accepted source byte-for-byte and is now committed by ASM-6.
- No task card, standing instruction, backlog, Claude output, runtime data or original work branch/worktree was replaced or removed.

Durable copies of the manifests: `evidence/main-integration-recovery.tsv` and `evidence/main-integration-supersession.tsv`.

## Deferred to J.3

Latest canonical card records remain on `work/ASM-6` and the original card worktrees. Main's local card files were preserved rather than overwritten because several root descriptions contain later owner evidence than the source branch (for example DESIGN-3's 10 MiB avatar ceiling and SETTINGS-3's continent requirement). J.3 owns navigation/spec/status and card-record reconciliation, plus the 8083 shutdown and final acceptance report. This J.2 does not claim those records are reconciled in main history.

## Check

`bash .local/asm6-j2-check.sh` (worktree-local): exit 0. It asserts main SHA, seven commits, no non-application paths in the publication delta, exact source parity, no remaining dirty overlap on published application paths, and the 150/11/139 recovery result. Verbatim output is in `log.md`.
