# OPS-11 — log

## Coordinator, 2026-11-13: stage S.1–S.2 dispatched

Owner, on the approach: «запусти отдельного агента на создание спеков по каждой фичи которая у нас есть. Согласен? думаешь имеет смысл?» and «ты координатор, самому делать не надо». Coordinator's answer: one agent per area, not one for everything (the whole code base does not fit one context); agents write the specs, the coordinator reviews them against the owner's decisions in the cards.

Seven executors (Sonnet 5.5) launched in parallel. Each writes one file into the coordinator's scratchpad `/private/tmp/claude-501/-Volumes-SSD-Dev-tknff-sifty/92c828a0-c188-40b2-b55c-12af52f00517/scratchpad/specs/` and runs no git command, so no two writers share this worktree: `collection.md`, `search.md`, `triage.md`, `application.md` (with the grounded model assessment), `settings.md` (with the current single profile), `data.md`, `inventory.md` (all non-terminal cards). Next: coordinator reviews each file; one executor then moves them into `docs/specs/` and this card folder, indexes them in `docs/MAP.md`, runs `bin/map-check` and commits. If the session dies before that, the scratchpad files may be lost: re-run the missing areas.

## Coordinator, 2026-11-13: search spec writer stopped twice

The search-and-catalogue writer (Sonnet 5.5) was stopped twice by an API safeguards error `[reasoning_extraction]` while reading sources (requests `req_011Cgz35YqKqfpNvoCo9vrtV`, `req_011Cgz3JjD2EBfZz5BdzEwmW`); no file written. The owner's fallback «Sonnet 4.6» cannot be selected: the agent launcher offers only sonnet (5.5), opus, haiku, fable. Question to the owner: run `search.md` on Opus 5.5. The other six writers continue.

Also stopped the same way: the local-data writer (request `req_011Cgz3NczpyAG9TkCVsw1qo`), but only after it had written `data.md` completely (13 sections, 30 KB); only its hand-back report is missing. `search.md` is the only missing spec.

## Coordinator, 2026-11-13: settings writer stopped; Sonnet 4.6 fallback prepared

The settings-and-profile writer (Sonnet 5.5) was stopped by the same safeguards error (`req_011Cgz3SoGq29HpYbSn4jQww`) before writing `settings.md`. Missing now: `search.md`, `settings.md`. Owner asked «можешь запустить на sonnet 4.6?». Claude Code accepts a full model id in an agent definition's `model:` field (changelog: "agents now accept the same model values as `--model`"), so the coordinator created `~/.claude/agents/executor-sonnet-4-6.md` (= `~/Dev/ai/agents/`, untracked there, not committed): a copy of `executor.md` with `model: claude-sonnet-4-6`. The running session does not see new agent types ("Agent type 'executor-sonnet-4-6' not found"); it becomes available after Claude Code is restarted. Resume after restart: spawn `executor-sonnet-4-6` (no `model` parameter) for `search.md` and `settings.md` with the same briefs as the other writers (brief text: see the spawn prompts of collection.md etc.; scratchpad files may be gone after restart — check `/private/tmp/claude-501/-Volumes-SSD-Dev-tknff-sifty/92c828a0-c188-40b2-b55c-12af52f00517/scratchpad/specs/`).

## Coordinator, 2026-11-13: progress

- `executor-sonnet-4-6` became available in the running session after all; `search.md` and `settings.md` relaunched on it.
- `collection.md` complete (200 lines, all sections) although its Sonnet 5.5 writer was stopped by safeguards afterwards (`req_011Cgz3ZfUD9ibrqVs9jxhQR`).
- `inventory.md` was cut off after 18 of 45 rows by a classifier stop; the same agent was asked to finish in small appends.
- Still running: triage, application (Sonnet 5.5), inventory, search, settings (Sonnet 4.6).

## Coordinator, 2026-11-13: application.md and inventory.md returned

