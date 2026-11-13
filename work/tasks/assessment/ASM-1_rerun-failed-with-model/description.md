# ASM-1 — Re-run failed assessments with a chosen model

**Epic:** assessment
**Created:** 2026-11-04
**Status:** founded (deferred by the owner on 2026-11-04 — keep, schedule when there is more time)
**Short card:** steps and acceptance criteria live here.

## Why this work exists

Assessments fail for reasons that a different model can fix: a small local model may return malformed JSON or run out of context, where a larger one succeeds.

Correction (2026-11-04): failed assessments are NOT stuck — they are already retried automatically. The worker claims rows `WHERE status IN ('pending','failed') AND next_at<=now` (`NextPendingQueueRow`), and a failure is recorded with an exponential-backoff `next_at` (`MarkQueueFailed`), so a failed row is re-claimed and retried once its backoff elapses; a server restart also resets interrupted rows to pending. Observed live: 19 failed fell to 3 after a restart, as transient failures cleared on retry. But the automatic retry always uses the same model that failed, so it fixes transient failures (timeout, one-off bad JSON) and never the systematic ones (a model too small in context, or consistently malformed output on a given posting) — those keep failing on every backoff retry.

What this card adds over the existing auto-retry: a direct owner action to re-run the stubbornly-failed assessments on an explicitly chosen (larger) model, without changing the stage model of the whole queue. The value is the model choice for the hard failures, not retrying per se.

## Notes

- Failed rows are in the assessment queue with `status='failed'` and an error. The queue row already carries a `provider`; the worker uses the row's provider (or falls back to the stage model when it is empty). The automatic backoff retry reuses that same provider, which is why a chosen override is what this card adds.
- A per-retry provider override already exists on the row (`override_provider`); this card surfaces it as a user action over the set of failed jobs, with a model picker.
- The error text is worth showing so the owner sees why they failed before choosing a model.
- Relevant code: `internal/background/assessment.go` (provider selection), the assessment-queue store methods, and the queue/catalogue UI.

## Steps

- [ ] 1.1 — Add an action to re-enqueue the failed assessments with a chosen provider applied as the retry override; check: triggering it moves the failed jobs back to pending with the chosen provider, and they are assessed on that model.
- [ ] 1.2 — Present the failed count and let the owner pick the model for the retry (from configured providers); check: the UI shows how many failed and which model the retry will use before running.

## Acceptance criteria

- The owner can re-run the failed assessments with an explicitly chosen model, without re-running the whole queue, and sees how many will be retried and on which model.

## Must not touch

- The success path of the queue and the stage-model configuration (the retry override is per-run, it does not change the configured stage).
