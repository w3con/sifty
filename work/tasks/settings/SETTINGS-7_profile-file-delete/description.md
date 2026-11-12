# SETTINGS-7 — Profile files cannot be deleted

**Epic:** settings
**Created:** 2026-11-12
**Status:** in progress
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Баг - не работает удаление файлов. Они не удаляются, а надпись DELETE - белая на белом.

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator diagnosis (2026-11-12)

- **Nothing is deleted because the server has no such route.** The profile view calls `DELETE /api/profile/documents/{id}` (`frontend/profile/api.ts:47`); `backend/internal/server/profile_api.go` registers GET, `/text`, `/file`, `/analyze`, `/cancel`, `/retry` and `/facts` for a document, but no DELETE. The request fails and the file stays (most likely lost in the move from the TypeScript backend).
- **White on white:** `frontend/profile/profile.css` gives `.profile-doc-confirm-delete` a literal red background with `!important`, so another rule wins or the button is rendered elsewhere; to be found in the browser. By the design system a destructive action uses the `danger-*` tokens, not literals.

## Proposed behaviour (to confirm with the owner at planning)

Deleting a file removes the stored file, its extracted text and its proposals that were not accepted; facts the owner already accepted stay in the profile (they are facts, the profile is a fact store, spec quick-match §15.1) and show that their source file was deleted. One confirmation, as today. The server answers with what was removed.

## Estimate (preliminary)

Complexity 3, AI time 30–45 min: the DELETE route through the existing profile service, one targeted test (accepted facts survive, pending proposals and the file are gone), the button colour with tokens, a browser check. Small enough for a parallel agent: it touches profile code only, not the INBOX-2 paths.

## Plan

**Short card:** steps and acceptance criteria live here. Authorised by the owner on 2026-11-12 ("выполни bin/work-handover SETTINGS-7, затем исправление по description.md карточки"). The proposed behaviour above is taken as agreed; the owner named these parts: the delete route, one test that accepted facts survive, the button colour through tokens, a browser check with Playwright on a copy of the data (not on `data/`). Production is neither built nor started.

## Steps

- [x] 1.1 — Add `DELETE /api/profile/documents/{id}` through the existing profile service: it removes the stored file, its extracted text and its unaccepted proposals, keeps accepted facts and marks their source as deleted, and answers with what was removed; check: `go test ./internal/profile/... ./internal/server/...` passes.
- [x] 1.2 — Add one test in `backend/internal/profile` that proves accepted facts survive while the file and the pending proposals are gone; check: the test fails without the service change and passes with it.
- [x] 1.3 — Find why the delete confirmation label is white on white and restyle `.profile-doc-confirm-delete` in `frontend/profile/profile.css` with the `danger-*` tokens, no literal colours and no `!important`; check: the label is readable in light and dark themes.
- [x] 1.4 — Check the whole flow in a browser through Playwright against a copy of the data directory (never `data/`) on a free port: confirm, file disappears, accepted facts remain; record the result in the card's comments; check: the run finishes and the copy is removed afterwards.

## Acceptance criteria

- `DELETE /api/profile/documents/{id}` removes the file, its extracted text and its unaccepted proposals; accepted facts stay in the profile and show that their source file was deleted.
- One automated test covers that behaviour and the Go suites for the touched packages pass.
- The delete button label is readable, coloured only through design tokens.
- The browser check ran on a copy of the data, and `data/` and `data-staging/` are byte-identical to before.
- Production was not built, started or deployed.

## Must not touch

- The INBOX-2 card, its worktree `.worktrees/INBOX-2` and any file it changes.
- `data/`, `data-staging/`, `bin/sifty`, `bin/sifty-staging` and the production port 8080.
- Anything outside profile code (backend profile service and its API, the profile view and its stylesheet, the new test).
