# SETTINGS-6 — Edit a source URL in Settings › Sources

**Epic:** settings
**Created:** 2026-11-11
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
еще надо сделать иконку карандаша рядом с url - чтобы можно было поменять

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Context (coordinator, 2026-11-11)

Came up when the Web3 Careers source turned out to have a typo in its address (`web3.careers` instead of `web3.career`), which silently stopped that source for two weeks; the coordinator fixed it by hand in the three `search.yaml` files.
Today: `PATCH /api/search/sites/{name}` only toggles `enabled`; GET/POST/DELETE exist. Settings › Sources shows the URL read-only.

## Proposed behaviour

- A pencil icon next to the URL in Settings › Sources turns it into an input with Save / Cancel (Enter / Escape).
- The server validates: absolute http(s) URL, not a duplicate of another source; on save it writes `search.yaml` through the existing configuration save (with its backup) and answers with the stored site.
- The source keeps its name, so collected vacancies, sightings and history stay attached to it.
- After saving, a quick reachability check (one HEAD/GET with the normal timeout) shows a plain note: reachable, or "did not answer — it will be tried at the next collection"; it never blocks the save.
- Separation: validation and saving in the server handler and config layer; the view only edits and shows the answer.

## Estimate (preliminary)

Complexity 3, AI time 30–45 min (extend PATCH, one targeted test for validation and that the name/history stay, the input in Settings, a browser check).
