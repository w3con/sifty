---
estimate: 8
---
# Profile: поля, единицы и редакторы

**Approval:** owner authorized execution on 2026-11-09: «ок, давай тогда следующую карточку.» This follows the coordinator's explanation that the next card implements Profile fields. Scope remains steps 3.1–3.4 below; complete results/time/cost breakdown reiterated before dispatch. Depends on technically accepted SETTINGS-2; does not depend on ASM-5. Approximate start14:15Z; upper stop boundary18:15Z (240min, including preparation/review). No production release or main integration is authorized.

## Полезный результат

Владелец вводит реальный опыт, навыки и условия; сохранённые поля читаются после перезагрузки и корректно передаются в подробную оценку/подготовку ответа. Демонстрация — браузерное заполнение, редактирование и просмотр сохранённых значений без платного inference для каждого поля.

## Решения координатора для проверки владельцем

- **Identity:** Display name, Legal name; убрать Headline. Gender — необязательный выбор Woman/Man/Non-binary/Self-describe/Prefer not to say. Отдельные необязательные Race и Ethnicity с подписями назначения «для анкет»: race допускает несколько выбранных значений (American Indian or Alaska Native, Asian, Black or African American, Native Hawaiian or Other Pacific Islander, White, Self-describe); ethnicity — Hispanic or Latino / Not Hispanic or Latino / Prefer not to say. У race также есть Prefer not to say. Пустое отличается от отказа отвечать. Citizenship/nationality не добавлять. Не угадывать эти значения по имени/документу; они не участвуют в поиске/скоринге/fit prompt и доступны только контуру ответов там, где нужны анкете, без автоматической отправки.
- **Skills:** Beginner / Proficient / Expert (Начальный/Уверенный/Эксперт) и Not set. Не переводить автоматически Production в Expert: старый уровень сохранить с пояснением до ручного выбора; прежняя информация не теряется. Убрать Context из формы, сохранить прежний текст в совместимых данных. Название до ASM-5 остаётся свободным. Уровень используется в подробной оценке и ответах, не как новый коэффициент быстрого балла.
- **Experience:** Company, Role, Start, End, Currently working here, Summary. Checkbox отключает End; его выключение возвращает прежний draft даты, а сохранённое active значение не содержит действующий End. Не выводить current employment из пустого End. Skills внутри записи убрать из формы, старые значения сохранить без потери.
- **Rich text:** готовый Tiptap с минимальной панелью bold/italic/lists/headings для Summary и больших текстовых пояснений. Хранить поддерживаемый документ и получать plain text для поиска/model context; существующий plain text не терять. Исключить embeds/произвольный HTML; не писать свой редактор. Форматирование не превращается в требование к LLM.
- **Даты:** готовый React DayPicker с ручным вводом и placeholder. Experience поддерживает точность month/year (`MM/YYYY`) и календарный выбор месяца/года; availability — дату (`DD/MM/YYYY`). Не дописывать выдуманный день к месячной точности. Проверить совместимость готовой библиотеки/лицензии до установки.
- **Roles/industries:** много значений, свободный ввод до ASM-5, без закрытого IT-only списка. Роли уже участвуют в быстром ранжировании, industries пока — подробная оценка/ответы; подсказки отражают фактическую реализацию, а не будущий scorer.
- **Compensation:** Minimum compensation / Desired compensation, EUR/USD; Before taxes / After taxes; Monthly / Yearly — период суммы. Не подставлять невыбранный налоговый режим/период как подтверждённый факт. Типизированные значения нормализуются для сравнения только при известных единицах/валюте/налоговом базисе; gross/net не пересчитывать без данных, никаких придуманных налогов/FX.
- **Crypto:** Fiat only / Crypto accepted / Crypto only; default Crypto accepted означает отсутствие ограничения по способу оплаты. Валюта ожидаемой суммы остаётся EUR/USD. **Equity/tokens:** toggle «Interested in equity/tokens in addition to base pay», default off — нет дополнительного пожелания, а не запрет. Доли/токены не покрывают minimum guaranteed pay. Это предложение координатора по ранее открытому смыслу, не цитата владельца.
- **Engagement:** Contract type Employment / Contract (оба допустимы), consultant относится к Contract. Working time Full-time / Part-time отдельно. Для Full-time часы скрыты; Part-time — Desired hours per week, необязательные min/max. Скрытое прежнее значение не применяется как действующее Full-time ограничение. Remote/Hybrid/Office отдельно. Work limits убрать как отдельный блок; сохранённые ограничения перенести в Other requirements без дублирования/потери.
- **Geography:** Current location (один country+optional city), Preferred work locations (несколько country+optional city) в виде выбранных тегов `Paris, France` / `Canada — any city`. Использовать готовый локальный список стран ISO, не внешний геокодер на каждый ввод; город — ручной ввод с отдельным country, без обещания всемирного каталога городов. Старые неоднозначные строки сохраняются как unstructured location до ручной правки.
- **Authorization:** по выбранной стране Eligible to work / Need visa sponsorship / Not set; разрешить добавить страну права работы независимо от желательных мест, чтобы не потерять существующие права. Current residence не доказывает eligibility. В желаемых местах — willingness to relocate и timezones/overlap, без второй копии проживания.
- **Timezones:** готовые названия регионов/городов с IANA ID (например Europe/Paris), не постоянный UTC offset. Desired overlap — необязательное число часов пересечения; без рабочего интервала вакансии результат «не сопоставлено». Не считать весь континент одним timezone.
- **Languages:** выбор языка и Pre-A1 — Novice, A1 — Beginner, A2 — Elementary, B1 — Intermediate, B2 — Upper-intermediate, C1 — Advanced, C2 — Proficient; отдельное Native. `A0` не выдавать за официальный CEFR уровень. Существующее произвольное значение сохраняется до ручной замены, не угадывается.
- **Availability:** notice period с числом/единицей, available-from с календарём; не обещать точную дату по расплывчатой старой строке. Время/часовой пояс не дублировать во втором master field.
- **Dealbreakers / Other requirements:** свободный текст с простым форматированием; LLM detailed assessment, не быстрые штрафы. Уже заданные структурированные условия повторно вводить не требуется.

