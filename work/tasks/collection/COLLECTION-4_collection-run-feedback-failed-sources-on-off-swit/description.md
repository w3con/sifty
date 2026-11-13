# COLLECTION-4 — Collection run feedback: failed sources, on/off switch, failed run, inte

**Epic:** collection
**Created:** 2026-11-13
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
1. Показывать ли в интерфейсе, что источник упал или ничего не вернул, и если да, то где? - информационным сообщением (полагаю полосато-оранжевым?) в inbox/shortlist и также на странице settings - в списке источников - каким то символом и цветом?
3. У нас нет toggle "сбор включен/выключен" - можно его добавить на страницу источников в настройках.
4. да. и выдавать сообщение.
5. что за "ручной" сбор? приложение запускает (должно) сбор при обновлении страницы - ЕСЛИ сбор еще не запущен или не был завершен в ближайшие 3 часа (ну не чаще чем в 3 часа можно запускать новый сбор). МОжно добавить в настройки, в "источники" как часто собирать вакансии.
---
Карточки заводить? - да
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator

Coordinator (2026-11-13): the words above are the owner's answers to the coordinator's collection questions (OPS-11) and the owner's «да» to capturing this card («да и да»). Agreed outcome, spec `docs/specs/collection.md` (after the OPS-11 release): a failed or empty source is shown as a warning notice in Inbox and Shortlist and as a coloured mark next to the source in Settings → Sources; a collection on/off switch on the Sources page (today only `collection_disabled` in runtime.yaml); a run in which every source failed or returned nothing is recorded as failed and the owner gets a message; collection starts on page load when none is running and none completed within the interval (default 3 h), the interval set on the Sources page; the daily 09:00 scheduler is replaced by the interval rule applied by the server too; the "Collect jobs" button stays; `request_timeout_ms` is wired and the seven unused runtime.yaml keys removed (owner accepted the coordinator's recommendations). Order: after INBOX-2 and COLLECTION-2 (same collector code). Not planned, not estimated.
