# ASM-6 J.1 — proposed card-granularity main integration inventory

Read-only proposal. Main was **not** changed. Branch: `work/ASM-6` at `.worktrees/ASM-6`.

## Source facts

- Canonical application stack: `d008650` → SETTINGS-1 `d6f5808` → SETTINGS-2 `5efd12a` → SETTINGS-3 `7f5c0a3` → DESIGN-3 initial `fe31cc3` → APP-8 `d2db849` → COLLECTION-1 `2b8a1d5` → shared integration `adfdadd` → finished DESIGN-3 `2f6c026`.
- These are linear ancestors. `adfdadd` adds the finished Overview/profile/navigation checkpoint to the stack; `2f6c026` adds the remaining branding, Settings, theme and fact-strip placement (15 frontend files).
- `main` is `4156d24`, a docs-only descendant of `d008650` (19 `work/tasks/**` paths). None of the application stack is in main.
- Root `main` has **53 `git status` entries covering 150 individual paths** (owner app/theme/tooling + task/docs). They are preserved untouched; the proposal never stashes, resets, checks out or stages them.
- Our merged J.1 tree adds `a46b9fb..2f6c026` (14 frontend files) onto `8677836` with three reconciled files: `frontend/RoleContent.tsx`, `frontend/main.tsx`, `frontend/theme.css`.
- Final app tree after **seven card commits** must equal this worktree's merged `backend/ frontend/ config/ docs/specs/` content. DESIGN-3 is ONE completed-card commit, not separate initial and finished commits.

## Proposed publication method — step 5 rejected; J.2 only, separately assigned

1. Create a temporary clean worktree at `4156d24`; never operate on the dirty main worktree files.
2. Apply each stage's application/config paths (below) in order and commit one Conventional Commits subject per card. Keep each card's path set exactly to its on-record delta; do not replay checkpoint history.
3. Exclude `work/tasks/**` and root process docs (`AGENTS.md`, `work/WORKFLOW.md`, `work/ROADMAP.md`, `docs/MAP.md`) from the app commits where root already has a newer dirty/divergent copy; reconcile card/navigation docs separately at J.3. This is what prevents the owner's 53 dirty/untracked entries from being overwritten.
4. Verify `git diff --quiet <prepared-final> <merged-ASM-6> -- backend frontend config docs/specs` before publication.
5. **REJECTED (coordinator, J.1 review): a bare `git update-ref refs/heads/main` with no checkout.** It would leave main's working tree/index inconsistent with `HEAD` and the owner would still be using the old app. J.2 must update the actual main files through a normal, recoverable publication while preserving originals. Do not execute main changes in J.1.
6. Required J.2 preservation shape: before touching main's worktree, back up every dirty/untracked path (or commit a recovery ref) so the owner's 150 local paths are recoverable byte-for-byte; then apply the prepared card commits to the real main checkout, never stash/reset/stage-all. Re-hash every dirty/untracked path before/after and show the same 150 paths / 53 status entries. If main's worktree cannot be updated without discarding local content, stop and return to the coordinator.


## Stage inventory

### 1. SETTINGS-1 — `feat(settings): add user profile workflow and fields`

- Source: `d008650` → `d6f5808`; 80 application/config/doc paths.
- Excluded from this commit: 17 `work/**` card/process paths (preserved/reconciled separately).

<details><summary>Exact paths</summary>

- `M` `backend/cmd/sifty/serve.go`
- `M` `backend/go.mod`
- `M` `backend/internal/assess/infer.go`
- `M` `backend/internal/assess/infer_test.go`
- `M` `backend/internal/background/assessment.go`
- `M` `backend/internal/background/collect.go`
- `M` `backend/internal/candidate/context.go`
- `M` `backend/internal/config/config.go`
- `A` `backend/internal/extract/extract.go`
- `M` `backend/internal/model/job.go`
- `A` `backend/internal/profile/analysis.go`
- `A` `backend/internal/profile/bootstrap.go`
- `A` `backend/internal/profile/contract.go`
- `A` `backend/internal/profile/documents.go`
- `A` `backend/internal/profile/documents_test.go`
- `A` `backend/internal/profile/docx_worker.go`
- `A` `backend/internal/profile/facts.go`
- `A` `backend/internal/profile/facts_service.go`
- `A` `backend/internal/profile/ranking.go`
- `A` `backend/internal/profile/ranking_test.go`
- `A` `backend/internal/profile/service.go`
- `A` `backend/internal/profile/service_test.go`
- `A` `backend/internal/prompts/profile-facts.md`
- `M` `backend/internal/prompts/prompts.go`
- `M` `backend/internal/server/drift_test.go`
- `A` `backend/internal/server/profile_api.go`
- `M` `backend/internal/server/routes.go`
- `M` `backend/internal/server/server.go`
- `A` `backend/internal/server/testdata/fixtures/49_GET_profile.json`
- `A` `backend/internal/server/testdata/fixtures/50_PUT_profile.json`
- `A` `backend/internal/server/testdata/fixtures/51_PUT_profile_missing_data.json`
- `A` `backend/internal/server/testdata/fixtures/52_GET_profile_documents.json`
- `A` `backend/internal/server/testdata/fixtures/53_POST_profile_documents.json`
- `A` `backend/internal/server/testdata/fixtures/54_GET_profile_document.json`
- `A` `backend/internal/server/testdata/fixtures/55_GET_profile_document_text.json`
- `A` `backend/internal/server/testdata/fixtures/56_GET_profile_document_file.json`
- `A` `backend/internal/server/testdata/fixtures/57_POST_profile_document_analyze.json`
- `A` `backend/internal/server/testdata/fixtures/58_POST_profile_document_cancel.json`
- `A` `backend/internal/server/testdata/fixtures/59_POST_profile_document_retry.json`
- `A` `backend/internal/server/testdata/fixtures/60_GET_profile_facts.json`
- `A` `backend/internal/server/testdata/fixtures/61_GET_profile_document_facts.json`
- `A` `backend/internal/server/testdata/fixtures/62_POST_profile_fact_accept.json`
- `A` `backend/internal/server/testdata/fixtures/63_POST_profile_fact_reject.json`
- `A` `backend/internal/server/testdata/fixtures/64_POST_profile_fact_resolve.json`
- `A` `backend/internal/server/testdata/fixtures/65_GET_profile_capabilities.json`
- `M` `backend/internal/store/core.go`
- `M` `backend/internal/store/jobs.go`
- `M` `backend/internal/store/migrate.go`
- `M` `backend/internal/store/migration_v14_test.go`
- `A` `backend/internal/store/profile_assessment_stale_test.go`
- `A` `backend/internal/store/profile_documents_storage.go`
- `A` `backend/internal/store/profile_fact_accept_test.go`
- `A` `backend/internal/store/profile_storage.go`
- `A` `backend/internal/store/profile_test.go`
- `M` `backend/internal/store/rescore.go`
- `M` `backend/internal/store/store.go`
- `M` `backend/internal/store/store_test.go`
- `A` `backend/workers/documents/index.mjs`
- `A` `backend/workers/documents/package-lock.json`
- `A` `backend/workers/documents/package.json`
- `M` `config/runtime.yaml`
- `M` `frontend/ApplicationConfirmation.tsx`
- `M` `frontend/Collection.tsx`
- `A` `frontend/Logo.tsx`
- `M` `frontend/RoleContent.tsx`
- `M` `frontend/api.ts`
- `M` `frontend/application.css`
- `M` `frontend/index.html`
- `M` `frontend/main.tsx`
- `M` `frontend/model/core.ts`
- `M` `frontend/package-lock.json`
- `M` `frontend/package.json`
- `A` `frontend/profile/ProfileScreen.tsx`
- `A` `frontend/profile/ProfileView.tsx`
- `A` `frontend/profile/api.ts`
- `A` `frontend/profile/contract.ts`
- `A` `frontend/profile/profile.css`
- `A` `frontend/profile/useProfile.ts`
- `A` `frontend/theme.css`
- `M` `frontend/vite.config.ts`
</details>

