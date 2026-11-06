# Development workflow

Roles: the owner defines product scope and approves completion; the coordinator prepares work, reviews code and accepts technical results; the executor implements the assigned stage.

## Sources and approval

[AGENTS.md](../AGENTS.md) contains mandatory operating rules. This document describes their application and the target workflow. [ROADMAP.md](ROADMAP.md) holds the current resume point and task order. Each task's own folder holds its canonical scope, plan, approval and evidence.

OpenCode is the primary agent tool. Shared, reusable rules belong in `/Volumes/SSD/Dev/ai`; project commands, architecture, model selection and release settings belong in this repository. Extracting and connecting the shared rules is separate work, not an implemented integration. Do not build further Claude Code adapters or compatibility features. Existing configuration has been backed up locally; its presence does not select the executor or authorize its use.

The owner approved the coordinator's full foundation plan, including four task states, new lifecycle automation and cleanup of obsolete directories after verified relocation, in [the foundation card](tasks/ops/OPS-4_repository-and-workflow-foundation/description.md) and [its plan](tasks/ops/OPS-4_repository-and-workflow-foundation/plan.md). Approval authorizes staged execution; the changes remain unimplemented until verified.

## Capture and preparation

A new bug report or idea is captured without starting implementation or switching away from the active task. Preserve the owner's words, reproduction information and links; update the matching card for a duplicate. An explicit correction to the current work changes its scope only as agreed with the owner.

There is one canonical card per task. Backlog and bug lists are navigational views. A roadmap records order and dependencies, while an epic describes a product outcome. Preparation belongs in the card rather than requiring its own state, extra agent or empty files. The approved target states are `backlog → in progress → review → done`; blocking adds a reason, and cancellation is an archived outcome distinct from success. The current commands have not yet been migrated to that model.

Before proposing implementation, the coordinator reads affected requirements, architecture, code, tests and related decisions; checks dependencies and possible contradictions; and prepares scope, exclusions, acceptance criteria and independently checkable steps. Product questions are settled before implementation. The owner approves the complete plan; the coordinator then handles technical details and review within it.

## Bounded execution and review

Coding uses `.opencode/agents/executor.md`, configured for `opencode-go/deepseek-v4.1-flash` through the owner's existing subscription. The coordinator maintains plans, records and instructions directly. An unavailable model or profile is a blocker, not authorization to use the coordinator's model or a direct billed API.

After plan approval, use `bin/work-handover` or `bin/work-resume` to obtain the assigned worktree brief. In a live session, set `WORK_SESSION=1` so dispatch prints the brief instead of starting a hidden process; exit code 97 is that handoff. The coordinator adds the exact assigned step IDs, expected checks and stopping point before calling the visible executor subagent. The current commands list all remaining steps; that list is context, not authorization to complete the entire card in one session.

The executor works in its assigned worktree on `work/<id>`. For each assigned step it implements the change, runs and records its check, closes the checkbox only after success, and commits the checkpoint. At the stage boundary it stops and hands back the branch/revision, completed work, check evidence, decisions, failures and corrections, remaining findings and the precise next action. Incomplete work is preserved and identified as incomplete.

The coordinator reviews the actual diff and check evidence. Corrections go back to the same executor while its context is useful. Once that stage passes, the next independent stage gets a fresh executor context in the same worktree. Two executors never write to that worktree concurrently. The owner does not need to approve each technical stage.

Verify the actual provider/model from executor session metadata, including resumed sessions; a profile setting or the executor's own statement is not proof of what ran. The first bounded implementation stage recorded available usage information and the review/correction effort. Its session metadata was independently checked: every assistant message named `opencode-go/deepseek-v4.1-flash`, with no fallback or provider switch. Verify resumed sessions the same way.

Watch current context separately from cumulative token usage. If context becomes too large for reliable work, save a durable checkpoint and transfer the remaining bounded work to a fresh executor. A numeric context threshold and its enforcement mechanism are not yet chosen or implemented; do not invent either.

## Product and dependency navigation

The product and architecture material now lives under `docs/product/` and `docs/architecture/`; the cross-cutting `docs/MAP.md` and its link validation remain steps 5.1–5.2. They connect user scenarios and requirements to frontend, backend, API, workers, storage, tests, decisions and task cards, and distinguish implemented, planned and known-broken behavior. Update affected links when preparing and completing work. Planning records live under `work/`; older MIT/private-repository descriptions are historical, while `LICENSE` defines the current Elastic License 2.0 terms.

The approved structural direction is `frontend/`, `backend/` including workers, `mcp/`, `legacy/app/`, `docs/`, `work/` and `tools/`, with shared root commands and configuration. `mcp/README.md` records the future TypeScript HTTP adapter; its implementation remains GO-9's scope and was not authorized by the structural move. Preserve legacy code until its parity gates permit removal, unfinished cards and unrelated worktrees. Keep shared hooks shared; inspect ownership and applicability before changing integration.

## Preview, acceptance and release

The target is continuous preview on staging at `localhost:8081`, including uncommitted development changes. Production at `localhost:8080` advances only after the owner approves the completed work on staging. The approved revision is checked and merged, production is verified, and only then is the card done. This is one owner release decision, not repeated approval for technical stages.

The target isolates binary, frontend bundle, workers and data between environments. Refreshing staging data is an explicit operation, separate from updating code. Failed checks, dirty copies, incomplete steps, unresolved blockers or a changed tested revision must prevent finishing, and failure must preserve recovery state. These protections and automation are planned, not yet established by the existing commands.

Currently staging requires `bin/sift-deploy --env staging`, that command copies a production database snapshot, and workers are shared. `bin/work-finish` merges without deployment, and `bin/sift-promote` is the explicit production command. Acceptance and finish now enforce the base branch, clean source and target checkouts (including untracked files), closed steps, a fit plan (present, non-empty, with steps and acceptance criteria, no malformed step line), resolved attention and no terminal outcome; they run checks at the captured source SHA and refuse to merge if the source revision, target revision or target branch moves. Failed checks and rejections preserve the work branch, worktree and uncommitted changes. Rejection commits only `rejected.md`, leaving pre-existing staged/unstaged target work untouched. A successful finish retains the worktree and branch when ignored local material is present, rather than deleting it. Automatic continuous preview and the release loop remain phase-6 work. The `make run-staging` target also uses the shared build artifacts and is not the isolated deployment path.

Runtime `data/` and `data-staging/` stay local. The proposed vacancy, user-provided, user-derived and application data layout is a separate migration task; structural relocation does not move live user files or split the database. The local incident record is `work/incidents.md`; it is ignored and untracked, and the earlier `ai/incidents.md` was removed from the index without rewriting Git history. Runtime/local logs and generated bundles are ignored, while task `log.md` evidence stays tracked. Secrets checks must cover tracked areas without banning valid task IDs or work-command references; publication-policy changes remain part of the foundation proposal.

## Restart and completion records

Before a context reset, update the active card's log and the roadmap's current resume point with approval status, completed checks, remaining work and the next action. Record any technical limitation or incident in the appropriate project record instead of silently routing around it. Never mark a checkpoint complete solely to create a clean stopping point.

After restarting OpenCode, read `AGENTS.md`, this document, the roadmap's current planning section and the active card's description, plan, attention and log. Then inspect the current branch, diff and worktrees. Resume only within recorded authorization. Agent-profile changes require restarting OpenCode; changing files does not reload an already-running session.
