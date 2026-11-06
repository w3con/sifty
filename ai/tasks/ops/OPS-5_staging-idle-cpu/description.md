# OPS-5 — High staging server CPU while the owner is idle

**Epic:** ops
**Created:** 2026-11-06
**Status:** backlog; reported, not investigated

## Why this work exists

Owner report: «sifty фронтенд жрет 45% цпу даже когда ничего не происходит. баг?»

The supplied Activity Monitor screenshot at 16:04:20 shows `sifty-staging` at 45.3% CPU, 21 threads and 165 idle wake-ups. This is the Go staging server process, not a browser frontend process. Owner inactivity does not establish whether collection, assessment or other background work was running. The cause and duration remain unverified.

## Preparation needed

Reproduce and measure sustained server CPU with background-job state recorded, distinguish normal collection/inference work from an idle loop or excessive UI polling, and prepare the smallest corrective plan. Do not modify runtime data or stop live work merely to diagnose it. No implementation or task switch is approved by this report.

## Acceptance criteria

- The load is explained with reproducible process/background-task evidence.
- If a defect is confirmed, a separately approved fix removes idle CPU waste without interrupting collection or assessment.

## Evidence

The owner supplied a local Activity Monitor screenshot named `Screenshot 2026-11-06 at 4.04.20 PM.png`. The image is not copied into this public repository. The visible server measurement is transcribed above; unrelated process data is omitted.