## Границы и слои

SQLite/typed contract — авторитетные данные; schema migration сохраняет старые значения и internal revision. Profile domain service — validation/совместимость/назначение полей; API/hooks — эффекты; components — render. Списки/подписи/field-use descriptors — единые domain/config ресурсы, не копии в UI/prompts. Reuse config/Dirs/provider. Никаких model/provider/портов/персональных примеров в application logic/views. Удаление поля из формы не означает удаление исходных данных. Минимальные адресные изменения fit/drafting inputs исключают identity-demographics из fit, а не просто скрывают их в UI.

## Оценка и шаги

Сложность 8; preparation 20–30мин, implementation 100–165мин, acceptance 30–45мин; всего **150–240мин**. Стоп на 240мин. Время review/сборки входит в строки; ожидание владельца исключено. Подписка, точная цена неизвестна; готовые open-source библиотеки без новой платной услуги, лицензии подтвердить перед использованием. Live LLM calls для этой карточки не нужны.

| Шаг | Результат / зачем | AI-время | Внешняя цена |
|---|---|---|---|
| 3.1 | Совместимый typed contract/migration и field-use inputs; сохранить данные и назначение полей | 35–55мин | Дополнительных сервисов нет |
| 3.2 | Identity/Skills/Experience/Languages/Availability, календари и rich text; понятный ввод | 45–70мин | Open-source packages, без платного API |
| 3.3 | Compensation/contract/hours/location/authorization/timezone; согласованные условия | 50–80мин | Локальные reference data, без платного API |
| 3.4 | Сборки, выборочная миграционная проверка и браузер; доказать сохранность и полезность | 20–35мин | Дополнительных сервисов нет |

Все шаги обязательны; никаких добавочных optional framework работ.

## Checkpoints

- [x] 3.1 — Расширить контракт и миграцию без потери старых значений и назначения полей.
- [x] 3.2 — Обновить персональные поля, опыт, навыки, языки и редакторы дат/текста.
- [x] 3.3 — Обновить условия оплаты, занятости, мест и права работы.
- [x] 3.4 — Проверить миграцию, реальный ввод и передачу данных по назначению.

## Acceptance criteria

- Владелец проходит заполнение всех групп на staging; Save/Cancel/no-op остаются независимыми. Пустое — Not set; gender/race/ethnicity необязательны, citizenship отсутствует.
- Старые уровни, текст, employment skills, location strings и суммы не исчезают молча; один адресный migration roundtrip тест на скрытую потерю/неверное переопределение.
- Demographics не оказываются в fit/fast-ranking inputs; проверяется фактическая сборка model input, не только скрытые поля. Нет модели при Save.
- Работающий checkbox current employment, ручные даты+календарь, rich-text roundtrip/plain-text input, part-time range, country-bound authorization видны в браузере.
- UI сообщает реальное назначение поля. Gross/net, FX и пустые данные не дают ложных подтверждённых сравнений. Новая формула оценки не заявляется выполненной.
- Build pass; broad suites SKIPPED; targeted скрытые checks отдельно. Записаны время и владелец/staging acceptance; production release отдельно.
