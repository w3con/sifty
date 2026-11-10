# SETTINGS-4 — Website и Social profiles: URL до предложенных фактов

**Epic:** settings
**Created:** 2026-11-09
**Status:** backlog
**Kind:** task

## Результат

Владелец добавляет адрес своего сайта либо публичного социального профиля, видит этапы fetch/text/analysis и подтверждает предложенные факты. Адрес и статус однозначны; «imported, 0 facts» не заменяет объяснение.

Точный исходный запрос находится в [SETTINGS-2](../SETTINGS-2_profile-workflow-and-fields/description.md). Это выделенный координатором результат, не новая идея сверх запроса. План предложен, выполнение не одобрено.

## Границы и зависимости

Зависит от SETTINGS-2, использует текущие documents/facts/config/provider/fetch adapters. Это источники профиля, не внешние источники вакансий. Полный social crawler, вход в аккаунты, бесконечный обход сайта, anti-bot обход и automatic application исключены.

## Оценка

- **Complexity:** 5 — существующий pipeline переиспользуется; неопределённость доступности внешних страниц.
- **AI preparation:** 15–25мин; **implementation:** 55–90мин; **acceptance:** 20–35мин.
- **Total AI elapsed:** 90–150мин, без внешнего fetch/model ожидания. Подписка; inference по выбранному provider, точная цена неизвестна. Полные шаги/проверка в plan.md.
