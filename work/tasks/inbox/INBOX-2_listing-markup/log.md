## 2026-11-13 — Round 6: влияние main в ветку work/INBOX-2 перед релизом

Координатор выдал исполнителю Round 6: влить main в `work/INBOX-2` перед релизом (не запускать `work-finish`, `sift-promote`, `sift-deploy`, не трогать main-чекаут, не пушить).

### Merge

База слияния: `c00f174`. В main за это время поступило 55 коммитов. Ключевые: INBOX-4 (unrestricted remote), ASSESSMENT-10 (метки ограничений), SETTINGS-8 (факты профиля), OPS-11 (покрытие спецификаций).

**Четыре конфликта:**

1. `backend/internal/matchcontract/contract.go` — main добавил тип `RegionEntry`, поле `CurrentLocation *ProfileSnapshot`, поля `Regions`/`RestrictionPhrases`/`WrapperPrefixes`/`TrailingQualifiers`/`Separators` на `Rules`, критерий `CriterionGeographyRemoteRestriction`, коды причин `ReasonRemoteRestricted*`. Ветка добавила `ExchangeRates map[string]float64` на `Rules` и `ReasonCompensationRangeCovers`. Оба набора сохранены.

2. `backend/internal/quickmatch/quickmatch.go` — main изменил подписи `evaluateGeography(profile, facts)` → `evaluateGeography(profile, facts, rules)` и `evaluateConditions(profile, facts)` → `evaluateConditions(profile, facts, rules)`. Ветка изменила `evaluateCompensation(profile, facts)` → `evaluateCompensation(profile, facts, rules.ExchangeRates)`. В итоге все три вызова обновлены.

3. `docs/MAP.md` — main обновил строку Profiles (добавил SETTINGS-9/S.1, «Spec updated OPS-11»), добавил строки Collection/Catalogue/Triage/Application preparation/Local data, убрал «not implemented» из Vacancy view. Ветка добавила строку Listing view, пометила Vacancy view как «Implemented in INBOX-2». В итоге: строка Listing view из ветки, строка Profiles из main (более новая), строка Vacancy view из ветки, новые строки из main.

4. `docs/specs/vacancy-view.md` — ветка заменила ✓/✕ кнопки на +/−. Main добавил описание роутинга требований после кнопок. В итоге: символы +/− из ветки вместе с текстом роутинга из main.

**Дополнительное исправление (не конфликт):** `backend/internal/server/routes.go` вызывал `jobextract.MergeModelConditions` со старой 2-аргументной сигнатурой; main изменил на 3 аргумента `(matchcontract.VacancyFacts, matchcontract.ExtractedConditions, []string)`. Исправлено импортом `matchconfig` и передачей `rules.RestrictionPhrases` через `matchconfig.Rules(cfg.Runtime.EffectiveQuickMatch(), cfg.VacancyView.ExchangeRates)`.

**Два тестовых файла ветки обновлены:** `backend/internal/jobextract/modelconditions_test.go` — 4 вызова `MergeModelConditions` получили третий аргумент `nil`. `backend/internal/jobextract/modellocation_test.go` (пришёл из main) — `matchconfig.Rules(config.DefaultQuickMatch())` → `matchconfig.Rules(config.DefaultQuickMatch(), nil)`.

**Версии миграций:** база слияния = 22, main остаётся на 22, ветка добавляет 23–27; коллизии нет.

**Итог тестов после слияния:**
- `go build ./...` — чисто.
- `go vet ./...` — чисто.
- `npx tsc --noEmit` — 0 ошибок.
- `go test -count=1 ./...` — 4 известных сбоя: `TestRouteDrift` (8 отсутствующих фикстур), `TestMigrationV14`, `TestMigrationV16ToV17PreservesData`, `TestSchema_FreshDatabase`. Остальные проходят, включая `matchsvc`, `quickmatch`, `profile`.

Коммит слияния: `df00e00`, сообщение `merge(INBOX-2): bring main into the branch before release`.

### Проверка UI после слияния (порт 9198, production copy)

Сервер: `scratchpad/r6bin/sifty serve --data-dir scratchpad/r6-data --port 9198 -dist scratchpad/r6bin/dist`.

**Скриншоты (8 файлов, все различны по MD5, префикс `R6-`, скрипт `screenshots/shoot-r6.mjs`):**
- `R6-950-band-1440.png` — job 950 (MTS — Senior Software Engineer, KamiwazaAI). Четыре плитки: Money ✓ ($150–250k, ≈ €230,000), Location (Silverthorne, нет вердикта), Work mode ✓ (Remote), Contract ? (not stated). What they ask: 8 требований, кнопки + и −. What they offer: Base salary, Equity, Health, Time off.
- `R6-950-ask-card.png` — ракурс на карточку What they ask job 950: required и nice-to-have с кнопками + и −.
- `R6-950-band-400.png` — job 950 при ширине 400 px; band в one-column.
- `R6-950-full-listing.png` — полный текст вакансии 950 (прокрутка вниз); headings и текст без тегов.
- `R6-1011-band-1440.png` — job 1011 (Senior Engineering Manager, Infrastructure, GlossGenius). Четыре плитки: Money ✓ ($230–280k, ≈ €258,000), Location (San Francisco), Work mode ✕ (Hybrid vs Remote), Contract ? (not stated). What they ask: 30 требований, кнопки + и −.
- `R6-1011-offer-card.png` — ракурс на карточку What they offer job 1011: Base salary, Equity, Health, Time off.
- `R6-1011-ask-buttons.png` — прокрутка к нижним нерешённым строкам job 1011; кнопки + и −.
- `R6-1294-remote-restriction.png` — job 1294 (Engineering Manager Knowledge Graph Platform, Reddit via We Work Remotely). Плитка Location: «US & Canada» без вердикта. Плитка Contract ✕ (Employment vs contract). + и − кнопки.

**Отступы overview-stack:** `gap: 20px`, `paddingBottom: 20px` (замерено через `window.getComputedStyle` на контейнере band-tile). Значение `gap: 12px` из `.overview-stack` в `theme.css` применяется между блоками (`FitsYouBand`, `div.what-cards`, `section.listing-panel`), тогда как 20 px — внутренний gap плиток и padding самого band-контейнера.

**Порт 9198 остановлен.**

---

## 2026-11-13 — L.5r9: кнопки + и − вместо ✓ и ✕

Координатор выдал исполнителю Round 5 с шестью пунктами: (1) роутинг только по словарю, (2) кнопки «Move to skills» / «Move to facts», (3) исправление коллизии ключей C/C++/C#/F#, (4) замена ✓/✕ кнопок на +/−, (5) обновление документации, (6) верификация. Затем координатор скорректировал объём: пункты 1–3 перенесены в новую карточку SETTINGS-10. Владелец: «по INBOX-2 там ОДНА И ТА ЖЕ карточка и overview делает в INBOX — и уже делает почему-то работу в профиле. это неправильно, должна быть отдельная карточка».

**Изменения (только пункт 4 и документация):**

В `frontend/WhatTheyAsk.tsx`: кнопка добавления изменена с `'✓'` на `'+'`; кнопка отклонения — с `'✕'` на `'−'`; кнопки вариантов — с `'✓ ${alt}'` на `'+ ${alt}'`. Знаки вердикта (✓/✕/?) на решённых строках и счётчики в заголовке не изменены.

В `frontend/QuickMatchPanel.tsx`: кнопка `fit-skill-action add` — с `'✓'` на `'+'`, добавлен `title="Add this skill to your profile"`; кнопка `fit-skill-action dismiss` — с `'✕'` на `'−'`, добавлен `title="Mark that you do not have this skill"`.

**Скриншоты (порт 9198, production copy, PID 72065):**
- `R5-skill-undecided.png` — job 950; строка навыка в состоянии «?»; кнопки показывают `+` и `−`.
- `R5-slash-before.png` — job 1279; строка «Datadog / OpenSearch» до нажатия.
- `R5-slash-alternatives.png` — после нажатия `+`: кнопки показывают `+ Datadog` и `+ OpenSearch`.
- `R5-list-row.png` — строка с запятым списком; кнопки `+` и `−`.
- `R5-nonsk-undecided.png` — job 1083; строка не-навыка в состоянии «?»; кнопки `+` и `−`.

**Таймирование** (3 нажатия `+` подряд на вакансии 950): каждый щелчок — 537–611 мс включая 500 мс намеренной паузы скрипта; реальный ответ API ≈ 37–111 мс. Запросов полного списка вакансий за три нажатия: 1 (только начальная загрузка).

**Итог тестов:**
- `go test -count=1 ./...` — проходят все, кроме четырёх известных (`TestRouteDrift`, `TestMigrationV14`, `TestMigrationV16ToV17PreservesData`, `TestSchema_FreshDatabase`).
- `go vet ./...` — чисто.
- `npx tsc --noEmit` — 0 ошибок.

**Порт 9198 остановлен.**

---

## 2026-11-13 — Issue A/B: «Datadog / OpenSearch» и «C++ or C» не обновлялись после ✕

После коммита `f71b54c` координатор запустил проверочный сервер (порт 9198, production copy) и обнаружил два дефекта: «Datadog / OpenSearch» не предлагал варианты по ✓, а «C++ or C» оставалась «?» после ✕.

### Issue A — «/» проверялся через `isFullVocabMatch` вместо `isAlternativeName`

В `vacancyai/classify.go` функция `ClassifySkillAdd` использовала `isFullVocabMatch` для разделителя «/» — точное совпадение со словарём. «Datadog» не находился точным совпадением (его нет в словаре как самостоятельной единицы), поэтому «Datadog / OpenSearch» не признавался набором альтернатив и кнопка ✓ добавляла всю фразу как один навык или подтверждённый факт.

**Исправление:** проверка заменена на `isAlternativeName` (словарь ИЛИ однослово + `IsSkillName`), как уже было реализовано для разделителя «or». Если хотя бы одна часть не проходит `isAlternativeName` (например, «reporting/planning», где «reporting» — глагольная форма), возвращается `ConfirmedFact`, а не `HasAlternatives`. «Datadog / OpenSearch» → `{HasAlternatives: true, Alternatives: ["Datadog", "OpenSearch"]}`.

**Новые тесты в `classify_test.go`:** «Datadog / OpenSearch» → alternatives; «reporting/planning» → ConfirmedFact; «C++ or C» → alternatives (оба прошли isAlternativeName через vocab match); «reporting, planning» → ConfirmedFact.

### Issue B — ✕ на «C++ or C» и «AWS, Kubernetes, Terraform, ArgoCD» не переключал строку

Нажатие ✕ вызывает `POST /api/profile/absent-skills`. Прежний обработчик записывал всю фразу как одну строку absent skill и не расщеплял её на альтернативы. `codeMatch` в `vacancyai/skillmatch.go` не проверял or/slash/comma-части при вынесении вердикта через `absentByGroup` / `absentByName`.

**Исправление в `profile_api.go`:** обработчик `POST /api/profile/absent-skills` теперь вызывает `ClassifySkillAdd(body.Name, nil)` и разбирает результат: при `HasAlternatives` — добавляет каждый вариант отдельно; при `SkillsToAdd` — добавляет каждую часть отдельно; при `ConfirmedFact` — добавляет полную фразу. Коллизия ключей (C++ и C оба дают ключ «c» в `skillKeyOf`) обработана отдельно (см. ниже).

**Исправление в `skillmatch.go`:** добавлены три блока проверки `absentFact` внутри `codeMatch`:
1. После проверки comma-частей в `haveByName`: если ни одна не найдена, проверяется, все ли части отмечены как absent — тогда `VerdictNo, ReasonMarkedAbsent`.
2. После разбивки по «/»: аналогичная проверка slash-частей на absent.
3. В новом блоке `anyOf && found==0` для or-требований: каждая or-часть проверяется на absent; если все absent — `VerdictNo`.

**Новые тесты в `codematch_test.go`:** поле `absentParts` добавлено в тестовую структуру; проверяются: «C++ or C» с «C++ or C» в absent → `VerdictNo`; «AWS, Kubernetes, Terraform, ArgoCD» с каждой из частей в absent → `VerdictNo`; «Datadog / OpenSearch» с «Datadog» в absent → `VerdictNo`.

### Баг коллизии ключей в absent-skills

`skillKeyOf("C++") = "c"` и `skillKeyOf("C") = "c"` (функция в `profile/service.go` убирает небуквенно-цифровые символы). Вызов `AddAbsentSkill("C++")` и `AddAbsentSkill("C")` записывал только одну строку — «C++», вторая отбрасывалась как дубликат. Поскольку C и C++ — разные группы словаря, `allAbsent` по условию «обе группы absent» не выполнялось, и строка оставалась «?».

**Исправление в `profile_api.go`:** после цикла сохранения альтернатив, если `storedCount < len(names)` (произошла коллизия), дополнительно записывается полная фраза `body.Name` («C++ or C»). `indexProfile` при сканировании «C++ or C» находит оба словарных группы C++ и C, устанавливает `absentByGroup["C++"]` и `absentByGroup["C"]`, и `allAbsent` возвращает `VerdictNo`. После нажатия ✕ на «C++ or C» в absent-skills появляются две записи: `("C++", "c")` и `("C++ or C", "c-or-c")`.

### Дефект регрессии пробелов обзора (рефактор L.5r1–r3)

При рефакторинге L.5r1–r3 обёртка `.overview` с `gap` была заменена на React-фрагмент без контейнера. Четыре элемента вкладки Overview (`FitsYouBand`, `div.what-cards`, `section.listing-panel`, `section.rail-card.ai-note`) рендерились как голые соседи без вертикального промежутка.

**Исправление в `frontend/RoleContent.tsx`:** `{tab==='overview'&&<>...</>}` заменено на `{tab==='overview'&&<div className="overview-stack">...</div>}`.

**Исправление в `frontend/theme.css`:** добавлено `.overview-stack{display:grid;gap:12px;min-width:0;margin-top:14px}` после существующих правил band/overview. Узкий вид (≤ 900 px) сохраняет single-column через уже имевшееся правило `.what-cards`.

### Итог тестов

- `go test ./internal/vacancyai/...` — все pass (включая новые кейсы classify_test и codematch_test).
- `go build ./...` — чисто.
- `npx tsc --noEmit` — 0 ошибок.
- Pre-existing failures (`TestRouteDrift`, `TestMigrationV14`, `TestMigrationV16ToV17PreservesData`, `TestSchema_FreshDatabase`) — без изменений.

### Скриншоты (port 9198, production copy)

- `IA-1279-before.png` — job 1279, строка «Datadog / OpenSearch» в состоянии «?», кнопки ✓/✕.
- `IA-1279-alternatives.png` — после нажатия ✓: строка показывает кнопки «✓ Datadog» и «✓ OpenSearch».
- `IA-1279-after-datadog.png` — после нажатия «Datadog»: строка разрешена.
- `IB-1083-before.png` — job 1083, строка «C++ or C» в состоянии «?».
- `IB-1083-after.png` — после нажатия ✕: строка показывает «Does not have: C++ or C»; счётчики изменились с «✓1 ?13» на «✓1 ✕1 ?12».
- `IB-1011-comma-before.png` — job 1011, строка «AWS, Kubernetes, Terraform, ArgoCD» в состоянии «?».
- `IB-1011-comma-after.png` — после нажатия ✕: строка показывает «Does not have: AWS».
- `overview-spacing-1440px.png` — 1440 px; зазор 12 px между всеми четырьмя панелями (измерен через `getBoundingClientRect`).
- `overview-spacing-1100px.png` — 1100 px; тот же зазор 12 px.
- `overview-spacing-400px.png` — 400 px (single column); зазор 12 px.

**Порт 9198 остановлен.**

---

## 2026-11-13 — Дефект 6: «Elixir or Erlang» оставалась «?» после выбора альтернативы

Координатор проверил коммит `2929ca4` и нашёл один дефект и несоответствие в скриншотах.

### Дефект: or-альтернативы не проверялись в `haveByName`

Вакансия 327, строка «Elixir or Erlang»: нажатие ✓ открывало кнопки «✓ Elixir» и «✓ Erlang»; нажатие «✓ Erlang» добавляло навык, но строка оставалась «?». Корневая причина: `codeMatch` в `vacancyai/skillmatch.go` не проверял части or-требования в `haveByName` и не проверял slash-части в блоке `leftover && !anyOf`.

**Анализ для «Elixir or Erlang»:** `scan` находит Elixir (группа vocabulary) и оставляет Erlang как leftover (`anyOf=true`). Код не входил в блок `leftover && !anyOf`. `idx.byGroup["Elixir"]` не находил «Erlang» (у пользователя Erlang, не Elixir). `len(groups)==1` → блок `len(groups)==0` пропускается. `allAbsent` завершается по другому условию. Итог: `false` → модель не вызывается (отключена) → «?».

**Анализ для «Datadog / OpenSearch»:** `scan` находит OpenSearch (группа Elasticsearch) и оставляет Datadog как leftover (`anyOf=false`). Существующий блок `leftover && !anyOf` разбивал по запятой, но не по «/». Итог: `false`.

**Исправление в `vacancyai/skillmatch.go`:**
1. В блоке `leftover && !anyOf` (после запятой-разбивки): добавлена slash-разбивка через `splitBySlash(r.Text)`. Охранник: каждая часть должна пройти `isAlternativeName` (известна словарю ИЛИ одно слово прошедшее `IsSkillName`). Любая часть найдена в `haveByName` → `VerdictMatch, ReasonSkillInProfile`. (Первая версия возвращала VerdictPartial при совпадении одной из slash-частей; исправлено на VerdictMatch, потому что «/» — разделитель альтернатив, любое совпадение достаточно.)
2. После блока `len(groups)==0`: добавлен блок `if anyOf && found == 0`. Вызывает `splitByOr(r.Text)`, проверяет все части через `isAlternativeName` (охранник предотвращает ложное совпадение «no go decisions» через `haveByName["go"]`). Любая or-часть найдена в `haveByName` → `VerdictMatch`. Также проверяет `idx.absentByName[skillOfferName(r.Text)]` для учёта ✕.

