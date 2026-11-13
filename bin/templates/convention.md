# Card-folder convention

A unit of work is a card. A card is a folder, not a file, so it survives any session or tool dying.

## Location

```
work/tasks/<epic>/<ID>_<slug>/
```

- `<epic>` is a lowercase short name grouping related cards (e.g. `sifty`, `harness`).
- `<ID>` is `<PREFIX>-<n>`: an uppercase prefix (default: the epic upper-cased) and a number allocated as one past the highest existing number for that prefix anywhere under `work/tasks/`.
- `<slug>` is a lowercase, dash-separated short title.
- A folder without any card file is not a card. Two card folders with the same ID are ambiguous: the commands refuse them rather than choosing one. Existing IDs are never renumbered.

Single-file cards that predate this convention stay directly under `work/tasks/` and are never moved.

## File set (full card)

| File | Author | Holds |
|------|--------|-------|
| `description.md` | owner | why the work exists, what it must and must not touch, and the explicit state metadata |
| `plan.md` | coordinator | steps grouped into phases, plus acceptance criteria |
| `log.md` | executor | each closed step with the verbatim output of the check that proved it |
| `comments.md` | anyone | running discussion and captured evidence |
| `attention.md` | anyone | items that block handover until resolved |

Conditional files, written when the lifecycle reaches that state:

| File | Written by | Meaning |
|------|-----------|---------|
| `done.md` | `work-finish` | the work was accepted and merged (successful outcome) |
| `rejected.md` | `work-finish --reject` | the work was reviewed and judged not acceptable (archived unsuccessful outcome) |
| `cancelled.md` | `work-cancel` | the work was abandoned before completion (archived unsuccessful outcome) |
| `question.md` | hand-authored | an unanswered business question; it blocks the card but is not a terminal outcome |

## States and metadata

The only normal states are `backlog`, `in progress`, `review` and `done`. Preparation (plan, dependencies, open questions) is content in a backlog card, not another state. A blocker is a reason attached to a nonterminal state, not a state of its own. Rejection and cancellation are archived outcomes distinct from successful `done`.

The smallest explicit metadata lives in `description.md`:

```markdown
**Status:** backlog
**Kind:** bug          # optional: bug, idea or task (default task)
**Blocker:** waiting on the owner   # optional, nonterminal cards only
```

Metadata is read only from the leading header block — before the first level-2 `##` section and outside fenced code blocks. A captured owner report is data: Markdown-looking `**Status:**`, `**Blocker:**`, checklist lines or `## Acceptance criteria` inside its fence never change card state or become plan steps.

Inference and precedence, used by `work-status`, `work-list`, `work-capture`, `work-handover`, `work-resume`, `work-accept` and `work-finish`:

1. An outcome file wins: `done.md` → `done`; `rejected.md` → `rejected`; `cancelled.md` → `cancelled`. This is why an old completed card with stale `**Status:** founded` or an unchecked template line remains completed and is not reopened.
2. `question.md`, a `**Blocker:**` line and open `attention.md` items block a nonterminal card without changing its state.
3. Otherwise the leading value of `**Status:**` decides. Legacy `founded`, `approved`, `new` and similar prose map to `backlog`; `descoped` maps to an archived cancellation even before a `cancelled.md` exists. An unknown value is reported, never guessed.
4. A card with no metadata is `backlog`.

Read-only navigation uses the active worktree's nonterminal state and exact next step, while showing the base state separately. Base cancellation, blockers, open attention and contradictions override the worktree because they are safety decisions; resume refuses them. Otherwise a contradictory active worktree is surfaced and refused, not hidden behind a healthy base copy. Commands that change metadata write only the current checkout's card.

Contradictions are refused: several outcome files, a `**Status:**` naming an outcome without its evidence file, a terminal outcome that still carries a blocker or open attention, several conflicting metadata lines, or duplicate IDs.

`work-status` changes `backlog`/`in progress`/`review` and records or clears a blocker. It refuses `done`: only `work-finish` writes that after the checked, owner-approved completion path. `work-finish --reject` writes `rejected.md`; `work-cancel` writes `cancelled.md`. Neither is treated as success.

## Short and capture cards

A short card carries only `description.md` plus `comments.md`. Its steps (as `- [ ]` lines) and its acceptance criteria live inside `description.md`. Create one with `work-new --short`.

`work-capture` creates a minimal backlog card for an incoming bug or idea: `description.md` plus `comments.md`, the owner's exact words, `**Kind:**`, and no plan, log or attention file. The wording is stored exactly as supplied, including outer whitespace, newlines and original CR/LF; only emptiness is validated. It does not dispatch an executor, cut a worktree, start work or start a server. Preparing the plan later adds the steps and acceptance criteria. `work-capture --update <id>` appends evidence to an explicitly matched existing card and refuses exact duplicate evidence.

## Steps, attention and navigation

- A step is a line `- [ ]` (open) or `- [x]` (closed) in `plan.md` (full card) or `description.md` (short card). The exact next open step is the first `- [ ]` line.
- An attention item is a line `- [ ]` in `attention.md`; while any is open, `work-handover`, `work-resume`, `work-accept` and `work-finish` refuse.
- `work-status <id>` shows the state, blocker, kind, card path, exact next open step and any active worktree. `work-list [--state ...] [--kind ...] [--epic ...]` derives backlog and bug views from canonical cards. Active worktree state is shown when it differs from the base checkout.
- `work/ROADMAP.md` remains the curated source for card sequence and epic outcomes. The commands link there; they do not duplicate its descriptions.
