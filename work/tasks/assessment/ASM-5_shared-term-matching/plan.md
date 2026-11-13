---
estimate: 8
---
# Нормализация названий с реальным применением

**Approval:** proposed, not approved. Depends on SETTINGS-3 and the preserved SETTINGS-1 implementation. Does not require URL sources or a vacancy aggregator.

## Результат и доказательство пользы

Владелец выбирает React в Profile и видит одно и то же объяснимое совпадение для React.js/Reactjs в вакансиях разных источников. Профессии/навыки вне IT доступны через межотраслевые данные; нишевые термины сохраняются без принуждения выбрать неверную категорию. Доказательство — работающий Profile selector, реальные сохранённые вакансии/оригинальные тексты и скоринг на staging, не только dictionary self-test.

## Источник, поддержка и время запуска

- База — pinned ESCO v1.2.1, English/French preferred/alternative labels и stable concept URI для occupations/skills. Не импортировать заодно все графы отношений/все languages. Это основа для нескольких отраслей, не обещание покрытия каждого нишевого понятия. До использования проверить доступность конкретного пакета, provenance и условия reuse/attribution. Download требует email через форму; предпочесть официальный прямой dataset/API export с pinned version без регистрации от имени владельца. Если получить данные в бюджет нельзя, сохранить результаты и пересогласовать, не подменять базу самодельным списком.
- Данные ESCO поддерживает Еврокомиссия. Снимок обновляется разработчиком как явное изменение данных при выпуске приложения, с source/version/hash; не скачивается при каждом Save/collect/start и не меняет результаты молча. Автообновляющийся сервис не строить.
- Локальное дополнение хранит новые canonical values и явно заданные aliases, не редактирует imported snapshot. Пользователь может выбрать найденный термин, добавить собственный или явно указать другое написание своего термина. Подтверждение alias — явный ввод; fuzzy search только предлагает варианты, не объединяет автоматически. Проектные общепринятые aliases (например React.js) — небольшие проверенные reference data с источником, не всеобщий вручную составленный словарь.
- Никакого LLM при появлении похожего слова, сборе или сохранении. Подтверждение extracted skill в Profile использует тот же resolver, но модель не пишет shared dictionary.
- Применение: после получения полного текста вакансии — известные label/alias spans; при Save профиля — разрешение выбранных значений; после изменения dictionary/profile — локальное обновление derived matching/ranking и stale-отметок без paid assessment. Сохранить original text и namespace/version сопоставления.

## Семантика

- Один concept ID не равен строке. Источник, язык и alias сохранены; одинаковое слово в нескольких concepts даёт ambiguous/unmapped, не случайный первый результат.
- Сохранять различие C / C++ / C#, React / React Native. Границы слова/фразы и регистр нормализуются аккуратно; не удалять всю пунктуацию. При отсутствии уверенного concept match оставить original exact terms в действующем поиске.
- `Director`, `Directeur`, `CSO` без уточнения не объявлять Sales Director. Иерархическое родство не означает равенство и не создаёт совпадение навыков автоматически. Наличие occupation→skill связи не означает, что пользователь владеет всеми навыками профессии.
- ESCO occupations/skills не превращать в справочник отраслей. Industries остаются расширяемыми пользовательскими значениями с явно подтверждёнными aliases; Blockchain/Web3/DeFi не объявлять автоматически одной отраслью/одинаковым опытом.
- Неизвестное название не мешает fetch/store/show; оно не становится подтверждённым отсутствием навыка. Пользователь не обязан ждать обновления словаря, чтобы искать нишевую работу.
- Обе стороны сравнения используют один resolver и одну версию. Current FTS/BM25 остаётся ранжированием: расширить query проверенными вариантами, сохранить group AND/OR/exclusions и дать reason с исходным совпавшим фрагментом. Не обещать структурированную salary/location weighted fit формулу.
- Для источников с query-параметрами варианты доступны существующему query builder; для listing-only сайтов продолжается discover→full text→local matching. Не заставлять каждый сайт принимать одинаковую taxonomy, не переписывать весь collection pipeline или enabling JobSpy в этой карточке.

## Реальные внешние скрапперы — установленное и неизвестное

- [COL-1 done](../../collection/COL-1_source-abstraction/done.md) описывает Source boundary, guarded fetch, generic и ATS/jobspy adapters; `backend/internal/source/jobspy.go` действительно существует.
- [Исторический backlog, пункт 6](../../../backlog.md) предлагал JobSpy для крупных boards, JSearch/SerpApi позже. Это не подтверждает текущую доступность, включение или полноту feeds. `background/collect.go` registry wiring проверять отдельно; наличие adapter не означает, что он зарегистрирован в текущем run.
- Собственные сайты владельца — независимый first-class путь; vocabulary не зависит от inventory агрегатора. Покупка feed, смена источников и обещание их coverage не входят.