**Решение по ✕:** нажатие ✕ на «Elixir or Erlang» записывает полную фразу через `addAbsentSkill("Elixir or Erlang")` → `absentByName["elixir or erlang"]`. Новый блок `anyOf && found==0` проверяет этот ключ и возвращает `VerdictNo, ReasonMarkedAbsent`. Строка становится ✕. Семантика: «у меня нет ни того, ни другого».

**Новый тест `TestOrAndSlashAlternativesNonVocabMatchedByPart`** в `vacancyai/codematch_test.go` (5 кейсов, 0 вызовов модели):
- «Erlang chosen from or-alternatives»: профиль имеет «Erlang» → VerdictMatch
- «Elixir chosen from or-alternatives»: профиль имеет «Elixir» → VerdictMatch
- «or-alternatives dismissed via ✕»: absent «Elixir or Erlang» → VerdictNo, ReasonMarkedAbsent
- «Datadog chosen from slash-alternatives»: профиль имеет «Datadog» → VerdictMatch
- «OpenSearch chosen from slash-alternatives»: профиль имеет «OpenSearch» → VerdictMatch

**Итог тестов:**
- `go test -count=1 ./internal/vacancyai/... ./internal/background/... ./internal/profile/...` — все pass.
- `go build ./...` — чисто.
- `npx tsc --noEmit` — 0 ошибок.

### Скриншоты (ретейк после обнаружения идентичных файлов)

Координатор обнаружил, что шесть файлов из коммита `2929ca4` (D5-*.png) были байт-идентичны (MD5 `47a2b6ffeb662b566c0baaa602d49f31`). Старые файлы удалены, сделан новый снимок на сервере порт 9198 (копия production, PID 26102, остановлен после съёмки).

Скрипт: `screenshots/shoot-defect5.mjs` заменён на версию из scratchpad (`shoot-d5.mjs`), которая покрывает Дефект 6 и Дефект 5 вместе.

---

## 2026-11-13 — Пять дефектов из ревью координатора (после коммита `0e6b911`)

Координатор проверил коммит `0e6b911` и вернул пять дефектов. Все исправлены в этой сессии до нового коммита.

### Дефект 1 — Регрессия тестов background

Два теста в `backend/internal/background/fitworker_test.go` падали: `TestWorkerScoresBothPassesEndToEndAndLeavesTheAssessmentQueueAlone` и `TestProfileChangeRunsOnlyTheMatchingPass`. Причина: заглушка extraction возвращала `"text":"Rust"`, но `haveByName` находит «Rust» в профиле по коду → 0 вызовов модели вместо ожидаемых 2.

**Исправление в `fitworker_test.go`:** цитата заглушки изменена на `"text":"TypeScript","quote":"You know Rust."` (TypeScript нет в профиле → модель вызывается; цитата «You know Rust.» присутствует в тексте вакансии `ruleDesc`/`otherDoc`). Заглушка matching теперь динамически извлекает ID фактов из промпта (вместо хардкода `evidenceId:"P1"`); когда факт не найден, возвращает `"verdict":"no"` с пустым `evidenceId`. Утверждение на строке 231 изменено с `Counts.Match != 1` на `Counts.No != 1`.

### Дефект 2 — Строка с запятым списком остаётся «?» после добавления навыков через ✓

Для вакансии 1050, строка «Python, Postgres, AWS, distributed systems»: нажатие ✓ добавляло 4 навыка в профиль, но строка оставалась «?», так как `codeMatch` проверял `haveByName` для всей фразы целиком, а не для отдельных частей.

**Исправление в `vacancyai/skillmatch.go` (функция `codeMatch`):** если `haveByName` по полной фразе не находит совпадения, вызывается `splitSkillParts(r.Text)`. При нескольких частях каждая проверяется в `haveByName`. Все найдены → `VerdictMatch, ReasonSkillInProfile`. Часть найдена → `VerdictPartial, ReasonSomeNamedSkills`.

**Новый тест `TestCommaListRequirementMatchedAfterAddingParts`** в `vacancyai/codematch_test.go`: полное совпадение (все 4 части в профиле → VerdictMatch, 0 вызовов модели) и частичное (3 из 4 → VerdictPartial, 0 вызовов модели).

### Дефект 3 — «Go or Rust» никогда не предлагал варианты

Поле `line.alternatives` заполнялось matching-проходом модели (отключён); в хранимых результатах оно отсутствовало. Поэтому `hasAlts` в `WhatTheyAsk.tsx` всегда было `false`, и нажатие ✓ на «Go or Rust» добавляло «Go or Rust» как один навык.

**Исправления:**

— `vacancyai/classify.go`: `SkillClassification` получил поле `Alternatives []string`. `ClassifySkillAdd` обнаруживает паттерны «X or Y» до разбивки по запятым; разбивает по ` or ` (без учёта регистра) и проверяет каждую часть через `isAlternativeName`. Все части допустимы → `{HasAlternatives: true, Alternatives: parts}`. Разделитель `/` обрабатывается аналогично, когда все части известны словарю. Новые вспомогательные функции `splitByOr`, `splitBySlash`.

— `isAlternativeName`: известна словарю (точное совпадение через `isFullVocabMatch`, не частичное сканирование) ИЛИ одно слово, прошедшее `IsSkillName`. Добавлена `isFullVocabMatch` в `skillmatch.go` — проверяет `v.terms[joinLower(toks)]` точно, без ложных срабатываний на подстроки (так «Familiarity with Go» не квалифицируется как альтернатива).

— `server/profile_api.go`: POST `/api/profile/skills` использует `cls.Alternatives` (обнаруженные сервером) в ответе `offer_alternatives`, с fallback на `body.Alternatives` (переданные клиентом).

— `frontend/WhatTheyAsk.tsx`: удалён параметр `alternatives` из типа `onAddSkill`. Добавлено состояние `serverAlts`. `handleAdd` сохраняет `r.alternatives` из ответа сервера в `serverAlts` и выставляет `showAlts=true`. Рендер использует `serverAlts` вместо `line.alternatives`.

— `frontend/RoleContent.tsx`: сигнатура `addSkill` изменена с `(name: string, alternatives?: string[])` на `(name: string)`.

**Исправленный баг в `IsSkillName`:** функция использовала `v.scanName` (частичное сканирование) для проверки словаря, из-за чего «Familiarity with Go or Rust» проходила проверку (потому что «Go» находился как подстрока). Исправлено заменой `scanName` на `isFullVocabMatch` как в `IsSkillName`, так и в `isAlternativeName`. Тест `TestClassifySkillAdd/Familiarity_with_Go_or_Rust` теперь проходит.

### Дефект 4 — DELETE confirmed-facts по-прежнему медленный

DELETE `/api/profile/confirmed-facts/{key}` занимал ~4,76 с. POST-маршруты работали быстро.

**Исправления в `profile/service.go`:** `RemoveConfirmedFact` и `UpdateSection` используют `updateSectionFast` (пропускает синхронный полный пересчёт). `Update` (полный PUT профиля) возвращён к `syncAfterSave` (сохраняет синхронный refresh контекста И `Recalculate`), так как `TestProfileSaveUpdatesDerivedContextAndFastRanking` требует обоих.

**Исправления в `server/profile_api.go`:** DELETE `/api/profile/confirmed-facts/{key}` и PUT `/api/profile/sections/{section}` вызывают `scheduleBackgroundRescore(d)` при изменении. PUT `/api/profile` уже содержал этот вызов.

**Замер на копии production (порт 9198):**
- DELETE `/api/profile/confirmed-facts/{key}` при изменении: ~20 мс (было 4,76 с).
- POST `/api/profile/skills` при изменении: ~34 мс.
- PUT `/api/profile/sections/skills` (без изменений): ~12 мс.

### Дефект 5 — Таблица соответствий

Решение владельца (2026-11-13): `"AI coding tools" должно остаться навыком — да`.

**Новое поле `ModifierWords []string`** в `config/skills.go` (yaml: `modifier_words`). Дефолтный список `modifier_words: [coding, testing, scripting, streaming, programming]` добавлен в `config/skills.default.yaml`. Структура `Vocabulary` получила поле `modifierWords map[string]bool`. В `IsSkillName`: слово из `modifierWords` внутри фразы из ≥ 2 слов → НЕ индикатор активности; одиночное → индикатор активности. Не-modifier -ing слова — индикатор активности только на первой или последней позиции.

**Таблица соответствий** (проверена на копии production, порт 9198):

| Фраза | Действие сервера | Ответ сервера |
|---|---|---|
| «AI coding tools» | skills_added | `["AI coding tools"]` |
| «AI coding tools experience» | fact_confirmed | `"AI coding tools experience"` |
| «Python» | skills_added (нет в профиле) | `["Python"]` |
| «Databricks» | skills_added | `["Databricks"]` |
| «TypeScript, Node» | skills_added | `["TypeScript", "Node"]` |
| «Go or Rust» | offer_alternatives | `["Go", "Rust"]` |
| «Coding, debugging, reviewing» | fact_confirmed | `"Coding, debugging, reviewing"` |
| Нетехническая фраза (✕) | fact_confirmed_absent | changed: True |
| «Go» (после выбора альтернативы) | skills_added | `["Go"]` |

**Скриншоты** (порт 9198, production copy; переснято после обнаружения идентичных файлов — см. запись Дефект 6 ниже):
- `D5-elixir-erlang-before.png` — job 327 «Software Engineer I, Full Stack (New Grad)»; строка «Elixir or Erlang» показывает «?» и кнопки ✓/✕.
- `D5-elixir-erlang-alts.png` — после нажатия ✓ строка показывает кнопки «✓ Elixir» и «✓ Erlang» (работает определение альтернатив для or-требований — Дефект 6).
- `D5-elixir-erlang-match.png` — после нажатия «✓ Erlang» строка стала зелёной ✓ (Дефект 6 подтверждён).
- `D5-ai-coding-tools-before.png` — job 898; строка «AI coding tools» с кнопками ✓/✕.
- `D5-ai-coding-tools-after.png` — строка «AI coding tools» стала зелёной ✓ после нажатия (добавлена как навык).
- `D5-coding-confirmed-after.png` — job 1050; строка «Coding, debugging, reviewing» стала зелёной ✓ после нажатия (подтверждённый факт).

**Порт 9198 остановлен.**

### Итог тестов после всех пяти исправлений

- `go test -count=1 ./internal/vacancyai/...` — pass (включая `TestCommaListRequirementMatchedAfterAddingParts`, тесты классификатора, `isFullVocabMatch`).
- `go test -count=1 ./internal/background/...` — pass (включая исправленные stub-тесты).
- `go test -count=1 ./internal/profile/...` — pass.
- `go build ./...` — чисто.
- `npx tsc --noEmit` — 0 ошибок.
- Pre-existing failures (`TestRouteDrift`, `TestMigrationV14`, `TestMigrationV16ToV17PreservesData`, `TestSchema_FreshDatabase`) — без изменений.

---

## 2026-11-13 — Часть A + Б: ✓ на строке навыка, быстрый ответ, исправление haveByName

### Часть A — ✓ на строке навыка не обновлял строку

**Корневая причина.** `codeMatch` в `vacancyai/skillmatch.go` имел два дефекта в проверке `haveByName`:
1. При наличии `leftover`-токенов или пустых `groups` проверка `haveByName` была добавлена корректно (логика), но `haveByName` не заполнялся правильно: `indexProfile` вызывал `skillOfferName(f.Value)` на значении факта вида `"Open source contributions (Proficient)"`, получая ключ `"open source contributions (proficient)"`, тогда как поиск шёл по `skillOfferName(r.Text)` = `"open source contributions"`. Несовпадение — строка не обновлялась.
2. Исправление (`skillmatch.go`): перед вызовом `skillOfferName` обрезается суффикс уровня: если значение факта содержит ` (`, берётся только часть до него. Факт `"Open source contributions (Proficient)"` теперь даёт ключ `"open source contributions"` — он совпадает с запросом по тексту требования.

**Новые/изменённые тесты (`codematch_test.go`):** в `TestHaveByNameMatchesSkillAddedByName` добавлен факт `"Open source contributions (Proficient)"` (с суффиксом уровня) и кейс `"non-vocab skill with level suffix matches requirement without level"`, который проверяет именно этот сценарий.

**Новый файл `vacancyai/classify.go` (ранее добавлен в предыдущей сессии):** `ClassifySkillAdd` и `IsSkillName` определяют, куда относить нажатие ✓: короткое имя без глагольных форм → навык, иначе → подтверждённый факт "Confirmed by you". Правила: ≤ 3 слова, ни одно не оканчивается на `-ing`. «AI coding tools» → подтверждённый факт (потому что «coding» оканчивается на -ing). «TypeScript, Node» → два навыка. «Coding, debugging, reviewing» → подтверждённый факт.

**Изменения в `vacancyai/confirmedfacts.go`:** убрана проверка `r.Kind == matchcontract.RequirementSkill`, теперь `checkConfirmedFacts` применяется к строкам любого вида, включая skill.

**Изменения в `config/skills.go` и `config/skills.default.yaml`:** добавлены `MaxSkillNameWords` и `VerbEndings` с дефолтами `3` и `["ing"]`.

### Часть Б — Медленный визуальный отклик

**`profile/service.go`:** добавлены `updateSectionFast` и `syncAfterSaveFast` — вариант `updateSection`, который не запускает полный пересчёт каталога. `AddSkill`, `AddAbsentSkill`, `AddConfirmedFact` теперь используют его; полный пересчёт перенесён в обработчик маршрута.

**`server/profile_api.go`:** добавлен `rescoreJobAndScheduleBackground` — синхронный пересчёт одной вакансии через `matchsvc.RecalculateJob`, затем фоновый (`scheduleBackgroundRescore` → `SyncDerivedContext`). Маршруты `POST /api/profile/skills`, `POST /api/profile/absent-skills`, `POST /api/profile/confirmed-facts` используют его вместо полного `PrioritizeModelWork`. Кнопка ✓ теперь: отправляет запрос → получает ответ → вызывает `refreshJobFit()` (GET `/api/jobs/:id`) — и строка обновляется, не дожидаясь полного пересчёта каталога.

**`frontend/RoleContent.tsx`:** добавлен `refreshJobFit()`, который вызывает `api(jobs/${job.id})` и обновляет `fullJob`. Обработчики `addSkill`, `dismissSkill`, `confirmHave`, `confirmDoNotHave` вызывают его после каждого API-ответа. Убрана функция `recheck()`.

**`frontend/WhatTheyAsk.tsx`:** `AskRow` показывает класс `pending` и символ `…` во время ожидания API-ответа. Добавлена поддержка вариантов (`alternatives`) через флаг `showAlts` и кнопки по каждому варианту.

**`frontend/theme.css`:** добавлены стили `.ask-row.pending` (opacity 0.6), `.ask-mark` при pending (opacity 0.5), flex-wrap для `.ask-actions` и `.ask-alts`.

### Скриншоты (port 9199, job 950, fresh copy of data/store.sqlite)

**`screenshots/L5r7-before-skill-add.png`** — строка «Cloud networking experience» (kind=skill, verdict=unclear) с кнопками ✓/✕; счётчики ✓2 ?6.

**`screenshots/L5r7-skill-add-pending.png`** — та же строка сразу после нажатия ✓: класс `pending`, символ `…`, кнопки заблокированы. API-запрос перехвачен и задержан на 2 с для съёмки.

**`screenshots/L5r7-after-first-skill-add.png`** — строка «Cloud networking experience» стала зелёной ✓; счётчики изменились с ✓2 ?6 на ✓3 ?5. Строка обновилась синхронно после `refreshJobFit()`, без перезагрузки списка.

**`screenshots/L5r7-after-second-skill-add.png`** — добавлена вторая строка «GenAI tool experience»; счётчики ✓4 ?4. Сводка API-вызовов: `/api/jobs` (полный список) = 1 (только начальная загрузка страницы), `POST /api/profile/skills` = 1, `/api/jobs/:id` (одиночный refresh) = 3. Полных перезагрузок каталога при нажатиях кнопок — 0.

**Порт 9199 остановлен.**

### Итог тестов

- `go test ./internal/vacancyai/...` — все pass (включая новый кейс `non-vocab skill with level suffix matches requirement without level` и `TestHaveByNameMatchesSkillAddedByName`).
- `go test ./internal/profile/...` — все pass.
- `go test ./internal/config/...` — все pass.
- `go build ./...` — чисто.
- `npx tsc --noEmit` — 0 ошибок.
- Pre-existing failures (`TestRouteDrift`, `TestMigrationV14`, `TestMigrationV16ToV17PreservesData`, `TestSchema_FreshDatabase`) без изменений.

---

## 2026-11-13 — L.5r7: доработка по ревью координатора

Пять замечаний координатора по первой версии L.5r7, закрыты в этой сессии.

**1. Tax-basis перенесён в EvaluateMoney.**

Прежде: проверка `jobTax != profTax` была в `evaluateCompensation` перед вызовом `EvaluateMoney`. Исправлено: проверка выполняется внутри `EvaluateMoney` (после блока «no profile goal», перед конвертацией). Оба места (tile и number) используют одну и ту же функцию — расхождения невозможны. Импорт `strings` удалён из `compensation.go`.

**2. Вакансия 979: расхождение tile/number устранено.**

Корневая причина: `vacancyview.ComputeMoneyResult` вызывал внутренний `cashFromSections` в качестве запасного пути, когда `facts.Compensation.Cash == nil`; `evaluateCompensation` при том же условии возвращал `compensation_job_missing`. Итог: tile показывал `unclear`, number — `not_comparable`.

Исправлено: `cashFromSections` и его вызов удалены из `vacancyview/money.go`. `ComputeMoneyResult` теперь читает только `facts.Compensation.Cash` (тот же источник, что и number). Параметр `sections` сохранён в сигнатуре (обратная совместимость с вызовами в `band.go`) как `_`. Удалена вспомогательная функция `min(a, b int)`.

Проверка на скрэтч-копии production: вакансия 979 → `not_stated` (было `unclear`). Количество расхождений после исправления: **0** (до: 1 известное).

**3. Обёртки удалены; тесты перенесены; добавлена таблица согласованности.**

Прежде: `vacancyview/money.go` содержал приватные обёртки `toAnnual` и `convertToProfile`, чтобы не трогать `TestToAnnual` / `TestConvertToProfile` из `band_test.go`.

