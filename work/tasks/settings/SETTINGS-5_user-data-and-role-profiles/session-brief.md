# Brief for the profiles coordinator session (2026-11-13)

The owner pastes this into a new Claude Code session opened in the main checkout. That session coordinates the profile work; the session that wrote this brief keeps APP-9 (application stage 1).

## Your role

You are the coordinator for SETTINGS-5 (two or more role profiles), and after it is finished, SETTINGS-10 (skills versus Confirmed by you in the profile). One session runs one card: the work gate binds this session to SETTINGS-5 until its folder has `done.md`; SETTINGS-10 can then start in the same session.

## Read first

`AGENTS.md`, `work/WORKFLOW.md`, the newest resume point in `work/ROADMAP.md`, the SETTINGS-5 card (`description.md`, `plan.md` — section 10 and the Approval section hold the approved stages S.1–S.6 and the owner's later additions — and `log.md` if present), `docs/specs/profiles.md`, and `work/tasks/settings/SETTINGS-10_skills-and-confirmed-facts/description.md`.

## Precondition

Start only after INBOX-2 is merged into main (its card folder has `done.md` and its squash commit is on main). INBOX-2 changed the profile (+ / − buttons on facts and skills); SETTINGS-5's plan was written before that.

## Steps

1. Re-check the approved SETTINGS-5 plan against main after INBOX-2. Tell the owner, in Russian, what changed and whether the plan still holds. A material change needs the owner's approval; otherwise the existing approval stands («утвердил уже», 2026-11-12).
2. Run `WORK_SESSION=1 bin/work-handover SETTINGS-5` and raise the executor with a plain `Agent` call (`subagent_type: executor`, Sonnet 5.5; if it refuses, `executor-sonnet-4-6`). Name the card in the prompt. Assign stages one at a time or in small groups, with their checks and stopping point.
3. Accept each stage yourself (read the diff, run the targeted tests) and report to the owner with the acceptance template from STYLE.md.
4. S.6 is the owner's review on a copy. Release only after the owner's explicit «да».
5. After SETTINGS-5 is done, plan SETTINGS-10 with the owner. The owner described the second piece of work as "work on the UI of the profile card itself"; confirm with the owner whether that is SETTINGS-10 (skills versus facts, move buttons, + / −) or a wider profile page redesign before writing the plan.

## Constraints

- Production data is read only; executors work on copies. Production is never built or released without the owner's explicit «да».
- Never push. Do not write into `.worktrees/INBOX-2` or `.worktrees/APP-9`.
- The git stash is shared between sessions: only a tagged stash, applied by SHA, then dropped.
- No `run_in_background` around agents, no `Workflow`, no `isolation`.
- The leak check refuses Cyrillic under `docs/`: translate owner quotes into English in specs; originals go into the card's own records under `work/`. Run `backend/scripts/leakcheck/commit-diff-check --staged` before committing docs.
- Keep `work/ROADMAP.md`, `docs/MAP.md` and `docs/specs/profiles.md` updated after each step.
- Before long builds or agents, hold the machine awake: `nohup caffeinate -dimsu -t 14400 >/dev/null 2>&1 & disown`.

## Coordination with APP-9 (the other session)

APP-9 adds a lookup trigger when a vacancy enters Shortlist and changes `apply()` in `frontend/main.tsx`; SETTINGS-5 S.4 makes stages per profile and S.5 changes the interface. Whichever card merges second rebases on main and resolves the overlap; the APP-9 trigger stays "vacancy entered Shortlist in any profile". Neither session edits the other's worktree.
