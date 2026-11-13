# DESIGN-2 — Separate frontend presentation from application logic

**Status:** backlog
**Kind:** task
**Created:** 2026-11-07
**Requirement:** R-DESIGN, R-SEARCH, R-TRI, R-SET, R-APP

## Owner request

```text
мне не надо объяснять. займись рефакторингом. этот код человеку не удобно читать
```

```text
и всех остальных .tsx

задача - отделить дизайн от логики
```

## Observable outcome and use

The owner and anyone changing the frontend can find and edit a screen's layout and styling without reading network requests, persistence, polling or workflow mutations. Every current `frontend/*.tsx` is covered. Small presentation components receive explicit data and callbacks; domain state and effects live in feature hooks/controllers, and static styles live in CSS. The task demonstrates this separation on the running staging app and in a review of the files a designer would edit for the header, source table, settings panels and answer editor.

## Scope

Refactor `main.tsx`, `Collection.tsx`, `Settings.tsx`, `RoleContent.tsx`, `ApplicationDraft.tsx`, `ApplicationConfirmation.tsx`, `DeleteRole.tsx`, `CatalogueFilters.tsx`, `Sources.tsx` and `Icon.tsx`, with their necessary new components, hooks, types, helpers and styles. Preserve the current appearance, text, endpoints and behavior. The coordinator authored the proposed implementation boundaries in plan.md; the owner's request above establishes the goal, not approval of that unwritten plan.

## Exclusions and preservation

No redesign, new user features, backend/worker changes, new UI framework, global state library, token-system project, legacy frontend rewrite or fixes for unrelated existing defects. Do not alter user data or production. Preserve the existing main-checkout changes in `ApplicationConfirmation.tsx` and `application.css`, and the separate design-token worktree. Do not resume or overwrite that worktree. Reconcile the scoped existing formatting without resetting, stashing away or discarding the owner's working copy.

## Estimate

- **Complexity:** 8 — ten coupled TSX files include shared catalogue state, polling/write ordering, modal focus, and editable application drafts. One task is appropriate because the promised outcome is a consistent presentation/logic boundary across the frontend; implementation is divided into separately reviewed stages.
- **AI preparation:** 10 minutes, including inspection and the complete plan.
- **AI implementation:** 75–115 minutes, including executor context loading, bounded stages, builds, staging setup and expected corrections.
- **AI acceptance:** 10–25 minutes for coordinator review, staging evidence and correction review.
- **Total AI elapsed time:** 95–150 minutes, excluding owner response time. Assumes the configured executor and local build tools work; external tool/provider delays are additional and must be reported. The upper bound is a stop boundary, not authorization to continue automatically.
- **Monetary/external costs:** executor uses the existing OpenCode Go subscription. Exact attributable model cost is unknown. No new paid services or dependencies are planned; dependency download delays are external.

## Approval

The owner approved the complete presented four-stage plan and 95–150 minute estimate on 2026-11-07 with «запускай». This authorizes bounded implementation and staging review, not production release. The coordinator authored the plan. Next action: delegate worktree preparation and obtain the lifecycle brief, then dispatch step 1.1 only.