Исправлено:
- Обёртки `toAnnual` и `convertToProfile` удалены из `vacancyview/money.go`.
- `TestToAnnual` и `TestConvertToProfile` удалены из `vacancyview/band_test.go`.
- В `quickmatch/money_test.go` добавлены `TestToAnnual` и `TestConvertToProfile`, вызывающие `quickmatch.ToAnnual` / `quickmatch.ConvertToProfile` напрямую.
- Добавлен `TestMoneyTileAndNumberCoherence` (пакет `quickmatch_test`): 10 кейсов (EUR/год ok, USD/год ok, USD mismatch, EUR/мес ok, EUR/час mismatch, нечёткий диапазон, граничный минимум, неизвестная валюта, пустая валюта, net vs gross), проверяет согласованность `EvaluateMoney` и `ComputeMoneyResult` (маппинг `not_comparable` → `not_stated`).

**4. Скриншот L5r6-after-confirm.png переснят.**

Первоначальный снимок был сделан, пока кнопка ✓ показывала `…` (API-вызов не завершился). Новый снимок: job 205 (Shakepay), строка «Financial, payments, or crypto-related systems experience» показывает зелёный ✓, кнопки ✓/✕ исчезли, справа в строке — обрезанный текст подтверждённого факта. Скрипт ждёт CSS-класс `match` на целевой строке (до 60 с).

Добавлен также `L5r6-profile-view.png` (ранее неотслеживаемый): профиль-пикер открыт над панелью обзора вакансии (Shakepay Senior SWE, job 205); показывает выпадающий список профилей с кнопкой «✓ Select».

**5. Процессы.**

Скрэтч-сервер (PID 76798, порт 9199) остановлен после съёмки. Перед финальным коммитом выполнена проверка процессов (см. ниже).

**Итог тестов после всех исправлений:**

- `go test ./internal/quickmatch/...` — все pass (включая `TestMoneyTileAndNumberCoherence`, `TestToAnnual`, `TestConvertToProfile`).
- `go test ./internal/vacancyview/...` — все pass.
- Pre-existing failures без изменений.

---

## 2026-11-13 — L.5r7: единая функция расчёта денег

Вынесена общая логика конвертации денег в `quickmatch.EvaluateMoney`. До этого `vacancyview.ComputeMoneyResult` и `quickmatch.evaluateCompensation` держали по своей копии `convertToProfile`, `toAnnual`, `fromAnnual`.

**Новый файл `quickmatch/money.go`:**

- `MoneyInput{Min, Max *float64, Currency, Period, TaxBasis string}` — входные данные по вакансии.
- `MoneyOutput{ConvertedMin, ConvertedMax float64, ApproxEUR string, Verdict string, Reason string}`.
- `EvaluateMoney(input, *ProfileCash, rates)` — возвращает `"ok"/"unclear"/"mismatch"/"not_comparable"`; `not_comparable` покрывает оба случая отсутствия профиля и несравнимых валют/периодов.
- Экспортированные `ToAnnual`, `FromAnnual`, `ConvertToProfile` — те же алгоритмы, что были в обоих пакетах.
- Приватная `approxEURStr` и `moneyFormatAmount` для строки `"≈ €N"`.

**`quickmatch/compensation.go`:**

- Удалены `convertToProfile`, `toAnnual`, `fromAnnual`.
- `evaluateCompensation` вызывает `EvaluateMoney`; tax-basis pre-check перенесён в `EvaluateMoney` по итогам ревью (см. запись выше).
- Verdict `"not_comparable"` → `ReasonCompensationIncomparable`; `"ok"` → `ReasonCompensationAligned`; `"mismatch"` → utility из `out.ConvertedMax / p.Minimum`; `"unclear"` → `ReasonCompensationRangeCovers`.

**`vacancyview/money.go`:**

- Добавлен импорт `quickmatch`.
- Удалены `approxEUR`, `convertToProfile`, `toAnnual`, `fromAnnual`.
- `ComputeMoneyResult` вызывает `quickmatch.EvaluateMoney`, использует `out.ApproxEUR`, verdict `"not_comparable"` → `"not_stated"`; обёртки `toAnnual` / `convertToProfile` удалены по итогам ревью (см. запись выше).

**Новый `quickmatch/money_test.go` (6 тестов):**

- `TestEvaluateMoneyThreeStates` — ok/unclear/mismatch и граничный случай нижней границы.
- `TestEvaluateMoneyPeriodConversion` — $8333/мес → ≈ $99996/год → ok.
- `TestEvaluateMoneyUnknownCurrency` — пустая валюта → `not_comparable`, ApproxEUR пусто.
- `TestEvaluateMoneyUSDByRate` — USD → EUR по курсу, ApproxEUR начинается с `"≈ €"`.
- `TestEvaluateMoneyNoProfile` — nil профиль → `not_comparable`, ApproxEUR всё равно заполнен.
- `TestEvaluateMoneyEmptyInput` — nil Max → `not_comparable`.

**Результат тестов:**

- `vacancyview` 9/9 pass (включая `TestToAnnual`, `TestConvertToProfile`, `TestCashCurrencyUnknown`, `TestComputeMoneyResultThreeStates`).
- `quickmatch` все pass (включая 6 новых).
- Pre-existing failures (8 маршрутов drift, 3 store/migration) без изменений — не затронуты.

---

## 2026-11-12 — L.5r6: «Confirmed by you» — ✓/✕ на строках не-навыков

Добавлен раздел профиля `confirmed-facts`. Нажатие ✓ или ✕ на строке вида «experience/other» (не skill) записывает факт `ConfirmedFact{Key, Text, Have}` в профиль. Следующий matching pass решает ту же требование во всех вакансиях без вызова модели.

**Бэкенд (Go):**

- `profile/contract.go` — добавлен тип `ConfirmedFact` и поле `ConfirmedFacts []ConfirmedFact` в `ProfileData`; `Normalize()` инициализирует срез.
- `profile/section.go` — константа `SectionConfirmedFacts`; `ParseSection`, `mergeSection`, `sectionSnapshot` знают о ней; `mergeConfirmedFacts` и экспортированная `NormalizeFactKey` (строчные, схлопнутые пробелы, отрезаны знаки препинания).
- `profile/service.go` — `AddConfirmedFact(text, have)` и `RemoveConfirmedFact(key)`.
- `server/profile_api.go` — `POST /api/profile/confirmed-facts` и `DELETE /api/profile/confirmed-facts/{key}`; при `jobId > 0` запускается приоритетный matching pass.
- `profile/matchfacts.go` — поля `confirmedHave` и `confirmedDoNotHave`; `MatchFactsAt` эмиттирует факты из `ConfirmedFacts`.
- `vacancyai/confirmedfacts.go` (новый файл) — `buildConfirmedFactsIndex` и `checkConfirmedFacts`; пропускает строки `RequirementSkill`.
- `vacancyai/match.go` — `checkConfirmedFacts` вызывается перед `codeMatch`.
- `server/testdata/fixtures/70_POST_...json`, `71_DELETE_...json` — фикстуры маршрутов; `drift_test.go` — правило нормализации для `{key}`.

**Фронтенд (React/TypeScript):**

- `profile/contract.ts` — интерфейс `ConfirmedFact`; поле `confirmedFacts` в `ProfileData`.
- `profile/api.ts` — `addConfirmedFact` и `removeConfirmedFact`.
- `WhatTheyAsk.tsx` — prop'ы `onConfirmHave`/`onConfirmDoNotHave`; `AskRow` показывает ✓/✕ только для не-skill строк (или только add/dismiss для skill строк); кнопки, которые не могут действовать, не показываются.
- `RoleContent.tsx` — `confirmHave` и `confirmDoNotHave` вызывают API, затем `recheck()`.
- `profile/ProfileView.tsx` — подраздел «Confirmed by you» рядом с «Skills you do not have»; каждый факт удаляем кнопкой ✕.
- `profile/useProfile.ts` — `removeConfirmedFact` в возвращаемом объекте.

**Тесты:**

- `vacancyai/confirmedfacts_test.go` (новый) — `TestConfirmedFactsCrossVacancy`: ✓ на experience в вакансии A → ✓ в вакансии B; ✕ → ✕; skill-строки не затрагиваются.
- `go test ./internal/vacancyai/...` ok, `go test ./internal/profile/...` ok.
- `npx tsc --noEmit` чист.
- `TestRouteDrift`: мои два маршрута убраны из списка «missing»; 5 pre-existing маршрутов по-прежнему пропущены (avatar, absent-skills, skills, sections/{section}, derived/refresh) — не моя область.

**Скриншоты (staging-копия, порт 9199, Shakepay Senior SWE, job 205):**

- `L5r6-before-confirm.png` — строка «Financial, payments, or crypto-related systems experience» с ✓/✕ кнопками.
- `L5r6-after-confirm.png` — та же строка после подтверждения: зелёный ✓, кнопки исчезли, справа текст факта (переснято в сессии 2026-11-13 — первый снимок был сделан пока API-вызов ещё шёл).
- `L5r6-profile-confirmed.png` — профиль «Confirmed by you»: `✓ Financial, payments, or crypto-related systems experience ✕`.
- `L5r6-profile-view.png` — профиль-пикер открыт над панелью обзора; выпадающий список «Tech Lead / + Add new profile».
- Строка стала match (зелёный ✓) сразу после завершения recheck.

**Порт 9199 остановлен.**

---

## 2026-11-12 — коррекция L.5r5: потеря валюты из модели (раунд 4 после приёмки координатора)

Дефект: деньги из model extraction теряли валюту. `cashText` в `jobextract.go` распознавал валюту только по ISO-коду (`currencyCodeRe`), но не по символу `$` в строке вроде `"$230,000-$280,000 per year"`. Следствия: пустой `code` → `approxEUR` выдавал `"≈ €280,000"` вместо `"≈ €230,000"` (пустая валюта ≡ EUR), `convertToProfile` пропускал конвертацию и сравнивал $280k прямо с порогом €90k → ложное ✓.

**Исправлено:**

1. **`jobextract/jobextract.go` — `cashText`:** После ISO-regexp добавлено определение валюты по символу в `value` (`$`→USD, `€`→EUR, `£`→GBP), затем та же проверка в `line`. Теперь `"$230,000-$280,000 per year"` → `code = "USD"`.

2. **`vacancyview/money.go` — `approxEUR`:** Ветка `jobCurrency == ""` теперь возвращает `""` вместо `"≈ €X"`. Пустая валюта не отображается как евро-эквивалент.

3. **`vacancyview/money.go` — `convertToProfile`:** Добавлен ранний `return 0, false` при `jobCurrency == ""`. Пустая валюта → incomparable → verdict "not_stated".

4. **`quickmatch/compensation.go` — `convertToProfile`:** То же самое раннее возвращение, обновлён комментарий. Число Fits you больше не считает деньги без валюты как совпадение.

**Два новых теста:**
- `TestCashCurrencyUnknown` (vacancyview): нулевая валюта → verdict "not_stated", approxEUR пусто, ShortRange показывается.
- `TestCashTextCurrencySymbolDetection` (jobextract): `"$230,000-$280,000 per year"` → `USD`; `"230,000-280,000 per year"` → пустая валюта.

**Результат:** id 950 → `$150–250k`, `≈ €230,000`, ✓; id 1011 → `$230–280k`, `≈ €258,000`, ✓; id 265 → `$155–215k`, `≈ €198,000`, ✓. 26 плиток из 651 показывают диапазон без символа валюты (genuinely unknown — нет ни знака, ни кода), verdict "not_stated", без approx; ранее они ошибочно считались ✓. Fill rate 27.8% (651/2344) на моей копии.

**Скриншот `L5r5-950-1440.png`:** band Money — `$150–250k`, hint "above your minimum · ≈ €230,000", ✓; offer tile big=`$150–250k`, small="Base salary, USD".

**Тесты:** `jobextract` 8/8, `vacancyview` 9/9, `quickmatch` pass. TypeScript check чист. 4 pre-existing провала неизменны.

**Порт 9199 остановлен.**

---

## 2026-11-12 — коррекции L.5r4 (раунд 3 после приёмки координатора)

Три дефекта из обзора координатора (раунд 3). Плюс два дополнительных дефекта, обнаруженных в ходе реализации.

**Defect 1 — What they ask: text breaking letter-by-letter:** `theme.css`: `.ask-text` — удалён `overflow-wrap:anywhere`, добавлен `flex:1 1 auto;min-width:0;word-wrap:break-word`. `.ask-evidence` — добавлены `max-width:35%;overflow:hidden;text-overflow:ellipsis`. `WhatTheyAsk.tsx`: inline evidence теперь только `line.profileValue || undefined`; длинная quote отображается через `title`-атрибут строки, не инлайново.

**Defect 2 — What they offer: salary tile inverted:** `WhatTheyOffer.tsx`: `OfferTile` — для `item.key === 'salary'` big text = `item.value` (диапазон «$150–250k»), small text = `item.label` («Base salary»). `OfferIcon` — добавлен `case 'salary'` с dollar-circle SVG.

**Defect 3 — Money fill rate 16% → 28.8%:** Три исправления.
- `routes.go`: перенос `MergeModelConditions` за пределы блока `facts != nil`; когда `facts == nil` и `extraction != nil`, создаётся минимальный валидный seed (`SchemaVersion`, `JobID`, `ExtractorVersion`, `RawSourceHash` из extraction) для прохода валидации `VacancyFacts.Validate()`.
- `jobextract/jobextract.go` (cashText): separator region теперь очищается от символов валюты (`$`, `€`, `£`) через `TrimFunc`, поэтому `"$230,000-$280,000"` парсится как диапазон (разделитель `-$` → `-`).
- `jobextract/modelconditions.go` (MergeModelConditions): compensation вызывает `cashText(v.Value, v.Value)` вместо `(v.Value, v.Quote)`, чтобы skip-эвристика equity не блокировала парсинг зарплатных диапазонов, чьи extraction-цитаты содержат «+ equity». Quote восстанавливается в `cash.Evidence.Quote = v.Quote` после парсинга.

**Целевой тест:** `jobextract/modelconditions_test.go` — `TestMergeModelConditionsZeroFactsWithSalaryExtraction` проверяет, что нулевой `VacancyFacts` (seed из extraction-метаданных) + extraction с compensation → `cash.Min/Max/Currency` заполнены, hash корректен.

**Fill rate:** 2344 вакансии в API; до исправления ~16%, после — 28.8% (674/2344). Оставшиеся 147 extraction-записей с compensation-значением не парсятся из-за genuinely ambiguous format («Competitive salary», «$181K base + 15% bonus index», decimal-point amounts «188,000.00»).

**Скриншоты** (`screenshots/L5r4-*.png`, shoot-L5r4.mjs, порт 9199):
- `L5r4-950-1440.png` — job 950 at 1440px: What they ask — строки не ломаются посимвольно; Proficiency in Python отображает evidence «Python (Proficient)» справа; offer card — salary tile big=«150–250k», small=«Base salary».
- `L5r4-950-1280.png` — job 950 at 1280px: аналогично; «5+ years software engineering experience» естественно переносится на 2 строки, не посимвольно.
- `L5r4-950-400.png` — job 950 at 400px: все ask-строки переносятся нормально.
- `L5r4-1011-1280.png` — job 1011 (нет stored facts, есть extraction): offer tile big=«230–280k», small=«Base salary / year» — зарплата из extraction показывается.
- `L5r4-100-1280.png` — job 100 (нет extraction): все тайлы «Not stated»; What they ask — «No requirements extracted yet»; What they offer — «No Compensation or Benefits section recognised».

**Тесты:** `jobextract` — 7/7 pass (1 новый: `TestMergeModelConditionsZeroFactsWithSalaryExtraction`). `vacancyview` — 8/8 pass. TypeScript check чист. 4 pre-existing провала неизменны.

**Компиляция:** `go build ./cmd/sifty/` — без ошибок.

**Порт 9199 остановлен** (PID 20995).

---

## 2026-11-12 — коррекции L.5r1–L.5r3 (раунд 2 после приёмки координатора)

Пять групп дефектов, найденных на свежей production-копии (2702 вакансии). Все исправлены до L.6.

**A1 — Layout (обзорная вкладка):** Removed the `.overview` / `.overview-main` / `.overview-rail` 2-column grid from `RoleContent.tsx`. Band → what-cards → listing → AI analysis note now stack full-width, matching the design canvas. The AI analysis note (`#ai-analysis-note`, needed by `aria-describedby`) moved below the listing section.

**A2 — ShortRange формат тайла Money:** Реализовано в предыдущей сессии: `formatShortRange` в `money.go`, новое поле `ShortRange` в `MoneyResult`, `moneyTile` в `band.go` использует его вместо `OriginalRange`. Тест `TestFormatShortRange` добавлен.

**A3 — What they offer:** `ComputeOfferItems` обновлён (`offer.go`): первый тайл — зарплатный диапазон из `MoneyResult` (ключ `salary`, метка «Base salary, USD / year»); затем только найденные пункты словаря (не все 8). Не найденные пункты опускаются. `NotStatedLabel` добавлен в `OfferItemDef` (`config/vacancy-view.go`) и во все записи обоих YAML-файлов (`vacancy-view.default.yaml`, `config/vacancy-view.yaml`). Новый экспортируемый хелпер `MoneyFromSections` в `band.go` позволяет `routes.go` передавать `*MoneyResult` в `ComputeOfferItems` без раскрытия интерфейса `sectionLike`. `routes.go` упрощён: `if facts != nil / else` заменён единым блоком с нулевыми фактами как значением по умолчанию.

**A4 — What they ask — счётчики и evidence:** `WhatTheyAsk.tsx`: счётчики ✓/✕/? теперь вычисляются из `countLineVerdicts(lines)` (а не из `youFit.counts`, которые равны нулю при отключённом matching-проходе). Добавлен span `.ask-evidence` в `AskRow`, показывающий `line.profileValue || line.quote` справа от текста требования.

**B — Work mode и contract вердикты:** Реализовано в предыдущей сессии: `workModeTile` и `contractTile` в `band.go` принимают `profile matchcontract.ProfileSnapshot`; вердикты `workModeVerdict` и `engagementVerdict` сравнивают цель профиля с фактом вакансии.

