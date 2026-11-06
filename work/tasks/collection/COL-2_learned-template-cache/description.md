# COL-2 — Learned per-host extraction template cache

**Epic:** collection
**Created:** 2026-11-04
**Status:** founded (reserved; not scheduled)

## Why this work exists

COL-1 built the deterministic extraction waterfall (JSON-LD, then a platform adapter, then generic HTML) and left a named, behaviourless seam for a learned fallback: `internal/source/hint.go` defines `ExtractionHint`, `HintValidator` and `HintStore` with no behaviour. This card fills that seam. When deterministic extraction cannot get the required fields from a host, a model learns a reusable per-host extraction hint once (a selector or template), the hint is cached, validated on every use, and invalidated the moment it stops producing the required fields, which triggers re-learning. The extraction unit is the host/platform, not the page, so the model is used once per host rather than on every posting.

This keeps the model off the common path and reserved for the long tail, as the project decided. It depends on nothing new from COL-1 beyond the seam already in place.

## Must not touch

- The deterministic waterfall and adapters COL-1 built; this adds a fallback behind them, it does not change their order or their exact-quote validation.
- The public-subtree rules for `sifty/`.

## Notes

- The seam to fill: `internal/source/hint.go` (`ExtractionHint`, `HintValidator`, `HintStore`).
- Model extraction is gated by the existing `model_extraction` config flag, which stays off by default.
- Not scheduled. This card reserves the number and records the design so COL-1's references to "the learned-extraction seam" have a home. Write `plan.md` before execution.

## Steps

- [ ] To be defined in plan.md.

## Acceptance criteria

- To be defined in plan.md.
