# OPS-4 — discussion and research

## Executor candidates — 2026-11-06

The following are candidates, not measured quality rankings. No executor or paid model trial was launched while preparing this plan. The installed OpenCode catalog lists these models; stored provider names indicate existing OpenAI and OpenCode Go authentication, not verified remaining quota or permission to invoke each model.

| Candidate | Proposed use | Published standard input/output USD per million tokens |
| --- | --- | --- |
| DeepSeek V4.1 Flash | Initial bounded implementation trial, including a focused Go change | OpenCode Go: 0.15 / 0.60 off-peak; 0.30 / 1.20 peak |
| GPT-6 Luna | Lower-cost alternative to evaluate if the initial executor is unsuitable | 0.10 / 0.50 up to 272K input; 0.20 / 0.75 above |
| GPT-6.1 Terra | Escalation candidate for implementation that needs more review/corrections | 1.00 / 5.00 up to 272K input; 2.00 / 7.50 above |

The coordinator's GPT-6.1 Sol standard rate is 2.00 / 10.00 up to 272K input, and 4.00 / 15.00 above. These are provider rates, not a prediction of the owner's bill: OpenAI is connected through OAuth and OpenCode Go accounts usage against subscription allowances. Cached input changes the comparison. The OpenCode catalog reports zero cost for its OpenAI entries, so those metadata do not establish that inference is free.

The first trial should implement one already-needed, bounded stage rather than create a separate benchmarking project. Review the actual diff, relevant Go tests (including race detection where applicable), compliance with the agreed scope and total effort including coordinator review and corrections. Escalate an unsuitable executor rather than repeating failed runs indefinitely. The current session's Task interface has no per-call model argument; project agent profiles must provide selection through a supported, visible agent dispatch path. OpenCode loads agent configuration at session start, so enabling a new profile requires a session restart.

Official sources inspected:

- https://opencode.ai/docs/go/ — models, subscription allowances and DeepSeek peak/off-peak rates. Availability can also depend on the workspace's enabled regions.
- https://developers.openai.com/api/docs/pricing — current standard OpenAI prices and the 272K input boundary.
- https://api-docs.deepseek.com/ — the direct API currently uses `deepseek-flash`, served by DeepSeek V4.1 Flash; this differs from OpenCode Go's `deepseek-v4.1-flash` model identifier.

## Provider comparison before the owner's choice

The owner asked to compare OpenCode Go, direct DeepSeek and Kimi. Go currently starts at USD 10/month and includes multiple models with separate usage allowances and five-hour/weekly limits; it is useful for trying executors when an existing subscription is active. Direct DeepSeek has published the same Flash peak/off-peak token rates and no monthly subscription requirement. For illustration, 10M uncached input tokens plus 1M output tokens cost USD 2.10 entirely off-peak, or USD 4.20 entirely at peak, before tax and excluding caching. This is not an estimate of this task's usage. Go's inclusion of a model and a saved key do not prove remaining allowance or enabled workspace access.

Kimi API is pay-as-you-go; Kimi Code and Kimi memberships are separate products. Kimi K3.1 Pro is available through Go at USD 2 / 10 per million uncached input/output tokens, with separate cache pricing. It is an alternative executor candidate, not automatically the economical choice relative to DeepSeek. Do not build a new default on K2.7: Kimi documents planned retirement of the K2 series by the end of December 2026. No new subscription, account top-up or provider/model configuration is authorized merely by recording this comparison.

Additional sources: https://api-docs.deepseek.com/quick_start/pricing/ and https://platform.kimi.ai/docs/guide/product-plans.md .

## Owner's executor decision — 2026-11-06

The owner confirmed an active OpenCode Go subscription and selected coding subagents using `opencode-go/deepseek-v4.1-flash`. The main coordinator retains planning, review and technical acceptance. This supersedes the open provider choice and the earlier proposal to use a different model as an automatic escalation path. An unavailable executor is a blocker to report, not permission to switch providers or spend through a direct API. The project profile is `.opencode/agents/executor.md`; loading it into the running session requires restarting OpenCode. This decision authorizes setting the profile and recording the role split, not execution of the full foundation plan.

## Publication policy proposal

Use separate named rule groups for credentials/private runtime content and for project formatting. Credential detection applies to all tracked paths, scanner sources and commit messages; explicit synthetic fixtures need narrow test-only handling. Work/task identifiers and `bin/work-*` references are valid project content, including in commit subjects. English formatting applies to application source and public technical documentation; work records may contain the owner's Russian wording. Detect accidental personal data artifacts by file/path rules and the whole-tree check as well as token patterns; no regex scanner is a proof that arbitrary prose contains no private information.

## Shared-hook correction

The coordinator's initial suggestion to remove knowledge-base hooks was premature. Shared agent hooks are configured globally; this repository also contains regular-file Git adapters and a submodule copy of shared tools. The correct implementation task is to establish ownership and applicability, repair local adapters, and avoid changing other projects' shared behavior. The inspected global tooling checkout has unrelated local changes; it is not part of this task's working tree.

## Bounded stages and reusable rules — 2026-11-06

The conversation established bounded, checkable executor stages with coordinator review, same-session corrections while useful, and fresh context for each next independent stage in the same worktree. Context-size handoff requires an on-disk checkpoint; it is distinct from a cumulative spend limit. No numerical threshold or automatic context enforcement has been selected. AGENTS.md, ai/WORKFLOW.md, the OpenCode executor profile and executor brief now carry those boundaries; command-side stage selection remains unimplemented.

The owner asked how reusable rules could live in `/Volumes/SSD/Dev/ai` and be imported or used by another project. The coordinator proposed a common tool-neutral rules source plus tool-specific loading and project-local commands/model/release settings. That extraction remains a proposal, not authorization to rewrite shared tooling. The owner then narrowed the tool direction: «я наверное не буду больше пользоваться Claude Code, поэтому давай забьем. Можешь просто сделать бэкап, но больше под них не ориентируйся». OpenCode is the primary target; no further Claude Code compatibility work is planned. Existing shared hooks and other projects are preserved.
