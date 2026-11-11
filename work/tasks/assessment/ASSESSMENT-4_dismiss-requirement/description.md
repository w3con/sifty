# ASSESSMENT-4 — Dismiss a vacancy requirement

**Epic:** assessment
**Created:** 2026-11-11
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
помимо add to experience  - надо еще "dismiss" - и это должно на будущее не показывать снова. make sense? хотя что делать если кликнут ошибочно? менюшку вывдавать, мол "вы уверены?"

Отклонённое попадает в список в профиле, вроде «Не важно для меня: async communication skills». - согласен. 

осле клика внизу на 5–10 секунд появляется «Dismissed · Undo»; - согласен. 

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Agreed in conversation (2026-11-11), before planning

- Dismiss means "not important to me": the requirement leaves You fit for that vacancy (it no longer counts as unclear at 0.5). It is not "I don't have it" (that is an honest "no") and not "the model is wrong" (that is Add to …).
- Dismissed items go to a profile list ("Not important to me: …"), visible and editable in the profile. The matching pass receives this list and recognises the same requirement in other vacancies in other wording (local model, no cost); such requirements are marked dismissed, not shown as missing.
- No confirmation dialog. After the click: "Dismissed · Undo" for 5–10 s; a collapsed "Dismissed (N)" block at the end of the panel to restore; the profile list to restore everywhere.
- Open for planning: whether the counts line shows "N dismissed"; whether a profile-list change re-matches all vacancies (cost: matching pass only).
- Depends on ASSESSMENT-3 (panel, matching pass, profile facts). Start after it is finished.
- Spec: extend `docs/specs/quick-match.md` §14 (panel, You fit, matching inputs) and the profile spec section that lists profile sections, after owner agreement on the plan.

## Estimate (preliminary)

Complexity 5 / AI time 2–3 h (profile field + contract, matcher input, evaluator exclusion, panel actions with undo, profile list UI, spec).
