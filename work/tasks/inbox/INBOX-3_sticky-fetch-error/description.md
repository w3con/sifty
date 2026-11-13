# INBOX-3 — Failed to fetch banner never clears

**Epic:** inbox
**Created:** 2026-11-11
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Кстати, при выключении модели (я полагаю) вверху выдается Failed to fetch - на серо-полосатом фоне. но кнопки "обновить" нет - и наверное не должно быть. значит фон должет быть оранжево полосатым (в соответствии с дизайн системой)? Это что за ошибка вообще? и кстати, даже когда продолжается работа оцекни/llm - ллоготип показывает, что работа идет, а оповещение - не пропадает. похоже на баг. запиши его куда нибудь

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator diagnosis (2026-11-11)

- "Failed to fetch" is the browser's own message when a request to the Sifty server gets no answer at all (server stopped or restarting). It is not about the model: a stopped model shows "Requirements could not be analysed — the local model is not running" inside the Quick Match panel. On 2026-11-11 the review server on 8094 was restarted several times by the executor, which matches.
- Why it never disappears: `frontend/main.tsx` polls `queue` every 2.5 s; a failed poll calls `setError(e.message)`, a successful poll never clears it. The banner has no close button. So one missed poll leaves the banner forever while everything (logo activity included) works again.
- Colour: by the design system an unknown or transient cause is **Retry** (grey stripes); **Broken** (orange) is "will not pass on its own". A server that comes back is Retry, so grey is right; the text and the stickiness are the defects.

## Proposed fix (to plan)

- An error raised by the poll clears itself on the next successful poll; errors from the owner's own actions stay until the next action or view change.
- Known cause in `frontend/messages.ts`: network failure to the own server → "Sifty server is not responding. Retrying…", Retry kind; no button needed.
- Estimate: complexity 2, AI time 20–30 min. Can ride along with INBOX-2 (same screen) if the owner agrees.

## Decision

2026-11-11, owner: «да» — fixed together with INBOX-2 as its step L.4b; INBOX-3 is closed by that card.