**C — Ложные срабатывания contract:** `vacancy-view.default.yaml` и `config/vacancy-view.yaml`: удалены одиночные слова `contractor`, `permanent`, `fixed-term`, `fixed term`; добавлены фразы `permanent role`, `permanent position`, `fixed-term contract`, `fixed term contract`. `ReadContractKind` в `contract.go` теперь пропускает секции с классом `"other"` или `""`. Новая функция `contractPhraseMatch` использует `\b...\b` для одиночных токенов (предотвращает совпадение «contractor» в «government contractor») и substring для фраз с пробелом/дефисом. Тест `TestReadContractKindFalsePositives` покрывает примеры из обзора координатора.

**D — Заполняемость Money (13,3% → ожидалось ~34%):** `jobextract/jobextract.go`: `cashSkipRe` заменён на word-boundary regexp (предотвращает совпадение «ote» в «remote»); добавлена проверка `hasSalaryCtx` — skip не применяется когда строка содержит «salary»/«base»/«wage»; > 2 сумм обрабатываются как multi-city range (берётся глобальный min/max). Те же исправления в `vacancyview/money.go` (`parseCashSkipRe`, `parseCashText`, `parseCashSkipRe`). Тест `TestParseCashTextSkipWords` покрывает конкретные id из обзора. Существующий тест `TestExtractEvidenceAndSectionPreservation` обновлён: «Base salary: ... plus bonus» теперь парсится (правильное поведение — контекст salary разрешает skip).

**E1 — Шаблон конфига:** `config/vacancy-view.yaml` добавлен в корень `config/` (рядом с `search.yaml` и другими шаблонами), с описанием всех полей и предупреждением о placeholder-курсах.

**E3 — Удалён `overview-b-mockup.png`:** `git rm work/tasks/inbox/INBOX-2_listing-markup/overview-b-mockup.png`.

**Тесты:** `vacancyview` — 8/8 pass (3 новых: `TestParseCashTextSkipWords`, `TestReadContractKindFalsePositives`, `TestFormatShortRange`); `config` — pass; `jobextract` — pass (1 ожидаемое изменение существующего теста). TypeScript check чист. 4 pre-existing провала (`server`, `store`) не изменились.

**Компиляция:** `go build ./... && npx tsc --noEmit` — без ошибок.

---

## 2026-11-12 — L.5r1–L.5r3 — Fits you band, What they ask/offer, full listing open

**Новые файлы бэкенда:**
- `backend/internal/vacancyview/` — пакет: `money.go` (три состояния вердикта: ok/unclear/mismatch; конвертация мин/макс; парсер cash-текста не пропускает зарплатный диапазон с «equity»), `band.go` (сборка четырёх тайлов + хинты), `contract.go` (контракт из явных слов), `vacancyview.go` (точка входа), `band_test.go` (тесты трёх состояний и job 950).
- `backend/internal/config/vacancy-view.go` и `vacancy-view.default.yaml` — обменные курсы и словарь offer-пунктов в конфигурации.

**Изменения бэкенда:**
- `matchcontract/contract.go`: константа `ReasonCompensationRangeCovers` (unclear-состояние, не считается mismatch).
- `quickmatch/compensation.go`: три состояния вердикта; конвертируются min и max.
- `model/job.go`: поле `Hint string` в `BandTile`; `Verdict` покрывает 5 состояний.
- `matchconfig/matchconfig.go`, `matchsvc/service.go`: загрузка vacancy-view конфига, передача в vacancyview и quickmatch.
- `server/routes.go`: `/api/jobs/:id` вызывает `vacancyview.Compute` и заполняет `job.Band` и `job.WhatTheyOffer`.
- `store/match_model.go`: `WhatTheyOffer` в `StoredMatchResult`.

**Новые файлы фронтенда:**
- `frontend/FitsYouBand.tsx`: левая колонка (число Fits you, bucket-pill, Refresh/Assess); правая — 4-колоночная сетка тайлов (иконка + вердикт-кружок + значение + хинт + approxValue).
- `frontend/WhatTheyAsk.tsx`: иконка-заголовок, счётчики ✓/✕/?, группы Required/Nice to have, кнопки ✓/✕ на строках без вердикта.
- `frontend/WhatTheyOffer.tsx`: иконка-заголовок, тайлы из offer-словаря.

**Изменения фронтенда:**
- `frontend/model/core.ts`: поле `hint` в `BandTile`, расширен тип `verdict`.
- `frontend/theme.css`: стили band-тайла (`.band-tile-head`, `.band-tile-vc`, `.band-tile-hint`), карточек с иконкой-заголовком (`.card-icon-head`), строк ask (`.ask-mark`, `.ask-text`) и offer-тайлов.
- `frontend/RoleContent.tsx`: `FitsYouBand` вынесен за пределы `.overview`-сетки (полная ширина); `useEffect` подгружает `/api/jobs/:id` для `band` и `whatTheyOffer`; удалены `localSections`, `listMode`, переключатель By section / Original, `.fact-strip`, `<QuickMatchPanel/>`; полный листинг всегда открыт.

**Решения по реализации:**
- Вердикт Location = `""` — без кружка (по указанию координатора, т.к. у профиля нет цели Location).
- Три состояния вердикта Money вместо двух: ok (нижняя граница ≥ minimum), unclear (minimum внутри диапазона), mismatch (верхняя граница < minimum).
- Парсер cash-текста пропускал диапазоны с «equity» — исправлено: skip только когда нет слова «salary»/«base»/«wage» рядом.

**Скриншоты** (`screenshots/L5r-*.png`, shoot-L5r.mjs, порт 9199):
- `L5r-job950-band.png` — job 950 (MTS Senior SE, KamiwazaAI): Money ✓ $150k–$250k (above minimum · ≈€230k), Location Silverthorne, Work mode Remote (?), Contract Not stated (?). What they ask (8 requirements, Proficiency in Python…) и What they offer (Equity, Health, Time off, Retirement, Learning, Equipment) видны ниже.
- `L5r-job100-no-extraction.png` — job 100 (Engineering Lead Wallets, Crossmint): все тайлы Not stated кроме Location; What they ask — «No requirements extracted yet»; What they offer — «No Compensation or Benefits section recognised».
- `L5r-job500-no-offer.png` — job 500 (Associate General Counsel, Mysten Labs): аналогично; Location — USA (Remote).
- `L5r-job950-full-listing.png` — листинг job 950 открыт без переключателя: заголовки h3 (Role Overview, Key Responsibilities), маркированный список без тегов.
- `L5r-job950-narrow.png` — 400 px viewport, Fits you band и число 75 корректно.

**Тесты:** все backend-тесты проходят (4 ранее известных провала неизменны). TypeScript check чист.

**Порт 9199 остановлен** (PID 17781).

---

## 2026-11-12 — коррекции после L.4e, раунд 2

**Числа по словарю секций (Коррекция 5)**

Сравнение до (L4e-verify, v26, 2773 вакансии с секциями) и после (inbox2-reshoot, v27, 2776 вакансий):

| Класс | До | После | Дельта |
|---|---|---|---|
| other | 6277 | 5320 | −957 |
| Requirements | 395 | 878 | +483 |
| About | 75 | 408 | +333 |
| Responsibilities | 255 | 560 | +305 |
| Compensation | 88 | 215 | +127 |
| Benefits | 175 | 276 | +101 |
| Nice to have | 153 | 225 | +72 |
| Legal/EEO | 18 | 64 | +46 |
| Role | 278 | 308 | +30 |
| Location | 47 | 57 | +10 |
| Process | 27 | 27 | 0 |
| **Итого** | **7788** | **8338** | **+550** |

Доля classified: 19,4% → 36,2%. «other» снизился с 80,6% до 63,8%.

**Исправление байтовой ошибки в `sections.go`**

При редактировании инструмент записал U+2019 (правая одиночная кавычка, `\xe2\x80\x99`) вместо U+0027 (прямой апостроф, `\x27`) во втором аргументе всех трёх `strings.ReplaceAll` в `normalizeHeading`. В результате: строка 78 заменяла U+2019 на U+2019 (no-op), строка 79 искала U+2019 вместо U+2018, строка 80 заменяла U+02BC на U+2019 вместо U+0027. Исправлено бинарной заменой через Python. Тесты `TestNormalizeHeading` и `TestSplitSectionsTypographicApostrophe` теперь проходят.

---

## 2026-11-12 — коррекции после L.4e (приёмка координатора)

Четыре дефекта, найденных после коммита L.4e, исправлены до L.6.

**Коррекция 1: `catalogueSeq` — атомарный счётчик вместо `MAX(id)`**

Прежний `CatalogueSeq()` возвращал `SELECT COALESCE(MAX(id), 0) FROM jobs` — не увеличивался при изменении статуса, блокировке компании, сохранении результатов матча или добавлении стоп-слова; фронтенд не узнавал об этих событиях до следующего ручного действия.

Исправление: `store.Store` получил поле `catalogueSeq atomic.Int64`. `Open()` загружает начальное значение из `id_sequence WHERE name='catalogue_seq'` (строка засевается миграцией v27 через `COALESCE(MAX(id), 0) FROM jobs`). Новый метод `BumpCatalogueSeq()` инкрементирует атомик и асинхронно сохраняет в `id_sequence` (best-effort). `CatalogueSeq()` читает только атомик без запроса к БД. Вызовы `BumpCatalogueSeq()` добавлены в: `Upsert`, `Seen`, `StatusWithReason`, `Delete`, `StoreRefetchResult`, `BlockJob`, `AddStopWord`, `RemoveStopWord`, `SaveMatchResult` (после `tx.Commit()`), `SaveFitResult`, `RescoreForProfile` (обе ветки). `targetVersion` поднят с 26 до 27.

Версионные проверки в `match_results_test.go`, `match_facts_test.go`, `match_model_test.go` обновлены 26→27. Все backend-тесты проходят (4 ранее известных провала неизменны).

**Коррекция 2: `AbortSignal.timeout` — только для GET, кнопки действий получают состояние ожидания**

Прежняя реализация добавляла `AbortSignal.timeout(10000)` ко всем вызовам `api()`, включая POST-действия (Assess now, Save and reassess и др.) — 10-секундный таймаут мог прервать запрос, который пользователь только что инициировал.

Исправление в `frontend/api.ts`: параметр `timeoutMs?: number` опциональный; `signal` устанавливается только когда он передан. Вызовы в `main.tsx`: `refresh()` — 30 000 мс, все poll GET-запросы — 10 000 мс, `filter-options` GET — 10 000 мс. POST-запросы остались без таймаута. Кнопка «Assess now» в `frontend/RoleContent.tsx` получила состояние `reassessing`: при нажатии показывает «Queuing…» и `disabled` до завершения (или ошибки).

Три новых теста в `frontend/messages.test.mjs`: TimeoutError DOMException → `ServerUnreachableError`; POST без таймаута не прерывается; `"database not responding"` → retry. Все 9 тестов проходят.

**Коррекция 3: словарь секций и нормализатор**

Нормализатор (`backend/internal/collect/normalise.go`): `isHeadingLike` теперь отклоняет текст, первый Unicode-символ которого — строчная буква (фрагменты предложений вроде «is as follows» из LinkedIn — 231 вакансия). Тест `bold_text_starting_with_lowercase_stays_as_paragraph` добавлен.

Классификатор секций (`backend/internal/collect/sections.go`): добавлена функция `normalizeHeading(s)` — нормализует типографские апострофы (`'`/`'`/`ʼ`) → `'`, `&` → `and`, обрезает конечные `:` и `.`, сворачивает пробелы, lowercase. `classifySection` теперь применяет три уровня: (1) точное совпадение после нормализации, (2) prefix-совпадение, (3) contains-совпадение через `containsWord`. Поля `Prefix []string` и `Contains []string` добавлены в `SectionClass`; `SectionEntry` в `config/sections.go` расширен аналогично; `ConfigSectionVocab` в `background/collect.go` копирует оба поля.

Словарь `config/sections.default.yaml` расширен: About — `prefix: [about]`; Responsibilities — `contains: [responsibilities]`, новые headings (in this role you'll, your impact, what you'll own и др.); Requirements — `contains: [requirements, qualifications]`, новые headings (who you are, about you, skills and experience и др.); Nice to have — новые варианты; Compensation — `contains: [compensation, salary]`, base salary, bonus; Benefits — `contains: [benefits]`; Legal/EEO — `contains: [privacy notice]`, явные heading для data privacy notice.

Новые тесты: `TestSplitSectionsTypographicApostrophe`, `TestSplitSectionsPrefixMatch`, `TestSplitSectionsContainsMatch`, `TestNormalizeHeading` в `sections_test.go` — все проходят.

Эффект на production-данных (2318 вакансий, job 950): «Competitive Compensation & Equity:» → Compensation, «Comprehensive Benefits:» → Benefits, «Who We Are» → About, «How to Apply» → Process, «Compensation:» → Compensation (все были other до этой коррекции).

**Коррекция D: скриншоты переснять с обновлённым словарём**

Переснято 4 файла в `screenshots/`: `L57-job950-by-section.png` (Compensation, Benefits, About, Process теперь показываются как метки классов, а не тексты заголовков), `L57-job2815-by-section.png` (About Chainalysis → About через prefix-правило), `L57-job2819-by-section.png` (Job Responsibilities → Responsibilities, Job Requirements → Requirements, About BIT: → About через contains/prefix), `L57-job8-original-key-responsibilities.png` (исправлен неправильный снимок из первой попытки пересъёмки — теперь показывает верный job 8 Amber Group). Скрипт `screenshots/shoot-L57.mjs` обновлён: хэш-навигация заменена кликами по `[data-job-id]`, добавлено ожидание загрузки секций через `waitForBySection()`, job 8 открывается из Shortlist-view. Порт 9199 остановлен.

---

## 2026-11-12 — L.4e — Poll without refetching the catalogue

Backend: `store.CatalogueSeq()` добавлен в `internal/store/jobs.go` — возвращает `SELECT COALESCE(MAX(id), 0) FROM jobs` (дешёвый штамп изменений). `queueStatus()` в `routes.go` включает `catalogueSeq` в ответ `GET /api/queue`. `GET /api/health` теперь возвращает 503 с `{"error":"database not responding"}` при ошибке БД (раньше — 400 через safeHandler).

Frontend: `api.ts` — добавлен `AbortSignal.timeout(10000)` ко всем fetch-запросам; timeout-DOMException перехвачен и преобразован в `ServerUnreachableError`. `main.tsx` — добавлен реф `lastCatalogueSeq` (инициализирован -1); в poll: `await api('health')` добавлен после queue/analysis (503 всплывает через Retry-баннер); `refresh()` вызывается только при изменении `q.catalogueSeq` (не при каждом тике).

Проверка Playwright (`screenshots/shoot-L4e.mjs`, 30-секундное окно): во время активного сбора (triggered auto-collect on mount) `catalogueSeq` менялся 4 раза (2844→2847, новые вакансии добавлялись); `/api/jobs` запрошен ровно 4 раза — по одному разу на изменение seq, ни разу не по таймеру. Queue и health опрашивались 10 раз каждый (~2.5-секундный интервал). Поле `catalogueSeq` присутствует в ответе queue.

---

## 2026-11-12 — L.5.7 — Playwright-скриншоты: By section, Original с h3, отключённый переключатель

Данные: `.backup` копия `data/store.sqlite` (2318 jobs, v22→v26 после запуска сервера), порт 9199. Сервер остановлен. Скрипт в scratchpad `scratchpad/shoot-L57.mjs`.

Сделано 7 скриншотов в `screenshots/L57-*.png`:

**L57-job25-toggle-disabled.png** — Job 25 (GiveDirectly, Hacker News). By section отключён (`disabled`, тултип «Available for listings with at least two headings»), кнопка Original активна. Листинг показывает 1 секцию без заголовков (plain paragraphs). Fallback на plain-text renderer не срабатывает — description_html есть, но 0 headed-секций.

**L57-job950-original-key-responsibilities.png** — Job 950 (KamiwazaAI, Wellfound). Original view: в листинг-тексте видны h3-заголовки («Role Overview» в верхней части экрана; скрипт подтвердил «Key Responsibilities» и 3 других h3 ниже видимой области). By section включён. Нет raw-тегов в тексте.

**L57-job950-by-section.png** — Job 950, By section: 14 блоков, первый открыт («MTS – Senior Software Engineer», preamble). Остальные свёрнуты с «+». Метки используют классы словаря: «Role», «Responsibilities», «Minimum Requirements», «Nice to have», «Purpose & Impact:», «Competitive Compensation & Equity:», «Comprehensive Benefits:», «Who We Are», «How to Apply», «Compensation:».

**L57-job950-collapsed-section-expanded.png** — Job 950, By section: пользователь кликнул второй раздел «Role», он раскрылся. Одновременно открыты «MTS – Senior Software Engineer» и «Role»; остальные свёрнуты. Контент «Role» — текст про Kamiwaza.

**L57-job2819-by-section.png** — Job 2819 (Matrixport/Paradigm). By section: 6 блоков, первый «Preamble» открыт. В preamble — cookie-consent страница LinkedIn (данные этой вакансии содержат страницу согласия cookies вместо описания — проблема сборщика до INBOX-2, не введена этой задачей). Прочие секции: «About BIT», «Benefits», «Job Responsibilities», «Requirements», «Preferred».

**L57-job2815-by-section.png** — Job 2815 (Chainalysis/Paradigm). By section: 5 блоков. Preamble открыт с настоящим описанием компании Chainalysis. Остальные: «In this job, you'll», «We're looking for candidates who have», «Nice to have experience», «Technologies we use».

**L57-job90-w3c-original.png** — Job 90 (Mysten Labs, Web3 Careers). Original: 30 параграфов, 0 элементов `<br>`, clean paragraph structure — нет лишних пустых строк (проблема, которую фиксировал L.5.1). By section отключён (1 секция, 0 заголовков). Пометка description_status: «available body html».

Ограничения данных: все Web3 Careers jobs в triage-базе production имеют 1 секцию (needs_refetch=1, данные до рефетча INBOX-2), поэтому "CoinList without the extra gaps" заменён на Mysten Labs (job 90) — он показывает чистый HTML без лишних br. Первый частично упавший запуск скрипта сохранил дублирующий файл `L57-job8-original-key-responsibilities.png` (показывает то же, что job25-toggle-disabled).

