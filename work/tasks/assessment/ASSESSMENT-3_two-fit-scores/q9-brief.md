# Q.9 brief — close-out of ASSESSMENT-3 (from coordinator, 2026-11-11)

Start from a clean context (`/clear`), read AGENTS.md, this card (log last entry first) and spec §14.13–14.14. Research mode is over for this card: matching quality continues in ASSESSMENT-5, the shortlist in ASSESSMENT-6, sections in INBOX-2. Do not start any of them.

1. **Checkpoint.** Commit the uncommitted work as it is (engagement check, matching stage and `think` option, removal of the local-only data rule, diagnostic live tests, `scripts/model-bench/`, spec §14.13, card files), in one or two commits with clear subjects.
2. **Matching off, You fit hidden** (spec §14.14): `quick_match.model_passes.matching` in configuration, default `false`; with it off the worker runs extraction only and never matching; the panel shows "Requirements" with ✓ / ✕ and the code-decided state only; no You fit number or counts line in the panel or the catalogue; Fits you unchanged. With it on, today's behaviour. One targeted test: with matching off the worker makes no matching call. Spec §14.14 marked implemented on the branch in the same commit.
3. **Review copy:** restart 8094 on the build with matching off; screenshots of Parity, Ethena Labs and one unrelated role; the catalogue row.
4. **Reconcile** spec §14 statuses (agreed / implemented on branch / released), `docs/MAP.md` for the quick-match requirement, and the attention file (open items that moved to other cards are marked moved, not done).
5. **Report** in the usual format and stop. The coordinator accepts and asks the owner's approval for the finish (squash to main and production release).
