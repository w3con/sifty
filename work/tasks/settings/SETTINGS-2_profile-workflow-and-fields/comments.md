# SETTINGS-2 — comments

Captured from the owner's report on 2026-11-09. No plan or work has started.

## Capture — 2026-11-09

**Kind:** idea

Owner's words added verbatim:

```text
1. Identity - нужно добавить Legal Name. Headline - это что? убрать. 
Добавить пол, национальность (можно как в американских анкетах - белый/черный/латинос/...).


2. Skills - давай подумаем. делать нормализацию или не надо? как библиотека оценки по весам будет различать React.js и React и Reactjs? Хочется простое решение. Есть идеи? 

Уровень - мне кажется лучше "Начальный/Уверенный/Эксперт" - это скорее для подготовки ответа на вакансию, чем при оценке. 

Кстати, надо ли нам разделять (иметь признак) что мы используем для оценки весовой, что для LLM работы? Это возможно поможет тебе быстро понимать, при проектировании/разработке как исползуется параметр, и может ли он быть произвольным или должен быть заданным. Что думаешь? нужно такое или это ненужная сущность? 

Контекст для скилла - это что? пока не пониаю что это, если нет объяснений - убирай. 

3. Exerience 

End date - или должен быть чекбокс рядом - все еще работаю (который disable end date).

Skills - кажется не надо - они общие для всего профиля. Предлагаю оставить только "Summary". 
Возможно, давай к таким большим текстовым полям добавим какую-нибудь простую библиотеку Wisywig - чтобы можно было делать жирным, итакликом, списки, хэдеры. 

4. Target roles and industries 

Вопрос опять - нужно ли использовать свободные/произвольные значения или pre-hardcoded (из списка)? 
Смотри, разные сайты вакансий имеют разную таксономию. Нужны ли нам таблицы соответствий? или как? Если человек хочет Директор по продажам, а у сайта вакансий есть Director, Directeur (на французском например), CSO, ... - нам нужно подумать видимо над общим решением. Как у нас сейчас сделано? как лучше сделать под неограниченный список сайтов? И чтобы желательно без LLM - потому что fetching должен быть автоматическим.
При этом, у нас есть профиль - и мы можем использовать LLM чтобы соотнести то, что напишет пользователь с тем что мы храним в базе, если мы позволяем писать произвольно. Но кажется лучше все таки давать выбирать из того что есть, чем гонять LLM даже единожды. Что думаешь?  Кстати, если делать, то можно указывать в профиле, что "эта информация используется при подготовке ответа на вакансии, не для оценки" (тултип при наведении на иконку вопроса рядом с таким полем). 

5. Compensation

Cash слово плохое. Просто минимум / максимум / ... или Payment minimum... 
Валюта - на выбор - Евро, доллары. Нужен выбор "можно в крипте / только в крипте / Неважно" (по умолчанию - неважно, а не "unknown").

Gross/net - поменять на checkbox/toggle - After taxes. 

Payment period - тоже поменять на checkbox/toggle - monthly / yearly. Это про оценку оплаты, а не про периодичность оплаты. 

Equity/tokens -  тоже поменять на checkbox/toggle


6. Engagement, hours and work modes

Engagement - а "консультант" это куда? МОжет лучше назвать "Permanent / Contract"? или как-то еще?
Если указан фуллтайм - поле ввода часов не должно появляться, а weekly hours надо назвать Desired hours? Стоит ли давать указывать именно часы вручную, а не выбор из списка (меньше 10, 10-20, не меньше 20, до 30, больше 30) ? что думаешь? как проще для нас сейчас (нам же надо будет потом это искать при оценке тоже). 

Work limits - это что? 

7. Geography and authorization

Локации - стоит ли их делать "облаком тэгов"? указываем страну только? или город тоже? разделяем понятия город и страна? наверное надо. 

надо сделать/объединить 2 группы полей: текущая локация и возможная локация. в каждой группе есть локация собственно, и возможность работать в этой локации. В каждой локации - это видимо набор тэгов. Типа я могу выбрать "France" "Paris" "Canada" - это возможные локации. Что думаешь? или надо "Paris, France"? 

Work eligibility - давай сделаем toggle между "Eligible to work" и "Need help with visa"? или плохо? 

Timezones / overlap - это в раздел "желательной локации" - и наверное лучше выбор из опций ("Запад америки, восток америки, европа, средняя азия, юго-восточная азия, япония/океания"? что думаешь? 

Про языки и Availability уже писал.


Dealbreakers - тот же вопрос про нормализацию/произвольный ввод. 
```

## Coordinator preparation summary (not owner wording, not an approved plan)

**Observable outcome.** The owner starts a real empty-profile flow (no preloaded profile data), adds sources and fields, saves sections independently without revision/rebase chatter, and receives actionable analysis errors. Demonstration is a later owner/browser walkthrough of the full workflow; no evidence exists today. This capture clears no data, changes no production configuration and makes no model calls. SETTINGS-1 technical acceptance is not owner acceptance; this intake does not close or release it.

**What the owner asked for (requests, not yet approved as implementation).**

- Fix the current profile workflow: diagnose and describe provider failures with a next action; make source states unambiguous; isolate each section save; show "Not set" instead of "Unknown"; open the first empty record's fields immediately (no extra Add click); date pickers (library) with manual entry and a format placeholder; predefined language levels with labels.
- Adjust profile fields: Identity (add Legal Name; remove Headline; add gender, nationality/ethnicity); Skills (normalization, level labels, provenance/use distinction, remove unexplained Context); Experience (end-date "still working" control, remove per-entry Skills, keep Summary, rich-text editor for long text); Target roles/industries (free-form vs list, cross-site taxonomy, aliases, use-for-application tooltip); Compensation wording and toggles; Engagement/hours naming and conditional hours; Geography (tag/location model, current vs desired, work eligibility toggle, timezone options); dealbreakers normalization.
- New URL source entry with fetch/parsing/analysis, and separate website/social-network groups.