---

## 2026-11-12 — L.5.6 — спецификация listing view

Создан `docs/specs/listing-view.md`: описание двух хранимых форм (`description`, `description_html`), таблицы `job_originals`, хранения секций (`job_sections`), API-поверхности (секции только в одиночном GET, не в списке), правил нормализации источника, поведения переключателя «By section» (включён при ≥ 2 headed-секций), меток секций и fallback-режима. Контракт Role в `docs/architecture/backend-api-contract.md` дополнен полями `descriptionHtml?` и `sections?` с типом `Section` и ссылкой на спецификацию. В таблицу спецификаций `docs/MAP.md` добавлена строка Listing view.

---

## 2026-11-12 — L.5.5 — фронтенд: переключатель «By section», секции-детали

**Изменения:**
- `frontend/model/core.ts`: тип `Section{heading?,class,content}` добавлен; `Job.sections?: Section[]` добавлен.
- `frontend/RoleContent.tsx`:
  - Новый стейт `listMode: 'original'|'section'` (начальное значение — 'original').
  - Локальный стейт `localSections: Section[]|null` — загружается через `api('jobs/${job.id}')` при открытии вакансии (список не возвращает секции); стейт сбрасывается при смене job.id; флаг `active` предотвращает гонку при быстрой смене вакансий.
  - «By section» включён только когда `headedSectionCount >= 2` (секций с непустым heading); иначе `disabled` с тултипом «Available for listings with at least two headings».
  - Компонент `SectionView`: рендерит список `<details>/<summary>` с открытым первым элементом; метка секции — имя класса для словарных классов (About, Requirements…) и текст heading для класса «other» (или «Preamble» для безголовочного preamble).
  - Оригинальный рендерер не тронут; переключение между «Original» и «By section» — кнопки в `.text-mode`.
- `frontend/theme.css`: добавлены стили `.section-view`, `.section-block`, `.section-label`, `.section-content`; используют CSS `details[open]` для индикатора «+»/«−».

**Playwright-проверки (`screenshots/L55-*.png`):**
- Вакансия «Head of Engineering @Lemon.io» (5 секций, 4 с заголовком): By section включён, 5 блоков, первый открыт, метки: «Preamble», «What You'll Do», «Who You Are», «Experience», «Why Lemon.io»; raw-теги в содержимом не видны; кликнутый блок раскрывается; возврат к «Original» работает.
- Вакансия «Software Engineer, AI Coding Tasks @Askable» (1 секция): By section отключён; виден серый стиль кнопки.
- Узкий экран (390 px): By section и секции-детали отображаются корректно.

---

## 2026-11-12 — L.5.4 — API: `sections` в ответе на один запрос

**Изменение** (`server/routes.go`, обработчик `GET /api/jobs/{id}`): после `d.Store.Get(id)` и Decorate, вызывается `d.Store.GetSections(id)`; если секции есть — `job.Sections` заполняется. Список (`GET /api/jobs`) не затронут.

**Измерение нагрузки на payload (scratchpad DB, 2814 вакансий):**
- `GET /api/jobs` (список): 49 507 711 байт — не изменился, секции в список не попадают (`json:"sections,omitempty"`, поле не задаётся).
- `GET /api/jobs/5` (одна вакансия): 22 511 байт без секций → 29 039 байт с секциями (+6 528 байт; 9 секций).

**Итог тестов:** только 4 pre-existing провала (`TestRouteDrift`, `TestMigrationV14`, `TestMigrationV16ToV17PreservesData`, `TestSchema_FreshDatabase`). Новых нет.

---

## 2026-11-12 — L.5.3 — хранилище секций, миграция v26, бэкфилл на копии data/store.sqlite

**Файлы:**
- `store/migrate.go`: `targetVersion` поднят с 25 до 26; Phase 15 добавляет `job_sections(job_id INTEGER PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE, sections TEXT NOT NULL)`.
- `store/sections.go` (новый): `saveSectionsInTx` — JSON-encode + INSERT OR REPLACE в транзакции; `GetSections` — чтение и decode; `SectionCounts{Normalised, Sectioned, Skipped}`; `BuildJobSections(ctx, vocab)` — ре-нормализует `description_html` (только эту колонку), разбивает на секции, пишет в `job_sections`, идемпотентен; `LogSectionCounts` — строчка в лог.
- `model/job.go`: `JobInput.Sections []Section json:"-"` (не сериализуется в API, пишется в транзакции).
- `store/jobs.go`: `Upsert` и `StoreRefetchResult` вызывают `saveSectionsInTx` когда `len(j.Sections) > 0`.
- `background/collect.go`: `ConfigSectionVocab(def config.SectionsDef) []collect.SectionClass` (экспортирована); основной цикл `defaultCollect` и `collectSources` вычисляют `j.Sections` перед `Upsert`; `collectSources` получил 6-й параметр `vocab []collect.SectionClass` (nil — пропустить).
- `background/collect_test.go`: все 8 вызовов `collectSources` дополнены `nil` как 6-й аргумент.
- `server/enrich.go`: `EnrichJob` получил 4-й параметр `vocab []collect.SectionClass`; перед `StoreRefetchResult` вычисляет `j.Sections` если vocab не nil и есть HTML.
- `server/enrich_test.go`, `server/routes.go`: вызовы `EnrichJob` дополнены `nil`.
- `cmd/sifty/serve.go`: после `BackfillDescriptionHTML` вызывается `BuildJobSections` со словарём из конфига; `deps.EnrichFn` устанавливается с vocab, чтобы рефетч тоже писал секции.
- `store/match_facts_test.go`, `store/match_model_test.go`, `store/match_results_test.go`: ожидаемая версия 25 → 26.

**Проверка на `.backup`-копии `data/store.sqlite` (2814 вакансий, v22 → v26):**
- Старт сервера: `startup: backfill description_html: converted=2750 needs_refetch=70 skipped=0`
- `sections: normalised=7 sectioned=2750 skipped=64`
- `job_sections` строк: 2750; всего секций: 7596; среднее: 2.76 на вакансию.
- Распределение классов: other 6160, Requirements 381, Role 267, Responsibilities 242, Benefits 163, Nice to have 140, Compensation 86, About 67, Location 46, Process 26, Legal/EEO 18.
- Байт-идентичность `description`: `sum(length(description))=13877623, count=2814` — совпадает с живой базой (v22). Расхождений нет.

**Итог тестов:**
```
go test ./... (сокращённо)
ok  github.com/w3con/sifty/internal/background  4.697s
ok  github.com/w3con/sifty/internal/collect      2.011s
ok  github.com/w3con/sifty/internal/config       1.748s
FAIL github.com/w3con/sifty/internal/server     (1 pre-existing: TestRouteDrift)
FAIL github.com/w3con/sifty/internal/store      (3 pre-existing: TestMigrationV14, TestMigrationV16ToV17PreservesData, TestSchema_FreshDatabase)
```
Новых провалов нет. Порт 9199 остановлен.

---

## 2026-11-12 — L.3 коррекция: `<br>` внутри `<strong>`/`<em>` в `renderPlain`

**Дефект:** Web3 Careers строки типа `<p><strong>About CoinList<br/><br/></strong>CoinList is building…</p>` давали в plain-тексте `About CoinListCoinList is building…` — заголовок приклеивался к следующему тексту. Причина: `renderPlain` для `strong`/`em` вызывал `n.Text()`, который сжимает всех потомков в одну строку и теряет позиции `<br>`.

**Исправление** (`backend/internal/collect/converter.go`, `renderPlain`): для `strong`/`em` вместо `sb.WriteString(strings.TrimSpace(n.Text()))` теперь вызывается `renderPlain(n, sb)` — рекурсия сохраняет `<br>` в правильной позиции.

**Тест** (`backend/internal/collect/converter_test.go`, `TestPlainFromStructuredBrInsideInline`): входные данные — `<p><strong>About CoinList<br><br></strong>CoinList is building…</p>`; проверяет наличие `"About CoinList\n"`, наличие `"CoinList is building"`, отсутствие `"About CoinListCoinList"`. Тест проходит после исправления.

**Верификация на реальных данных:**
- Строка id=27 (l36-data, job_originals, 364 КБ, `html/page`): `description_html` содержит `<strong>About CoinList<br/><br/></strong>CoinList is building…`; `PlainFromStructured` на этом фрагменте даёт `About CoinList\n\nCoinList is building…` (разрыв строки есть, слипания нет).
- Throwaway `verifier_br` (собран, запущен, удалён): `OK: no gluing detected`.

**Анализ staging Web3 Careers (l36-data, 132 строки с description_html):**
- Строк с `<strong>Heading<br/><br/></strong>Text` (паттерн, который закрывает текущее исправление): 43 вхождения в 9+ строках.
- Строк с `<strong>Heading</strong>Text` без `<br>` (L.5-дефект): 0 в staging-копии; в l36-data 2 вхождения в 2 строках (оба — inline-жирный внутри предложения, не заголовок + тело).

**Итог тестов:**
```
go test ./... 2>&1 (сокращённо)
ok  github.com/w3con/sifty/internal/collect      0.533s
ok  github.com/w3con/sifty/internal/source       2.662s
FAIL github.com/w3con/sifty/internal/store       (3 pre-existing: TestMigrationV14, TestMigrationV16ToV17PreservesData, TestSchema_FreshDatabase)
FAIL github.com/w3con/sifty/internal/server      (1 pre-existing: TestRouteDrift)
```
Все новые тесты проходят. Ранее существовавшие провалы не изменились.

---

## 2026-11-12 — L.3.5 — живая коллекция Remote3 и WWR в throwaway DB

Команда: `sifty serve --data-dir <scratchpad>/inbox2-l35-data/data --port 9199`, коллекция запущена через `POST /api/search {"force": true}`.

Результат:
```
collect: upserted 108 postings, skipped 0 by stop-word
```

| Источник | Всего | description_html | needs_refetch | original_kind |
|---|---|---|---|---|
| Remote3 | 8 | 8 | 0 | html/page |
| WWR | 100 | 100 | 0 | rss/item |

Все 108 вакансий имеют `description_html`; ни одна не требует рефетча. Оригиналы: 8 — `html/page`, 100 — `rss/item`.

Качество описаний:
- Remote3: 8/8 имеют `available structured description`; 5/8 содержат `<li>`, 3/8 — `<h*>`. В 4 из 8 описаниях есть артефакт "Back to jobs" из breadcrumb-навигации страницы — ожидаемое поведение (не область этой карточки).
- WWR: 100/100 — `feed description; may be excerpt`; 94/100 содержат `<li>`, 43/100 — `<h*>`. Ни одна не содержит raw-тегов в plain-тексте.

---

## 2026-11-12 — L.3.6 — бэкфилл и рефетч на копии data-staging

**Копия:** `data-staging/store.sqlite` → scratchpad (live-файл не тронут).

**До бэкфилла:**
- Схема: v18 → после открытия через store.Open: v25 (миграции v24, v25 применены автоматически)
- 2783 вакансии, `description_html IS NULL` у всех 2783

**Результат бэкфилла (`BackfillDescriptionHTML`):**
- Конвертировано (description_html заполнен): 2719
- needs_refetch=1 (склеенный текст, >= 500 байт, нет переносов): 220
- Пропущено (description = ''): 64

**После бэкфилла:**
- description_html missing: 64 (все — пустые описания, не могут быть заполнены)
- needs_refetch=1: 220

**Рефетч Web3 Careers (61 вакансия, needs_refetch=1 у этих строк):**
- 1 вакансия проверена вручную, 60 — скриптом (3 сек между запросами)
- Успешно: 61/61 (HTTP 200 от сервера)
- Статус после рефетча: `available body html` у всех 61
- needs_refetch=1 у Web3 Careers: 0

**Рефетч Remote3:**
- Не потребовался: backfill успешно структурировал 40 "newlines"-строк из inventory (в отличие от прогноза 2026-11-11, сделанного до L.2b, когда Remote3 использовал HTML-скрапинг вместо RSS).

**Итоговое состояние staging-копии:**

| Источник | Всего | html_set | needs_refetch |
|---|---|---|---|
| Other (ATS и прочие) | 2145 | 2082 | 158 |
| WWR | 238 | 237 | 1 |
| Remotive | 227 | 227 | 0 |
| Web3 Careers | 132 | 132 | 0 |
| Remote3 | 41 | 41 | 0 |
| **Итого** | **2783** | **2719** | **159** |

Оставшиеся needs_refetch=159 — склеенные строки из «Other» (158) и WWR (1), не входящие в область L.3.

**Целостность description:** Описания нерефетченных Web3 Careers строк (kind:metadata, 52 строки) сравнены с оригинальным data-staging — байт-идентичны. Пример (id=97–101): hex(description) совпадает.

**job_originals:** 61 строка (только что рефетченные Web3 Careers вакансии).

Ports used: 9199 (stopped before this entry). Live bases not touched.

---

## 2026-11-12 — L.2c + L.3.1–L.3.4 — оригинал, бэкфилл, рефетч-путь, тесты

**L.2c.1 — миграции v24 и v25 (`store/migrate.go`):**
- v24: таблица `job_originals(job_id INTEGER PRIMARY KEY REFERENCES jobs(id) ON DELETE CASCADE, kind TEXT NOT NULL, fetched_at TEXT NOT NULL, body BLOB NOT NULL)`.
- v25: колонка `needs_refetch INTEGER NOT NULL DEFAULT 0` на таблице `jobs` (через `addColumnIfMissing`).
- `targetVersion` поднят с 23 до 25.

**L.2c.1 — хранилище оригинала (`store/originals.go`):**
- `saveOriginalInTx(tx, jobID, kind, fetchedAt, body)`: gzip-сжимает тело, пишет через `INSERT OR REPLACE`.
- `(s *Store) GetOriginal(jobID)`: читает и распаковывает.

**L.2c.2 — адаптеры и модель (`model/job.go`, адаптеры источников):**
- `model.JobInput` получил поля `Original []byte` и `OriginalKind string` с тегом `json:"-"`.
- Все адаптеры проставляют оба поля: Greenhouse, Lever, Ashby — `html/api`; Generic, Remote3, WWR, Web3 Careers — `html/page`; ParseFeed — `rss/item`.
- `store.Upsert` вызывает `saveOriginalInTx` в той же транзакции, если `len(j.Original) > 0`.

**L.2c.3 — тесты (`store/originals_test.go`):**
- `TestOriginalWrittenWithVacancyInOneTransaction`: upsert → GetOriginal, проверяет kind/fetchedAt/body.
- `TestRepeatedCollectionNoSecondRow`: два апсёрта одного URL с разными body → второй побеждает, строка одна.
- `TestAPIResponseDoesNotContainOriginal`: marshal `JobInput` и `Job` → нет "secret page body", нет `"html/page"` как JSON-значения, нет `"originalKind"`.

**L.3.1 — бэкфилл (`store/backfill.go`):**
- `BackfillDescriptionHTML()`: запрос `description_html IS NULL AND description != ''`; для каждой строки вызывает `collect.StructuredHTML`; для «склеенных» строк (без `\n`, len >= 500 байт) ставит `needs_refetch=1`; `description` не трогает.
- Нет сетевого клиента — компилятор гарантирует отсутствие сетевых вызовов.

**L.3.2 — рефетч-путь (`store/jobs.go`):**
- `store.StoreRefetchResult(id, j)`: обновляет `description`, `description_status`, `description_html`, сбрасывает `needs_refetch=0`, опционально `published_at`/`published_basis`, сохраняет оригинал, обновляет FTS. Единственный путь, который может заменить описание legacy-строки.
- `server/enrich.go` рефакторинг `EnrichJob`: убран прямой вызов `client.GetDetailWithDiscipline`; используется `buildServerRegistry(client).For(hostOf(job.OriginalURL))`; вызывается `src.Resolve(ctx, ref)`, результат пишется через `s.StoreRefetchResult`.

**L.3.3 — Web3 Careers «Apply Now» и пустые параграфы:**
- `web3CareerExtractDescription` Layout 2: последний дочерний элемент контейнера удаляется, если его текст — `"apply now:"` или `"apply now"`.
- `collect/converter.go` `wrapTopLevelInlines`: блочные элементы с пустым текстом пропускаются (пустой `<p>` не попадает в вывод).

**L.3.4 — тесты (`store/backfill_test.go`):**
- `TestBackfillPreservesDescriptionHashes`: SHA-256 `description` до и после бэкфилла совпадают.
- `TestBackfillNoNetworkRequired`: вызов `BackfillDescriptionHTML` без сетевого клиента компилируется — гарантия отсутствия сети.
- `TestRefetchBypassesLegacyGuard`: Upsert не меняет описание legacy-строки; `StoreRefetchResult` меняет.
- `TestSanitizerRemovesScriptsAndAttributes`: документирует, что санитизация происходит в слое collect (ссылка на `TestStructuredHTMLSanitiser`).

**Обновлены версии в тестах:** `match_facts_test.go`, `match_model_test.go`, `match_results_test.go` — проверка версии 23 → 25.

**collect/collect_test.go:** `assertParity` переведена с `reflect.DeepEqual` на JSON-сравнение (поля `json:"-"` исключаются).

**Итог тестов:**
```
go test -count=1 ./internal/collect/... ./internal/source/... ./internal/background/... ./internal/fetch/... ./internal/store/... ./internal/server/...
ok  github.com/w3con/sifty/internal/collect     0.514s
ok  github.com/w3con/sifty/internal/source      0.262s
ok  github.com/w3con/sifty/internal/background  4.558s
ok  github.com/w3con/sifty/internal/fetch       0.614s
FAIL github.com/w3con/sifty/internal/store      (3 pre-existing: TestMigrationV14, TestMigrationV16ToV17PreservesData, TestSchema_FreshDatabase)
FAIL github.com/w3con/sifty/internal/server     (1 pre-existing: TestRouteDrift)
```
Все новые тесты проходят. Ранее существовавшие провалы не изменились.

---

## 2026-11-11 — L.2b исправление (раунд 2) — Web3 Careers нативный макет, Remote3 RSS, WWR реальная фикстура, конвертер

