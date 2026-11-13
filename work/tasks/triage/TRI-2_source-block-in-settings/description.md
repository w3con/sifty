# TRI-2 — Block source from Settings, not from Triage

**Epic:** triage
**Created:** 2026-11-05
**Status:** founded (plan.md before execution)

## Why this work exists

TRI-1 let the owner block a source from the per-role "Block…" control in Triage. The owner wants source blocking to be a Settings action instead: a source is blocked once, not from whichever role happens to be open. This card removes "Block source" from Triage (leaving only "Block company") and adds the ability to create a source block in Settings, which TRI-1 did not provide — Settings can list and remove block rules but not create them, so removing the Triage button without this would leave no way to block a source.

## Must not touch

The TRI-1 block-matching semantics (company by normalised name; source-blocked only when every source of a role is blocked). The stop-word work (COL-6). The dedup, stage, assessment and collection paths. The navigation restructure (separate card).

## Notes

A source block created from Settings is not tied to a job, so it needs a direct create endpoint (`POST /api/block-rules {kind,value}`) and a `GET /api/sources` list; `GET /api/block-rules` and `DELETE /api/block-rules/{id}` from TRI-1 are reused.
