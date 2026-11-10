# SETTINGS-2 — Profile: исправить workflow и уточнить поля

**Epic:** settings
**Created:** 2026-11-09
**Status:** review
**Kind:** bug

**Integration status,2026-11-10:** technical implementation accepted; source published to main in commit `7f80bbf`, final application source `58af432`. Card remains review, not done; production is not updated. Owner acceptance/release is separate.

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Profile

Candidate profile
Only confirmed data is shown. Empty values remain unknown; legacy configuration is retained on the server and is not edited here.

Revision 1
Close profile
Derived context/ranking is not confirmed current (stored context predates the confirmed profile; refresh context). The saved profile is intact; refresh context before relying on scores.

Quick match uses target roles and skill names as search signals. An unknown skill level is not production experience, and free-text dealbreakers are checked only in the detailed LLM assessment.

Analysis started for profile.pdf. 

profile.pdf
error
pdf · 7,888 text characters · 0 facts

extraction provider "qwen3.5-9b" failed: Post "http://127.0.0.1:11434/api/chat": context deadline exceeded

Analyze
Retry
View text 

faq.md
missing
md · 0 text characters · 0 facts

Configured file is missing under the data directory

No local original or cached text is available for this source. 

- не работает, и сообщения об ошибках не ясны и не понятно что делать. 

website
imported
remote_profile · 9,215 text characters · 0 facts

Analyze
View text  - это что значит? 
website
imported
remote_profile · 9,215 text characters · 0 facts

Analyze
View text  website (какой?) импортирован, распарсен, но никаких фактов не найдено? или он не был еще анализирован? если нет - то почему? когда он должен быть проанализирован? нужно же поле, куда вставить URL? 
Может нужна отдельная группа для "Social networks" и отдельная для "Your website"? - и когда пользователь вставляет ссылку - начинается fetch/parsing/analysis? 

ithub
error
remote_profile · 1,963 text characters · 0 facts

extraction provider "qwen3.5-9b" failed: Post "http://127.0.0.1:11434/api/chat": context deadline exceeded

Analyze
Retry
View text 


Ок, я понимаю, что ты взял из Базы Данных то, что есть - но не включал анализ. Давай по другому - пусть эти поля будут пустыми (раз уж анализа все равно нет - и я введу сообщения, так мы пройдем по реальному workflow). 

Посмотри на баги, какие можно поправить быстро, типа настройки LLM - поправь, там где нужны поля новые, например для ввода URL или группы - сделай карточку. 

Еще момент - если данных нет пусть будет "не задано" а не "неизвестно". ну типа Not set или какое-то общеупотребительное слово. 

Когда я хочу добавить Experience или Skills - сначала появляется кнопка Добавить, которую мне надо кликнуть, чтобы появились уже поля добавления. Зачем этот первый экстра-шаг? если записей в этой группе еще нет - пусть сразу открывается поле(я) для редактирования первого. 

При сохранении добавлений - у ВСЕХ открытых групп появляется куча информации:
The profile changed to revision 3. Your draft is kept below; nothing was overwritten. Compare this section, then choose explicitly.

Latest stored section
Skills
Unknown
Your kept draft
The form below still contains your draft and any validation error. Rebasing replaces this section on the next Save.

Rebase kept draft (replaces this section)
Discard draft and use latest 

зачем? во первых зачем обновляются ВСЕ группы, а не та, которую сохраняют? Во-вторых зачем писать всю эту информацию и спрашивать rebase или discard? Не все пользователи понимают rebase, и зачем это второй вопрос, если вверху группы уже есть кнопка cancel? и при этом сохренеие куда-то происходит, раз версии есть, но потом ни откуда версии востановить нельзя. 

Короче - убрать запись других груп кроме сохраненных, убрать версионность, убрать всю информацию про сохранение/ребейз. При нажатии на save надо просто сохранить ту группу которая открыта, со всеми правками или без правок что произошли. (ну если правок не было - можно не сохранять наверное, хотя возможно лучше и сохранить - подумай.)

Все что касается выбора дат - должен быть календарь (готовая библиотека) для простоты UI (c возможностью вручную вписать даты - нужен placeholder для демонстрации формата даты)

языки уровень - должны быть понятные предзаписанные уровни - A1,A2,B1, ...  с подписями типа A0 - Novice, A1 - Beginner, A2 - ....


пока сделай карточку, я еще сейчас напишу отдельно по каждому пункту где какие изменения нужны. 
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Подготовленный объём — 2026-11-09

Владелец поручил продолжить планирование. Теперь [plan.md](plan.md) содержит предложение по исправлению существующего workflow; исходный полный отчёт выше сохранён как первичная запись, не как список работ одной карточки. Поля вынесены в [SETTINGS-3](../SETTINGS-3_profile-fields/plan.md), URL-источники в [SETTINGS-4](../SETTINGS-4_profile-url-sources/plan.md), нормализация в [ASM-5](../../assessment/ASM-5_shared-term-matching/plan.md). Это устраняет пересечение scope, а не дублирует баги в нескольких карточках.

**Уточнённая оценка этой карточки:** complexity 5; AI preparation 15–25мин, implementation 55–90мин, acceptance 20–35мин; всего 90–150мин без ожидания владельца и внешнего provider latency. Подписка, точная цена неизвестна; облачный fallback не включён. Верхняя граница — остановка и переоценка. Прежняя общая intake-оценка не прибавляется к этим временам.

Выполнение этой карточки одобрено владельцем 2026-11-09: «Ладно, тогда запускай первую карточку как ты предложил, из плана.» Остальные планы не запущены. Source baseline — сохранённый SETTINGS-1 HEAD `d6f5808`, отдельная рабочая копия SETTINGS-2 без интеграции snapshot в main. Запрошенный пустой staging-старт выполняется только с восстановимым снимком текущих данных, не очищением production. Точный объём и оценка остаются в plan.md.

**Результат 2026-11-09:** техническая реализация принята в `.worktrees/SETTINGS-2`, все шаги закрыты; отчёт координатора скопирован в [acceptance.md](acceptance.md). Результат был доступен на http://localhost:8081 → Profile. Исходники сведены в main коммитом `7f80bbf` (итоговый `58af432`); ожидается отдельное принятие владельцем, production не обновлён, качество фактов из настоящего документа пока не доказано. Следующие карточки не запущены.
