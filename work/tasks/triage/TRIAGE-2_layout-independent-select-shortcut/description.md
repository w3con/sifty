# TRIAGE-2 — Layout-independent Select shortcut

**Epic:** triage
**Created:** 2026-11-10
**Status:** review
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Можно сделать так, чтобы Y (select) работало на любой раскладке, не только на английской?
```

## Why this work exists

Observable outcome: pressing the physical Y key selects on every keyboard layout, not only the English one. Beneficiary: the owner triaging jobs. Concrete use: in the existing triage/select view, `н` on a Russian layout and `y` on an English layout, both with `event.code === 'KeyY'`, must run the same Select action as the current `y/KeyY` path; typing in an editable field must not select. Demonstrated within this task in the existing browser tooling against the running frontend.

Approval: owner «да. потом - Давай все закоммитим, squash commits / push?» approved the exact small plan shown in chat immediately before this card was opened — fix only Y/Select across layouts.

## Preparation — coordinator plan (transcribed)

Scope: use the physical `KeyboardEvent.code === 'KeyY'` instead of the locale-dependent `y`; preserve the existing editable-target, modifier (Ctrl/Alt/Meta), composition and dialog guards, and leave every other shortcut untouched. The existing `event.code` is the data/config/domain input, the existing frontend handler in `frontend/main.tsx` is the logic, and the existing button is presentation. No persisted data or settings are added.

Preliminary estimate: complexity 1; AI preparation ~1 min, fix ~1 min, frontend build 1–2 min, real-browser verification 1–2 min; total AI elapsed 3–5 min with the upper boundary measured from approval. No external monetary costs.

Excluded: other shortcut keys, backend changes, user-data changes, deploy/release, history rewrite, integration and push.

## Steps

- [x] K.1 — Use physical KeyY for Select; build frontend and verify layout-independent action and input guard in a real browser.

## Acceptance criteria

- `y` on an English layout and `н` on a Russian layout, both with `event.code === 'KeyY'`, invoke Select.
- Shift behavior is unchanged and the existing editable/modifier/composition/dialog guards still suppress Select.
- The frontend build passes.

## Must not touch

Other shortcuts, backend, user data, persistence, settings, deploy, history rewrite, integration and push.

## Approved follow-through: integration and repository publication

The coordinator proposed the following complete Git operation after technical acceptance, and the owner answered verbatim «да, делай». This separately approves integration and normal push and supersedes the earlier exclusion of those operations for this follow-through only; production release, data changes and rewriting existing history remain excluded. The beneficiary is the owner using the main checkout and its GitHub backup; demonstrated use is the accepted keyboard fix in main plus the reviewed documents/design assets committed and main successfully published to origin/main.

Complexity:3 because of existing uncommitted documentation/design assets, preservation requirements and remote history uncertainty. AI preparation/review approximately1minute, integration of the keyboard fix approximately1minute, grouping and committing pending records/assets approximately2–4minutes, remote verification and normal push approximately1–2minutes; total approved AI elapsed4–7minutes with shared review included. No new paid service is needed; external network/credential delays are separate and uncertain. Preserve every existing worktree and recovery ref. Existing source/build/browser evidence already covers the unchanged accepted fix; do not rerun broad suites.

- [x] K.2 — Integrate the accepted Y fix once and commit reviewed pending workflow records, plans and design assets in meaningful groups.
- [ ] K.3 — Verify remote history and publish the reviewed main branch with a normal push, then record the resulting clean/synchronized state.

K.2 concrete results: one application-fix commit containing only the accepted main.tsx change and its current task records/navigation; separate Conventional Commits for workflow instructions, planning records/runtime documentation, and owner-approved design assets. Explicitly inspect all staged paths/blobs, preserve unrelated runtime/local data and secrets guards, and retain the exact pre-operation source and dirty-file recovery material. Existing-history squash is not approved; no amend, reset, force push or existing worktree removal. Any guard finding must be diagnosed rather than bypassed. Historical benchmark/run evidence is data, not permission to rerun operations.

K.3 concrete result: after coordinator review of K.2's actual diff/commits, refresh origin/main, verify ancestry and push main normally to its configured origin. Do not overwrite another actor's commits. Report the actual remote revision, ahead/behind counts and pending paths. No automatic production deployment follows from publication.

Acceptance for this approved follow-through: accepted three-line keyboard change present in main; all intended existing dirty/untracked documents and design assets accounted for; real commit/secret guards applied; existing history and recovery branches preserved; normal push succeeds and local main matches the verified remote ref. Inputs/configuration are Git's existing index/refs and configured origin; logic is existing Git/work commands, with no application data/configuration or presentation changes beyond the accepted fix.