## Слои

Versioned reference snapshot + SQLite local terms/aliases — data; existing config/Dirs — пути и параметры; общий domain resolver/matcher — logic; Profile combobox/reason views — presentation. Никаких aliases, user examples, adjustable weights, provider/model и machine paths в views или рассыпанных regex. Не создавать taxonomy service/framework. Существующие profile/source adapters переиспользовать.

## Исследование и источники — проверено 2026-11-09

- [ESCO download](https://esco.ec.europa.eu/en/use-esco/download): бесплатные datasets, 28 языков, форматы, pinned versions; форма email и отдельный API путь. Русского в опубликованном списке языков нет; русский пользовательский текст не выдавать за автоматически переведённый.
- [Структура данных](https://esco.ec.europa.eu/en/structure-esco-downloadable-datasets): occupations/skills, concept URI, preferred/non-preferred terms и отдельные relations.
- [ESCO services](https://esco.ec.europa.eu/en/use-esco/use-esco-services-api): hosted/local API; EUPL относится к API software, его не встраивать и не путать с лицензией data. [EU reuse notice](https://commission.europa.eu/legal-notice_en): attribution/изменения, с учётом возможных dataset exceptions.
- [O*NET](https://www.onetcenter.org/database.html): межотраслевые occupations, alternate/job titles, software/other skills, downloadable data; [data license](https://www.onetcenter.org/license_db.html). Реальная альтернатива, не второй обязательный import. Слияние двух taxonomy добавит неоднозначность и не входит в первый результат.
- Полный ESCO package/покрытие React, blockchain и reptile care пока не скачаны и не проверены. Исследование подтвердило формат/назначение, не исчерпывающую пригодность отдельных терминов. Одна угаданная ссылка terms-use вернула 404; использованы официальные download/API/reuse страницы, условия конкретного data release ещё проверить при получении.

## Оценка и все шаги

Сложность 8. Preparation 20–30мин, implementation 75–115мин, acceptance 25–35мин; **120–180мин**. Stop/re-estimate на 180мин; внешние download задержки отдельно. Нет runtime/model/платного API, ESCO download бесплатен; стоимость coding subscription неизвестна. Проверка/контекст/review уже внутри строк.

| Шаг | Результат / зачем | AI-время | Внешняя цена |
|---|---|---|---|
| 5.1 | Проверенный pinned ESCO snapshot и маленькое local overlay; конкретный источник вместо ручного всесветного словаря | 30–45мин | Бесплатные data; доступность download отдельно |
| 5.2 | Resolver и расширяемый Profile selector с aliases; одинаковая семантика ввода | 30–45мин | Новых сервисов нет |
| 5.3 | Подключение к existing query/ranking и reasons; доказать пользу на вакансиях любых источников | 35–55мин | Только локальный расчёт |
| 5.4 | Focused alias invariants, build и browser; исключить ложные объединения | 25–35мин | Новых сервисов нет |

Все шаги обязательны для этой отдельно одобряемой карточки. Работа над ней не условие запуска остальных исправлений Profile. Ни отдельной research-only стадии без применения, ни опционального taxonomy management UI сверх простого ввода aliases.

## Checkpoints

- [ ] 5.1 — Зафиксировать проверенный ESCO snapshot, происхождение и расширяемые локальные значения.
- [ ] 5.2 — Включить общий resolver в Profile selection и явные aliases.
- [ ] 5.3 — Подключить те же варианты к текущему поиску, ранжированию и причинам совпадения.
- [ ] 5.4 — Подтвердить межотраслевой и нишевый workflow без LLM.

## Acceptance criteria

- В приложении React/React.js/Reactjs дают одно выбранное понятие и одинаковое совпадение; React Native не подмешивается. C/C++/C# и ambiguous CSO/Director не склеиваются. Минимальный targeted-тест этих скрытых false-positive/alias-score invariants, не матрица UI.
- Показан один non-IT импортированный concept и один пользовательский нишевый термин, отсутствующий в snapshot, с фактическим совпадением в original vacancy text. Не заявлять конкретный animal-care термин найденным без проверки данных.
- Вакансии собственных sources сохраняются и показываются без словаря/агрегатора; no match/unmapped не выдаётся за доказанное отсутствие опыта. Ничего не фильтруется из каталога скрыто.
- Source/version attribution единообразны; смена aliases локально пересчитывает derived results без model/сети, original text сохранён, owner corrections не перезаписаны.
- Пользователь видит working selector и actual ranking/reasons в браузере; passing importer tests без подключения недостаточно. Build отдельно; broad suites SKIPPED; записаны время/owner review.