- `application.md` complete. Writer's main findings (from reading code, nothing run): answer generation cannot run — `backend/cmd/sifty/serve.go:154` passes `nil` as the inference function to `NewPreparationWorker`, which has no default (`backend/internal/background/preparation.go:61`, coordinator verified both lines); PATCH /application ignores the edited answer; /remember cannot work as the UI calls it; "Assess now" does nothing while the queue switch is off but says "queued"; field-name mismatches between UI and API (job_id vs jobId); the assessment request omits the evidence bank its prompt refers to. Production revisions: 63 "Inspection access failure", 55 "Questionnaire inspected", 2 "Model draft prepared" (coordinator query, read-only).
- `inventory.md` complete: 45 rows. Cancel candidates: INBOX-1, COL-7, SET-4, INBOX-3, DESIGN-2. Re-plan/stale: DESIGN-1, GO-5, GO-6, GO-7, GO-9, ASM-1, SET-1, APP-3, COL-2, COL-4, OPS-8, SETTINGS-5, main copy of INBOX-2 plan. Eight review cards are in main and production but say otherwise (APP-8, ASM-6, COLLECTION-1, DESIGN-3, SETTINGS-1/2/3, TRIAGE-2); closing them needs the OPS-9 item 4 path. GO-7 step 1.2 (purge applied roles after 30 days) conflicts with TRIAGE-1.
- Still running: triage (Sonnet 5.5), search and settings (Sonnet 4.6). ASSESSMENT-10 executor (Sonnet 4.6) running in its own worktree.

## Coordinator, 2026-11-13: triage.md returned

`triage.md` complete (12 behaviour sections). Writer's contradictions: screens are Inbox / Shortlist / Archive since DESIGN-3 while MAP, NAVIGATION-1 and APP-8 say Triage / Pre-selected / Preparation; the model assessment queue enqueues every vacancy every 30 s while enabled (MAP says after Select); "Block company" writes block_rules that the UI can neither show nor remove (15 in production); "Applied removed after 30 days" is promised but PurgeArchive is never called (GO-7); delete's "collection will not restore it" relies on a URL-only tombstone; workflow_history is written, never read, and deleted with the vacancy (matters for TRIAGE-1). Still running: search and settings (Sonnet 4.6); ASSESSMENT-10 executor.

## Owner, 2026-11-13, on application preparation

Owner: «Подготовка ответов к отклику в production не работает. - нормально никогда и не работала - не была нормально спроектирована. скорее была заглушка.»

Consequence for `application.md`: its status line and purpose must say that answer preparation is a stub that was never designed, not a broken feature; its defects are recorded as the state of the stub, and no repair card is proposed. A real design of application preparation would be a new card when the owner wants it.

## Coordinator, 2026-11-13: search.md returned

`search.md` complete (360 lines, Sonnet 4.6). Contradictions: `store/jobs.go` List still sorts by the model assessment score on the server, overridden at once by the client's `sortCatalogue` (no visible effect, live code); the expressions editor says changes "apply to the next collection" without saying they never filter the visible list. Overlap to resolve at review: its section 13 (Archive) duplicates triage.md section 9. Still running: settings (Sonnet 4.6); ASSESSMENT-10 executor.

## Coordinator, 2026-11-13: settings.md returned; framing error

`settings.md` complete (17 sections, Sonnet 4.6); its writer did not read the SETTINGS-4/5/6/9, SET-1, OPS-7, SETTINGS-7/8 cards. Contradictions: the Advanced save notice names `.local/` but backups go to `config-backups/`; the `matching` stage is settable only through raw YAML; the status bar has CSS but no component. All seven files now exist in the scratchpad.

Owner asked «ты понимаешь значение слова specs для функционалности?». Coordinator's answer: the brief it wrote asked the agents to describe what the code does ("Written from the code of main"), which yields as-built descriptions, not specifications; a specification states how a feature must behave, is agreed by the owner and is the reference the code is checked against. The seven files are material for the specs, not specs. Proposed rework, awaiting the owner's confirmation: per area, rules of intended behaviour from the owner's decisions in the cards; where only the code decides, a question to the owner; each rule marked implemented / not implemented / implemented with a deviation (code reference only as evidence); never-designed features (application preparation) marked "not designed". Do not move the seven files into `docs/specs/` as they are.

## Owner, 2026-11-13: rework to feature specifications

Owner: «да, переделывай. Еще момент - спеки мы пишем для фичей, не для задач. Т.е. например для "оценка fit" или "сбор вакансий" или "отображение информации о вакансии", или "подготовка отклика". Т.е. по сути скорее для эпиков. Не для кнопок или транзакций»

