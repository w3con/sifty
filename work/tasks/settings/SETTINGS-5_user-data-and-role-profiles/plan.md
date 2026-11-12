# SETTINGS-5 — User и ролевые Profiles: отложенный проект

**Состояние:** отложено владельцем. Это сохранённый проект, не разрешение на реализацию. Исполняемых checkpoints пока нет.

**Авторство:** модель и backend-предложение написал координатор. Владелец согласовал общий уровень «Мои данные» над профилями и уточнил, что Triage, Preparation и отказы должны быть отдельными по профилям: «да отдельные. сохрани все это в карточке, но сейчас мы это делать не будем. отложим на потом.» Один человек, несколько вариантов позиционирования; другие люди, регистрация, сессии и logout — гораздо позже.

## 1. Цель и интерфейсная граница

Общие факты о человеке хранятся один раз. Ролевой Profile определяет, какую работу человек ищет и какие данные использует для этого. Страница «Мои данные» доступна из глобального меню Profiles; отдельная кнопка User в шапке сейчас не нужна. Переключатель профиля задаёт контекст всего приложения. Локальные вкладки профилей внутри страницы убираются. Дизайнерский dropdown уже существует в DESIGN-3, но backend на момент обсуждения поддерживает только один профиль и создание нового выключено.

В списке вакансий показывается оценка активного профиля. Сравнение профилей можно раскрывать в карточке; один «максимальный балл среди всех профилей» не подставляется вместо балла выбранного профиля.

## 2. Общие данные User и данные Profile

| User — общие факты | Profile — вариант поиска работы |
|---|---|
| Имя, контакты, фотография | Название: CTO, Principal Engineer и т.п. |
| Подтверждённые навыки и их уровни | Какие навыки использовать |
| История работы и подтверждённый опыт | Какой опыт включить |
| Языки и уровни | Целевые роли и отрасли |
| Текущее местоположение, право на работу | Желаемые места, переезд и условия работы |
| Библиотека исходных документов | Выбранные документы/CV |
| Личные ответы для анкет | Оплата, занятость, часы, дополнительные требования |

В профиле хранятся ссылки на общие навыки/опыт, а не копии массивов. Go можно включить в инженерный профиль и не включать в CTO; сам факт знания и его подтверждённый уровень остаётся один. Один документ может использоваться несколькими профилями. Общие факты обновляются один раз, профильная выборка не превращается в отдельную противоречащую биографию.

Сохранить нынешний подход типизированного JSON, стабильных ID элементов и ревизий, где он подходит; не переводить каждое поле в отдельную SQL-таблицу. Разделение нынешних Geography/Availability проходит по смыслу: текущее положение/допуски — факты User, желаемые места/пересечение времени/условия — Profile. Точное место notice/доступности уточнить при подготовке реализации, не копировать раздел целиком механически.

## 3. Явные ID и операции с профилями

Сейчас LocalProfileID="local", а CurrentProfile() выбирает первую строку ORDER BY id LIMIT1. Нужны получение/сохранение по ID, список, создание, переименование, копирование и архивирование Profile. Существующий CreateProfile после вставки тоже возвращает CurrentProfile и требует исправления.

Сохранённый activeProfileId нужен интерфейсу, но не является неявным входом фоновой работы. Каждый запрос оценки/подготовки передаёт profileId; задача закрепляет конкретные версии. Запущенная для CTO работа не должна после переключения прочитать Engineer как «текущий» профиль.

Архивирование Profile сохраняет общие документы, факты и историю откликов. Идентификатор/снимок использованного профиля в старой заявке остаётся доступным.

## 4. Документы и предложения фактов

Документы сейчас связаны с profile_id. Владельцем библиотеки становится User; связи Profile–Document определяют выбранные источники. Оригинал хранится один раз через существующий Dirs; переименование сущностей не требует физического переноса всех файлов. Отвязка документа от профиля не удаляет его из библиотеки или другого профиля.

У предложения факта явный адресат: «работал в X» — общий факт User; «ищу контракт» — условие определённого Profile. Принятие обновляет правильную сущность атомарно и убирает карточку из предложений. Смена активного профиля не перенаправляет уже созданное предложение на другой адресат. Сохранить цитаты, происхождение, документы-источники, состояния конфликтов и историю решений.

## 5. Оценки, контексты и очереди по профилям