### 2. SETTINGS-2 — `feat(settings): add profile workflow and document fields`

- Source: `d6f5808` → `5efd12a`; 23 application/config/doc paths.
- Excluded from this commit: 7 `work/**` card/process paths (preserved/reconciled separately).

<details><summary>Exact paths</summary>

- `M` `backend/internal/assess/assess.go`
- `M` `backend/internal/assess/infer.go`
- `M` `backend/internal/assess/infer_test.go`
- `M` `backend/internal/profile/analysis.go`
- `M` `backend/internal/profile/bootstrap.go`
- `M` `backend/internal/profile/contract.go`
- `M` `backend/internal/profile/documents.go`
- `A` `backend/internal/profile/extraction_options_test.go`
- `A` `backend/internal/profile/live_extraction_probe_test.go`
- `A` `backend/internal/profile/section.go`
- `A` `backend/internal/profile/section_test.go`
- `M` `backend/internal/profile/service.go`
- `M` `backend/internal/profile/service_test.go`
- `M` `backend/internal/server/profile_api.go`
- `M` `backend/internal/store/profile_storage.go`
- `M` `frontend/main.tsx`
- `M` `frontend/profile/ProfileScreen.tsx`
- `M` `frontend/profile/ProfileView.tsx`
- `M` `frontend/profile/api.ts`
- `M` `frontend/profile/contract.ts`
- `A` `frontend/profile/documentPresentation.ts`
- `M` `frontend/profile/profile.css`
- `M` `frontend/profile/useProfile.ts`
</details>

### 3. SETTINGS-3 — `feat(settings): add rich profile fields and reference data`

- Source: `5efd12a` → `7f5c0a3`; 23 application/config/doc paths.
- Excluded from this commit: 6 `work/**` card/process paths (preserved/reconciled separately).

<details><summary>Exact paths</summary>

- `M` `backend/internal/profile/contract.go`
- `M` `backend/internal/profile/facts.go`
- `A` `backend/internal/profile/fields.go`
- `A` `backend/internal/profile/profile_fields_test.go`
- `M` `backend/internal/profile/ranking.go`
- `A` `backend/internal/profile/rich_text.go`
- `M` `backend/internal/profile/section.go`
- `M` `backend/internal/profile/service.go`
- `M` `backend/internal/server/profile_api.go`
- `M` `frontend/package-lock.json`
- `M` `frontend/package.json`
- `A` `frontend/profile/DateInputs.tsx`
- `M` `frontend/profile/ProfileView.tsx`
- `A` `frontend/profile/ReferenceFields.tsx`
- `A` `frontend/profile/RichTextEditor.tsx`
- `A` `frontend/profile/RichTextView.tsx`
- `M` `frontend/profile/contract.ts`
- `A` `frontend/profile/datePickerConfig.ts`
- `M` `frontend/profile/profile.css`
- `A` `frontend/profile/referenceData.ts`
- `A` `frontend/profile/richText.ts`
- `A` `frontend/profile/sectionErrors.ts`
- `M` `frontend/profile/useProfile.ts`
</details>

### 4. DESIGN-3 initial slice — NOT a separate commit; folded into completed DESIGN-3 (stage 7)

- Source: `7f5c0a3` → `fe31cc3`; 16 application/config/doc paths.
- Excluded from this commit: 7 `work/**` card/process paths (preserved/reconciled separately).

<details><summary>Exact paths</summary>

- `M` `backend/go.mod`
- `M` `backend/go.sum`
- `M` `backend/internal/config/config.go`
- `A` `backend/internal/profile/avatar.go`
- `A` `backend/internal/profile/avatar_test.go`
- `M` `backend/internal/server/profile_api.go`
- `M` `config/runtime.yaml`
- `M` `frontend/Icon.tsx`
- `M` `frontend/main.tsx`
- `M` `frontend/profile/ProfileScreen.tsx`
- `M` `frontend/profile/ProfileView.tsx`
- `M` `frontend/profile/api.ts`
- `M` `frontend/profile/contract.ts`
- `M` `frontend/profile/profile.css`
- `M` `frontend/profile/useProfile.ts`
- `M` `frontend/style.css`
</details>

