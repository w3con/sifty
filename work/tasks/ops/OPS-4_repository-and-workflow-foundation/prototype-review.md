# Prototype review: current state, deferred automation and tests

**Date:** 2026-11-06
**Author:** coordinator
**Basis:** static review of the OPS-4 worktree at `0b1965b` and its uncommitted changes; existing check evidence was read, not rerun. No executor, build, test suite or deployment was launched for this review.

## Owner direction

«Отключи большинство тестов- они не нужны на данном этапе.» «Сохрани свои наблюдения в отдельный файл - займемся тестами когда функциональность будет готов.» «Сейчас по сути идет быстрое прототипирование.» Subsequently: «я бы вообще все выключил»; «кажется что это нам дает прирост 80% работы на 5% нужности».

The owner also requested a short review of accumulated changes, a minimal completion proposal and an analysis of what is lost by not completing the original plan. The coordinator authored the large plan and is responsible for its excessive scope and execution overhead. The percentage above is the owner's assessment, not a measured cost/benefit statistic.

## What is ready and where

- Phases 1–5 were technically reviewed: 18 of 23 steps. Source is relocated to `frontend/`, `backend/`, `legacy/app/`; records/docs/shared wiring are under `work/`, `docs/`, `tools/`. Earlier evidence covers builds and representative runtime checks at the new paths.
- Root GitHub workflows, safer merge/preservation guards, task capture/navigation and the requirement/data map exist. Their development cost is already spent; keeping them does not require building more workflow automation.
- Main remains `c0181a1` with the old tree. Production and the running staging instance have not been advanced to this work. Physical main relocation is not complete.
- Preview step 6.1 is committed at `0b1965b` and checked by its executor, but has not been technically accepted. The original plan checkbox is not acceptance. The supervisor is not connected to work handover/resume and has not been activated on real staging.
- Step 6.2 is partial: `_deploy_lib.py`, `sift-deploy`, `work-test` had uncommitted edits and `sift-data-refresh` was untracked when work stopped. Those edits are preserved. The partial artifact-publishing refactor has incompatible function call signatures. Do not use it as a working deployment path.
- `make run-staging` still builds/uses the shared binary, bundle and worker sources. The new environment document describes guarantees that this partial implementation has not established; its environment/rollback sections are not verified operational instructions.

## What is lost by stopping the original plan

| Deferred part | Practical consequence | Prototype alternative |
|---|---|---|
| Finishing/reviewing continuous preview | No verified automatic refresh for every uncommitted frontend/backend/worker edit; the new preview command remains an experiment. | Rebuild and restart staging deliberately after a meaningful change. |
| Full binary/bundle/worker isolation and explicit refresh tooling | No general guarantee that every entrypoint separates environments or preserves staging edits. Shared worker sources can expose production to development changes. | Use one reviewed staging launch with its own binary, bundle, worker copy and existing data; avoid shared Makefile paths and implicit data refresh. |
| Owner-approved completion/release automation | Approval does not automatically perform the checked merge, deploy, verification and final task closure. | Coordinator performs those actions explicitly on the owner's release instruction. Keep existing branch/cleanliness/preservation guards. |
| Full scratch failure-recovery and clean-clone verification | No comprehensive claim about rollback combinations or installation on another machine. | Limit the completion claim to this local prototype, its build and the owner-reviewed staging behavior. Preserve recovery copies before live actions. |
| Final synchronization of instructions and handback | Documentation may contradict executable state and the relocated app remains unseen by the owner. | This portion is still necessary: state the one actual staging command, current paths and actual release boundary accurately. |

The structural move, existing product functionality, task records and map are not lost by deferring these parts. The remaining work is mostly developer convenience and operational assurance. It is possible to finish a useful structural task without completing an automated release platform. It is not accurate to mark the entire original plan done unchanged.

## Minimal completion proposal — not implementation authorization

1. Preserve the unfinished deployment/preview work as recoverable work, then select one coherent manual staging path based on the reviewed application move. Do not mix a half-finished new deploy library with the older command.
2. Build backend and frontend at their new locations; launch staging using a staging-only binary, bundle and worker copy against the existing staging data. Do not refresh/copy production data as part of a code update. Verify which process owns the port before stopping anything.
3. Let the owner inspect the running site: listing/navigation, opening a vacancy, settings and the current application-confirmation dialog. Report the source revision and known limitations. New requested confirmation responses remain backlog work.
4. Update the actual run instructions. Re-scope the original card explicitly, leaving deferred automation open or moving it to a separately prepared follow-up. Do not tick unimplemented original steps or use force to bypass completion guards.
5. Only after owner approval, preserve/reconcile main's local incident record, generated bundle and initialized private submodule per the inventory; then perform the agreed merge/release. Keep the newer worktree incident record. No history rewrite or deletion of unrelated worktrees.

