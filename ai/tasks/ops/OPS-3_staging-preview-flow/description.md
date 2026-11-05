# OPS-3 — Staging previews in-progress work; prod stays main

**Epic:** ops
**Created:** 2026-11-05
**Status:** founded (plan.md written; awaiting "Execute the plan")

## Why this work exists

The deploy loop built in OPS-1 does not match the owner's model of the two environments. The owner's model is: the staging environment (port 8081) shows work in progress, before it is merged, so he can look at a change and decide; the production environment (port 8080) shows only what is merged into the main line, updated on his explicit word. OPS-1 built it differently: the acceptance command merges the change into main first and only then deploys it to staging, so staging shows already-merged code rather than work under review; and both environments share one compiled binary and one frontend bundle, so they cannot show different code at the same time. The result is that the owner cannot preview a change on 8081 while 8080 keeps showing the accepted version, which is the whole point he asked for. This card corrects the loop to his model without breaking the work-management harness.

## Must not touch

The application's behaviour (collection, scoring, serving). The work-management harness's one-card-per-conversation gate and its merge-to-main-on-finish contract, beyond removing the incorrect automatic staging deploy. The coordinator-gated public-repo push (GO-1). Production is updated only on the owner's word, never automatically.

## Notes

The two environments must be able to run different code at the same time, which requires each to own its compiled binary and its frontend bundle rather than sharing them. Staging builds and serves whatever checkout it is run from, so the coordinator previews a feature branch on 8081 (from that branch's working copy) while production keeps serving main on 8080. Finishing a card merges it to main as the harness already does, but no longer force-deploys staging; production moves to the new main only through the promote command on the owner's confirmation.