Coordinator's feature list: collection (`collection.md`), catalogue and search (`catalogue.md`), triage and decisions (`triage.md`), vacancy view (update `vacancy-view.md`), fit assessment (`quick-match.md`, not rewritten now), application preparation (`application.md`, "not designed"), candidate profile (update `profiles.md` with what is implemented today), local data (`data.md`). No settings spec: sources go to collection, search expressions to catalogue, models to fit assessment and application preparation. The seven as-built files are kept as material in `as-built/` in this folder. Common brief: `brief-specs.md`. Writers: `executor-sonnet-4-6`, output in the scratchpad `specs-v2/`.

## 2026-11-13 — triage spec draft returned

Writer `executor-sonnet-4-6` (a708d07df43c2f369) wrote `specs-v2/triage.md` (copied into this folder from the scratchpad). 65 rules: 58 implemented, 4 not implemented, 4 deviates, 1 owner decision needed (two rules carry two tags). Its questions for the owner: (1) canonical screen names Inbox/Shortlist/Archive versus Triage/Preparation/Archive; (2) should the model assessment queue take only Shortlist vacancies instead of every vacancy every 30 s; (3) GO-7 archive purge versus TRIAGE-1 recognising past applications — keep an application ledger before purge, or scope TRIAGE-1 to unpurged vacancies; (4) "Block company": stay Inbox-only and reversible, or move to the unified exclusion list of NAVIGATION-2/3; (5) a management screen for company blocks (15 in production), and where. Coordinator review against owner decisions pending (S.3).

## 2026-11-13 — local data spec draft returned

Writer `executor-sonnet-4-6` (a6b0dd3567a79a2b1) wrote `specs-v2/data.md` (copied here). 29 rules: 22 implemented, 4 not implemented (OPS-7, OPS-8 ×2, GO-6), 3 owner decision needed, 0 deviates. Its questions: (1) a command to initialise a new data directory from repository templates, or a documented manual copy; (2) applied-vacancy purge — wire the 30-day function, make the period configurable, or remove the dead function (overlaps triage question 3, GO-7 versus TRIAGE-1); (3) should the deploy snapshot also cover owner files (config/, answers/, feedback/, evidence/, profile photo) so a rollback restores everything; (4) moving the data directory to another machine — tested manual procedure or an export/import command. Coordinator review pending (S.3).

## 2026-11-13 — application preparation spec draft returned

Writer `executor-sonnet-4-6` (aeb959f878a821004) wrote `specs-v2/application.md` (copied here). 21 rules: 7 implemented, 2 deviate (evidence bank omitted from the assessment request; field edits not saved), 10 not implemented (ASM-1, APP-3 ×3, APP-4 ×2, APP-5, APP-6, APP-7, APP-8 unreleased), 2 owner decision needed (assessment queue scope; shortlist re-run on a stronger model). Its nine questions: answers store (existing answers store, Obsidian kb/, new); what is prepared first (fixed common list, questions seen in inspected forms, both); who writes the first answer (model drafts and owner confirms, or owner writes); answer scope and matching (model or exact wording); sensitive answers with unresolved facts (draft or blank); when drafting starts (tab open, button, background — decisions record and APP-3 disagree); shape of the "think" dialogue; queue limited to shortlisted vacancies (same as triage question 2); shortlist re-run automatic or on request and default model (ASSESSMENT-6). Coordinator check pending (S.3): the feature is "not designed" per the owner, so the review must confirm the draft lists only decided rules and does not invent a design.

## 2026-11-13 — catalogue and search spec draft returned

Writer `executor-sonnet-4-6` (adfbbadec6496cfd9) wrote `specs-v2/catalogue.md` (copied here). 73 rules: 69 implemented, 2 not implemented (INBOX-2 L.4c row placeholder, L.4e poll without refetching the list), 2 deviate (domain block kind without a creation path in the interface; 30-day archive deletion never called, GO-7). Its questions: (1) domain blocks — expose next to company blocks, keep code-only, or remove; (2) Settings → Search after SETTINGS-5 moves search expressions to the profile — remove, redirect, or note; (3) the 2.5 s full-list poll (~35 MB) — wait for INBOX-2 L.4e or fix separately; (4) catalogue text search is client-side substring only — permanent, or full-text search still a goal. Coordinator check pending (S.3): 73 rules is high for a feature-level spec — check that rules are outcomes, not screen elements; check whether the server-side sort by model score (found in the as-built round) is covered.

