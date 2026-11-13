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

## Plan (coordinator, 2026-11-13) — stage 1 of the application path

Waiting for the owner's approval. Specification: `docs/specs/application.md` §3.0 (stage 1) and §3.6 rules 20–25, §4 (search providers).

**Outcome.** For every shortlisted vacancy Sifty knows the best place to apply: the employer's own page or applicant-tracking-system (ATS) page when it can be found, otherwise the aggregator's page. Shortlist shows which one it is; Apply opens it in a separate browser window beside Sifty; the aggregator link stays available as a secondary link. Lookup runs when a vacancy enters Shortlist, once for the vacancies already in Shortlist, and on demand from a "Find employer page" button.

**Facts found while planning (production database, read-only, 2026-11-13).**
- 59 vacancies in Shortlist (`status='saved'`); 22 have an apply link. 15 are Wellfound with no apply link; others are web3.career, We Work Remotely, venture-portfolio boards (Framework, Electric Capital, Multicoin), Hacker News, LinkedIn, ATS pages.
- The `jobs` table already has `apply_url` and a free-text `apply_status` (about a dozen texts, written by the legacy TypeScript resolver `legacy/app/src/discovery.ts` `hydrateCandidate`). The Go collector sets `ApplyURL: nil` for HTML sources (`backend/internal/collect/parse.go:147,209,285`), and Lever/Ashby `applyUrl` fields are parsed but not mapped (`backend/internal/source/lever.go:50`, `ashby.go:52`). Apply links are therefore set unevenly since the Go port.
- Apply in Shortlist already opens `externalApplyUrl(job)` (apply link, else original) in a new tab and then shows the outcome dialog (`frontend/main.tsx:106-115`).
- ATS clients for Greenhouse, Lever and Ashby boards exist in `backend/internal/source/`; per-host pacing exists in `backend/internal/background/discipline.go`.

**Must not touch.** Form inspection and the headless-browser worker (APP-4, APP-5); answer drafting and draft editing (stage 2); the assessment queue; Quick Match scoring; profiles code (SETTINGS-5 runs in parallel in another session — see coordination below); `.worktrees/INBOX-2`; production data (copies only). No form filling, no POST to any employer site.

**Coordination with SETTINGS-5.** Both cards touch the stage-change path (APP-9 adds a lookup trigger when a vacancy enters Shortlist; SETTINGS-5 S.4 makes stages per profile) and `frontend/main.tsx` (APP-9 changes `apply()`). The card merged second rebases on main and resolves; the trigger stays "vacancy entered Shortlist in any profile".

**Acceptance criteria.**
- Vacancies from Greenhouse, Lever and Ashby sources carry their ATS page as the direct apply link; aggregator pages that link to an employer or ATS page carry that link (legacy resolver rules ported, intermediaries such as LinkedIn excluded).
- Lookup order per vacancy: collected direct link → aggregator page link → ATS public boards by company name (Greenhouse, Lever, Ashby, Workable) → web search for the company's careers page and the vacancy on it. A candidate is accepted when title and location match; an ambiguous candidate is confirmed by the extraction-stage model; a failed lookup records an honest state ("employer page not found") and keeps the aggregator link.
- Result stored per vacancy: apply link, how it was found (collected, aggregator page, ATS board, web search), when, and the aggregator link kept separately; `apply_status` texts replaced by a small fixed set of states.
- Web search providers are an ordered list in a YAML config file (Tavily, Brave, DuckDuckGo HTML) with per-provider key taken from an environment variable or a git-ignored key file named in the config; keys never in the YAML; a provider that fails is logged with its reason and the next one is tried; requests are paced.
- Shortlist and the vacancy view show "Employer page" or "Via <aggregator>" next to Apply, a "Find employer page" button when nothing direct is known, and the aggregator link as a secondary link; Apply opens the best link in a separate window placed beside Sifty (falls back to a tab if the browser refuses), then shows the outcome dialog as today.
- On a production copy: lookup over the current Shortlist; report counts by method, and a sample of ten direct links opened and read by the executor (right company, right vacancy).
- Spec `docs/specs/application.md` rules 20–25 and §4 tags updated; `docs/MAP.md` row updated.

- [ ] **A.1 — Direct links at collection** (45–60 min): map ATS `applyUrl`/page URLs; port the legacy aggregator-page rule (employer or ATS link on the vacancy page, intermediaries excluded); fixed state set; targeted tests per source.
- [ ] **A.2 — ATS board lookup** (60–90 min): company-name to board-slug candidates; query Greenhouse, Lever, Ashby, Workable public boards through the existing clients; title and location matching; targeted tests with recorded board responses.
- [ ] **A.3 — Web search chain and careers page** (60–75 min): provider chain config and clients (Tavily, Brave, DuckDuckGo HTML); query for the company's careers page; on the careers page detect an embedded ATS (then A.2) or find the vacancy link by title; pacing; tests with recorded responses.
- [ ] **A.4 — Lookup worker, model confirmation, API** (45–60 min): run on entering Shortlist, once for the current Shortlist, and on demand (`POST jobs/{id}/employer-lookup`); ambiguous candidates confirmed by the extraction-stage model; result persisted; lookup never blocks Apply.
- [ ] **A.5 — Interface** (45–60 min): link kind next to Apply, "Find employer page" button, aggregator link as secondary, Apply in a separate window beside Sifty.
- [ ] **A.6 — Check on a production copy** (30–45 min): lookup over the copied Shortlist with the owner's Tavily key; counts by method; ten-link sample read; spec and map updated.

**Estimate.** Complexity 5. Implementation 5–6.5 h, coordinator acceptance 30 min. Paid use: Tavily queries — about 150–200 in A.6 (59 vacancies × 2–3), plus a few dozen in development; model confirmations only for ambiguous matches (extraction-stage model). Release is separate and needs the owner's «да».
