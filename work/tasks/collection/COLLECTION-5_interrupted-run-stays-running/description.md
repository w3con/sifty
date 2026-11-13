# COLLECTION-5 — An interrupted collection run stays 'running' after a server restart

**Epic:** collection
**Created:** 2026-11-13
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Finding by the INBOX-2 coordinator during the release on 2026-11-13 (not the owner's words): the daily collection that started 18:27 was interrupted when the server stopped for the release; collection_runs row 21 has no finish time and the server keeps reporting the run as running, because startup recovery exists only for the assessment queue, not for collection runs. Expected: on startup an unfinished run is closed as interrupted, and the next run can start.
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.