## 2026-11-13 — candidate profile spec update returned

Writer `executor-sonnet-4-6` (ac7ebc4735fa1c274) wrote `specs-v2/profiles.md` (copied here; update of `docs/specs/profiles.md`). 60 rules: 33 implemented, 12 not implemented (SETTINGS-5), 5 not implemented (INBOX-2 L.5r6, "Confirmed by you"), 8 deviate, 2 untagged cross-references to quick-match. Its questions: (1) SETTINGS-9 geography "not set" — fix standalone now or inside SETTINGS-5 S.1 (the inventory already proposes S.1); (2) SETTINGS-4 website and social URL sources — still planned, after SETTINGS-5?; (3) "Confirmed by you" facts — My data (shared by all profiles) or per profile; the draft assumes My data, no card decides it. Coordinator check pending (S.3): every owner decision of the existing `profiles.md` must survive (diff against `docs/specs/profiles.md`); 8 deviations to be listed.

## 2026-11-13 — collection spec draft returned

Writer `executor-sonnet-4-6` (ad9188ab46e118fd2) wrote `specs-v2/collection.md` (copied here). 77 rules: 61 implemented, 6 not implemented (INBOX-2 ×4: stored original, structured form, plain text from it, Remote3 category guard; COL-2 per-source rules and repair; COL-4 per-source run events), 9 deviate, 1 owner decision needed. Its questions: (1) show a failing or empty source during a run, and where (MAP and search.yaml say yes, code silent, COL-4 plans it); (2) do search expressions filter what is stored at collection or only search and ranking (Search page and search.yaml say collection; code does not use them there; relates to SETTINGS-5 S.2); (3) "Collect jobs" when collection is disabled in runtime.yaml says "Collection started." — report it specifically; (4) a run where every source failed or returned nothing — "failed" instead of "complete"; (5) does a manual or on-open run count as today's run for the daily scheduler. Coordinator check pending (S.3): 77 rules, check feature level; COLLECTION-2 (Remotive category pages) must appear among open cards.

## 2026-11-13 — vacancy view spec update returned (all seven drafts in)

Writer `executor-sonnet-4-6` (a28191a932455e115) wrote `specs-v2/vacancy-view.md` (copied here; update of `docs/specs/vacancy-view.md`). 25 rules: 1 implemented (header with actions), 24 not implemented — INBOX-2 (on its branch, unreleased), 0 deviate, 0 owner decision. Changes it reports: draft status format, ✓ routing rule (skill name ≤ 3 words without -ing → Skills, otherwise → Confirmed by you), three-state money verdict, HTML full listing, unified money function, sections 4 and 6 added, mockup B preamble dropped, owner quotes kept. Its questions: (1) money verdict — three states (✓ lower bound meets minimum, ? minimum inside range, ✕ upper bound below minimum) as built on the INBOX-2 branch, or ✓/✕ only as in the earlier spec; (2) Location tile verdict automatic once the profile has a location goal, or a separate decision; (3) "Confirmed by you" list — stay in the Skills section next to "Skills you do not have" or move to About me (the INBOX-2 L.5r6 text says About me).

All seven drafts are in `specs-v2/`. Next: S.3 coordinator review against owner decisions, then one combined question list for the owner.

## 2026-11-13 — S.3 review, round 1: application, vacancy view, profile

Corrections sent back to the same writers (SendMessage), files to be re-copied when they return.
- **profiles.md** (ac7ebc4735fa1c274): the draft lost agreed decisions of `docs/specs/profiles.md` — computed facts in My data, dealbreakers, search editor on the profile page and sources in Settings, one fetch per source for all profiles, shared vacancy extraction, You fit per profile, staleness per profile versus shared facts, worker order by the active profile's toggle, Search section line and chips, archiving keeps shared data, the out-of-scope list. Seven "Today …" rules tagged [deviates] only restate that SETTINGS-5 is unbuilt — to be folded into [not implemented] tags. Its question 3 ("Confirmed by you" shared or per profile) is already decided by the old spec's My data table. "Four states" lists five. Experience match wording changed — to be checked against code. S.6 tag wrong (S.4). First profile name is the owner's choice, not "Blockchain".
- **vacancy-view.md** (a28191a932455e115): canvas link dropped; Location rule still waits for SETTINGS-8 and ASSESSMENT-8, both released — the old spec's condition for a Location verdict holds, so the rule becomes "the tile shows the geography verdict", with a confirmation for the owner instead of an open question; question 3 already decided (About me group); rule 21 contradicts itself; "day × 260" and the three-state money verdict are branch additions beyond the agreed text — one question; fill-rate rule dropped; scope names a settings spec that no longer exists; config keys to be given by YAML name (`max_skill_name_words`, `verb_endings` in `skills.default.yaml` on the INBOX-2 branch — they are configuration, not code).
- **application.md** (aeb959f878a821004): as-built link points at itself; APP-8 is in main since 2026-11-10 and released with every later production build, not "pending release"; sibling links; rule 5 (evidence bank omitted from the assessment request) has no card — the coordinator proposes one to the owner.

