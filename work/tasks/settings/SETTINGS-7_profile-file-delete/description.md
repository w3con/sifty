# SETTINGS-7 — Profile files cannot be deleted

**Epic:** settings
**Created:** 2026-11-12
**Status:** backlog
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
