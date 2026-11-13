# APP-9 — Find a shortlisted vacancy on the employer's own site and offer to apply

**Epic:** application-preparation
**Created:** 2026-11-13
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Кстати, надо добавить карточку для таких случает (которых много) - мы должны искать вакансию на сайте работодателя (для тех вакансий что в shortlist) и если находится -  предлагать сразу идти туда (там кстати и регистрация обычно не требуется).  Заведешь карточку?

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Owner decisions — 2026-11-13

Recorded verbatim; the English rendering is in `docs/specs/application.md` §3.0, §3.6, §4 and §5 question 9.

Stages:

```text
полный цикл - это вместе с ответами и даже автоматически заполненные. Это идеал, к чему мы стремимся. Итеративно. Поэтому это наш третий, финальный этап. Идеал
твой второй вариант - это наш второй этап.
Наш первый этап - это твой первый вариант.
```

Priority: «ХОчется уже чтобы был полный цикл, а качество каждого этапа улучшать можно потом.» Order: «Порядок что ты предлагаешь. - да, согласен.» (INBOX-2 release → APP-9 → stage 2 design session and answers → quality cards.)

Why the employer's site: «wellfound - много хороших вакансий, но технические ограничения самого wellfound не позволяют кандидатам не из америки апплаиться.»

The owner's path:

```text
1) Находить ссылку на форму заполнения (обычно это сайт аггрегатор) - уже должно быть найдено при разборе вакансии в самом начале, после ее fetch.
2) Пытаться найти в ссылку на сайт работодателя (сначала искать на странице вакансии у аггрегатора, если не найдено - надо видимо искать в гугле, находить сайт работодателя, находить его careers page, как бы она не называлась, находить вакансию.
3) Если найдена прямая ссылка - дальше работаем только с ней.
4) Парсим вопросы на странице вакансии (аггрегатора или компании)
5) Смотрим что есть в профиле пользователя, формируем в базе данных предлагаемые вопросы на каждый ответ.
6) показываем пользователю для утверждения. (это наверное уже как раз что на этапе 2).
7) пользователь может изменить текст предложенный. Мы (llm) должны посмотреть что в изменениях и сохранить факт в профиль пользователя. Возможно, в профиле пользователя (в смысле в структуре данных) должны быть такие заранее подготовеленные вопросы на ответы - чтобы не спрашивать LLM каждый раз написать вопрос на типовой ответ.
8) Даже "cover letter" и персонализированные ответы типа "почему вы хотите работать в компани XYZ?" сохраняются - из них можно вытащить полезную информацию при ответе на вопросы в других вакансиях.
9) При подготовке ответов - сначала находятся типовые ответы (если в вакансии только типовые вопросы) и LLM смотрит факты о пользователи и предыдущие ответы и саму вакансию чтобы составить Cover letter.
```

On the coordinator's amendments: rule 7 with owner confirmation of extracted facts — «7 ок»; all other proposals (separate window instead of iframe, applicant-tracking-system boards before web search, reuse facts not text, sensitive questions left to the owner, extension in stage 3) — «со всеми твоими другими предложениями согласен».

Search providers: «у меня есть Tavily, но вообще это тоже надо вытащить в Settings - чтобы можно было задавать ключи Tavily/Brave/любого поискового движка. Как fallback надо выбрать кого-то типа DDG? ... Такие "fallbacks" тоже должны задававаться в настройках. Пока, для простоты - в yaml. Что будет нужно позже вынесем в отдельные поля.» Coordinator's reading, to be confirmed by the owner: web search is part of stage 1 because the Tavily key exists; recommended chain Tavily → Brave → DuckDuckGo.

Open: forms behind sign-in (LinkedIn, Wellfound) — owner asked «что мы можем сделать? ... предлагать пользователю залогиниться ... или же - держать у себя справочник вопросов разных агрегаторов? что-то еще?»; coordinator recommends a catalogue of the fixed form part plus pasting the form text in stage 2, and the extension in stage 3 (spec §5 question 9). Not part of stage 1.

Stage 1 scope for the plan: form link at parse time; employer lookup (aggregator page → ATS boards → web search → careers page); search provider chain in YAML; Apply opens the direct link in a separate window. Estimate 5–7 h. Plan to be written after the INBOX-2 release.

Owner, later on 2026-11-13: web search in stage 1 — «Подтверждаю». Forms behind sign-in — «Так ты сам уже написал, как с ними быть» quoting options 3 (paste the form text, stage 2) and 4 (extension, stage 3); recorded as `docs/specs/application.md` rule 27.