## 2026-11-13 — S.3 review, round 2: triage, data, catalogue, collection; application corrected

- **application.md** returned with the round-1 corrections (as-built link, APP-8 released, sibling links, queue question cross-referenced); counts unchanged (7 / 2 / 10 / 2). Re-copied.
- **triage.md** (a708d07df43c2f369): too close to screens and storage — rewrite at outcome level (~30–40 rules): no storage values or table names, keyboard as one or two rules (layout-independent shortcuts: Y done, N and A deviate), screen details dropped, configuration table without database tables; catalogue spec is `catalogue.md`; purge becomes [owner decision needed] here (single place for GO-7 versus TRIAGE-1, options: keep an application record before purge / never purge / scope TRIAGE-1); "Unknown" block folded into the block-management question; tab-name question carries the coordinator's recommendation (Inbox / Shortlist / Archive canonical, came with DESIGN-3 which is in production).
- **data.md** (a6b0dd3567a79a2b1): purge rule points to triage; initialise-new-directory and move-to-another-machine merged into one question (coordinator recommends a documented manual procedure); path resolution moved to configuration; sibling links.
- **catalogue.md** (adfbbadec6496cfd9): factual error — it says search expressions filter at collection; the only callers of `BuildFTSQueryExprs` / `BuildFTSQuery` are `backend/internal/profile/ranking.go:216,228` (ranking of stored vacancies), so collection.md rule 71 is right. Also `config.go:30` says groups combine with OR, the draft says AND — to verify. Cut screen detail (~25–35 rules), move keyboard / counts / Archive / blocks / purge to triage and collection-time exclusion to collection; check the server-side order by old model score. Questions: poll question dropped (INBOX-2 L.4e decided), Settings → Search question dropped (agreed profiles spec keeps stop words, sources, providers, schedule in Settings), domain blocks folded into triage's block question; full-text search question kept.
- **collection.md** (ad9188ab46e118fd2): rules tagged [deviates] describe the code instead of the intent (32, 34, 52, 62, 74–76) — rewrite intent first; 64 is implemented, not a deviation; condense adapters and fetch safety (~40–50 rules); sibling links; COLLECTION-1 is in production since 2026-11-10 (card in review until OPS-9); run-feedback questions merged into one (coordinator: yes, one card); runtime.yaml keys parsed without effect become a question.

- **data.md** corrected (a6b0dd3567a79a2b1): 27 rules — 20 implemented, 4 not implemented (OPS-7, OPS-8 ×2, GO-6), 2 owner decision, 1 pointer to triage. Questions: fresh start / move to another machine (coordinator recommends a documented manual procedure); deploy snapshot to cover owner files. Re-copied.

- **vacancy-view.md** corrected (a28191a932455e115): 27 rules — 1 implemented, 25 not implemented on the INBOX-2 branch, 1 not implemented without a card (Location tile geography verdict; branch `band.go` leaves the verdict empty). Questions: accept the branch additions to money (? state, day × 260); confirm the Location tile shows the geography verdict. Re-copied. A card for the Location verdict is to be proposed to the owner.

- **profiles.md** corrected (ac7ebc4735fa1c274): 60 rules — 32 implemented, 20 not implemented (SETTINGS-5), 5 not implemented (INBOX-2 L.5r6), 1 deviates (SETTINGS-9), 2 cross-references. Lost decisions restored (checked by search: dealbreakers, computed facts, shared extraction, archive keeps shared data, worker order, out-of-scope). Questions: SETTINGS-9 inside SETTINGS-5 S.1 (coordinator recommends); SETTINGS-4 still wanted and after SETTINGS-5. Re-copied.