Сейчас быстрый балл хранится в jobs.match_score, последний подробный результат — в jobs.assessment, Context() возвращает последний общий снимок, очередь assessment_queue уникальна только по job_id. Эти singleton-зависимости нужно заменить профильными.

Предлагаемая запись быстрого результата:

```text
profile_job_scores
  profile_id
  job_id
  quick_score
  user_revision
  profile_revision
  input_fingerprint
```

Подробные оценки, контексты и очередь также учитывают профиль и версии фактических входов. В fingerprint входят версия вакансии/контекста и версия алгоритма/настроек. Профили с одинаковым номером revision не взаимозаменяемы.

Изменился Profile — устарели его результаты. Изменился общий факт — устарели затронутые результаты. Переключение профиля читает его сохранённые результаты и не запускает модель. Отсутствующая оценка — «не оценено», а не ноль. Локальный пересчёт и дорогой LLM-запуск разделены.

Сборка контекста явно объединяет User + выбранный Profile + выбранные документы. Чужие неподключённые документы/условия профиля не подмешиваются через глобальный «последний контекст». Сохранить исключение необязательных демографических ответов из matching/fit/preparation/extraction.

Важное ограничение нынешнего BM25: его 0–100 нормируется относительно совпадений запроса, поэтому80 у CTO и70 у Engineer ещё не доказывают объективно лучший профиль. Автоматический выбор лучшего профиля по максимуму такого числа не вводится. Новый детерминированный scorer — отдельная задача, не скрытая часть этой миграции.

## 6. Состояния вакансий и отклики — согласованная развилка

**Triage/Preparation/отказы отдельные для каждого Profile.** Неподходящая по условиям CTO вакансия может подходить инженерному профилю. Нужна связь Profile–Job со стадией, причиной отказа и историей ручных переходов. Общий каталог вакансий не копируется по профилям. Существующие глобальные блокировки не становятся профильными без отдельного решения.

**Факт реального отклика общий для User.** История хранит дату, использованный профиль/ревизию и данные вакансии. Возврат в Triage, отклонение/удаление карточки либо архивирование профиля не стирает факт подачи. При переключении прежний отклик виден во всех профилях. Открытие Apply само по себе не является подачей — нужно явное подтверждение, как сейчас.

Подготовка заявки закрепляет профиль, user/profile revisions, документы и контекст. Уже подготовленные ответы не меняются незаметно при переключении профиля. Исторический снимок сохраняется, даже если исходный профиль/документ позднее отвязан или архивирован. Пояснение отказа относится к решению в профиле и не превращается автоматически в общий факт о навыках.

## 7. Предлагаемый API

```text
GET/PATCH  /api/me
GET/POST   /api/me/documents
GET/POST   /api/profiles
GET/PATCH  /api/profiles/{id}
PUT        /api/profiles/{id}/sections/{section}
GET        /api/jobs?profileId=...
POST       /api/jobs/{id}/assess       { profileId }
POST       /api/jobs/{id}/prepare      { profileId }
```

Для интерфейса сервер отдаёт согласованную проекцию общих и профильных данных. Переиспользуются нынешние store/domain/API границы, файловый resolver, provider adapters и конфигурация. Никаких новых accounts/session/permission/framework механизмов ради одного локального User.

## 8. Миграция и проверка

Из нынешнего единственного профиля создать одного User и первый ролевой Profile. Общие факты перенести в User, условия поиска оставить профилю. Сохранить стабильные ID, оригиналы, цитаты, историю ревизий и откликов; привязать имеющиеся документы/навыки к начальному профилю. Старые сериализованные ревизии не переписывать как будто они всегда имели новую схему.

Локальные результаты пересчитать по профилю. Старые подробные оценки оставить с их исходными данными и статусом актуальности; если новые входы нельзя доказать — считать устаревшими, не запускать LLM автоматически. Не восстанавливать отсутствующие события или происхождение догадками.

При подготовке исполнения нужны адресные проверки скрытых рисков: миграция существующей базы, изоляция данных/оценок двух профилей, очередь сохраняет исходный профиль при переключении, изменение общей базы не теряет выборки, отсутствие модельных вызовов при переключении, история подачи сохраняется после возврата/удаления. Видимое поведение проверяется на работающем приложении. Полные suite по умолчанию не запускаются.

## 9. Возобновление

Не начинать сейчас. Предварительная complexity13/130–220min в description.md — дизайн-оценка, не утверждённая стоимость исполнения. При возврате сверить с уже интегрированным кодом, разбить на ограниченные полезные результаты, дать полный план/время владельцу и получить отдельное разрешение. Текущий дизайн DESIGN-3 сохранить и переиспользовать.