### 5. APP-8 — `feat(applications): record expired application outcome`

- Source: `fe31cc3` → `d2db849`; 10 application/config/doc paths.
- Excluded from this commit: 7 `work/**` card/process paths (preserved/reconciled separately).

<details><summary>Exact paths</summary>

- `M` `backend/internal/model/job.go`
- `M` `backend/internal/server/routes.go`
- `M` `backend/internal/store/jobs.go`
- `M` `backend/internal/store/migrate.go`
- `A` `backend/internal/store/outcome_reason_test.go`
- `M` `frontend/ApplicationConfirmation.tsx`
- `M` `frontend/application.css`
- `M` `frontend/main.tsx`
- `M` `frontend/model/core.ts`
- `M` `frontend/model/workflow.ts`
</details>

### 6. COLLECTION-1 — `feat(collection): retain publication date and age`

- Source: `d2db849` → `2b8a1d5`; 31 application/config/doc paths.
- Excluded from this commit: 7 `work/**` card/process paths (preserved/reconciled separately).

<details><summary>Exact paths</summary>

- `M` `backend/internal/collect/feed.go`
- `M` `backend/internal/collect/parse.go`
- `A` `backend/internal/collect/published_test.go`
- `M` `backend/internal/collect/testdata/feed.json`
- `M` `backend/internal/collect/testdata/jsonld.json`
- `M` `backend/internal/collect/testdata/links_getro.json`
- `A` `backend/internal/model/date.go`
- `A` `backend/internal/model/date_test.go`
- `M` `backend/internal/model/job.go`
- `M` `backend/internal/server/enrich.go`
- `M` `backend/internal/server/enrich_test.go`
- `A` `backend/internal/server/match_test.go`
- `M` `backend/internal/source/ashby.go`
- `M` `backend/internal/source/generic.go`
- `M` `backend/internal/source/getro.go`
- `M` `backend/internal/source/greenhouse.go`
- `M` `backend/internal/source/greenhouse_test.go`
- `M` `backend/internal/source/lever.go`
- `A` `backend/internal/source/match.go`
- `A` `backend/internal/source/match_test.go`
- `M` `backend/internal/store/jobs.go`
- `M` `backend/internal/store/migrate.go`
- `M` `backend/internal/store/migration_v14_test.go`
- `M` `backend/internal/store/outcome_reason_test.go`
- `A` `backend/internal/store/published_test.go`
- `M` `backend/internal/store/store_test.go`
- `M` `frontend/RoleContent.tsx`
- `A` `frontend/dates.ts`
- `M` `frontend/filters.ts`
- `M` `frontend/main.tsx`
- `M` `frontend/model/core.ts`
</details>

### 7. DESIGN-3 completed — `feat(design): add reusable design system, profile navigation and finished layout` (single completed-card commit; includes stage 4 slice)

- Source: `2b8a1d5` → `2f6c026`; 25 application/config/doc paths.
- Excluded from this commit: 14 `work/**` card/process paths (preserved/reconciled separately).

<details><summary>Exact paths</summary>

- `M` `frontend/ApplicationConfirmation.tsx`
- `M` `frontend/ApplicationDraft.tsx`
- `A` `frontend/Brand.tsx`
- `M` `frontend/CatalogueFilters.tsx`
- `M` `frontend/Collection.tsx`
- `A` `frontend/ListResizer.tsx`
- `M` `frontend/Logo.tsx`
- `M` `frontend/RoleContent.tsx`
- `M` `frontend/Settings.tsx`
- `A` `frontend/location.ts`
- `M` `frontend/main.tsx`
- `A` `frontend/messages.ts`
- `A` `frontend/profile/ProfileSwitcher.tsx`
- `M` `frontend/profile/ProfileView.tsx`
- `M` `frontend/profile/ReferenceFields.tsx`
- `M` `frontend/profile/api.ts`
- `M` `frontend/profile/contract.ts`
- `M` `frontend/profile/documentPresentation.ts`
- `M` `frontend/profile/profile.css`
- `M` `frontend/profile/referenceData.ts`
- `M` `frontend/profile/useProfile.ts`
- `A` `frontend/public/fonts/tiny5-OFL.txt`
- `A` `frontend/public/fonts/tiny5.woff2`
- `M` `frontend/theme.css`
- `M` `frontend/workflow.ts`
</details>

### 7. ASM-6 — `feat(assessment): add deterministic Quick Match`

- Source: `2f6c026` → merged J.1 worktree; 51 application/config/doc paths. This is the ASM-6 card delta over finished DESIGN-3; the already-merged Design files appear in stage 7 and the three reconciled files appear here in their combined form.

<details><summary>Exact paths (2f6c026 vs merged working tree)</summary>

