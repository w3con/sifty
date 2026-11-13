---
estimate: 5
---
# Profile: макет, переиспользуемые блоки и загрузки

**Approval:** owner «хорошо, делай одним прогоном» approves the coordinator's corrected25–45min full scope including persistent photo. The former115–190min estimate is withdrawn as inflated, not a remaining allowance. One executor, one combined implementation pass and one final browser/build acceptance. Start2026-11-09T17:26Z, stop boundary18:11Z (owner wait excluded). Coordinator ses_e4a7d8341ffe1Xqrv4Rmx52jqw.

## Наблюдаемый результат и применение

Владелец открывает Profile на staging и видит экран в композиции frontend/Claude outputs/profile.png, с реальными сохранёнными данными и новыми полями. Документы добавляются заметной кнопкой Add files и перетаскиванием; фотография действительно загружается и остаётся после перезапуска. Несколько карточек/групп используют одни визуальные компоненты: изменение их общего оформления видно сразу у нескольких потребителей. Отдельная неиспользуемая галерея не создаётся.

## Точный объём

- На широком экране: шапка профиля с фотографией/инициалами и две колонки. Слева документы, компактные навыки, целевые роли/отрасли и предложения фактов; справа оплата, способ/время работы, география/право работы, языки/доступность и дополнительные условия. Опыт и дополнительные личные поля размещаются согласованно с карточками, не исчезают ради сходства с картинкой. На узком экране — одна читаемая колонка без горизонтальной прокрутки.
- Макет задаёт отступы, размеры, поверхности, типографику, теги и группы выбора. Значения/тексты берутся из текущего Profile. Beginner/Proficient/Expert, новые условия и отсутствие citizenship сохраняются; старые подписи из картинки не возвращаются.
- Небольшой общий набор секционных карточек, подписанных значений/статистик, тегов и групп выбора извлекается из реальных потребителей Profile. Использовать существующие CSS tokens/компоненты там, где они подходят; без нового UI framework, Storybook, каталога всех возможных блоков или переписывания остальных экранов. Иконка Profile в навигации — user существующего Icon adapter.
- Documents about you имеет Add files/зону drop с доступным обычным file picker. Действуют текущие разрешённые типы и лимиты backend capabilities. Существующие upload/Analyze/Retry/View text/Original и принятие фактов сохраняются; загрузка/выбор не запускают модель. Несколько файлов проходят существующий upload-путь с отдельным состоянием/ошибкой, успешные загрузки не теряются при неудаче следующей.
- Фото трактуется как аватар из макета: загрузить, просмотреть, заменить и удалить. Оригинал/ссылка хранятся устойчиво через существующие data-directory resolver/storage/config; поддерживаемые форматы и лимит задаются один раз в существующей конфигурации/возможностях, а сервер проверяет фактический тип. Фото не отправляется модели и не анализируется как CV. Замена не удаляет действующее фото, пока новое не сохранено; отмена не меняет сохранённое. После reload/restart показывается сохранённый аватар, при отсутствии — инициалы/иконка из реального имени.
- В картинке есть обещания, которых текущий продукт не выполняет: Can send/default-CV, автоматическое приложение фотографии, штрафы быстрого балла и числовые gaps. Этот план не изображает их работающими переключателями/счётчиками. Использовать реальные состояния документов/фактов и честные подсказки назначения полей, не добавляя новый scorer/автоотправку.

## Слои и сохранность

SQLite/Profile и файловое хранилище через существующий Dirs — авторитетные данные; существующая конфигурация/capabilities — лимиты/типы; domain services — сохранение/замена/проверка фото; hooks/API — эффекты; визуальные блоки — данные/callbacks/CSS. Не хранить значения владельца/макета, пути машины, адреса/порты, provider/model и настраиваемые лимиты в логике/представлении. Не создавать вторую settings-систему. Сохранять документы, факты, текущий профиль и независимые Save/Cancel/no-op. Baseline — технически принятый SETTINGS-3, не грязный main; оригинальные worktrees/Designer сохраняются. Staging обновляется только после проверок с текущими данными и восстановимым снимком; production/main integration отдельно.

## Оценка и обязательные шаги

**Complexity5:** bounded Profile layout/components plus a small persistent avatar addition using existing services. One outcome, not five separate executor stages. Former60–120 and115–190min estimates superseded by the corrected approved **25–45min**. Preferred-location continents remain separately captured geography work, not this appearance/photo approval.

AI preparation3–5min, implementation17–30min, acceptance5–10min; total25–45min excluding owner wait/external delays. Preparation is included in the first row, not added again. OpenCode Go subscription; attributable monetary cost unknown, no new paid services. At18:11Z preserve progress and report rather than silently extending.

| Шаг | Конкретный результат и зачем | AI-время | Внешняя цена |
|---|---|---|---|
| D.1 / appearance and documents | Two-column mockup composition, shared blocks reused in real sections, responsive layout, user icon and Add files/drop on existing upload; recognizable usable screen |10–15min including preparation|No new services/model calls|
| D.1 / avatar | Actual upload/storage/read/replace/delete; the photo control persists its result |10–20min|Local storage; no paid service|
| D.1 / final acceptance | Builds, real desktop/mobile browser, isolated real uploads/restart and safe staging transition; demonstrated result |5–10min|No new services|

All three parts are mandatory within one assigned stage; no optional infrastructure improvements. Reuse current card/tag/field implementations instead of creating an additional parallel abstraction.

## Checkpoints

- [x] D.1 — Одним прогоном оформить Profile, переиспользовать блоки, подключить документы и сохраняемый аватар; проверить и передать staging.

## Acceptance criteria

- Реальный Chromium на desktop/mobile показывает узнаваемую композицию макета, все согласованные поля и user icon. В отчёте есть рядом target/current screenshots, а не только build success.
- Один общий визуальный блок используется минимум двумя действительными группами; демонстрационная правка общего стиля влияет на обоих, затем откатывается до финального вида. Не клонированные CSS-копии/компоненты с разными именами.
- Add files/drop действительно сохраняют разрешённые документы; Analyze остаётся явным действием. Не вызываются поиск/модель при приёмочной загрузке; успешный файл сохраняется при ошибке следующего.
- Фото реально загружается/читается после reload и перезапуска из устойчивого хранилища, заменяется/удаляется. Ошибка замены не теряет предыдущую фотографию; один адресный скрытый check допустим для этого свойства, а не для видимых кнопок/лейблов.
- Исходные документы, факты и значения Profile сохранены; пустые значения не подменены примерами макета, поля/независимые сохранения не деградировали. Нет фиктивных Can send/gap-scoring/auto-attach обещаний.
- Актуальные frontend/backend builds pass, broad suites SKIPPED. Сохранены данные/точка восстановления, staging здоров, production неизменён. Фактическое время и owner acceptance отдельно от технической проверки.
