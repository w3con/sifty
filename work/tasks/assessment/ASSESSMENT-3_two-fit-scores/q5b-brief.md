# Q.5b brief — small panel fixes (from coordinator, 2026-11-11)

Q.5 is accepted with these fixes. Keep the scope this small.

1. Counts belong to You fit. Put "N match · N partial · N no · N unclear" directly under the You fit number (its own column), not across both columns, so it reads as the breakdown of You fit. In "Matches", the goal lines (Role, Contract type, …) and the requirement lines must not make the header count look like it contradicts the counts line: show goal lines first and separated by a thin divider, or give them their own small heading — pick the quieter option and show it in a screenshot.
2. Alternatives: add `alternatives` to FitLine (backend, omitempty; `matchcontract/fit.go`, `quickmatch/fit.go`) and show them in the panel, e.g. "Backend language (Kotlin · Python · Go)". Test it.
3. Retake `panel-22-before-experience.png` (it is a byte copy of the "after" image) or delete it and say so in q5-panel.md.
4. Block `POST /api/search` in any browser session against a copy from now on.

`npm run build`, the frontend tests and the touched Go packages must pass. Stop after Q.5b; then Q.6 per plan.md plus: implement a lock shared by every Sifty process on the machine for model calls (prod, staging and review servers use the same Ollama), with a test.

Spec: reflect items 1 and 2 in `docs/specs/quick-match.md` §14.4 (what the panel shows and where the counts belong), marked as implemented on the branch, not released. Every following step updates the spec in the same commit as the behaviour it changes.

Context: this brief is self-contained. Start from a clean context (`/clear`), read AGENTS.md, then this card (description, plan, attention, log, this brief).