- `M` `backend/cmd/sifty/serve.go`
- `M` `backend/internal/background/collect.go`
- `M` `backend/internal/background/scheduler.go`
- `M` `backend/internal/background/trigger.go`
- `M` `backend/internal/config/config.go`
- `M` `backend/internal/config/holder.go`
- `M` `backend/internal/config/holder_test.go`
- `M` `backend/internal/config/save.go`
- `A` `backend/internal/jobextract/extract_test.go`
- `A` `backend/internal/jobextract/jobextract.go`
- `A` `backend/internal/matchconfig/matchconfig.go`
- `A` `backend/internal/matchcontract/contract.go`
- `A` `backend/internal/matchcontract/identity.go`
- `A` `backend/internal/matchcontract/validate.go`
- `A` `backend/internal/matchsvc/service.go`
- `A` `backend/internal/matchsvc/service_test.go`
- `M` `backend/internal/model/job.go`
- `A` `backend/internal/profile/quickmatch.go`
- `M` `backend/internal/profile/ranking_test.go`
- `M` `backend/internal/profile/service.go`
- `A` `backend/internal/quickmatch/compensation.go`
- `A` `backend/internal/quickmatch/conditions.go`
- `A` `backend/internal/quickmatch/evaluate_test.go`
- `A` `backend/internal/quickmatch/geography.go`
- `A` `backend/internal/quickmatch/professional.go`
- `A` `backend/internal/quickmatch/quickmatch.go`
- `A` `backend/internal/quickmatch/role.go`
- `A` `backend/internal/quickmatch/text.go`
- `M` `backend/internal/server/routes.go`
- `A` `backend/internal/store/match_facts.go`
- `A` `backend/internal/store/match_facts_test.go`
- `A` `backend/internal/store/match_results.go`
- `A` `backend/internal/store/match_results_test.go`
- `M` `backend/internal/store/migrate.go`
- `M` `backend/internal/store/migration_v14_test.go`
- `M` `backend/internal/store/rescore.go`
- `M` `backend/internal/store/store.go`
- `M` `backend/internal/store/store_test.go`
- `M` `config/runtime.yaml`
- `M` `docs/MAP.md`
- `A` `docs/specs/quick-match.md`
- `A` `frontend/QuickMatchPanel.tsx`
- `M` `frontend/RoleContent.tsx`
- `M` `frontend/filters.ts`
- `M` `frontend/main.tsx`
- `M` `frontend/model/core.ts`
- `M` `frontend/profile/ProfileScreen.tsx`
- `M` `frontend/profile/ProfileView.tsx`
- `A` `frontend/quickmatch.ts`
- `M` `frontend/theme.css`
- `M` `frontend/workflow.ts`
</details>

## Root main dirty/untracked preservation manifest

25 tracked modifications + 125 untracked files (53 total including staged entries).

