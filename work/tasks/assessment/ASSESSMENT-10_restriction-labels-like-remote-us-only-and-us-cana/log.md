
## 2026-11-13 — coordinator check of the executor hand-back (N.1–N.2)

Executor `executor-sonnet-4-6` (aa85224a3fafdac17) reported N.1 (`b0d1b9f`) and N.2 (`800e423`) on `work/ASSESSMENT-10`, both ticked. Checked by the coordinator: diff `b6e7c58..800e423` (13 files); `resolveRestrictionLabel` tries the full label first, then paren-aware decomposition and configured separators (letter separators at word boundaries); wrapper prefixes, trailing qualifiers and separators live in `backend/internal/config/regions.default.yaml` with `quick_match` override; the hard-coded `strippedForms` list is gone; targeted test `TestResolveRestrictionLabelForms` covers the nine labels and a vocabulary name containing "and"; spec quick-match §4.1 updated. Probe result reported: 1934 vacancies, ?→✕ 6 (1153, 1175, 1196, 1201, 1279, 1294), ?→✓ 0, 1136 unchanged.

Not accepted yet: the probe baseline is "old configuration on new code", so changes caused by the code itself (vacancy 17, any over-splitting) are invisible. Correction sent to the same executor: true before/after against the base tree `b6e7c58` with a per-vacancy verdict diff; show where the new rule fields enter the rules identity (stored results re-evaluated without a model call) or bump the evaluator version; record N.2 in comments.md.

## 2026-11-13 — coordinator acceptance

Executor returned the corrected N.2 (`36db7cf`): true before/after on a fresh production copy (base tree `b6e7c58` versus the branch, one line per vacancy): 8 changed — 17, 1153, 1175, 1196, 1201, 1279, 1294 unknown → not covered; 1262 "Remote (Global)" unknown → not applicable (worldwide; correct); 1119, 1281, 1136 unchanged. Rules identity covers the new fields (`matchcontract/identity.go:8` hashes the whole `Rules`; `matchsvc/service.go:298`), so stored results re-evaluate at startup without a model call; no version bump. Coordinator ran `go build ./...` and the tests of quickmatch, matchsvc, config, matchcontract on the branch: pass; data probes skip without their environment variables. Accepted technically; report in `acceptance.md`, limits in `attention.md`. Release waits for the owner's «да»; main must be merged into the branch first.

## Owner, 2026-11-13: release

Owner: «Да, выпускай.» Release by the same executor (aa85224a3fafdac17): merge main into the branch, mark the spec released, `bin/work-finish` with release; coordinator checks production after.

## 2026-11-13 — released; coordinator check of production

Executor report: main merged into the branch (`fb43ae8`), spec marked released (`71c3a80`), `bin/work-finish` exit 0, squash commit on main `105e82e`, production `71c3a802fe79-20261113T022044.178759Z`. Coordinator checked on port 8080: health 200; `/api/version` prints that version; vacancy 17 now carries `remote_restricted_not_covered`; 1136 still does; 1262 has no restriction line (worldwide). The executor stashed the untracked `legacy/app/shoot-L57.mjs` to let work-finish run and restored it; the stash list holds no leftover entry. Worktree `.worktrees/ASSESSMENT-10` kept (holds a production copy in scratch/data); remove on the owner's word.

## Worktree removed, 2026-11-13

Owner: «да и да, удали». The coordinator removed `.worktrees/ASSESSMENT-10` together with its production copy in `scratch/data` (290 MB); it held no uncommitted changes. Branch `work/ASSESSMENT-10` is kept. In the same step ten untracked one-off scripts in `legacy/app/` (shoot-*, nav-explore*, check-row-buttons*, explore-ui, find-skill-comma) were deleted from the main checkout; no code referenced them.