- **triage.md** corrected (a708d07df43c2f369): 38 rules (was 65) — 30 implemented, 5 deviate, 4 not implemented, 1 owner decision (purge). Questions: tab names (coordinator: Inbox / Shortlist / Archive canonical); assessment queue scope; purge versus past-application history (a/b/c); Block company narrow or unified exclusion; block management screen and the "Unknown" guard. Re-copied.

## Owner, 2026-11-13: answers to the collection questions

Questions as relayed in chat: (1) show a failed or empty source, and where; (2) do search expressions filter collection or only search and ranking; (3) report "Collect jobs" when collection is disabled; (4) a run where every source failed → "failed"; (5) does a manual or on-open run count as today's run for the daily scheduler.

Owner, verbatim:

«1. Показывать ли в интерфейсе, что источник упал или ничего не вернул, и если да, то где? - информационным сообщением (полагаю полосато-оранжевым?) в inbox/shortlist и также на странице settings - в списке источников - каким то символом и цветом?
2. Fetch все вакансии источника, затем поиск исходя из поисковых запросов в профиле пользователя (сейчас они еще пока в settings, но запланирован перенос в профиль).
3. У нас нет toggle "сбор включен/выключен" - можно его добавить на страницу источников в настройках.
4. да. и выдавать сообщение.
5. что за "ручной" сбор? приложение запускает (должно) сбор при обновлении страницы - ЕСЛИ сбор еще не запущен или не был завершен в ближайшие 3 часа (ну не чаще чем в 3 часа можно запускать новый сбор). МОжно добавить в настройки, в "источники" как часто собирать вакансии.»

Coordinator's reading for the spec: (1) a failed or empty source is shown as an information notice in Inbox and Shortlist and as a mark with a colour next to the source in Settings → Sources; (2) collection fetches every posting of a source and stores it; the active profile's search expressions then decide what the profile's Inbox shows (matches the agreed profiles spec: a vacancy appears in the Inbox of the profile whose search found it) — today the expressions only rank; (3) a collection on/off switch on the Sources page (today only `collection_disabled` in runtime.yaml); (4) a run where every source failed or returned nothing is "failed" and the owner gets a message; (5) collection starts on page load when no run is in progress and none completed within the interval (default 3 h); the interval is set on the Sources page. Open: the owner asked what "manual" collection is — it is the "Collect jobs" button in Settings → Sources, which starts a run at once, skipping the 3 h pause; whether to keep it and the daily 09:00 scheduler (which runs without the page open) goes into the combined question list.

- **catalogue.md** corrected (adfbbadec6496cfd9): 24 rules (was 73) — 15 implemented, 3 implemented with known limits, 2 not implemented (INBOX-2 L.4c, L.4e), 1 not implemented (SETTINGS-5: profile expressions decide the Inbox, owner answer quoted), 3 deviate. Key finding: in production the search expressions have no visible effect at all — the BM25 ranking that uses them writes a legacy score field and is called only by the testable collection function, not by the production run; the catalogue is sorted by Fits you; the screen caption "apply to the next collection run" is wrong. Server still orders by the old model score; the client overrides it. One question left: in-list text search client-side only, or database-backed search. Re-copied.

- **collection.md** corrected (ad9188ab46e118fd2): 51 rules (was 77) — 36 implemented, 10 not implemented (INBOX-2 ×3, COL-2, COL-4, SETTINGS-5, no card ×4: the owner-decided run feedback, on/off switch, failed status, interval setting), 5 deviate. Owner answers folded in with quotes. Questions left: keep "Collect jobs"; keep the daily 09:00 scheduler or replace by the interval rule; unused runtime.yaml keys — remove or wire. Re-copied.

All seven drafts reviewed and corrected. Next: one combined question list for the owner (end of S.3).

## 2026-11-13 — combined questions

`questions.md`: 14 questions with coordinator recommendations, application design questions deferred to a design session, four findings without a card. Shown to the owner in chat.

## Owner, 2026-11-13: answers to the combined questions

Owner: «14 вопросов - согласен со всеми твоими предлжоениями. но по 7 - (б) - тогда нужно добавить возможность удалить сообщения в shortlist и archive. Кнопка?»

