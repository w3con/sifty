# Q.7b brief — larger local model trial (from coordinator, 2026-11-11)

Start from a clean context (`/clear`), read AGENTS.md, this card and spec §14.12. Q.7 is accepted. Run only after the owner approved the download in chat.

1. `ollama pull qwen3.5:35b-a3b` (MoE, about 3B active parameters; about 23 GB). Do not change the default model in `config/providers.yaml` of the repository; add the candidate as a separate provider entry and select it for the `extraction` stage only in the review copy's `providers.yaml`.
2. Extraction: re-run the Q.3b regression set (6 vacancies, `expected.json`) with the candidate, 2 runs; same table as `q3b-regression-results.md`.
3. Matching: re-match the 43 vacancies of the Q.6 set with the candidate (extraction from 9B kept, so only the matcher differs), twice, to measure stability (same verdict in both runs).
4. Report next to the 9B numbers: verdict shares, You fit of the unrelated roles and of Parity / Ethena Labs / Growth Lead / Product Designer, the hand-check items from Q.7 that 9B got wrong (fixed or not), share of identical verdicts across the two runs, time per vacancy (extraction and matching), memory used, projected backfill.
5. No code change unless a provider option is needed to run it; spec §14.12.5 gets the measured result, marked as a trial.

Stop with the report; the choice of model is the owner's.
