# OPS-4 — attention

## Active coordination

The sole coordinator is `ses_e4ed90c39ffeVGdkVhVpkwmHD2` (this live session). On the owner's instruction «сам разберись», it reconciled the duplicate coordination. Session `ses_e4e6d31c5ffe0YJzYvSwzbVexM` completed with `finish: stop` at 17:36:14, and its resumed executor `ses_e4e70ff06ffeR5KzHXIu5fgaUm` completed at 17:35:06. Neither needs killing or restarting. Do not resume the secondary coordinator or its executor while the recorded coordinator is active. Preserve both histories. All subsequent stages are dispatched only by the recorded coordinator after checking the previous writer stopped.

The owner approved the complete plan, including cleanup of obsolete directories after verified relocation. No unanswered product question blocks execution. The separate data migration remains a backlog item rather than authorization to move live data.

The live session now exposes the configured `executor` subagent. The owner reaffirmed «продолжай. deepseek используется через opencode go». Dispatch remains restricted to `opencode-go/deepseek-v4.1-flash`; verify inference metadata after the initial visible invocation before closing step 1.2. No unanswered attention item blocks handover.
