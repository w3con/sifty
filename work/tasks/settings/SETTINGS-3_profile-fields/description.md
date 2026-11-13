# SETTINGS-3 — Понятные поля и редакторы Profile

**Epic:** settings
**Created:** 2026-11-09
**Status:** review
**Kind:** task

**Integration status,2026-11-10:** technical implementation accepted; source published to main in commit `c827cee`, final application source `58af432`. Card remains review, not done; production is not updated. Owner acceptance/release is separate. The continent preference below remains captured and unimplemented.

## Зачем и для кого

Владелец заполняет опыт, условия работы и географию без непонятных терминов и свободных значений там, где нужен выбор. Исходные требования дословно сохранены в [обсуждении Profile](../SETTINGS-2_profile-workflow-and-fields/comments.md); этот документ и plan.md написал координатор.

Owner: «С остальными твоими комментами согласен, там где ты до конца не определился - реши как тебе кажется будет лучше, потом я еще раз посмотрю». Citizenship исключено словами «полагаю гражданство не нужно». Это согласие с направлением, не запуск нового плана.

## Границы

Поля/миграция/редакторы/правильная передача в существующий context входят; новый weighted engine, URL fetch, универсальная нормализация и генерация/отправка CV не входят. Зависит от независимого Save в SETTINGS-2. Условия в форме не выдаются за уже подключённые к быстрому баллу. Сохранять прежние значения при миграции.

## Предварительная оценка

- **Complexity:** 8 — одна согласованная модель Profile затрагивает schema/validation, facts и UI; оставляем одной карточкой ради целостной миграции, а workflow/URL/словарь уже вынесены.
- **AI preparation:** 20–30мин; **implementation:** 100–165мин; **acceptance:** 30–45мин.
- **Total AI elapsed:** 150–240мин, без ожидания владельца и внешних задержек; подписка, относимая цена неизвестна.
- Полные шаги/критерии в plan.md. Выполнение одобрено владельцем 2026-11-09: «ок, давай тогда следующую карточку.» Только эта карточка, включая проверку staging; интеграция в main и production отдельно.

## Implemented result —2026-11-09

All four steps technically accepted through3d69ef5 in .worktrees/SETTINGS-3. Staging http://localhost:8081→Profile served the updated fields, preserving current profile revision10,8documents and10facts. Coordinator report copied to [acceptance.md](acceptance.md). Source later published to main (`c827cee`, final `58af432`); owner functional acceptance and production release separate, card review not done. Visual mockup, user icon, discoverable upload and real avatar belong to the next proposed DESIGN-3 plan; repeated website-analysis timeout recorded as evidence in SETTINGS-2, no model retry initiated.

## New geography requirement —2026-11-09, captured only

Exact owner wording is in comments.md. Preferred work locations must accept a continent as well as a country or country+city. Concrete use: the owner can select Europe once to express willingness to work anywhere in Europe, without enumerating countries. A continent selection covers its member locations; it does not mean an employment authorization or visa answer for every country. Residence and country-specific work authorization stay independent. This is a new requirement, not implemented by the accepted country/city editor and not approval of the separate appearance/photo plan.

Provisional added complexity3: typed location scope, local continent membership/reference data, compatible section-save/context handling and visible choice/tags must agree. AI preparation5–10min, implementation20–30min, acceptance5–10min; total30–50min excluding owner wait/external latency, within the existing subscription with unknown attributable price and no new paid service. This is an incremental backlog estimate, not extra time already spent or authorization to reopen accepted source. Prepare the exact step/results/cost and acceptance before execution; preserve original country/city values. No new scoring formula is implied.
