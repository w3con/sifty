---
estimate: 5
---
# URL-источники Profile

**Approval:** proposed; execution not approved. Depends on SETTINGS-2. SETTINGS-3 can precede or follow without changing source-ingestion semantics.

## Результат и демонстрация

Владелец в Website или Social profiles вводит URL и нажимает **Add and analyze**. Это явное действие запускает fetch → извлечение текста → один настроенный analysis, после чего владелец принимает/отклоняет предложения. Вставка/печать URL, открытие страницы и обычный Save не запускают модель. Websites и Social profiles — категории одного document/fact pipeline, не новые независимые профили.

## Решения и ограничения

- Один Website и несколько Social profiles; если существующих сайтов больше, сохранить их, не отрезать данные. Показывать hostname, полный адрес по ссылке, дату получения, этап и фактический outcome. `website` без адреса не выдавать за новый добавленный URL.
- Сохранить исходный/final URL, текст и content hash; факт ссылается на документ/версию и точную цитату. Новые факты не перетирают подтверждённые автоматически.
- Fetch только через существующий guarded client/path resolution; профильная страница не проходит через JobInput/job-posting parser. Reuse существующее извлечение читаемого текста; не дублировать HTTP/provider инфраструктуру.
- Обрабатывается указанная публичная страница, а не все ссылки домена. Redirect/403/login/no text/timeout объясняются с действием. Публичность URL не гарантирует доступности social profile без входа; не обещать поддержанный LinkedIn crawler. Для закрытой страницы доступны уже существующие upload/paste-text пути.
- При сохранённом тексте Retry analysis не скачивает страницу повторно. **Refresh source** — явное действие; неизменный hash не повторяет inference; изменение текста делает старые предложения неактуальными, но не уничтожает принятый профиль. После содержательного изменения анализ запускается только в явно запрошенном действии refresh-and-analyze.
- Single run per document/content hash, cancel/run guards, без скрытых бесконечных retries. Повторный Add того же URL не создаёт повторных вызовов. Конфигурация существующих лимитов/timeouts — единственный источник; новый лимит, если необходим, только в runtime config.
- Успех с нулём фактов показывать отдельно от «ещё не анализировали». Экран содержит один понятный следующий шаг; технический transport dump не основное сообщение.
- Ни массовых model calls, ни provider switch. В приёмке максимум одна короткая synthetic URL→facts проверка на выбранном provider; если она не удалась, критерий полезности не принят. Личный URL анализируется владельцем в реальном workflow, не беззвучным acceptance-циклом.

## Слои и исключения

SQLite documents/facts — authoritative inputs; configured client/provider/Dirs — сеть, параметры и пути; Profile application service — fetch/parse/analyze effects; hooks/views — form/status/actions. Не вшивать URL примеров, личные профили, адреса окружений или provider/model в application logic/views. Никакого plugin framework, external vacancy aggregator, login automation или production deploy.

## Оценка и шаги

Сложность 5. Preparation 15–25мин, implementation 55–90мин, acceptance 20–35мин; **90–150мин** всего, stop/re-estimate на 150мин. Review/сборки входят; external latency отдельно. Точная inference цена неизвестна; local configured provider не имеет облачного per-call счёта, но может задержать работу. Переход на billed API — отдельный scope/cost, не fallback.

| Шаг | Результат / зачем | AI-время | Внешняя цена |
|---|---|---|---|
| 4.1 | Контракт URL/category/hash и guarded fetch→text; превратить ссылку в реальный источник | 30–50мин | Публичный HTTP; тарифов новых сервисов нет |
| 4.2 | Явный analyze/retry/refresh/cancel через existing facts; исключить повторные вызовы и потерю правок | 30–50мин | Один короткий configured inference при проверке; цена неизвестна |
| 4.3 | Website/Social UI, браузер и focused hidden checks; пройти полный полезный workflow | 30–50мин | Дополнительных сервисов нет |

Все строки обязательны. Полный crawler и social-specific платные adapters не optional additions этой карточки.

## Checkpoints

- [ ] 4.1 — Добавить URL-контракт и извлечение текста через существующий HTTP-клиент.
- [ ] 4.2 — Подключить явный анализ/повтор/обновление с source/run guards.
- [ ] 4.3 — Показать группы Website/Social и проверить настоящий URL→факты workflow.

## Acceptance criteria

- Браузер: Add and analyze публичной страницы → понятные этапы → attributable facts → accept/reject → сохранённый Profile после reload.
- URL/hostname и состояния различают fetch failure, text ready, never analyzed, analyzed-empty, model error. Закрытый social URL даёт честный fallback upload/paste, не ложный успех.
- Typing/GET/Save не вызывают модель; duplicate Add/Retry не запускают одновременные повторные inference. Один targeted-тест скрытого повторного вызова и late old-hash publication с reuse existing coverage.
- Текст источника, исходные файлы и ручные правки сохраняются; guarded fetch сохраняет проверку redirects/private IP. Build и browser отдельно от SKIPPED broad suites.
- Записать фактическое время, calls и стоимость если доступна; отсутствие end-to-end evidence означает непроверенный результат, не delivered. Production untouched.
