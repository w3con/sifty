# OPS-4 — attention

- [x] The owner subsequently authorized «6.1 - доделай». Continuous preview is now running from the main checkout on 8081 and ready for owner review. The general path/configuration refactor remains the separate OPS-7 task; 6.2–6.3 are still deferred and automated suites remain disabled.

## Active coordination

Final approval supersedes the earlier release hold: owner said «принимай и закрывай». Complete 7.1/7.2 under the prototype test policy, then execute the normal pinned-source squash/release path with explicit `--adopt-legacy` for this first production transition. No old-history rewrite or new feature work is authorized. All mechanics are assigned to the executor; current final-stage stop boundary 23:35 CET.

Current approval: continue 6.2 and 6.3 only, using the continuation scope/estimate in plan.md. 6.2 is technically accepted; 6.3 is implemented and awaiting coordinator review. Live production remains owner-gated and still needs the explicit `--adopt-legacy` first transition; no full suites, existing-history rewrite, generalized path refactor or enforcement plugin is included.

The sole coordinator is `ses_e4ed90c39ffeVGdkVhVpkwmHD2` (this live session). On the owner's instruction «сам разберись», it reconciled the duplicate coordination. Session `ses_e4e6d31c5ffe0YJzYvSwzbVexM` completed with `finish: stop` at 17:36:14, and its resumed executor `ses_e4e70ff06ffeR5KzHXIu5fgaUm` completed at 17:35:06. Neither needs killing or restarting. Do not resume the secondary coordinator or its executor while the recorded coordinator is active. Preserve both histories. All subsequent stages are dispatched only by the recorded coordinator after checking the previous writer stopped.

The owner approved the complete plan, including cleanup of obsolete directories after verified relocation. No unanswered product question blocks execution. The separate data migration remains a backlog item rather than authorization to move live data.

The live session now exposes the configured `executor` subagent. The owner reaffirmed «продолжай. deepseek используется через opencode go». Dispatch remains restricted to `opencode-go/deepseek-v4.1-flash`; verify inference metadata after the initial visible invocation before closing step 1.2. No unanswered attention item blocks handover.