Automatic preview/release integration, generalized environment refresh/rollback and an exhaustive verification matrix are proposed follow-up work, not needed to see the relocated prototype on staging. No new follow-up card or changed task plan has been created by this proposal.

## Test policy now implemented

- `bin/work-test` reports SKIPPED by default; `--full` or `WORK_TEST_MODE=full` is explicit opt-in to the preserved suite.
- `bin/sift-accept` skips automated suites by default and builds backend/frontend for app/build changes. Build failure still fails the command. Output does not present skipped tests as passed.
- GitHub's workflow is manual-only (`workflow_dispatch`), preserving its existing jobs for a deliberate later run.
- Direct test commands and test source files remain available. Agents must not invoke them or develop new tests as a default prototype step without an explicit owner request.
- Compilation/build, existing Git preservation/readiness guards and actual secret scanning remain operational. They are not a repeated automated test suite. Functionality is reviewed on the working staging site.

## Test observations to revisit after functionality is ready

Static inventory found 100 Python test methods under `bin/tests/` and 39 scanner test methods; subtests add more cases. These counts are not runtime measurements. Older Go and legacy suites were inspected selectively, not audited exhaustively.

### Clear candidates for removal or consolidation

- `bin/work-test:41–80` before this change: standalone compilation overlaps commands' `--help` execution/imports. Preserve missing-command detection if consolidating.
- `bin/work-test:57–70`: command-name and no-`@` assertions constrain documentation form rather than application behavior.
- `backend/internal/store/fts_query_test.go:55–79`: two substring checks repeat rules already covered by exact query comparisons above.
- `bin/tests/test_work_note.py`: combine output/spacing preservation into the repeated-note scenario; retain failure-exit propagation.
- `bin/tests/test_capture.py`: combine fresh-card/no-dispatch/no-worktree and staging-sentinel assertions into one creation scenario.
- `bin/tests/test_navigation.py:test_generated_views_agree_after_a_transition`: fold its assertions into the existing state-transition test.
- `backend/scripts/leakcheck/tests/test_policy.py:test_real_secret_assignment_fails_everywhere`: its path loop never passes the path to the function, repeating the identical check five times. Keep a single unit check and the real path-sensitive integration cases.
- Scanner self-test and `test_policy.py` overlap on placeholders/environment assignments, formatting areas, compatibility allowance and diagnostic redaction. Consolidate their examples while preserving every distinct credential form.

### Expensive repetition and slow-test causes

- The original GitHub workflow ran scanner self-test directly and again inside the harness job. One run is sufficient.
- Every full harness run includes preview subprocess/proxy tests. The existing log records four preview tests at 58.970 seconds using fake build commands. Keep them opt-in while the supervisor is not the prototype's trusted runtime.
- `_preview_lib.py:346–366` polls child existence for up to ten seconds after signalling without reaping via `wait()`. Zombie-child handling is a plausible source of slow shutdowns; this is a static finding, not a newly reproduced timing result.
- The task log contains 14 recorded check commands containing `bin/work-test`, apart from focused checks and coordinator calls. Some follow real fixes; full-suite repetition should not be the default after every edit.
- Acceptance followed immediately by finish currently repeats acceptance; a later simplification should use one final checked revision rather than repeated identical full runs. Do not use `--force` as a substitute for a coherent process.

### Valuable checks to preserve as source

Preserve worktree/uncommitted-file protection, tested-revision movement, database migrations/data integrity, cancellation/blocker enforcement, one preview owner and failed-build/recovery scenarios. Similar assertions at library, command and installed-hook boundaries may protect different wiring and are not automatically duplicates. Do not remove the whole legacy test tree: some tests import the current frontend (for example `legacy/app/test/usability-filters.test.ts`).

Runtime alone does not explain five hours of work: map tests were recorded around 0.015 seconds, focused task tests took seconds and preview tests about a minute. Much of the cost came from designing/writing infrastructure and tests, long executor reasoning, repeated handoffs, coordination defects and delayed staging feedback.