All 14 recommendations in `questions.md` are accepted as the owner's decisions (the recommendations were written by the coordinator; the owner approved them). Question 7: (b) — applied vacancies are never deleted automatically; the 30-day promise is removed. The owner asks for a way to delete vacancies in Shortlist and Archive: it exists already — the "Delete role…" button in Shortlist and both Archive tabs (`frontend/main.tsx:168`, confirmation dialog `frontend/DeleteRole.tsx`), permanent, with a tombstone. Told to the owner; noted that deleting an applied vacancy also deletes the record of that application.

## 2026-11-13 — S.3 closed; S.4 scope decided by the coordinator

S.3 ticked: seven drafts reviewed, corrected by their writers, owner answered the combined questions. S.4 scope (coordinator's technical decision): `collection.md`, `catalogue.md`, `triage.md`, `application.md`, `data.md` go into `docs/specs/`, `profiles.md` replaces the old one. `vacancy-view.md` stays in this card's `specs-v2/` until INBOX-2 is released: the INBOX-2 branch edits `docs/specs/vacancy-view.md`, its MAP row, and adds `docs/specs/listing-view.md` (stored description forms, overlapping collection.md rules on stored forms); replacing the file now would collide with the live INBOX-2 session. After the INBOX-2 release, a short follow-up reconciles vacancy-view.md and listing-view.md with collection.md.

## 2026-11-13 — S.4 complete (executor, work/OPS-11)

**What changed.** All 14 owner decisions of 2026-11-13 applied to `specs-v2/` drafts: each answered question left §5 and updated or added a rule with an owner attribution tag; Q7 uses the owner's words «(б) — applied vacancies are never deleted automatically; the 30-day promise is removed.»; canonical screen names Inbox / Shortlist / Archive applied throughout. Five specs and updated profiles.md published to `docs/specs/`. Main merged before MAP edits (commit 02c6d64 added the OPS-11 task-index row; no MAP conflict). `docs/MAP.md` updated: 7 spec table rows (5 new + Profiles + Vacancy view unchanged), user-outcome prose in R-COL, R-SEARCH, R-TRI, R-APP, R-DATA replaced with spec links, R-TRI gains the canonical screen names sentence, 24 card-row spec links added. SETTINGS-5 description.md and plan.md "deferred" header replaced with approved line; SETTINGS-9 added to plan.md §10 S.1.

**Commits.** d903bbf (apply owner decisions to specs-v2), 0076aeb (publish specs to docs/specs/), 1653c8d (MAP updates), ef589c1 (SETTINGS-5 card header correction).

**map-check.** `OK: 5 active document(s), 14 requirement(s), 83 canonical card(s), 2 legacy card(s)`

**Not placed.** vacancy-view.md stays in specs-v2/ (INBOX-2 live session owns docs/specs/vacancy-view.md). Q9 and Q10 from combined questions (money verdict, Location tile) are about vacancy-view.md and are deferred with it. COL-7 has "none needed" in inventory's covering spec column, so no collection.md link was added to its MAP row.

## 2026-11-13 — coordinator check of S.4

Executor (a68229ef95e0761eb) commits `d903bbf`, `0076aeb`, `1653c8d`, `ef589c1`, `f6b3445` plus merge of main `e4facb7`. Checked: `bin/map-check` OK; six status lines set; no "[owner decision needed]" left except one design question in application.md; triage rule 25 applies option (b). Corrected by the coordinator: rule 25 put a paraphrase in the owner's quotation marks — replaced by the owner's actual words; the open-cards line for GO-7 still said it wires the purge — now says step 1.2 is superseded. Not yet placed: decisions 9 and 10 (money "?" state and day rate; Location tile verdict) belong to vacancy-view.md, which waits for the INBOX-2 release; GO-7 card itself still plans the purge — to be re-planned.

## Owner, 2026-11-13: release and cards

Owner: «да и да» — yes to releasing OPS-11 and yes to capturing the proposed cards. Captured on main: COLLECTION-4 (collection run feedback), TRIAGE-3 (remove the 30-day deletion promise; GO-7 step 1.2 superseded), TRIAGE-4 (N and A keys, "Not suitable" reason, search expressions caption), INBOX-4 (Location tile geography verdict, after INBOX-2); indexed in MAP (main `c0ca700`). Release by the S.4 executor: merge main, `bin/work-finish` with release.