**Проблема (найдена координатором после первого раунда L.2b):** у 15 из 16 собранных Web3 Careers вакансий в `description` присутствовал текст "Join talent pool" из шапки страницы. Причина — адаптер не мог выбрать блок описания через CSS-селектор `.trix-content`: HTML страницы содержит `style="color: #b3b3b3"class=trix-content` без пробела между атрибутами, и HTML-парсер принимает `class=trix-content` как дубль атрибута `class` и игнорирует его; элемент получает только первый класс (`text-right-...`). Статически заданный заголовок "Join talent pool" находится в хроме страницы, а не в контейнере `.trix-content`; фолбэк на полное тело страницы захватывал его.

**Исправление Web3 Careers нативного макета** (`source/web3careers.go`, `web3CareerExtractDescription`): Layout 2 теперь использует `div.text-dark-grey-text` — это внешний контейнер описания, который HTML-парсер правильно разбирает (класс задан стандартно, без дублирования). Bondex-виджет ("Join talent pool") удаляется через `a[href*="bondex.app"] → parent.Remove()` перед извлечением. Layout 1 (Greenhouse `[data-qa=...]`) по-прежнему отрабатывает первым, `div.text-dark-grey-text` используется только для нативного макета.

**Новые тесты** (`source/web3careers_test.go`):
- `TestWeb3CareersResolveNativeLayout` — фикстура `testdata/web3career_job_native.html` (реальная страница, 80 КБ, MEYA HOLDINGS LTD). Проверяет: нет "Join talent pool", есть `<h3>` и `<li>`, нет сырых тегов в plain-описании, starts with tag.
- `TestWeb3CareersResolveExtractsContentOnly` — переведён на реальную Greenhouse-страницу (`testdata/web3career_job.html`, 67 КБ, Anchorage Digital). Проверяет: нет "Join talent pool", нет "Apply" в chrome-позиции, есть `<li>` и `<strong>`, апострофы целы.

**Remote3 — переработан на RSS** (`source/remote3.go`, тесты): `Discover` использует `GET /api/rss` вместо HTML-скрапинга (Next.js SSR не содержит ссылок на вакансии). `Resolve` выбирает `[class*="RemoteJobs_jobDescription"]` на странице вакансии. Фикстуры заменены реальными: `testdata/remote3_listing.html` ← реальный RSS (4629 байт, 8 элементов); `testdata/remote3_job.html` ← реальная страница (110 КБ, Robinhood Wallet).

**WWR — реальная RSS-фикстура** (`source/testdata/wwr_rss.xml`): заменена на двухэлементную выборку из реального `weworkremotely.com/remote-jobs.rss` (Smart Suites, Vector Outsourcing). `testdata/wwr_job_body.html` — рукописная (Cloudflare-заблокированные страницы недоступны; тестирует только body-фолбэк путь). Тест `TestWWRDiscoverRSSEntityEscaped` проверяет `<strong>` и `<li>` вместо `<h3>`.

**Конвертер — `wrapTopLevelInlines`** (`collect/converter.go`): после `renderSanitised` отдельный проход группирует top-level текстовые узлы и инлайн-элементы (`strong`, `em`, `br`) в `<p>`. Блочные элементы (`h*`, `p`, `ul`, `ol`) проходят насквозь. Тест `TestStructuredHTMLWrapsTopLevelText`. Золотой файл `testdata/feed.json` обновлён.

**Итог тестов:**
```
go test ./internal/collect/... ./internal/source/... ./internal/background/... ./internal/fetch/... -count=1
ok  github.com/w3con/sifty/internal/collect     0.316s
ok  github.com/w3con/sifty/internal/source      0.411s
ok  github.com/w3con/sifty/internal/background  4.343s
ok  github.com/w3con/sifty/internal/fetch       0.295s
```

**Живая проверка Web3 Careers (throwaway dir inbox2-l2b2-data, порт 9299, search.yaml: только Web3 Careers):**
```
2026/11/11 22:37:44 collect: upserted 16 postings, skipped 0 by stop-word
```

```sql
SELECT count(*) FROM jobs;  -- 16

SELECT sum(CASE WHEN description LIKE '%Join talent pool%' THEN 1 ELSE 0 END)
FROM jobs;  -- 0 (было 15 в предыдущем прогоне)

SELECT description_status, count(*) FROM jobs GROUP BY description_status;
-- available body html | 16

-- Нет raw-тегов в plain description:
SELECT id, CASE WHEN description LIKE '%<h3>%' OR description LIKE '%<ul>%'
                     OR description LIKE '%<li>%' OR description LIKE '%<p>%'
           THEN 'RAW' ELSE 'ok' END FROM jobs;
-- все: ok

-- Нет double-encoded entities:
SELECT id, CASE WHEN description LIKE '%&amp;amp;%' THEN 'DOUBLE' ELSE 'ok' END FROM jobs;
-- все: ok

-- "Apply" присутствует в 15 из 16 вакансий — ожидаемо:
-- у нативного макета (job 1, MEYA HOLDINGS) контейнер div.text-dark-grey-text заканчивается
-- <p>Apply Now:</p> — незначительный хром, не относится к Bondex-виджету;
-- в остальных Greenhouse-страницах слово "Apply" встречается в тексте самой вакансии.
SELECT sum(CASE WHEN description LIKE '%Apply%' THEN 1 ELSE 0 END) FROM jobs;  -- 15
```

---

## 2026-11-11 — L.2b — Remote3, WWR, Cloudflare-403, Web3 Careers

**Remote3** (`source/remote3.go`): адаптер регистрируется по `Remote3Pattern = "remote3\.co$"`. `Discover` фильтрует ссылки: пропускает `/jobs/<category>` (category pages), оставляет только `/remote-jobs/<slug>` (vacancy pages). `Resolve` читает тело страницы через `collect.StructuredHTML + plainOrFallback`; JSON-LD на этом источнике нет. Тесты: `TestRemote3RegistryPattern`, `TestRemote3DiscoverFiltersJobPaths` (fixture `testdata/remote3_listing.html`), `TestRemote3ResolveUsesBody` (fixture `testdata/remote3_job.html`).

**We Work Remotely** (`source/wwr.go`): адаптер регистрируется по `WWRPattern = "weworkremotely\.com$"`. `Discover` строит URL RSS-ленты (`/remote-jobs.rss`), разбирает через `collect.ParseFeed`; entity-escaped HTML из RSS (`&lt;h3&gt;…`) разворачивается нативно. Если запись в RSS уже полная — возвращается без запроса detail-страницы. `Resolve` при 403 записывает blocked-статус без retry; для доступных страниц пробует CSS-селекторы `section.listing-container` и аналоги, затем `StructuredHTML`. Экспортируется `wwrExtractContentFromBody` и `resolveWWRBody` для тестов. Тесты: `TestWWRRegistryPattern`, `TestWWRDiscoverRSSEntityEscaped`, `TestWWRPageBodyChromeStripping`, `TestWWRResolveBodyConversion`.

**Cloudflare 403** (`fetch/traffic_test.go`): `TestCloudflare403NoHTTPRetry` запускает counting-сервер, возвращающий 403 с телом "Just a moment..." (Cloudflare), проверяет: первый вызов — `BlockedResult{Reason:"403"}`, счётчик сервера = 1; второй вызов — cooldown error, счётчик по-прежнему 1. Retry не происходит.

**Web3 Careers** — исправление Discover (`source/web3careers.go`): `StructuredJobs` на домашней странице `https://web3.career` возвращает вакансии с URL домашней страницы (JSON-LD не содержит поле url). Новый метод: `web3CareerJobRefs` находит через goquery все `<a href>`, чей путь совпадает с `web3CareerJobPathRe = ^/[a-z0-9][a-z0-9-]*/\d{5,}$`, дедуплицирует по абсолютному URL. Описания на этапе Discovery не заполняются; Resolve читает тело detail-страницы. Тест переименован в `TestWeb3CareersDiscoverExtractsAnchorLinks` (fixture `testdata/web3career_listing.html`): 3 уникальных ref из 4 ссылок, категорийные пути и короткие числа исключены.

**Регистрация адаптеров**: Remote3 и WWR добавлены в `background/collect.go` (`buildRegistry`) и `server/enrich.go` (`buildServerRegistry`).

**Итог тестов:**
```
go test ./internal/collect/... ./internal/source/... ./internal/background/... ./internal/fetch/... -count=1
ok  github.com/w3con/sifty/internal/collect     0.323s
ok  github.com/w3con/sifty/internal/source      0.483s
ok  github.com/w3con/sifty/internal/background  4.523s
ok  github.com/w3con/sifty/internal/fetch       0.475s
```

**Живая проверка web3.career** (throwaway data-w3c-l2b, порт 9199, одна итерация, search.yaml: только Web3 Careers enabled):
```
2026/11/11 22:01:16 collect: upserted 16 postings, skipped 0 by stop-word
```

```sql
SELECT id, title, description_status, length(description_html), length(description)
FROM jobs ORDER BY id;
-- 16 строк, description_status = "available body html" у всех
-- description_html: 2807–11444 байт; description: 2748–10914 байт

-- Нет raw-тегов в plain description:
SELECT id, CASE WHEN description LIKE '%<h3>%' OR description LIKE '%<ul>%'
                THEN 'RAW' ELSE 'ok' END FROM jobs;
-- все: ok

-- Нет double-encoded entities:
SELECT id, CASE WHEN description LIKE '%&amp;amp;%' THEN 'DOUBLE' ELSE 'ok' END FROM jobs;
-- все: ok

-- Апострофы целы (job 3, текст содержит их):
SELECT substr(description, 400, 400) FROM jobs WHERE id=3;
-- ... CleanSpark (Nasdaq: CLSK), is a market-leading data center developer...
```

description_html содержит `<h*>`, `<li>`, `<p>` у 14 из 16 строк (2 — только `<p>`, т.к. те вакансии написаны сплошным текстом). Chrome-текст листинга ("Apply Join talent pool 16h ago") присутствует в начале description до первого `<p>` — это ожидаемое поведение `StructuredHTML`, не артефакт L.2b.

---

## 2026-11-11 — L.2 исправление — шаблон Web3 Careers, тесты, энрич-путь

Исправлено: (1) шаблон реестра `web3\.career$` не совпадает с хостом `web3.careers` из search.yaml — исправлен на `web3\.careers?$` (константа `Web3CareerPattern` в source/web3careers.go); (2) адаптер не был зарегистрирован в `server/enrich.go`; (3) тестов на адаптер не было.

Добавлены тесты (`source/web3careers_test.go`):
- `TestWeb3CareerRegistryPattern` — реестр отдаёт `*Web3CareersSource` для `web3.career` и `web3.careers`, fallback для `jobs.lever.co`
- `TestWeb3CareersResolveUsesBody` — Resolve читает тело страницы, а не JSON-LD; апострофы и `&` не портятся; `description_html` содержит `<h*>` и `<li>`, plain-текст не содержит сырых тегов

```
go test ./internal/source/... -run TestWeb3Career -v
=== RUN   TestWeb3CareerRegistryPattern
--- PASS: TestWeb3CareerRegistryPattern (0.00s)
=== RUN   TestWeb3CareersResolveUsesBody
--- PASS: TestWeb3CareersResolveUsesBody (0.00s)
PASS
ok  github.com/w3con/sifty/internal/source  0.37s
```

Живая проверка Web3 Careers (throwaway dir data-w3c, порт 9199):
```
2026/11/11 18:22:51 collect: discover "Web3 Careers" (https://web3.careers): web3careers discover "https://web3.careers": fetch: GET "https://web3.careers": Get "https://web3.careers": dial tcp 192.64.119.131:443: i/o timeout
```

Хост web3.careers недоступен с этой машины в момент проверки — i/o timeout. Запись в журнале показывает `web3careers discover`, а не `generic discover`: правильный адаптер срабатывает. SQL-проверку для этого источника провести невозможно до восстановления сетевого доступа.

---

## 2026-11-11 — L.2 — Structured form at collection

Check: throwaway DB at `/private/tmp/…/inbox2-l2-data/data/store.sqlite`; collection ran against Remotive only (Web3 Careers timed out — host unreachable from this machine; at time of original check there were no unit tests on Web3CareersSource — this was a false statement in the original entry; tests and pattern fix added in the correction above).

```
sqlite3 store.sqlite "PRAGMA user_version;"
23

sqlite3 store.sqlite "SELECT count(*) as total, count(description_html) as html_set, count(CASE WHEN description_html IS NULL OR description_html='' THEN 1 END) as html_missing FROM jobs;"
46|46|0
```

Sample description_html (Remotive RSS, actual job listing):
```
<p>Nov 04, 2026 - IAPWE is hiring a remote Freelance Writer. 💸 Salary: $50-$75 /hour. 📍Location: Worldwide.</p><p>Our organization is seeking content writers…</p><ul><li>Health &amp; beauty</li><li>Fitness</li>…</ul><p>Requirements:</p><ul><li>We ask that all work be completed using a word processor…</li></ul>
```

Schema v23 migrated correctly; all 46 new rows have both `description` (plain) and `description_html` (structured); no row is missing either. Store tests that expected v22 updated to v23 (match_facts_test.go, match_model_test.go, match_results_test.go). Pre-existing failures (TestMigrationV14, TestMigrationV16ToV17PreservesData, TestSchema_FreshDatabase) untouched.
Exit: 0

## 2026-11-11 — L.1 — Inventory per source

Check: `cd /Volumes/SSD/Dev/tknff/sifty/.worktrees/INBOX-2 && test -s work/tasks/inbox/INBOX-2_listing-markup/inventory.md && echo 'inventory.md exists (table in card: 'work/tasks/inbox/INBOX-2_listing-markup'/inventory.md)' && python3 -I work/tasks/inbox/INBOX-2_listing-markup/inventory_counts.py | diff - work/tasks/inbox/INBOX-2_listing-markup/inventory_counts_output.txt && echo 'counts reproduce byte-for-byte' && python3 -I work/tasks/inbox/INBOX-2_listing-markup/inventory_counts.py | sed -n '1,28p'`
Exit: 0

```
inventory.md exists (table in card: work/tasks/inbox/INBOX-2_listing-markup/inventory.md)
counts reproduce byte-for-byte
== 1. Totals (whole table, 2783 vacancies)
&amp;                                126
&lt; or tag-name&gt; (escaped)        73
<p>/<li>/<br> tags                   161
any entity                           168
any real tag                         161
contains newline                    2107
kind:empty                            64
kind:escaped                          13
kind:glued                            57
kind:html                            161
kind:metadata                        381
kind:newlines                       2107

== 2. Kind per source (primary source = earliest sighting); columns are counts
source                         total      html   escaped  newlines     glued  metadata     empty
Paradigm                         836        80         0       697         0         1        58
Wellfound                        400         0         0       399         1         0         0
We Work Remotely                 265        27         0       237         0         0         1
Remotive                         226         0         0       226         0         0         0
Multicoin capital                192        20         0        47         1       122         2
A16z                             172         0         0        37         9       124         2
Hacker News — Who is hiring?     167         0         0       167         0         0         0
Electric Capital                 141         5         0        79         0        57         0
Web3 Careers                     132         2        13         6        46        65         0
Framework ventures                89        27         0        54         0         7         1
Cryptocurrency Jobs               65         0         0        61         0         4         0
Remote3                           41         0         0        40         0         1         0
```

## 2026-11-12 — note

Коррекции после L.3 (раунд приёмки, 2026-11-12):

1. serve.go: добавлен вызов BackfillDescriptionHTML после store.Open (строка 85) до запуска сервера; логирует converted/needs_refetch/skipped, только если converted>0 || needs_refetch>0.
2. TestStartupPathBackfillsLegacyRows: новый тест в store/backfill_test.go — raw-INSERT строк (обход Upsert), затем BackfillDescriptionHTML, проверка description_html заполнен и description байт-идентичен; glued-строка получает needs_refetch=1.
3. model.Job.NeedsRefetch bool json:needsRefetch,omitempty — добавлено поле; store/jobs.go: get() выбирает needs_refetch из БД, сканирует, проставляет на Job.
4. frontend/model/core.ts Job type: needsRefetch?: boolean добавлен.
5. frontend/RoleContent.tsx: useEffect с ref-ключом job.id — срабатывает однократно при открытии вакансии с needsRefetch=true; вызывает refreshSource(), нет retry.
6. remote3ExtractDescription: цикл удаляет ведущие chrome-элементы (пустой текст = logo-ссылки, 'back to jobs') перед collect.StructuredHTML; test TestRemote3ResolveExtractsContentOnly дополнен проверкой отсутствия 'Back to jobs'.
7. Живая коллекция Remote3 (throwaway DB, порт 9199, остановлен): 8 вакансий, ни в одной 'Back to jobs' нет.
8. Проверка needsRefetch через API: сервер на 9199 против копии L.3.6, job id=8 → needsRefetch:true; job id=7 (не помечен) → поле отсутствует (omitempty). Сервер остановлен.

Тесты после коррекций:
- go test ./internal/collect ./internal/source ./internal/background ./internal/fetch ./internal/store ./internal/server: только 4 ранее известных провала (TestMigrationV14, TestMigrationV16ToV17PreservesData, TestSchema_FreshDatabase, TestRouteDrift).

Уточнение по 40 Remote3-строкам: это страницы категорий (/jobs/pr-jobs и т.д.), а не вакансии; company=Unknown; см. report-L3.md.

## 2026-11-12 — note

Коррекция needs_refetch — исправление эвристики бэкфилла (2026-11-12):

Дефект: все 159 строк с needs_refetch=1 в staging-копии содержали реальные HTML-теги в description (LinkedIn ATS rows, 81 из 159). Старая эвристика '>=500 байт + нет newlines' помечала их независимо от наличия тегов. Эффект: открытие вакансии запускало рефетч на защищённые страницы; при успехе hash description менялся и триггерилась переработка моделью.

Исправление в BackfillDescriptionHTML (store/backfill.go):
- Новый предикат: нет literal HTML тега (< + буква или /) И нет newlines И len >= 300 байт (пороговое значение из inventory карточки).
- Добавлен вспомогательный descHasHTMLTag(s) — O(n) скан без regexp.
- Строки с escaped markup (strong&gt;) не содержат literal тега → остаются помеченными (это Web3 Careers с повреждёнными описаниями).
- Строки с реальными тегами (LinkedIn ATS <p><strong>...) → description_html заполнен, needs_refetch=0.

