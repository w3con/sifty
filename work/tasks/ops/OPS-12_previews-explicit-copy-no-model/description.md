# OPS-12 — Card previews: explicit port and fresh data copy, no background model wo

**Epic:** ops
**Created:** 2026-11-13
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Coordinator proposal, 2026-11-13, after the owner noted that several sessions compete for the local model and retired staging («Меня смущает, что когда у нас несколько сессий - они начинают конфликтовать за локальную модель»; «убирай стейджинг»): (1) bin/sift-preview still defaults to the retired staging port 8081 and data-staging — make it require --port and a fresh copy of production data, or make the copy itself; (2) a preview server runs Sifty's background model work (extraction, fit) on its copy and queues on the machine-wide model lock with production — start previews with model work off unless the card's check needs it.
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.
