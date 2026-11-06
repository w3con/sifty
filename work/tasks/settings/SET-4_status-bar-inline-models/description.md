# SET-4 — Taller status bar with inline model selection

**Epic:** settings
**Created:** 2026-11-04
**Status:** founded
**Short card:** steps and acceptance criteria live here.

## Why this work exists

The bottom status bar now shows the live assessment and application-preparation models with labels and icons, read from the running session config. The owner wants two improvements: give the bar a little vertical height and padding so the text is not cramped, and let the assessment and application models be changed directly in the bar without opening Settings. The extraction stage stays in Settings only (it is a rarely used fallback). Mobile width is not a constraint for this product.

The agreed interaction: the shown model text next to each icon is itself the control — clicking "qwen3.5-9b · qwen3.5:9b" opens a selector of configured providers; choosing one sends the stage change, the config reloads live, and the bar and the worker pick up the new model immediately. No separate labelled dropdowns are added to the bar.

## Notes

- The bar is `app/web/main.tsx` (`.queue-status` / `.queue-provider`); the live models come from `GET /api/stage-providers`, already polled. Changing a stage is `PUT /api/stage-providers`, which reloads the in-memory config live.
- The uncommitted status-bar rework from this session (live models, labels, target/pencil icons, fixed running suffix) is the starting point.
- UI code under `app/` (or `sifty/` after GO-5) follows the no-card-identifier rule in comments.

## Steps

- [ ] 1.1 — Give the status bar vertical height and padding so the text has breathing room; check: the bar is visibly taller with inner padding and nothing is clipped.
- [ ] 1.2 — Make the assessment model text in the bar a selector of configured providers that, on choice, sends the stage change and reflects the new model live; check: picking a different model in the bar changes `stages.assessment` and the bar shows the new model without a page reload.
- [ ] 1.3 — Do the same for the application-preparation model; check: picking a model in the bar changes `stages.preparation` live.

## Acceptance criteria

- The status bar is taller with padding.
- The assessment and application-preparation models are changeable directly in the bar by clicking the shown model, with the change applied live via the existing stage-provider endpoint; extraction remains Settings-only.

## Must not touch

- The Settings stage-model section (it stays as the full control, including extraction).
- The assessment and preparation pipelines.
