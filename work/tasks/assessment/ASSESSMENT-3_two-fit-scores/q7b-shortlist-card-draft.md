# DRAFT: Shortlist re-check on a stronger model (not registered, not approved)

Draft for the owner. Nothing here is built. It becomes a card only after the owner reads it and the coordinator registers it.

## What this is

When the owner puts a vacancy on the shortlist, Sifty runs both model passes again for that vacancy on a model the owner chose for this purpose (Haiku 5.5 is the recommendation from the Q.7b measurements: 52 of 52 expected requirements, 27 of 27 conditions) and recomputes both scores. The first line (every collected vacancy) stays on the local model; the stronger model is spent only on the few vacancies the owner has already selected.

## Why

The local models find about 80 percent of the expected requirements and invent matches on shared words. A hosted model costs about 0.1 to 0.2 US cent per extraction and is slow (about 21 s per vacancy), which is acceptable for a handful of vacancies a day and not for 100 to 1000 per run.

## Behaviour

- Trigger: a vacancy enters the shortlist. A manual "re-check" action may also exist.
- Pass 1 again: extraction on the shortlist extraction model (the vacancy text leaves the machine only if that provider is hosted; this is the owner's choice of provider, there is no data rule).
- Pass 2 again: matching on the shortlist matching model, over the new requirements.
- Result: stored beside the first-line result, never replacing it silently; the view says which model produced the numbers and shows the first-line numbers next to them when they differ.
- Both scores are recomputed by the existing pure evaluator; no new scoring rule.
- Configuration: two new optional stages in `providers.yaml` (for example `shortlist_extraction`, `shortlist_matching`), falling back to the first-line stages when absent. The cost guard of section 14.7 applies.

## Must not touch

Production on port 8080, merge to main, push, the first-line queue and its ordering, the detailed assessment queue and its switch, profile data and revisions, workflow and Apply history, process files, other worktrees.

## Acceptance

1. A vacancy added to the shortlist gets numbers computed by the shortlist models, labelled with the model name.
2. Removing it from the shortlist does not delete the first-line numbers.
3. With the shortlist stages unset, nothing changes.
4. A hosted provider without the cost permission is refused with the usual message.
5. Verified in a real browser on a review copy; tests for the stage fallback and the stored-result separation.

## Checkpoints

- [ ] S.1 Add the two optional stages and their fallback to the configuration and validation.
- [ ] S.2 Store a shortlist result per vacancy beside the first-line result (keyed by the model names and the description hash).
- [ ] S.3 Run both passes for a vacancy on the shortlist stages and recompute both scores.
- [ ] S.4 Trigger on shortlist entry and a manual action; keep the first-line queue untouched.
- [ ] S.5 Show the model name and the first-line numbers next to the shortlist numbers in the panel.
- [ ] S.6 Update section 14 of the quick-match specification and measure on ten real shortlist vacancies.

## Open questions for the owner

- Which hosted model is the shortlist default, and is a subscription route (CLI) acceptable for it?
- Should a shortlist re-check run automatically or only on request?
- Is the stronger model allowed to see profile facts (the owner has said there is no data rule; this only records it as a choice).