**Verified current-code findings (read-only coordinator inspection).**

- `.worktrees/SETTINGS-1/backend/internal/profile/ranking.go`: `FastRankingQuery` composes the existing configured FTS expression with target roles plus skill names as one OR group; it does not add a structured weighted salary/location engine.
- `.worktrees/SETTINGS-1/backend/internal/store/rescore.go`: ranking is SQLite FTS5/BM25 relevance, not structured weighted matching; no shared alias/normalization layer.
- `.worktrees/SETTINGS-1/frontend/profile/useProfile.ts`: `saveSection` saves one section but marks every other open draft stale when the global revision changes, which explains the reported cross-section chatter.
- `.worktrees/SETTINGS-1/backend/internal/profile/contract.go`: identity has `displayName`/`headline`; skill levels are Production/Familiar/Learning or empty, while name/context are free-form; geography is a current string plus desired/eligibility/timezones string arrays, not structured location objects.

**Coordinator recommendations — not approved.**

- One shared canonical ID/alias layer for profile and vacancy values, with documented use by field (search, quick ranking, LLM assessment, drafting) rather than user-facing toggles or a new entity.
- Section-scoped saves with a hidden concurrency guard; saving an unchanged section may be a no-op.
- Structured country/city locations with per-country work authorization.
- Do not equate a generic "Director" or ambiguous "CSO" with a sales director.

**Open questions for the continuing interview (no decision recorded).**

- Nationality/citizenship vs race/ethnicity distinction.
- Minimum/target vs minimum/maximum pay.
- What an equity/token preference means.
- Current location vs desired location and timezone/overlap semantics.
- Free-form vs predefined values, and how aliases are maintained without LLM calls.

**Future scope split — preparation only; separate implementation cards must be proposed and approved before work (complexity >= 8). Do not create those cards now.**

- **A. Existing workflow bugs:** provider diagnosis/errors, save isolation, "Not set", empty first rows, dates, language levels.
- **B. Profile field adjustments:** identity, skills, experience, targets, compensation, engagement/hours, geography/authorization.
- **C. URL/source entry:** fetch, parsing, analysis state and errors.
- **D. Shared aliases/normalization:** aligned with INBOX-1 search/scoring semantics.

One intake card continues the interview now. No checkbox steps and no approved plan; this card does not authorize implementation, a worktree, deployment, data clearing, production changes or model calls.

**Preliminary estimate for all requested work — explicitly provisional and unapproved.**

- **Complexity:** 13 — UI, API/schema, source ingestion and matching semantics across unresolved questions.
- **AI preparation:** 30–60 min.
- **AI implementation:** 4–7 h.
- **AI acceptance:** 1–2 h.
- **Total AI elapsed:** 5.5–10 h, excluding owner replies and external fetch/model delays.
- Exact provider costs are unknown; no model call was authorized or performed in this capture. Capture and notes took roughly 5–10 min and are not the implementation estimate. Re-estimate and itemize the independent scopes before any approval.

## Planning continuation — 2026-11-09

Owner instruction: «продолжай планирование карточки(ек)». This authorizes planning, not implementation. Owner excluded citizenship: «полагаю гражданство не нужно.» Owner approved the coordinator's other recommendations in principle: «С остальными твоими комментами согласен, там где ты до конца не определился - реши как тебе кажется будет лучше, потом я еще раз посмотрю». This is not approval of the new costed execution plans.

The coordinator's plans now separate four outcomes: this card owns workflow defects; SETTINGS-3 owns profile fields; SETTINGS-4 owns explicit URL-source ingestion; ASM-5 owns shared terms actually connected to profile selection and existing ranking. New field definitions below are coordinator decisions for owner review, not quotations authored by the owner. Earlier no-split/no-plan statements describe the capture stage and are superseded for planning only.

Preserve the owner's niche-source workflow: web3.career and other owner-selected sites work independently of JobSpy or a future aggregator. A vocabulary describes skills/roles; it is not a job feed or a prerequisite for collecting a vacancy. Unknown terms remain usable original text. Automatic LLM synonym discovery is not proposed.

The first capture executor introduced an extra «и» in the owner's hours question and incorrectly described the existing skill-level enum as free-form; both were corrected during coordinator planning review. Its earlier exact-verbatim verification claim was therefore inaccurate.

The previous 5.5–10h intake estimate is superseded by the itemized four-card proposal: 7.5–12h AI elapsed, excluding owner replies and external delays. The increase reflects explicit schema-preserving field changes, real provider/source verification, and taxonomy integration into current ranking rather than an unused dictionary. Individual cards may be approved independently; workflow fixes do not depend on normalization.

## Capture — 2026-11-09

**Kind:** bug

Owner's words added verbatim:

```text
Баги:

website
Failed
remote_profile · 9,215 text characters

Source: https://tikonoff.com/

The selected model took too long to respond. Try again. If it keeps timing out, check the selected model in Settings.Open Settings

Technical details
Retry analysis
View text 

The selected model took too long to respond. Try again. If it keeps timing out, check the selected model in Settings.Open Settings

Technical details
extraction provider "qwen3.5-9b" failed: Post "http://127.0.0.1:11434/api/chat": context deadline exceeded
Retry analysis
View text 

но много фактов было добавлено из CV например. добавка предложенных фактов  - работает.
```