Расширен TestStartupPathBackfillsLegacyRows:
- id=1004: повторённый <p>ATS structured content...</p> (>600 байт, нет newline) → description_html set, needs_refetch=0.
- Все description хэши не изменились.

Бэкфилл на свежей копии data-staging/store.sqlite (scratchpad, не live):
- converted=2731, needs_refetch=70, skipped=0
- html_null=65 (пустые descriptions, не затронуты)
- Помеченные строки по доменам: web3.career:59, coinbase.com:4, job-boards.greenhouse.io:3, jobs.lever.co:1, jobs.ashbyhq.com:1, jobs.multicoin.capital:1, wellfound.com:1
- Первые 120 байт описаний 5 помеченных строк (все — escaped HTML или plain glued text без literal тегов):
  id=27 (web3.career): 'strong&gt;About CoinList br&gt; br&gt; /strong&gt;...'
  id=28 (web3.career): 'The Role You ll be the second person in BCB...'
  id=29 (web3.career): 'About Zscaler Serving thousands of enterprise customers...'
  id=30 (web3.career): 'WHO WE ARE: Zinnia is the leading technology platform...'
  id=31 (web3.career): 'Who We Are At OKX, we believe that the future will be reshaped...'

Live bases not touched. No refetch run.

## 2026-11-12 — L.4.1

Frontend: добавлен descriptionHtml?: string в тип Job (frontend/model/core.ts). Создан frontend/listingHtml.ts — чистый модуль (без React): parseListingHtml() использует browser DOMParser (inert parse), белый список h2-h4/p/ul/ol/li/strong/em/br; script/style удаляются с контентом; остальные теги разворачиваются в дочерние; атрибуты не передаются. В RoleContent.tsx: ListingText получила descriptionHtml?: string; при наличии delegирует в HtmlListingText; fallback на прежний plain-text-рендерер когда descriptionHtml отсутствует. НИКОГДА не используется dangerouslySetInnerHTML или innerHTML на живом элементе. В theme.css добавлены: .listing-text h2/h3/h4 (font-size, font-weight, margin, color, text-transform:none, letter-spacing:normal), .listing-text ol (list-style:decimal, same margin/padding как ul), .listing-text strong (font-weight:600), .listing-text em (font-style:italic). Добавлен :is(h2,h3,h4):first-child{margin-top:0}. Дефект найден и исправлен: глобальный h3{text-transform:uppercase} из style.css делал h3 заглавными — добавлен text-transform:none к .listing-text h3.

Check: `cd frontend && npx --no-install tsc --noEmit && echo 'tsc: OK'; node --test dates.test.mjs quickmatch.test.mjs 2>&1 | tail -5`
Exit: 0

```
tsc: OK
# fail 0
# cancelled 0
# skipped 0
# todo 0
# duration_ms 83.316834
```

## 2026-11-12 — L.4.2

Playwright-скрипт: screenshots/shoot-L4.mjs. Импорт playwright 1.65.0 из npx-кеша (~/.npm/_npx/e41f203b7505f1fb/node_modules/playwright/index.mjs). Сервер запущен на порту 9199 против копии staging (никогда не live). Данные подготовлены в DB-копии: id=1 description_html=NULL (fallback), id=2 description_html=hostile string (security). Результаты 7 кейсов: [1] Parity id=84 — .listing-text present; [2] LinkedIn id=8 — ul count=3, p count=6; [3] Numbered list id=13 — ol count=1; [4] Web3 Careers id=27 needs_refetch=true→undefined (cleared), 0 браузерных запросов к web3.career (рефетч делает сервер); [5] Fallback id=1 — .listing-text: 1, .listing-missing: 0, raw tags in textContent: false; [6] Hostile — все 5 ассертов PASS (window.__pwn undefined, no script, no img, no style=, no on* attr, Title рендерится как h3). Первый прогон: needs_refetch до=true, после=undefined. Второй прогон (после CSS fix): needs_refetch=undefined уже до (рефетч из первого прогона).

Check: `node work/tasks/inbox/INBOX-2_listing-markup/screenshots/shoot-L4.mjs`
Exit: 0

```

[1] Parity (id=84) desktop
  saved L4-parity.png
  .listing-text present: true

[2] LinkedIn with HTML (id=8)
  ul count: 3, p count: 6
  saved L4-linkedin-html.png

[3] Numbered list (id=13)
  .listing-text ol count: 1
  saved L4-numbered-list.png

[4] Web3 Careers needs_refetch (id=27)
  needsRefetch before open: undefined
  needsRefetch after open+wait: undefined
  requests to web3.career: 0 []
  saved L4-web3careers-refetch.png

[5] Fallback (id=1, description_html=NULL)
  .listing-text: 1, .listing-missing: 0
  raw tags in textContent: false
  saved L4-fallback.png

[6] Hostile case (id=2)
  PASS: window.__pwn is undefined
  innerHTML checks: {"hasScript":false,"hasImg":false,"hasStyleAttr":false,"hasOnAttr":false,"h3texts":["Title"],"ptexts":["x","text"]}
  PASS: no <script>
  PASS: no <img>
  PASS: no style=
  PASS: no on* attr
  PASS: "Title" renders as h3
  saved L4-hostile.png

[7] Phone-width Parity (id=84, 390×844)
  saved L4-phone.png

All cases done.
```

## 2026-11-12 — L.4.3

Просмотр скриншотов (Read tool):

L4-parity.png: Parity Tech Lead (id=84, desktop 1440x900). Listing text status «detail unavailable: Error: HTTP 429» — Parity использует Cloudflare, detail-страница заблокирована. description_html = '<p>lead tech lead blockchain +3</p>' (только ключевые слова из metadata). Первая видимая строка: «lead tech lead blockchain +3». HTML-рендерер работает корректно — контент из descriptionHtml без raw-тегов. Это metadata-строка с минимальным контентом; полного описания нет — не дефект L.4.

L4-linkedin-html.png: Amber Group Full-Stack Software Engineer (id=8, desktop). Текст начинается с параграфов «AMBR is building...». Раздел «Key Responsibilities» рендерится как <p><strong>Key Responsibilities</strong></p> (LinkedIn использует <strong> вместо h2/h3). Ниже bullet list (<ul><li>) с отступом и disc-маркерами. Всё чисто, тегов не видно. Последняя видимая строка: «Maintain production reliability through troubleshooting, backups, and recovery procedures».

L4-numbered-list.png: Discernis Senior Software Engineer (id=13, desktop). Headings рендерятся корректно: «About Discernis», «How we work», «The role», «What you'll do» — видны как полужирные заголовки чуть крупнее тела. Параграфы с 15px 1.65 line-height. Bullet list (<ul><li>) внизу с disc-маркерами. Нумерованный список (<ol>) подтверждён программно (count=1) но ниже видимой части скриншота. Первая строка: «About Discernis» (h2). Последняя видимая: «Help interview and mentor as the team grows».

L4-web3careers-refetch.png: CoinList Senior Smart Contract Engineer (id=27, Archive/Applied view). Успешный рефетч при первом открытии (needs_refetch 1→0). Скриншот показывает рефетченный контент: «About CoinList» и «About This Role» как полужирные заголовки (<p><strong>...</strong>), bullet list с точками. «build smart contracts» рендерится полужирным (<strong>). Текст чистый, тегов не видно. Первая строка: «About CoinList». Последняя видимая: «company direction, and revenue».

L4-fallback.png: Lemon.io Senior Back-end Developer (id=1, description_html=NULL, fallback). Fallback plain-text рендерер работает. Параграфы с текстом вакансии. Строки-разделители («Requirements for Senior Back-end Developers with Golang») рендерятся как обычные параграфы (lead-in у этой строки нет — не оканчивается на ':'). Нет HTML-тегов в textContent. Первая видимая строка: «Oct 27, 2026 - Lemon.io is hiring a remote Senior Back-end Developer.». Последняя видимая: «You own problems end-to-end: you spot the issue, design and build the solution, and ship it to production».