| Path | SHA-256 | Kind |
|---|---|---|
| `AGENTS.md` | `b76cf6a8ac69a8b1fd9fbf68967ae51d114f1c64d29efc5aa33f460632576d42` | modified |
| `bin/templates/executor-brief.md` | `67ce6a9b6c59d0b920c7127081b8cb611908a03e2e68a12cfc2790714f0d7d44` | modified |
| `docs/MAP.md` | `c30e6ec22cdce5c8f2591a23477c5be7138c246ff4da782f0a6dc9fdb7633627` | modified |
| `docs/architecture/preview-and-environments.md` | `b4d72683e6effb3eeaf30a7cb55bb78afa0b3e33029ad5052cb7c9d6406fa446` | modified |
| `frontend/ApplicationConfirmation.tsx` | `e2231b8d7ec71865d4d20b42730aff577bc6d148762fa543e8ed85127d8cc649` | modified |
| `frontend/Collection.tsx` | `f06d8c71713bf45df72c9fb37f3e6a4561937109756670ae79fb0c72c00737f6` | modified |
| `frontend/application.css` | `525a2c84cdfd64438655cf06d047d7b079b438a29aee8b9b9a357995cc9e8b03` | modified |
| `frontend/index.html` | `17dbfe2c2e916fab6b2d516519b0fae0e1f59e7f3e87054726320726b13e0748` | modified |
| `frontend/main.tsx` | `3fd568200b9a96c63f2b88d2dbc34e9accb90424669bf74cfefd1947a6ce1bb4` | modified |
| `frontend/package-lock.json` | `b015e4f256b5d54caba5f9398d06fc91d016f0b351f35a9312b786fbcdb2e6cc` | modified |
| `frontend/package.json` | `ea9af18c1847922ae0dd86ea1a6b675113f7c61e77a00e28ec46b04da2735b13` | modified |
| `frontend/vite.config.ts` | `504681bc7ff5907bd60e4fc228ac1ba2bb376c58c09640622bf3d769e1c66659` | modified |
| `work/ROADMAP.md` | `5e81f8019ea0936f9bc00cb9802ccf2501b537b74b72a9cd378d13ec81105aef` | modified |
| `work/WORKFLOW.md` | `3332dd0bce26446521f0a5be58ff1f166ed3e7e5c058ae35980ed42ce4daf29b` | modified |
| `work/tasks/application-preparation/APP-8_expired-application-outcome/description.md` | `57523697635c6dc9ea3ebbfef2a0fed720263fdce412a78d06e71661cb0c6c6f` | modified |
| `work/tasks/assessment/ASM-6_deterministic-quick-match/attention.md` | `691a139a4b692668c5655d74ceef6dc953e5b933b71445e5a58dc0cb42ef6b4d` | modified |
| `work/tasks/assessment/ASM-6_deterministic-quick-match/description.md` | `737a244cda5b0d060c78364b9527633bb664e814a7c8b1d49656abb71ef135e8` | modified |
| `work/tasks/assessment/ASM-6_deterministic-quick-match/log.md` | `1c0e295851ae8dc61460ed8f0842423ba65f1c6de3b624d55a7d866c0b31c370` | modified |
| `work/tasks/assessment/ASM-6_deterministic-quick-match/plan.md` | `71c6f0ba98631b521f377f2a75019c24f5f6a4926bad110991dd5e03e8da7a6e` | modified |
| `work/tasks/assessment/ASSESSMENT-1_user-calibration-of-quick-match-weights/description.md` | `375744dab3463d4bf4493a9cb61a1c775a370e4492afe190b144c8df60c0af44` | modified |
| `work/tasks/collection/COLLECTION-1_publication-date-and-age/description.md` | `133b88cf9c0e7200e7156fb91d87a8bd856f98d65b6347989f12a72c888bdaca` | modified |
| `work/tasks/design/DESIGN-3_reusable-design-blocks/description.md` | `24e8149236d88e6078de5c49563b578809cdaf9fff5b2a1434bf4c6bb8aedf1d` | modified |
| `work/tasks/inbox/INBOX-1_search-catalogue-overview-quick-match/description.md` | `295e9fe7640a2fbc55025ba9acf658a69e644fa0c321a67602f4fca73962ecdd` | modified |
| `work/tasks/settings/SETTINGS-5_user-data-and-role-profiles/description.md` | `f37362d7682decca6b042d0b1255e6dde65d5e3642cb6d1ad4933a2c9649304c` | modified |
| `work/tasks/triage/TRIAGE-1_recognize-previously-applied-vacancy-duplicates/description.md` | `162c4a15f1e33023719cff284b5c0336054d7fa450fb7527c3a07b7bd74c379f` | modified |
| `docs/specs/quick-match.md` | `24b5d9cb4e9a1cf6bd3a1f0c1b33fde0e6ccbee662f0d8fec76849c1eeb5e6b3` | untracked |
| `frontend/Claude outputs/A-collection.png` | `db3d1b7b9e8687ff4311fbaa00acf74dedf9042e8dbb6f807c33cac6301b892b` | untracked |
| `frontend/Claude outputs/A-triage.png` | `95b265ef470c8d2ebd98f50d75625feea561f26b3d7333515838a35a551ebaba` | untracked |
| `frontend/Claude outputs/B-collection.png` | `e999ad97d1ac81fb967acdabd524289eb270ef7865a468704883ea483e912a6a` | untracked |
| `frontend/Claude outputs/B-triage.png` | `c6a78db5839eb4559bf9401e61abfa63f2455ee7fb333cbcfb55b1e4c53cca90` | untracked |
| `frontend/Claude outputs/B2-H1.png` | `a27ce8b0874557a87e81bb3e544290fe5643bd06aeadb8fad4d62cd8284913d7` | untracked |
| `frontend/Claude outputs/B2-H2.png` | `dd53ed055cc3612be5f034aa4925d3fc58edde654510641b45ae87d4324d8ebc` | untracked |
| `frontend/Claude outputs/B2-H3.png` | `42393ed681b9ecddbddd58ac370e08a8b83832f1c3e5ffef02c7752d1350f919` | untracked |
| `frontend/Claude outputs/brand-states.png` | `0d9c76f33b2e32daf368efb9ad00a0ef838b2e128aa69c65d9fc6e76f08666b5` | untracked |
| `frontend/Claude outputs/card.png` | `f4ce12935767d6387e2575892c41ee5abedde129b0755a022706053d75917ca0` | untracked |
| `frontend/Claude outputs/code-profile-1440.png` | `f0f5df0d54516f7d74a4d91da6fa859f4e5d45d42bd968dae99e7b7630e98db5` | untracked |
| `frontend/Claude outputs/code-profile-files.png` | `ea25ed2c2ac9db54291c334a8c5a83c734b24daf4d4eef6c63c89df5c147b840` | untracked |
| `frontend/Claude outputs/code-profile-files2.png` | `9b870d101c6e0c4518c7f05a3b02a734c3e016188b8c0f7e8232bd7ec9da4f3b` | untracked |
| `frontend/Claude outputs/code-profile-pill.png` | `3d57cc028eee82bfc5423cd02b80c75dff8bbcf202b04caf1b819c3b78fd4509` | untracked |
| `frontend/Claude outputs/code-profile-tabs.png` | `d0956863a15161f1031356623bb04883765ec248b2ce43ad92191a91ff1464b7` | untracked |
| `frontend/Claude outputs/del-12.png` | `606f786392454e8461047b47ebf0816c7f2169c496a227c97f9ad1906ffc4c7d` | untracked |
| `frontend/Claude outputs/del-3.png` | `fdc0f83b3998208534b6ae8d585ce74a9c985a206542628808090350fdd56d36` | untracked |
| `frontend/Claude outputs/design-system-applied.png` | `0fc760fbe6cd11bccbfeb690e3120bac170b1a2e9509ad779a0083bf81b93488` | untracked |
| `frontend/Claude outputs/edit-A.png` | `0d503574b1287905eb3217d1e2b2c59612a79524266f2b48b1c9c93e3a75f9f3` | untracked |
| `frontend/Claude outputs/edit-B.png` | `331b635448b0a8f5c556739b19211a53eab4bc6322f25b0a8262ac644f1f56cb` | untracked |
| `frontend/Claude outputs/edit-C.png` | `75952346dcf68a9d98d27d1916e58b137cd5896ed1efebeb17ac8ba263da9c9d` | untracked |
| `frontend/Claude outputs/error-styles.png` | `e8f90044ed888ff1666ff6da835ae6d4f4b5df752589bf40bd7615d6db557896` | untracked |
| `frontend/Claude outputs/facts-flow.png` | `5f7a0548236cec5f292913f67d6bc1782de2ca04704e62e5da1c5842ccc6ffe3` | untracked |
| `frontend/Claude outputs/facts-icons-unknowns.png` | `972a25cae7464bdd120b2f7624a1a2492597ab7d37e18f9bee3d1d466123c1b3` | untracked |
| `frontend/Claude outputs/facts-in-tabs-row.png` | `1596c9eb862ec724275a621d34d560d97b7f7dd35429664bf4cb4bfeda58185f` | untracked |
| `frontend/Claude outputs/filters-new2.png` | `dc6a3a9ef4f8e97ff8b0c2068222dfcfe516b0c2dcc7406c36ab14a4bdceaf41` | untracked |
| `frontend/Claude outputs/filters-v3.png` | `5f1d2bd52a95ef8560b1d2c1f68027148c19cffcde87729c2482684432b89d2a` | untracked |
| `frontend/Claude outputs/header-centered.png` | `484693345091d03225fb28bb6dcbe54735bd8721ec8fd02a75aa48d66d37913a` | untracked |
| `frontend/Claude outputs/header-structure.png` | `adc97bd93e613a3193c2e6eaadf968c88007ea5f279e181f17cee8da46150ca1` | untracked |
| `frontend/Claude outputs/honk-gradient-options.png` | `874906eceb1e5e45ff3c6618f4560093a9b94bb33a8f549698cdeba291cfb043` | untracked |
| `frontend/Claude outputs/location-compact.png` | `92c210c405302e6c4476a5525c39f19c91e8f54a89f9ab2b41ab7fdda61a941d` | untracked |
| `frontend/Claude outputs/logo-cute-bold.png` | `e613038d824fdba81e9cc2a9bfeb4cd6b805b1ab5181942071a9971418c663d5` | untracked |
| `frontend/Claude outputs/logo-fonts-2.png` | `685ac96db7cdb77817fbba4d319862f0537461fe3597a0263d16e9789cfd5728` | untracked |
| `frontend/Claude outputs/logo-fonts-3.png` | `67ef54980603c09a4504ce47e84baf4f8d0103eddbc27f1683b5e39fd23a0c71` | untracked |
| `frontend/Claude outputs/logo-fonts-gradient.png` | `760813bdeb6a76c05d24c0be944087bb8b11814d9349c9f3571f598135d73a4d` | untracked |
| `frontend/Claude outputs/logo-fonts.png` | `f3a8d104a06e579885b2279662b3242aac5116e77e3050f80090e001b6e71b85` | untracked |
| `frontend/Claude outputs/logo-honk-palettes.png` | `3e617093227a855bdd40fa7e4931f5d1fee77a3a970d4dfe35190b8d963d0843` | untracked |
| `frontend/Claude outputs/logo-tiny5-sifting.png` | `95deb9181dbb851ca300b4639e606a92d7547995d97740b556f81d3a291e50d3` | untracked |
| `frontend/Claude outputs/message-levels.png` | `bdf053686c8f0fce434412b5b0e22035f46e37d21361cdf57160ba74cf57ac38` | untracked |
| `frontend/Claude outputs/messages-four.png` | `16f6fa475304601b5f2e328f97c1b2c33d2a479a206cc0cafcaaec70344700a4` | untracked |
| `frontend/Claude outputs/overview-a.png` | `e13999e3c682b596a551e30d7b2264a0827ca7e87232c2e1dc0db77200b5993b` | untracked |
| `frontend/Claude outputs/overview-b.png` | `c286f6c6234e05bf8dcfececdb428eaf0a0fb5a5b203855311a08455153e072b` | untracked |
| `frontend/Claude outputs/overview-code-2.png` | `984c39a6bde6c7756316fdcd808350e6063d543151ac233579a0bb2b45e6f3bf` | untracked |
| `frontend/Claude outputs/overview-code.png` | `35b6a4ba94666f27a9176a013a0bba22265254b6fad3f117cfa0fd0d286dc4c4` | untracked |
| `frontend/Claude outputs/overview-facts-icons.png` | `8fd76545100bfeedfdaa39c30f9ce20999f337080c8d88f9b62f3ad7697a9aa4` | untracked |
| `frontend/Claude outputs/overview-strip-expired.png` | `a00e13379ea268918504e0413731f3cfec274a25e3884db277c813600f56f367` | untracked |
| `frontend/Claude outputs/overview-strip.png` | `9851186b7d18ee60bcd497c7e7dfca6e7e8eb0f979e179a5221dd0ac74efdd0f` | untracked |
| `frontend/Claude outputs/paper-tones.png` | `b4c52af081c576178204d7b032bca31ec5ad4abde7394c6481789a7e45cd8476` | untracked |
| `frontend/Claude outputs/profile-1920-columns.png` | `74b01e102c8a128a90d1dc16081507bdd941367b4a72f7c3e35408e400d8b1bb` | untracked |
| `frontend/Claude outputs/profile-1920-full.png` | `abf53e3bb0ca3311b7b571646df60cd6ee244e118ffde5aa2c97067b59fc85aa` | untracked |
| `frontend/Claude outputs/profile-1920-now.png` | `dbf29efc9c2676eb696404bac66c5ee3c3b39f10c56e8561d35007abea17f8d6` | untracked |
| `frontend/Claude outputs/profile-edit.png` | `6c100eb2ffabc6450b79559381e4bf4d5c0ce98a21bf26b432caddfc49af4389` | untracked |
| `frontend/Claude outputs/profile-tabs-options.png` | `f26a028237cb6a9aa224670aa5f35ab6021ad8a44a282cbee44f7c42428e869b` | untracked |
| `frontend/Claude outputs/profile.png` | `cd4fe905fcfaf35d36a3de015f4f5c8b099f26c4b641606bc8504a8034cb4338` | untracked |
| `frontend/Claude outputs/profile2-1440.png` | `b727eae7bb81cb1faa93094ada204bed916d0a83000c0c661cf47ca2de106651` | untracked |
| `frontend/Claude outputs/profile2-1920.png` | `066aa0feefeee411ab5fb4dad98eccb843b588efa4d6fb461373db3c07e9841f` | untracked |
| `frontend/Claude outputs/profile3-1440.png` | `44622b8fce5f47bb7a29e87ea8e3c5fdb8cf7f5c3e7aef5e623e7bdde10adf23` | untracked |
| `frontend/Claude outputs/profile3-1920.png` | `1f616e05bbcf20ddc437dff507072e7d75e534f9aebc78e26dcf69f15997dbed` | untracked |
| `frontend/Claude outputs/profile4-1440.png` | `2f2f61e02555f1fb692711e245e1c2691af9c7606a0d17d12c19bae765168498` | untracked |
| `frontend/Claude outputs/profile4-1920.png` | `11401754fe0b47a89718cb9491459f2b21e71be39468cb6ae3392334e9c0304d` | untracked |
| `frontend/Claude outputs/profile5-1440.png` | `0193382b8f06a57abc556cd80cc7a94efaf1dd1980f0cda81fa69a543c7f7253` | untracked |
| `frontend/Claude outputs/profile5-1920.png` | `7cc84b86c72d63f6a5955ae5caa87ab6c8606f6da77b41ec022f85d3921bec80` | untracked |
| `frontend/Claude outputs/profile6-1440-full.png` | `c31fb8d6be3fcf6c92b5ccffcc3e197f72b48c7d62749e936c7236d2afc6273d` | untracked |
| `frontend/Claude outputs/profile6-1440.png` | `1809eaab98f516cd27acf9f2ddbd5c1fd36faf7d770918b85e8fcd13eaaaf68a` | untracked |
| `frontend/Claude outputs/profile6-1920.png` | `c2342ef0493e4c08ca5bde93136ea2c8334361c116c5d39b4a6cdafd00e70e18` | untracked |
| `frontend/Claude outputs/profile7-1440.png` | `e86b5e91cb715292b0b9dc5e369d48ecc914c4d9e40c6bc01fa57c9c93c4c575` | untracked |
| `frontend/Claude outputs/profile7-1920.png` | `7c22dc93469c303a668be3fc029b9cb66aea7f1dadc431f7d62ac43cf94b4807` | untracked |
| `frontend/Claude outputs/quick-match-placement.png` | `2ca4fa25dcae9004e02cea144ce198efb8bfe6e8ca3dae81550f31f849beb153` | untracked |
| `frontend/Claude outputs/rail-header-separation.png` | `2550cd1862a80b9c430aac6fd2521dc370849c1cf7d13a5c0f56590133dd2e21` | untracked |
| `frontend/Claude outputs/rail-panel.png` | `de0620e760ee66b55b296080e243502b13a44f510b227e74f296ae51a1a437be` | untracked |
| `frontend/Claude outputs/regions-edit.png` | `013f3a72708f1ccf614c7211e610485ce11e3cd4360a44dce2733dfcfcf5c273` | untracked |
| `frontend/Claude outputs/rename.png` | `d1240eca55e1687a5dea93853aa27cf7e5167fc98e9f48a2cf5364ad9cb1b1b1` | untracked |
| `frontend/Claude outputs/rp0.png` | `81d00fabbfb1873154fa2b3b315363243efba398b424360f9ec8d0dc7b9c7077` | untracked |
| `frontend/Claude outputs/rp1.png` | `b9217639c36917c7424a276a592475b77c01239f166c97df5e19a27600420861` | untracked |
| `frontend/Claude outputs/rp2.png` | `7f2fa85d1c07b963aac7c19ff74403b38074dd89b1782ac29d9aa120e2c15c82` | untracked |
| `frontend/Claude outputs/rp3.png` | `8bbd2a7b269ae08b70990fe83a733602969e2247d89dca95caa532b6c669f914` | untracked |
| `frontend/Claude outputs/search-new.png` | `d7e553ed25bce5cfcb784aa1c015303022723cca091184ebd3326cb42d40d641` | untracked |
| `frontend/Claude outputs/settings-models-code.png` | `48b8950c68734127a5665a470f0a8e46f295ca86e0b08d0438248cc3c1faab82` | untracked |
| `frontend/Claude outputs/settings-models.png` | `cdcd6394cc7c81f3fc4927d867bbd37198946a091a7a475cb3f9553f6ad2595c` | untracked |
| `frontend/Claude outputs/settings-queue-code.png` | `1a2e28d5fe4d96b4d988459278b4454d5ebecf304930678dd8ef962eb816d4e1` | untracked |
| `frontend/Claude outputs/settings-queue.png` | `733d119b85cc9f41990695736b1905b265b1c74f83d93dd0e48ff877627ce4cf` | untracked |
| `frontend/Claude outputs/settings-sources.png` | `dcd1987ef38160f6fe033541a0aa943ec67386f76e3bf8730bf0e1a4e674dcf5` | untracked |
| `frontend/Claude outputs/sheet-layout.png` | `f84b80620c1239fe6e1d1c5060ba64396d982a7bae6294460469cc9749a07843` | untracked |
| `frontend/Claude outputs/sifty-logo-motion.html` | `65502f075b72a5969d72d8bcf31ae1d7b5bfd2a63a1ba5f5075736db13f99c51` | untracked |
| `frontend/Claude outputs/sifty-next-steps.md` | `836ffc5aff813899b54fc752002e5c403a74a415c4c17f7b682fa3e072f0a248` | untracked |
| `frontend/Claude outputs/skill-levels.png` | `18d1558fd083f265b9ed3ad9dce6daf9a69d50bc4b4555744e3dabd1162126ef` | untracked |
| `frontend/Claude outputs/skills-grey.png` | `c980a308d7ebb2cc164ccafe0c3d1ffb3e99fe31bece057b3c29e40fef16b178` | untracked |
| `frontend/Claude outputs/sort-toggle.png` | `bffc17aeed121770144cbe56b110e4754d8b51a91bd4aa5d48f8c8ed1c5e50f1` | untracked |
| `frontend/Claude outputs/switcher-menu.png` | `f3fa265f7e7d6f407fa559ab8898744f4c27d29f1c0e9adb34f9a1bada9a79f9` | untracked |
| `frontend/Claude outputs/tags-edit.png` | `2ad057ae041ac6a012af3c1683a48cbeb5f2ba2afd5c87e5da4ad2efc862d4a9` | untracked |
| `frontend/Logo.tsx` | `ab25507cec174472e17dd95757bfad12951d399302eca93cad07a2da3601ee31` | untracked |
| `frontend/theme.css` | `edcbddad545f2775717895db57f610d18632ccf48b00f7480604df7bb3e4f359` | untracked |
| `work/tasks/application-preparation/APP-8_expired-application-outcome/attention.md` | `6642b180d19411da0832fa216a4e8bb713eb0804ae90ff12bc71c2847451d173` | untracked |
| `work/tasks/application-preparation/APP-8_expired-application-outcome/log.md` | `004f4c0649ed81c39784e2432e6a1f311fe385f949d2ec76b6e912f3a2af2bd9` | untracked |
| `work/tasks/application-preparation/APP-8_expired-application-outcome/plan.md` | `304aca56523e17b747afd68898cdd1acb3312c95632ecdadec285f1ff1e5edea` | untracked |
| `work/tasks/assessment/ASM-5_shared-term-matching/description.md` | `2fb5a5c4fcb7bd9f3c6d2bbf81f783b21e632f5e034522f399fff185fe51ebbf` | untracked |
| `work/tasks/assessment/ASM-5_shared-term-matching/plan.md` | `5d88cb08aad299bb62ac11479b1f7fbca446d1a7692030b288c08ba22a1b6753` | untracked |
| `work/tasks/assessment/ASM-6_deterministic-quick-match/preparation-acceptance.md` | `f59c1595f434f4c9de2823c051e2a5a8b9395f961fb4f47454d50462bcf8f18e` | untracked |
| `work/tasks/collection/COL-7_production-refresh-and-assess/baseline-20261109T114700Z.json` | `dfd0ae237096e7572db1b6ac0779a808e73b5b76687eddcc58b2e0aa65943710` | untracked |
| `work/tasks/collection/COL-7_production-refresh-and-assess/description.md` | `7f4e95f79946f30a4218bcbd082a5ea6fb8b7329f5d03c218d6cd9c9ba69342d` | untracked |
| `work/tasks/collection/COL-7_production-refresh-and-assess/plan.md` | `c29460e16688bbe8562166bbbed25e34f80b5f7d4ae53dc4925d561b43dd1f1d` | untracked |
| `work/tasks/collection/COL-7_production-refresh-and-assess/run-7.1-start.json` | `1fc341af223cfc64e4c20399e7f72ed355f49237f04e150ee1073a77beafe79d` | untracked |
| `work/tasks/collection/COLLECTION-1_publication-date-and-age/attention.md` | `88e43087368dfec423c12e9aca88800052ad44489131c3794dc745de71f6cb25` | untracked |
| `work/tasks/collection/COLLECTION-1_publication-date-and-age/log.md` | `d4f4442f81ba73b38688484021e52cbd45d3dd6c8a9929400a47f02e91c50579` | untracked |
| `work/tasks/collection/COLLECTION-1_publication-date-and-age/plan.md` | `65d1b854029062dc4e0be237c269ee25df0bd8a845ea5e43434a717f22f64683` | untracked |
| `work/tasks/design/DESIGN-2_separate-presentation-and-logic/attention.md` | `e8c6b4c2e30c5f794429c42940d278f9453eaf674671b2c8d5d4ed66de9536a2` | untracked |
| `work/tasks/design/DESIGN-2_separate-presentation-and-logic/comments.md` | `44d12105d90604aa37439776723c7e88424dc5a45e7e36fbbe6717b93a67cd7b` | untracked |
| `work/tasks/design/DESIGN-2_separate-presentation-and-logic/description.md` | `c4e1d20e925ffbf8173a118be5a4d82d9aa97ca1e35aabc6dbcc17d208ff30f7` | untracked |
| `work/tasks/design/DESIGN-2_separate-presentation-and-logic/log.md` | `21503da319fadf9968d421bbe290ba929b8f4b94d0bbc6a6c0e5539f3a389cba` | untracked |
| `work/tasks/design/DESIGN-2_separate-presentation-and-logic/plan.md` | `774d5e53489eaf724719d8ee865dcce632be273f6671a4187fd42efa6bdce10a` | untracked |
| `work/tasks/design/DESIGN-3_reusable-design-blocks/attention.md` | `f2f81b56d2fb6bbd6562d84d717f66fecdbde0dd417c12375dfe976c789f9ba4` | untracked |
| `work/tasks/design/DESIGN-3_reusable-design-blocks/log.md` | `2c5f07f3547c9f9b481c855394598815b6c0b80e0af22bd03504b5491e031f19` | untracked |
| `work/tasks/design/DESIGN-3_reusable-design-blocks/plan.md` | `5874a9ed50f86b85dcbbf435b6d271e6c89094a2e13a712e7bbb859ed2738420` | untracked |
| `work/tasks/inbox/INBOX-1_search-catalogue-overview-quick-match/plan.md` | `983295df01f8d70db09425f31bf5af5925d8bd93d6019d49444add50c3cf5370` | untracked |
| `work/tasks/settings/SETTINGS-2_profile-workflow-and-fields/attention.md` | `7c8744f9fbd411c8773b9188f7fa68e67fdcb1c1345a9580353795703d15a622` | untracked |
| `work/tasks/settings/SETTINGS-2_profile-workflow-and-fields/description.md` | `93903f08a86ae49e10d71701bb301edac12bce3af6800c37797b18db33348daa` | untracked |
| `work/tasks/settings/SETTINGS-2_profile-workflow-and-fields/log.md` | `1919e8eb57dd7d61f84b350af5ddd5b13c5988298e71d46f0be83896a5332fc6` | untracked |
| `work/tasks/settings/SETTINGS-2_profile-workflow-and-fields/plan.md` | `f258e849f394c8952646212f1d6a0977a27383d196d0147e41fcb2218ae92dfd` | untracked |
| `work/tasks/settings/SETTINGS-3_profile-fields/attention.md` | `df2fb155a4fcbb3932b816e9e2de395f875a0312a2cc71bf0217d9daee5200bb` | untracked |
| `work/tasks/settings/SETTINGS-3_profile-fields/description.md` | `0d4dc85b86ae43f49494948c843621e37f167d32fcc645e67cb154517da229f3` | untracked |
| `work/tasks/settings/SETTINGS-3_profile-fields/log.md` | `3c5ef7b7839a7598294ddd875757785665847781a6e1d2f8e8a846e2c02595f5` | untracked |
| `work/tasks/settings/SETTINGS-3_profile-fields/plan.md` | `cc91f7a1097938cd6c7b7fbc077f6907b94ad5703af3345e3e1a2fbfe194d462` | untracked |
| `work/tasks/settings/SETTINGS-4_profile-url-sources/description.md` | `12bd3112f2886d573defc8ef83a12adfabd571251491f867f4c989492ed244ec` | untracked |
| `work/tasks/settings/SETTINGS-4_profile-url-sources/plan.md` | `a6a3c80260fa30783be4b1a5f678177e2de9610397bc1487e7fce736abe4333c` | untracked |
| `work/tasks/settings/SETTINGS-5_user-data-and-role-profiles/plan.md` | `ca7cfaf9a4459c102e3d43bf1946f4de7e70da8a412b21f01f2103dbc2c4b89a` | untracked |
