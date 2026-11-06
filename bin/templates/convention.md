# Card-folder convention

A unit of work is a card. A card is a folder, not a file, so it survives any session or tool dying.

## Location

```
work/tasks/<epic>/<ID>_<slug>/
```

- `<epic>` is a lowercase short name grouping related cards (e.g. `sifty`, `harness`).
- `<ID>` is `<PREFIX>-<n>`: an uppercase prefix (default: the epic upper-cased) and a number allocated as one past the highest existing number for that prefix anywhere under `work/tasks/`.
- `<slug>` is a lowercase, dash-separated short title.

Single-file cards that predate this convention stay directly under `work/tasks/` and are never moved.

## File set (full card)

| File | Author | Holds |
|------|--------|-------|
| `description.md` | owner | why the work exists, what it must and must not touch |
| `plan.md` | coordinator | steps grouped into phases, plus acceptance criteria |
| `log.md` | executor | each closed step with the verbatim output of the check that proved it |
| `comments.md` | anyone | running discussion |
| `attention.md` | anyone | items that block handover until resolved |

Conditional files, written when the lifecycle reaches that state:

| File | Written by | Meaning |
|------|-----------|---------|
| `done.md` | `work-finish` | the work is accepted and merged |
| `rejected.md` | `work-finish --reject` | the work was judged not acceptable |
| `cancelled.md` | hand-authored | the work was abandoned before completion |
| `question.md` | hand-authored | a business question stops the work |

## Short card

A short card carries only `description.md` plus `comments.md`. Its steps (as `- [ ]` lines) and its acceptance criteria live inside `description.md`. Create one with `work-new --short`.

## Steps and attention

- A step is a line `- [ ]` (open) or `- [x]` (closed) in `plan.md` (full card) or `description.md` (short card).
- An attention item is a line `- [ ]` in `attention.md`; while any is open, `work-handover` refuses.