L4-hostile.png: Lemon.io Head of Engineering (id=2, hostile descriptionHtml). Rendered content: «x» (p), «Title» (h3, смешанный регистр после CSS-фикса text-transform:none), «text» (p). Нет скриптов, нет img, нет onclick/onerror, нет style=color:red. «Title» отображается как heading с правильным font-weight и цветом (#18181b). «text» — параграф. Статус «available structured description» — сервер возвращает description_html как есть (hostile строку), но frontend санитизирует при рендере.

L4-phone.png: Parity Tech Lead (id=84, 390x844). Layout адаптируется — detail panel на полную ширину. «lead tech lead blockchain +3» читается корректно. «By section» / «Original» кнопки видны. Статус «Error: HTTP 429» в subtitle panel.

Дефект обнаружен и исправлен в рамках L.4: global h3{text-transform:uppercase} из style.css — добавлен text-transform:none;letter-spacing:normal к .listing-text h3 (аналогично .detail .rail-card h3). Дефект вне L.4 (не исправлялся): Remote3 Back to jobs артефакт в 4 из 8 вакансий (известен из L.3.5).

## 2026-11-12 — L.4b — sticky "Failed to fetch" banner

**Дефект:** Ошибка фонового поллинга (api('queue') fails) не очищалась автоматически. "Failed to fetch" от браузера показывалась бесконечно даже после восстановления сервера.

**Исправление:**
- `frontend/messages.ts`: добавлена `messageText(raw: string): string` — маппинг "Failed to fetch" (case-insensitive) → "Sifty server is not responding. Retrying…", прочие ошибки возвращаются без изменений.
- `frontend/main.tsx`: добавлен `pollError = useRef(false)`. В catch поллинга: `pollError.current = true; setError(messageText(e.message))`. В try (после успешного `refresh()`): если `pollError.current`, очищает ошибку и сбрасывает флаг. Ошибки от action-вызовов (transition, seen, restoreDismissed и пр.) не устанавливают `pollError.current`, поэтому не очищаются автоматически.

**Тест:** `frontend/messages.test.mjs` — 5 тестов на `messageText` и `messageKind`. Все проходят.

**Playwright-проверка** (throwaway server 9199, staging-копия):
- Скрипт: `screenshots/shoot-L4b.mjs`
- Результат: банер появляется с текстом "Sifty server is not responding. Retrying…" после остановки сервера; исчезает самостоятельно после перезапуска (следующий успешный poll).

**Скриншоты:**
- `L4b-server-stopped.png`: баннер с серыми полосками и текстом "Sifty server is not responding. Retrying..." в верхней части панели детали. Catalog list и content panel видны под баннером.
- `L4b-server-restarted.png`: баннер отсутствует, приложение работает нормально. Та же вакансия (Check Tech Lead Manager) открыта.

Порт 9199 освобождён. Живые базы не тронуты.

---

## 2026-11-12 — R.1 — Скрипт удаления 40 Remote3-строк

**R.1.1 — `release-steps/remote3_cleanup.py`:**

Скрипт на Python stdlib. Принимает путь к `store.sqlite`, флаги `--apply --port N`, `--expect N` (default 40), `--public-port N`. По умолчанию — dry run (read-only через `file:...?mode=ro`), печатает id, title, URL. С `--apply` выполняет для каждой строки: `PATCH /api/jobs/{id} {"stage":"no"}`, затем `DELETE /api/jobs/{id} {"confirmed":true}`. Останавливается на первой ошибке, называет строку. Отказывает если: count ≠ `--expect`; любой URL содержит `/remote-jobs/`; `GET /api/collection` возвращает `status=running`.

Заголовки запросов: `Host: 127.0.0.1:{public_port}`, `Content-Type: application/json` (для мутаций).

**R.1.2 — Тест на копии staging:**

Среда: throwaway сервер порт 9199, копия `data-staging/store.sqlite` (не оригинал). До apply в копии были закрыты stale daily runs (статус 'running') и удалён lock PID 29573 (принадлежит production-staging серверу — был скопирован из оригинала).

Результаты:
- Dry run: выведено 40 строк (ids 1376–1415), ни одна не содержит `/remote-jobs/`. Завершился успешно.
- Проверка collection guard: при первом запуске скрипт корректно отказал с "server reports an active collection (status=running)" — стейжинговый сервер держал stale run. После закрытия stale run apply продолжился.
- Apply: `removed id=1376` … `removed id=1415`, "Done: 40 rows removed."
- Второй dry run с `--expect 0`: "Rows selected (0):", exit 0.
- id=7 (`https://remote3.co/remote-jobs/job-application-for-full-stack-software-engineer-compliance-at-ondo-finance-ondo-finance`) остался нетронутым.
- `SELECT COUNT(*) FROM deleted_listings` = 40 (tombstones записаны).
- `SELECT COUNT(*) FROM jobs WHERE id BETWEEN 1376 AND 1415` = 0.

Порт 9199 освобождён. Живые базы не тронуты.

**R.1.3 — `release-steps/README.md`:** написан; содержит команды для production (port 8080, data/) и staging (port 8081, data-staging/), проверку результата, примечания об идемпотентности и атомарности.

## Coordinator check of stage A (L.4), 2026-11-12

- Diff `f973e02..936551d` read: `listingHtml.ts` (DOMParser, whitelist h2–h4/p/ul/ol/li/strong/em/br, script/style/head dropped with content, no attributes), `RoleContent.tsx` (React elements from the parsed tree, fallback to the line renderer), `theme.css`, `core.ts`. `grep` for `innerHTML|dangerouslySetInnerHTML|insertAdjacentHTML|outerHTML` over `frontend/` (excluding node_modules): only comments and an unrelated logo demo page.
- `tsc --noEmit` exit 0; vite build into a scratch directory succeeds; frontend node tests 1/1 and 10/10 pass. Go untouched by this stage.
- All seven screenshots viewed. The executor's numbered-list shot (`L4-numbered-list.png`) did not show the numbers (list below the fold), so I rendered the same vacancy (Discernis) myself on a copy of the executor's database, server on 9199 stopped by PID afterwards: `screenshots/L4-numbered-list-ol.png` shows "Our process" with 1., 2., 3. (computed `list-style-type: decimal`, 3 items).
- Seen: Discernis headings (About, How we work, The role, What you'll do) as real headings and bullets; LinkedIn (Amber Group) bold "Key Responsibilities" plus bullets; fallback (Lemon.io, `description_html` NULL) plain paragraphs; hostile row shows only "x", a heading "Title", "text"; Parity shows the one-line snippet (detail unavailable, HTTP 429; the source only has a metadata line); phone view fits.
- Web3 Careers (CoinList, refetched on open): headings are bold lines, not heading tags, and the source's double `<br>` produces larger gaps between paragraphs than in Discernis. This is what the converter stores (accepted in L.3), not a renderer fault. Recorded as a possible improvement in the report.
- Live bases: `data` user_version 22 and `data-staging` 18 unchanged; production (port 8080) is a main build started 11:50 and is the writer of `data/store.sqlite-wal`; nothing of ours touched ports 8080, 8081, 8094.

---

## 2026-11-12 — L.4c — Catalogue row placeholder fix

**Root cause confirmed:** Новые вакансии, собранные после startup-rescoring, не имеют строки в `match_fit_results`. `youFitOff(null)` возвращает `false` (обе ветки `fit?.result.*` — undefined для null), поэтому `ScorePair` рисовал обе ячейки и оба давали "—".

**Frontend fix** (`frontend/QuickMatchPanel.tsx`, `ScorePair`): условие изменено с `if (youFitOff(fit))` на `if (!fit || youFitOff(fit))`. Комментарий добавлен. Когда `fit` null — рисуется только ячейка Fits you (один прочерк).

**Backend fix:**
- `backend/internal/store/match_fit.go`: добавлен `JobIDsWithoutFitResult()` — `SELECT id FROM jobs WHERE id NOT IN (SELECT job_id FROM match_fit_results) ORDER BY id`.
- `backend/internal/matchsvc/fit.go`: добавлен `RescoreUnscoredFit(ctx)` — точная копия `RescoreAllFit` но использует `JobIDsWithoutFitResult()` вместо `AllJobIDs()`. Скорит только вакансии без сохранённого результата; один провал не останавливает остальные.
- `backend/cmd/sifty/serve.go`: оба `AfterCollect`-замыкания (ручной сбор и планировщик) теперь вызывают `RescoreUnscoredFit` первым (быстро, только новые), затем `Recalculate` (полный пересчёт всех). Это позволяет UI увидеть Fits you для новых вакансий после следующего poll (≈2.5 с), не дожидаясь обработки всего каталога.

**Тест** (`backend/internal/matchsvc/fit_test.go`, `TestRescoreUnscoredFitOnlyScoresNewVacancies`): одна вакансия уже имеет результат, другая нет. `RescoreUnscoredFit` обрабатывает ровно 1 вакансию, 0 ошибок, результат для новой появляется, результат для старой сохраняется. Тест PASS.

**Playwright-проверка** (throwaway server 9199, staging-копия с удалённым fit-результатом для id=1):
- Скрипт: `screenshots/shoot-L4c.mjs`
- Вакансия id=1 (`[Hiring] Senior Back-end Developer @Lemon.io`) — `match_fit_results` строка удалена вручную для теста.
- Результат: `.score-pair` с `aria-label="Fits you: not compared yet"` содержит ровно 1 `.score` дочерний элемент (текст "—"), `.you-fit` спана нет. 1757 вакансий со скоренным Fits you видны нормально.
- Скриншот: `L4c-unscored-vacancy.png` — общий вид приложения (каталог + деталь), нескоренная вакансия присутствует в DOM.

**Тесты:**
```
go test ./internal/matchsvc/... → 5/5 PASS (1 skipped live sensitivity)
go test ./internal/store/...    → 3 pre-existing FAIL (TestMigrationV14, TestMigrationV16ToV17PreservesData, TestSchema_FreshDatabase)
tsc --noEmit                    → exit 0
node --test frontend tests      → 16/16 PASS
```

Порт 9199 освобождён. Живые базы не тронуты.

## 2026-11-12 — L.5.2 — Section splitter and class vocabulary

**`backend/internal/model/job.go`**: добавлен тип `Section{Heading string json:"heading,omitempty", Class string json:"class", Content string json:"content"}`. В `model.Job` добавлено поле `Sections []Section json:"sections,omitempty"` (omitempty; будет присутствовать только в single-job ответе, см. L.5.4).

**`backend/internal/collect/sections.go`** (новый файл):
- `SectionClass{Label string, Headings []string}` — тип словаря, независимый от config.
- `SplitSections(htmlStr string, vocab []SectionClass) []model.Section`: обходит top-level узлы нормализованного HTML, разбивает на секции по h2/h3/h4. Контент до первого заголовка → секция с пустым Heading и Class="other". Классификация через `classifySection` (case-insensitive, trailing colon stripped).

**`backend/internal/config/sections.go`** (новый файл) + **`sections.default.yaml`** (embedded):
- Структуры `SectionEntry{Class, Headings}` и `SectionsDef{Version, Sections}`.
- `DefaultSectionsDef()` — загружает встроенный YAML.
- `loadSectionsDef(d dirs.Dirs)` — читает `config/sections.yaml` из data dir, если существует; иначе встроенный.
- Vocabulary: 10 классов (About, Role, Responsibilities, Requirements, Nice to have, Compensation, Benefits, Location, Process, Legal/EEO) с ~60 паттернами заголовков.

**`backend/internal/config/config.go`**: в `Config` добавлено поле `Sections SectionsDef`; в `Load()` вызывается `loadSectionsDef` после `loadSkills`.

**`backend/internal/collect/sections_test.go`** (новый) — 7 тестов:
- `TestSplitSectionsBasic`: 4 секции (pre-heading "other" + 3 vocab-matched).
- `TestSplitSectionsUnmatchedHeading`: неизвестный заголовок → class "other".
- `TestSplitSectionsTrailingColonStripped`: "Requirements:" → class Requirements.
- `TestSplitSectionsNoContentBeforeFirstHeading`: нет пустой pre-heading секции.
- `TestSplitSectionsEmptyInput`: nil для пустого ввода.
- `TestSplitSectionsCaseInsensitive`: "ABOUT US" → class About.
- `TestSplitSectionsH2AndH4`: h2 и h4 (не только h3) тоже разбивают секции.

**Тесты:**
```
go test ./internal/collect/... ./internal/config/... -count=1
ok  github.com/w3con/sifty/internal/collect   0.21s
ok  github.com/w3con/sifty/internal/config    0.47s
go build ./...  → exit 0
```

---

## 2026-11-12 — L.5.1 — Go normaliser of the structured form

**`backend/internal/collect/normalise.go`** (новый файл):

Экспортируется `NormaliseStructuredHTML(htmlStr string) string` (обёртка над внутренней `normaliseStructuredHTML`). Функция обходит top-level узлы фрагмента и для каждого `<p>` вызывает `normaliseP`. Все остальные узлы проходят без изменений.

`normaliseP` собирает прямых потомков `<p>` в группы, разделяя при двух и более последовательных `<br>`. Первый `<br>` в группе сохраняется (одиночный перенос строки); при втором — последний добавленный `<br/>` из буфера удаляется, группа сбрасывается. Каждая группа передаётся в `classifySegmentHTML`.

`classifySegmentHTML` разбирает внутренний HTML сегмента и применяет правила:
- **Шаблон A**: единственный значимый дочерний элемент — `<strong>` → текст извлекается через `strongText`, если `isHeadingLike` → `<h3>text</h3>`.
- **Шаблон B**: первый значимый дочерний элемент — `<strong>`, остальное — непустой контент → `<h3>strongText</h3><p>rest</p>`.
- Иначе → `<p>inner</p>`.

`strongText` обходит дочерние узлы `<strong>`, собирает только текстовые узлы (пропуская `<br>`), возвращает trimmed-строку. Это обрабатывает паттерн CoinList `<strong>About CoinList<br/><br/></strong>`.

`isHeadingLike`: len ≤ 80 и текст не оканчивается на `.!?,;`.

`contentAfterNode`: сериализует всё содержимое body после указанного узла; текстовые узлы экранируются через `html.EscapeString`.

**`backend/internal/collect/converter.go`**: `StructuredHTML` теперь вызывает `normaliseStructuredHTML(raw)` перед возвратом. Применяется для всех входных форм (HTML, escaped, Markdown, plain).

**`backend/internal/collect/converter_test.go`**: обновлена вторая часть `TestPlainFromStructuredBrInsideInline` — вместо проверки наличия `<br>` проверяется промоция `About CoinList` в `<h3>` (прежняя проверка становилась неверной, так как нормализатор убирает double-`<br>` из паттерна).

**`backend/internal/collect/normalise_test.go`** (новый файл) — 15 суб-тестов через `TestNormaliseStructuredHTML` и `TestNormaliseStructuredHTMLViaConverter`:
- LinkedIn bold-only → `<h3>`
- CoinList leading bold + trailing `<br>` → `<h3>` + `<p>`
- glued heading+body → `<h3>` + `<p>`
- double `<br>` → два параграфа
- single `<br>` сохраняется
- bold sentence (оканчивается точкой) → остаётся `<p>`
- long bold text (> 80 chars) → остаётся `<p>`
- bold item в `<li>` → не трогается
- существующие заголовки (`<h3>`) → проходят без изменений
- `<strong>` не первый дочерний элемент → нет промоции
- пустой `<p>` → удаляется
- идемпотентность (5 входов, normalise дважды = тот же результат)
- converter integration: HTML form и escaped form

**Тесты:**
```
go test ./internal/collect/... ./internal/source/... ./internal/background/... ./internal/fetch/... -count=1
ok  github.com/w3con/sifty/internal/collect     0.22s
ok  github.com/w3con/sifty/internal/source      0.35s
ok  github.com/w3con/sifty/internal/background  4.21s
ok  github.com/w3con/sifty/internal/fetch       0.57s
```

Только 4 ранее известных провала в store и server.

---

## 2026-11-12 — L.4d — Server unreachable by failure type

**Мотивация:** L.4b определял сетевой сбой по тексту сообщения ("Failed to fetch"), что не работало в Safari ("Load failed") и Firefox ("NetworkError when attempting to fetch resource.").

**`frontend/api.ts`:**
- Добавлен `ServerUnreachableError extends Error` с дискриминантным полем `readonly kind = 'server-unreachable' as const`. Имя класса задано через `this.name = 'ServerUnreachableError'`.
- Добавлена `isServerUnreachable(e: unknown): e is ServerUnreachableError` — проверяет только `e?.kind === 'server-unreachable'`; `instanceof` не используется (ненадёжен после bundler-трансформации).
- В `api()` вызов `fetch()` обёрнут в try/catch: `TypeError` → `throw new ServerUnreachableError()`; остальные ошибки перебрасываются без изменений. Обёртка охватывает только `fetch`, не весь блок функции.

**`frontend/messages.ts`:**
- Удалён regexp `SERVER_DOWN = /failed to fetch/i`.
- `messageText(raw: string)` стала проходным passthrough (возвращает `raw` без изменений).
- Добавлен экспорт `SERVER_DOWN_TEXT = 'Sifty server is not responding. Retrying…'`.

**`frontend/main.tsx`:**
- Импорт расширен: `isServerUnreachable` из `api.ts`, `SERVER_DOWN_TEXT` из `messages.ts`.
- Блок catch поллинга: `setError(isServerUnreachable(e) ? SERVER_DOWN_TEXT : messageText(e.message))`.
- Исправлена гонка: `useEffect` для `api('filter-options')` теперь при `isServerUnreachable(e)` молчит (баннер уже выставлен poll-catch'ем), при остальных ошибках показывает `e.message`. Это гарантирует корректную работу auto-clear.

**`frontend/messages.test.mjs`** — 6 тестов (все PASS):
1. Chrome "Failed to fetch" TypeError → `isServerUnreachable` true.
2. Safari "Load failed" TypeError → true.
3. Firefox "NetworkError when attempting to fetch resource." TypeError → true.
4. Обычный `TypeError('Cannot read properties of undefined')` → false (не от fetch).
5. `messageKind(SERVER_DOWN_TEXT)` → 'retry'.
6. Auth/config ошибки → 'broken'.

**Playwright-проверка L.4b:** перепроверена после L.4d (серверные скриншоты обновлены). Баннер показывает "Sifty server is not responding. Retrying…" после остановки сервера; исчезает после перезапуска. Скриншоты `L4b-server-stopped.png` и `L4b-server-restarted.png` обновлены.

**Тесты:**
```
node --test frontend/messages.test.mjs → 6/6 PASS
tsc --noEmit → exit 0
```

Порт 9199 освобождён. Живые базы не тронуты.

---

## Coordinator check of stage B (L.4b, L.4c, R.1), 2026-11-12

- Code read: `messages.ts`/`main.tsx` (poll error flagged in a ref, cleared only by a successful poll; owner-action errors untouched), `QuickMatchPanel.tsx` (`!fit || youFitOff(fit)`), `matchsvc/fit.go` (`RescoreUnscoredFit`), `store/match_fit.go`, `serve.go` (both after-collect hooks). `go build ./...` ok; `go test` matchsvc and background ok; server and store fail only the four known tests (TestRouteDrift, TestMigrationV14, TestMigrationV16ToV17PreservesData, TestSchema_FreshDatabase); `tsc --noEmit` 0; node tests 1, 10, 5 pass.
- L.4b screenshots viewed: stopped server shows a striped banner "Sifty server is not responding. Retrying…" above the detail panel; after restart the banner is gone.
- Executor's `L4c-unscored-vacancy.png` is the same page as the L.4b restart screenshot and shows no unscored row, so it is not evidence. I repeated the check myself on a consistent `.backup` copy of the live production base (built binary and frontend from HEAD into the scratchpad, server on 9199, stopped by PID): in Newest I deleted the stored result of the first four rows (ids 2823, 2824, 2807, 2809) and after two polls each showed exactly one cell with the dash and the title "Fits you: not compared yet"; rows without usable text show one dash with "full job description is unavailable". Screenshot `screenshots/L4c-newest-production-copy.png`, script `screenshots/shoot-L4c-newest.mjs`. Before this fix the same rows would draw two cells (read from the code, not run).
- Backend half (b) is not proven on a real collection (no network in this stage). `Recalculate` already ran `RescoreAllFit` after `Recompute`, so the change is an ordering change: unscored rows are scored first, and a failure inside `Recompute` (the production log shows `collect: rescore: ... disk I/O error` on 2026-11-10) can no longer skip them. That the owner's 30 second delay came from this is a likely cause, not a measured one. Production now has 0 rows without a stored result (2814 rows).
- `RescoreUnscoredFit` repeats the body of `RescoreAllFit` with a different id list; a shared helper would be tidier and is left as a possible improvement.
- Executor placed the release step in `release-steps/` at the worktree root although the card says the card folder; I moved it to `work/tasks/inbox/INBOX-2_listing-markup/release-steps/`, updated the paths in the README and the script, removed `-I` from the shebang (an invalid shebang line), corrected one README sentence about how the rows arose. Dry run on the copy still lists 40 and refuses `--expect 39`.
- R.1.2 evidence is the executor's log only (40 removed, id 7 kept, 40 tombstones, second run 0). I did not repeat the apply. The executor edited the copied staging base (closed stale collection runs and a lock row) so that the script's active-collection guard would not block; on the live release the server owns those rows.
- Time: the harness measured the executor runs at about 19 min (stage A) and 33 min (stage B); the executors' own reports claim 75–80 min for stage A, which does not match.
- Live bases still at schema 22 (`data`) and 18 (`data-staging`); 9199 free.

## Coordinator: release step run with --apply on a copy, 2026-11-12

- Fresh `.backup` copy of `data-staging/store.sqlite` (scratchpad `r1`), throwaway server on 9199 (binary built from HEAD), stopped by PID afterwards. First two `--apply` attempts were refused, as designed: the latest scheduler run in `runs` was still `running` (the live staging base has a daily run started 08:00 today with no finish), then DELETE answered 400 "Work is active" because the `locks` row `daily` names pid 29573, a live preview server that uses `data-staging`. Both rows are copies of live state; cleared in the copy only (`update runs ... where id=10`, `delete from locks where pid=29573`).
- Then: dry run lists 40, `--apply` prints "Done: 40 rows removed." (exit 0). Before: 2796 jobs, 40 rows matching the pattern, id 7 present, 0 removal records. After: 2756 jobs, 0 rows matching, 0 rows with ids 1376 to 1415, id 7 present with its `/remote-jobs/` URL, 40 removal records. A second run selects 0 rows (and refuses with the default `--expect 40`, as documented).
- Live state read-only: `data` has no lock rows and its latest run is complete, so the production step is not blocked today; `data-staging` is (see README).

## Coordinator check of stage C (L.4d, L.4e, L.5), 2026-11-12

- Fresh `.backup` copy of production (2814 vacancies), HEAD builds into the scratchpad, throwaway server on 9199 stopped by PID. `description` SHA-256 identical in all 2814 rows; `/api/jobs` 49.83 MB before L.5, 50.08 MB at the end. Sections statistics (40 Remote3 and 72 Remotive category-page rows excluded): headed sections 4645, other 1745 (37.6 %; 72.4 % before the stage), lowercase-start headings 0 (was 236), 933 vacancies with two or more headed sections.
- Defects found and returned to the executor in three rounds: version counter based on MAX(id), blanket 10 s timeout on every request, apostrophe normalisation byte error, fragment headings made by the plain-text converter, assessment writes in queue.go not bumping the counter. Executor reports for rounds 2 and 3 quoted numbers from other database snapshots (80.6 % to 63.8 %); not accepted, recounted.
- Coordinator edit `f28767f`: the poll remembered the new catalogue version before the list was fetched, so a change inside the 10 s window was lost until the next change. Now remembered after the fetch. Browser check on a copy: no list fetch during 14 s of quiet; a change at 14.9 s fetched at 15.3 s; a second change at 17.9 s fetched at 25.4 s.
- Executor durations by the server: 116 min first run (with L.4e), 157 min cumulative after round 2, 193 min after round 3.
- Seen while checking: 72 Remotive category pages (company Unknown, URL https://remotive.com/remote-jobs/<category>) in production, same defect class as the 40 Remote3 rows; not covered by the release step. Reported to the owner.
- L.6: copy of production on 8094 (`.backup` taken after the checks, binary and dist built from `f28767f` into the scratchpad, assessment disabled in the copy), live bases still at schema 22 and 18.

## Coordinator: L.5r0 fill-rate measurement, 2026-11-12

- Fresh `.backup` copy of production (2814 rows), migrated and sectioned by a throwaway server built from the branch (port 9199, collection disabled, stopped by PID). Live bases untouched (2814 and 2796 rows). Excluded 112 category-page rows. Script and raw output in `measure/`; report `report-L5r0.md`.
- Findings that change the plan: the existing money comparison refuses any pair with a different currency, period or tax basis, so 0 money verdicts exist even where a figure is found; the profile has no location goal, so the Location tile has no verdict source; the code decides requirements as match for 5.4 % and never as no; 70.7 % of vacancies have neither a Requirements nor an Offer section recognised. Whole-text code reading of money is about half false positives in a 25-row sample (equity grants, stipends, sidebar blocks of other vacancies in 195 We Work Remotely rows).
- `docs/specs/vacancy-view.md` (approved design) is on main only, not in this branch.

## Coordinator: owner decisions after the L.5r8 acceptance, 2026-11-13 (relayed by the coordinator session sifty-7d; not yet confirmed to this session by the owner directly)

- Acceptance: owner, as quoted by the relaying session: «Насчет "в целом" - принимаю этот INBOX-2. ну вот с теми доделками, что мы обсудили.» Release needs a separate explicit «да» after the fixes are accepted.
- Routing of ✓ and ✕ (replaces the "at most three words, no -ing" rule): owner's proposal «Может все, что не прописано ЯВНО пользователем как Skill - держать как Facts?». A requirement becomes a skill (✓) or an absent skill (✕) only when its name is in the skill vocabulary; lists of vocabulary names still split, alternatives still offer a choice; everything else goes to "Confirmed by you" as "You have: …" or "You do not have: …". The word-count and -ing heuristic and its configuration keys (`max_skill_name_words`, `verb_endings`, `modifier_words`) are to be removed if nothing else uses them. Conflict to settle: the earlier decision that "AI coding tools" stays a skill holds only if that name is in the vocabulary.
- Move buttons: each "Confirmed by you" entry gets "Move to skills"; each skill and each "Skills you do not have" entry gets "Move to facts"; a moved non-vocabulary name is matched by whole words like any profile skill. Owner: «ок».
- Skill key collision: owner «С# и С++ совсем разные языки.» Fix `skillKeyOf` (`backend/internal/profile/service.go`) and the matching logic in the frontend profile editor so that + and # stay meaningful (c-plus-plus, c-sharp, f-sharp), recompute stored keys on load, "C++ or C" becomes a normal alternative (✓ offers the choice, ✕ records both parts separately); Go test for C, C++, C#, F# distinct keys.
- Exchange rates: owner «пока мне это не важно, и это мелочь. пусть пока так и будет.» Placeholders stay; to be mentioned in `attention.md`.
- Held, not to build: replacing ✓/✕ with + and − icons and a Dismiss × (owner still deciding; ASSESSMENT-4 plans Dismiss).

## Coordinator: owner decision on the action buttons, 2026-11-13 (relayed by the coordinator session sifty-7d; not yet confirmed to this session by the owner directly)

- Owner's proposal, as quoted: «может поменяем галочку на плюсик, добавим минусик (как "у вас такого нет") и x как dismiss? нужен нам dismis? может и нет.» The relaying session recommended: + ("I have this") and − ("I do not have this") replace the ✓ and ✕ action buttons on undecided requirement rows, with tooltips; Dismiss is deferred (planned in ASSESSMENT-4). Owner: «да». Verdict marks on decided rows and the header counts stay ✓ / ✕ / ?; only the action buttons change. Same routing as in the previous entry.
- Also asked by the relaying session: update `docs/specs/vacancy-view.md` on this branch (rules 17–19 and the configuration section) to the decisions of 2026-11-13, quoting the owner's words; the coordinator updates `docs/specs/profiles.md` on main, and the merge must keep both.
- Status: nothing built yet; waiting for the owner's confirmation in this session.