## 10. Re-plan against the current code (coordinator, 2026-11-12)

Owner, 2026-11-12: «и еще - пора сделать возможность использовать несколько профилей, что думаешь? запустим еще одного агента параллельно?» and «да» to re-planning now. Implementation starts **after INBOX-2 is merged** (both change the schema and collection; parallel migrations would collide). Sections 1–8 above stay the design; this section adapts it to what landed since 2026-11-09.

### What changed in the code since the design

- Two scores (ASSESSMENT-3): `match_fit_results` holds one result per **job**; Fits you is computed from the single profile `local`; You fit is hidden (spec quick-match §14.14). Model extraction (`match_vacancy_extractions`) is per vacancy text and does not depend on the profile — it stays shared across profiles. Requirement matching is per profile when it returns (ASSESSMENT-5).
- The profile is a fact store (spec quick-match §15.1); ✓ / ✕ skills and "Skills you do not have" are facts.
- Search: one `search.yaml` (sites + expressions). The owner has just narrowed it to blockchain work; the previous broad engineering queries are in `data/config-backups/search-20261110-*.yaml`.
- Worker order follows the Inbox toggle; the model lock is machine-wide.

### Proposed stages (each a separately checkable result)

- [ ] **S.1 — User, profiles, migration** (60–90 min). Tables/records for User (facts: identity, skills, experience, languages, documents, ✓/✕ lists, answers) and Profile (name, target roles and industries, compensation, engagement, hours, work modes, wanted places, dealbreakers, selected skills/experience/documents, search expressions). Migration turns `local` into User + first Profile, preserving ids, revisions, quotes and history. API: list / create / rename / copy / archive profiles, get/patch User. Targeted tests: migration preserves the profile byte-for-byte in meaning; two profiles do not see each other's goals.
- [ ] **S.2 — Search per profile** (45–75 min). Each profile has its own search expressions; sources (sites) stay shared. A collection run uses the union of active (non-archived) profiles' expressions and records which profile's expression matched each vacancy. One shared catalogue; a vacancy is not copied per profile.
- [ ] **S.3 — Scores per profile** (60–90 min). `match_fit_results` keyed by (profile, job); Fits you per profile from that profile's goals and the shared User facts; extraction shared; the worker works for all profiles, ordering by the active profile's toggle; switching profiles reads stored results and never calls a model. Targeted test: switching does not call the model; a profile change marks only that profile's results stale.
- [ ] **S.4 — Triage and decisions per profile** (60–90 min). Per (profile, job): stage, rejection reason, history of manual moves; the fact of an application stays User-level and is visible in every profile (design §6). Migration puts today's states into the first profile.
- [ ] **S.5 — Interface** (60–90 min). The existing header switcher (`frontend/profile/ProfileSwitcher.tsx`, "Add new profile" disabled today) creates / renames / copies / archives; Inbox, Shortlist and Archive show the active profile's vacancies and scores; the profile page separates "My data" (User) from the profile's own sections (DESIGN-3 components).
- [ ] **S.6 — Owner review on a copy** (30 min): two profiles (Blockchain — today's; General engineering — seeded from the 2026-11-10 search backup), switching, scores, decisions.

Out of this card: detailed assessment and application preparation per profile (the detailed queue is off; design §5–6 kept for later); You fit per profile (ASSESSMENT-5).

### Estimate

Complexity 13 (unchanged). AI time about 5.5–8 h for S.1–S.6. No paid services.

### Questions for the owner before approval

1. A vacancy found only by the engineering profile's queries — shown in the blockchain profile's Inbox too (with that profile's low Fits you), or only in the profile whose queries found it? Coordinator: only in the profile whose queries found it; the others reach it through search.
2. Sources: one shared list (coordinator's recommendation) or a separate list per profile?
3. The second profile "General engineering" seeded from the old queries at S.6 — yes?

### Owner answers (2026-11-12)

1. «Только в нашедшем профиле» — a vacancy appears in the Inbox of the profile whose queries found it; other profiles reach it through search.
2. «Общий список» — sources are shared; profiles differ by search expressions.
3. «Да» — S.6 seeds a second profile "General engineering" from the 2026-11-10 search backup.

Plan ready for the owner's go-ahead; start after INBOX-2 is merged.
