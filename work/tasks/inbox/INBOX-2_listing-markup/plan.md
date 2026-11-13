# INBOX-2 — Listing text keeps its markup: plan (coordinator, 2026-11-11)

## What is wrong today (measured on data-staging, 2 780 vacancies)

- Every source turns the page into plain text through one function, `collect.CleanHTML` (`backend/internal/collect/text.go`): headings, list items and numbering are lost; only a newline before some block tags survives. Feeds and JSON-LD (`collect/parse.go`) go through the same `text()`.
- Some sources bypass it and store raw HTML in `jobs.description`: 161 vacancies contain `<p>`, `<li>` or `<br>` (e.g. LinkedIn via jobspy, web3.career detail pages). The Overview shows these tags literally.
- 126 vacancies contain `&amp;` and similar entities; some JSON-LD texts arrive glued without any newline ("About UsParity is…").
- About 2 100 have newlines, which the current `ListingText` renders as paragraphs and bullets — readable but without headings or numbering.
- `snapshots` keep only the normalised record, not the page, so the original markup of existing vacancies cannot be recovered without fetching the page again.

## Outcome

The Overview › Original shows the listing with its headings, bulleted and numbered lists and paragraphs, styled by the page CSS; no tags or entities are visible. Then, optionally, By section groups the same text under its headings.

## Decisions to keep

- **Two forms of one text.** A new stored form with structure (`jobs.description_html`, a sanitised subset: h2–h4, p, ul, ol, li, strong, em, br; everything else unwrapped, no attributes, no scripts, no links unless decided otherwise) and the existing plain `jobs.description`, derived from it by one function. Plain text stays the input of search (FTS), Quick Match facts and the ASSESSMENT-3 model passes.
- **No silent re-analysis.** For vacancies that exist today, `description` stays byte-identical unless the page is fetched again; otherwise every description hash changes and the model re-reads ~2 800 vacancies (~13 h). The structured form for them is built locally from what is stored.
- **Separation.** Conversion and sanitising live in the collect layer (one place, used by every source); storage in the store; rendering in a presentation component that maps the allowed tags to the page's existing typography classes. No inline styles, no `dangerouslySetInnerHTML` of unsanitised data.
- **Starts after ASSESSMENT-3 is merged** (both touch `RoleContent.tsx` and the description path); cut the worktree from `main` then.

## Steps

- [x] **L.1 — Inventory per source** (30–40 min). For each source in `search.yaml`, three stored samples: raw form (HTML / escaped HTML / plain with newlines / glued), entities, what the page actually offers (fetch at most 3 pages per source, politely). Table in the card. Check: coordinator reviews the table; it decides L.3.
- [x] **L.2 — Structured form at collection** (60–90 min). One converter `collect.StructuredHTML` (sanitised subset above) plus `collect.PlainFromStructured` (the plain text, entities decoded, blocks separated by blank lines, list items as "- " / "1. " lines). All adapters, feeds and JSON-LD use them; jobspy/LinkedIn HTML goes through them too. Additive migration for `description_html`. One targeted test for the sanitiser (scripts, attributes, event handlers removed) and one for the plain derivation. Check: a fresh collection on the review copy (collection enabled for this step only, one or two sources) stores both forms.
- [x] **L.2b — Collection defects found in L.1** (new, 45–75 min): Remote3, WWR, Cloudflare-403, Web3 Careers Discover fix. Done: see log.md 2026-11-11.
- [x] **L.2c — Keep the original** (20–30 min, owner agreed 2026-11-11, spec §15.2.1). Stored as received, never rendered, never returned by the API.
  - [x] L.2c.1 Additive migration v24: table `job_originals(job_id PRIMARY KEY, kind, fetched_at, body BLOB gzip)`; store methods to save, read and replace one row.
  - [x] L.2c.2 Adapters, feeds, JSON-LD and jobspy pass the body they already hold (page HTML or feed item) in the job input; `Upsert` writes it in the same transaction as the vacancy. No extra fetch for any source.
  - [x] L.2c.3 Targeted test: the original is written with the vacancy in one transaction, a repeated collection of an unchanged vacancy does not add a second row, and no API response carries it.
- [x] **L.3 — Existing vacancies, locally** (40–60 min, plus refetch 20–30 min and polish 10–15 min). Stage ends here; report «готово к приёмке».
  - [x] L.3.1 Local backfill, idempotent, no network: stored raw or escaped HTML → structured form through the converter; plain text with newlines → structured form from lines (bullet lines → ul, "1." lines → ol, short lines ending with ":" → h3); glued text → one paragraph and a `needs_refetch` mark (additive column). `description` is not touched for any of them.
  - [x] L.3.2 Refetch path: Refresh and opening a vacancy marked `needs_refetch` fetch through the source adapter (existing rate limits, one source at a time) and write description, `description_html` and the original together; this path, and only this path, may replace `description` of a row that has no `description_html`. Collection keeps the guard.
  - [x] L.3.3 Web3 Careers: no trailing "Apply Now:"; the converter never emits an empty paragraph (first or any).
  - [x] L.3.4 Targeted tests: `description` of existing rows is byte-identical after the backfill (hash compare); the sanitiser removes scripts and attributes; the backfill and the guard cause no fetch; the guard is bypassed only by the refetch path.
  - [x] L.3.5 Live collection of Remote3 and We Work Remotely into a throwaway database (not `data` or `data-staging`): descriptions are vacancy text, `description_html` and the original are stored. Look at the stored text, not only at counts.
  - [x] L.3.6 On a copy of `data-staging/store.sqlite` (scratchpad, never the live file): run the backfill, then refetch the 59 damaged Web3 Careers and the 40 Remote3 vacancies politely. Report counts per kind before and after, hashes of unchanged rows, refetch outcomes. The live bases are changed only by the release.
- [x] **L.4 — Original view** (30–45 min, plus Playwright setup 10–15 min). Browser checks are made with Playwright (headless Chromium, `npx playwright`, version 1.65.0, browsers already cached) against a throwaway server on port 9199 with a fresh copy of `data-staging/store.sqlite`; screenshots go to `screenshots/` in this card folder (owner, 2026-11-12).
  - [x] L.4.1 Frontend type carries `descriptionHtml`; `ListingText` renders it through a whitelist renderer (React elements built from a parsed tree, never `innerHTML` / `dangerouslySetInnerHTML`): h2–h4, p, ul, ol, li, strong, em, br, everything else dropped; uses the existing listing-text typography (`theme.css`), headings and the numbered list get the matching styles; falls back to the current line renderer when `descriptionHtml` is absent.
  - [x] L.4.2 Playwright script (kept in the card folder as `screenshots/shoot-L4.mjs`) that starts nothing itself: the coordinator's brief names the server command; it opens Overview › Original and saves screenshots for: Parity; a LinkedIn vacancy with HTML; a vacancy with a numbered list; a Web3 Careers vacancy refetched in this check (marked `needs_refetch` before opening, mark cleared after, one polite request); a vacancy without `description_html` (fallback); a narrow phone-width view of one of them. The script also asserts in the page that no `<script>`, inline `style` attribute or visible tag text appears inside the listing text.
  - [x] L.4.3 Look at every screenshot and write in the log what is visible (headings, lists, numbering), not that a file exists.
- [x] **L.4b — Sticky "Failed to fetch" banner (INBOX-3, owner agreed)** (20–30 min). Per INBOX-3 diagnosis: an error raised by the background poll clears on the next successful poll; network failure to the own server is classified in `frontend/messages.ts` as Retry with the text "Sifty server is not responding. Retrying…"; errors from the owner's actions unchanged. Check with Playwright (screenshots in `screenshots/`): stop and start the throwaway server, the banner appears and disappears by itself.
- [x] **L.4c — Catalogue row placeholder** (10–15 min, coordinator, 2026-11-11, from the owner's screenshot after the ASSESSMENT-3 release). With matching off, a row whose vacancy waits for extraction shows two dashes ("– –"): the second is a leftover You fit placeholder. Show one placeholder for Fits you only; a row without usable text keeps its single "–". Root cause found on production data: these rows have no stored two-score result at all (`fit` is null, so `youFitOff` is false and two cells are drawn); they were collected at 15:14 after the startup rescoring at 15:13 and get a result only when the worker reaches them. Fix both: (a) the row draws one cell when there is no result; (b) collection rescoring: after a collection run, compute the two-score result locally (no model) for the new vacancies at once, so Fits you appears immediately and the worker only refines it. Owner's words: «переключился в newest - и спустя 30 секунд - все еще ничего.», «обновляются только вторые цифры, но не первые». +15–20 min. Check with Playwright on a fresh copy of production data (`data/store.sqlite`, copied, never the live file): Newest, first rows, screenshots in `screenshots/`.
- [x] **R.1 — Release step: remove the 40 Remote3 category rows** (15–20 min; owner, 2026-11-12: «удалить 40 строк Remote3, вакансию id 7 оставить, штатным удалением, при выпуске; сейчас живые базы не трогать»). The rows are category pages (`original_url` like `https://remote3.co/jobs/%`), in both live bases ids 1376–1415, all in Triage; id 7 is a real vacancy at `/remote-jobs/…` and is not selected.
  - [x] R.1.1 `release-steps/remote3_cleanup.py` (Python standard library): selects the rows read-only from `store.sqlite` by the URL pattern, prints id, title and URL (dry run is the default), and with `--apply --port N` removes each one through the running server's own API: move to rejected, then `DELETE /api/jobs/{id}` with `{"confirmed":true}`, so tombstones and cleanup run. Refuses to run when the selected count differs from `--expect` (default 40), when the pattern would match a `/remote-jobs/` row, or when the server reports active collection. Stops on the first error and says which row.
  - [x] R.1.2 Tested only on a fresh copy of `data-staging/store.sqlite` and a throwaway server on port 9199: dry run lists 40, apply removes 40, id 7 remains, a second run finds 0, tombstones exist. Result in the log.
  - [x] R.1.3 `release-steps/README.md` in this card folder: when to run it (after the release has started the new server, before the first collection), the exact commands for production (port 8080, data `data`) and staging (port 8081, data `data-staging`), how to check the result, and that the step needs the release backup to exist. The live bases are not touched in this card.
- [x] **L.4d — Server unreachable by failure type** (15–20 min; owner, 2026-11-12: «определять по типу сбоя (TypeError), а не по тексту Chrome — Safari «Load failed», Firefox «NetworkError…»»). In `frontend/api.ts` a rejected `fetch` (network failure to the own server) becomes one dedicated error type; `messages.ts` and `main.tsx` classify by that type, not by the browser's message text; a TypeError raised by the page's own code is not mistaken for it. Targeted node test with the three browser wordings.
- [x] **L.5 — Sections, section classes, headings from bold text (required; owner, 2026-11-12)** (75–105 min). One Go splitter for the view and later for the extraction. Presentation of the plain `description` and every ASSESSMENT-3 read stay unchanged.
  - [x] L.5.1 Go normaliser of the structured form, used by the converter: a paragraph that is only bold text (LinkedIn «Key Responsibilities») and a leading bold run followed by text (Web3 Careers «About CoinList», glued «About Us» + text) become headings; double line breaks become paragraph breaks and empty paragraphs disappear (the extra gaps in Web3 Careers); idempotent; never touches `description`. Targeted tests on saved fragments of real pages.
  - [x] L.5.2 Section splitter (heading → section, in original order, text before the first heading is its own section) and class assignment from a heading vocabulary kept in configuration (About, Role, Responsibilities, Requirements, Nice to have, Compensation, Benefits, Location, Process, Legal/EEO; other), following the existing configuration convention; targeted test.
  - [x] L.5.3 Storage: additive, derived data, written with the vacancy in the same transaction; startup step for existing vacancies (re-normalise the stored structured form where the normaliser changes it, build sections), no network, idempotent. Check on a `.backup` copy of `data/store.sqlite`: `description` byte-identical in every row (hash compare), counts of rows changed and sections per class.
  - [x] L.5.4 API exposes `sections` (heading, class, content); `/api/jobs` payload is not made materially larger (measure before and after, prefer the single-job response if a list row would grow).
  - [x] L.5.5 Frontend: By section is enabled only when a listing has at least two headed sections, otherwise stays disabled with its tooltip; each section shows its class name (own heading text when the class is other), is collapsible; Original unchanged apart from the new headings.
  - [x] L.5.6 Spec of the listing view updated (find it under `docs/specs/`).
  - [x] L.5.7 Playwright on a fresh `.backup` copy of `data/store.sqlite`, screenshots in `screenshots/`: By section for three vacancies, LinkedIn «Key Responsibilities» as a heading in Original, Web3 Careers (CoinList) without the extra gaps, one vacancy with the toggle disabled, a collapsed section. Look at each and log what is visible.
- [x] **L.6 — Owner review on the review copy (8094)** (20–30 min, owner present). Done: owner accepted the vacancy view on 2026-11-13 («принимаю этот INBOX-2», with the + and − buttons) and confirmed the release («да, выпускай»).

## Acceptance criteria

Added by the coordinator on 2026-11-11 when work-handover refused the card for lacking this section; every line restates the Outcome or a step's Check above, nothing new is decided here.

- Overview › Original shows headings, bulleted and numbered lists and paragraphs in the page typography; no HTML tags or entities are visible (checked in the browser on Parity, a LinkedIn vacancy with HTML, one with a numbered list, one glued).
- A fresh collection stores both `description` and `description_html`; the sanitiser removes scripts, attributes and event handlers (targeted test).
- For vacancies that exist before the change, `description` is byte-identical (hashes unchanged) except those refetched on purpose; counts per kind are reported before and after.
- Every newly collected or refetched vacancy has its original stored as received; the view never renders it.
- The release step removes exactly the 40 Remote3 category rows through the standard deletion and leaves id 7; it is proven on a copy only.
- Search, Quick Match facts and the ASSESSMENT-3 passes still read the plain `description`.
- Without a stored `description_html` the view falls back to the current line renderer.
- If L.5 is built: By section is enabled only for listings with at least two headings, and sections are stored with the vacancy and exposed through the API.

## Estimate

Complexity 8. Required L.1–L.4 + L.6: about 3–4.5 h of AI time. Optional L.5: +45–75 min. Total with L.5: 4–5.5 h; with L.4b (INBOX-3) +20–30 min. No paid model or service; L.1 and L.2 fetch a few pages from job boards.

## Addition (coordinator, 2026-11-11): one section splitter for the view and the extraction

The ASSESSMENT-3 research found that the extraction also needs the text divided into sections (56 of 97 texts split by headings or label lines, 16 glued). There must be one splitter: L.5 builds sections in Go from the structured form, stores them with the vacancy (not only in the frontend) and exposes them through the API; the view uses them for By section. Feeding sections to the extraction is not part of this card; it is available for ASSESSMENT-5 or a later card. Estimate of L.5 unchanged except storing sections (+15–20 min).

## Owner decisions after L.1 (2026-11-11)

Owner: «Что делать с повреждёнными вакансиями Web3 Careers (59 штук) и с Remote3 (40)? - и загрузить сейчас, чтобы не мучаться, и сделать механизм загрузки при открытии вакансии.» On markdown: «здесь мне нужно больше информации. это про то, чтобы сделать какой-то общий для всех механизм?» — coordinator: yes, one converter for every input form (HTML, escaped HTML, markdown, plain lines) producing the same structured form; markdown is one more input branch of it. On We Work Remotely: «надо разобраться с дефектом».

Changes to the steps (added estimates in brackets):
- **L.2** — the converter accepts four input forms: HTML; entity-escaped HTML (decode once, then parse as HTML — the cause of the 161 literal-tag and 126 `&amp;` rows); markdown (Wellfound search API: headings, bold, lists, links → the same subset, links unwrapped to text); plain text with lines. Web3 Careers: read the job page body instead of its broken JSON-LD. (+20–30 min)
- **L.2b — Collection defects found in L.1** (new, 45–75 min): Remote3 resolves to a category page instead of the vacancy page — fix the resolver so the vacancy page is fetched; We Work Remotely stores the whole page body with page chrome — take the description from the RSS item (it carries headings and lists) and, where only the page body exists, keep only the job's content block; Cloudflare-blocked job pages are not retried. One targeted test per fix on a saved page in testdata.
- **L.3** — in addition to the local conversion: refetch now, politely (existing rate limits, one source at a time), the 59 damaged Web3 Careers vacancies and the 40 Remote3 vacancies (after L.2b); for We Work Remotely re-derive from the RSS where the item is still in the feed, otherwise cut the known chrome locally; keep the refetch-on-open mechanism for the rest. A refetched description changes its hash, so the ASSESSMENT-3 worker re-extracts those vacancies (about 100 × 11 s, local model, free) — expected, not a defect. (+20–30 min)

Revised total: required about 4.5–6 h of AI time, with L.5 5.5–7 h. Paid services: none; network: about 100 job pages plus the feeds.

## Addition after the owner's pipeline decision (2026-11-11, spec docs/specs/quick-match.md §15)

The owner agreed to add to this card («2 и 3 - согласен»):
- **L.2c — Keep the original** (20–30 min): store the fetched HTML / feed item of every newly collected vacancy as received (additive column or table, compressed if large), never rendered; used for re-parsing without a new fetch (§15.2.1, and later COL-2). Existing vacancies get it when refetched.
- **L.5 extended — section classes** (+20–30 min): the splitter also finds headings glued to the following text ("About UsParity…") and classifies each section by a heading vocabulary in configuration (About, Role, Responsibilities, Requirements, Nice to have, Compensation, Benefits, Location, Process, Legal/EEO; other); classes stored with the sections and returned by the API. By section shows the classes as section names. Routing sections to scoring is not part of this card (ASSESSMENT-5).
Per-source parsing rules and their repair are COL-2, not this card.
Revised total: required about 5–7 h, with L.5 6–8 h.

## Addition (coordinator, 2026-11-12): from OPS-10 H.4

- [x] **L.4e — Poll without refetching the catalogue** (30–45 min; named L.4d when added, renamed because L.4d is already the failure-type step above; to be done after L.5.7 and before L.6): the UI refetches the whole jobs list (~35 MB, ~2.7 s of server time) every 2.5 s, which keeps production almost permanently busy (measured in OPS-10 H.4). The poll keeps reading the queue and the worker status; the jobs list is fetched on load, on the owner's actions and when the server reports a change (a revision or last-change stamp on an existing cheap endpoint), not on every tick. In `frontend/api.ts` give requests a timeout; show `/api/health` 503 ("database not responding") through the existing Retry banner. Check on a copy: network panel over 30 s shows no repeated jobs list; the list still updates after a collection run.

## Approval

2026-11-11, owner: «с ней, утверждаю.» — the plan with the optional L.5 (By section) is approved. Start after ASSESSMENT-3 is merged into main.

## Owner review of L.5 (2026-11-12): returned — rework to mockup B

Owner: «Ничего не устраивает. 1. Это не тот layout что ты мне показывал… 2. Смотрю секцию Compensation: … base salary between $150,000 and $250,000 … но справа в колонке с Key Facts - ничего нет. 3. Кстати - почему то 2 секции Compensation. 4. Не было идеи вытащить все секции из вакансии. Было важно 2 секции: что хотят от кандидата, и что предлагают… Для этой же цели делался блок Key Facts.» Coordinator: the mockup B of 2026-11-09 (now `overview-b-mockup.png` in this folder) was not carried into the plan or a spec; that was the coordinator's miss. Agreed behaviour is now `docs/specs/vacancy-view.md`.

- [x] **L.5b — By section per mockup B** (60–90 min): fixed groups What they ask / What they offer / What you'll do (pending the owner's answer) / Other text (collapsed), merging same-class sections; state labels "recognised" / "not found in listing"; availability rule per spec §2. Section classes stay as built (they feed the groups). SUPERSEDED by L.5r1–L.5r3 (approved design, owner 2026-11-12); not built.
- [x] **L.5c — Key facts from merged facts** (45–75 min): Key facts read the same merged facts as Fits you; code reads salary ranges, work mode, location and contract type from the classified What-they-offer sections, then the model's conditions, each with its origin label (spec §3). Check: the owner's example (base salary $150,000–$250,000) shows in Key facts and in Fits you. SUPERSEDED by L.5r1 (Fits you band reads the merged facts); not built as written.
- [x] **L.6** — owner review again on 8094 after L.5b/L.5c, screenshots side by side with the mockup. SUPERSEDED: the review happened on the L.5r build, see the L.6 line above.

Owner, 2026-11-12: «нет, ты другой рисовал» — mockup B is not the layout the owner means; the visual layout is to be confirmed with the owner before L.5b starts (the group logic of spec vacancy-view §2–3 stands).

## Rework after the approved design (2026-11-12)

Owner approved the canvas "Sifty — vacancy view" (https://claude.ai/artifact/NV9n28fNGjTVruEqPsoRkp): «Мне нравится этот дизайн.»; «Секция full listing - должна сразу быть открытой, кнопки "Show" не надо.»; seniority tile removed. Spec: `docs/specs/vacancy-view.md` §2–3 (replaces L.5b/L.5c above).

- [x] **L.5r0 — Fill rates first** (30 min, measurement only, on a fresh production copy): share of vacancies where each tile gets a value (Money, Location, Work mode, Contract) from structured fields / section code / model conditions; share with a recognised Requirements section and an Offer section; share of requirements decided ✓/✕ by code. Report before building.
- [x] **L.5r1 — Fits you band** (60–90 min): tiles per profile goal with verdict fills; replaces the Key facts and Quick match cards; values from merged facts (spec §3), including code reading salary ranges and contract words in offer sections.
- [x] **L.5r2 — What they ask / What they offer cards** (60–90 min): per spec §2.3; offer vocabulary in configuration.
- [x] **L.5r3 — Full listing open, switch removed** (15–20 min).
- [x] **L.6** — owner review on 8094, screenshots side by side with the canvas. SUPERSEDED: see the L.6 line above.

### Decisions after L.5r0 (2026-11-12)

- Money: owner «Пересчёт по курсу из настроек» — rates in configuration; tile shows the original range, «≈ €…» and ✓/✕ against the minimum; read only from Compensation/Benefits sections and model conditions (spec §3).
- Location: no verdict colour for now (coordinator's default; the owner's question was unclear, the value is shown).
- Work mode from text only as a weak hint behind the model; contract only from explicit words (executor's recommendations accepted).
- What they ask uses the model's extraction regardless of a recognised Requirements section (82 % of vacancies have one) — not the 22.6 % section rate.
- Bring the spec into the branch: `git merge main` before L.5r1.
- We Work Remotely pages with other jobs' salary blocks (195): collection defect, recorded in COLLECTION-2.

### Owner, 2026-11-12 (review of the work in progress): ✓ does nothing on non-skill rows

Owner: «принять (v) не работает.» Screenshot: ✓ on "Reviewing AI-generated PRs", "Financial, payments, or crypto-related systems experience", "High-growth or fast-moving product environments experience" (kind experience) does nothing; ✕ on "React Native" works. Cause: the ✓ handler only adds skills (`quickAddSkillNames` returns nothing for kind ≠ skill), but the new card shows ✓/✕ on every "?" row.
Required in L.5r2 (spec vacancy-view §2.3 and profiles §2): ✓ on a non-skill requirement records a fact "You have: <requirement text>" in a "Confirmed by you" list (profile, About me, removable, next to "Skills you do not have"); ✕ records "You do not have: <requirement text>" in the same way. Code marks the same requirement (normalised text) in other vacancies ✓ / ✕ without asking again; similar wording is ASSESSMENT-5's job. A button that cannot act is never shown.

- [x] **L.5r6 — Confirmed by you: ✓/✕ on non-skill requirements** (60–90 min). Profile section `confirmed-facts` stores `ConfirmedFact{Key, Text, Have}`. ✓ on a non-skill "?" row calls `POST /api/profile/confirmed-facts`; ✕ calls the same with `have=false`. Profile section in About me (next to "Skills you do not have"), deletable. `MatchFacts` emits `confirmedHave`/`confirmedDoNotHave` facts; `vacancyai` checks them before `codeMatch` for any kind. Same normalised text in other vacancies is decided ✓/✕ without asking. Profile revision increments → Fits you recalculates. No buttons shown for a row that cannot be handled. Cross-vacancy Go test. Screenshot before and after ✓ on an experience row.
- [x] **L.5r9 — Replace ✓ / ✕ action buttons with + / − on undecided requirement rows** (scope corrected: vocabulary routing, move-to-skills / facts buttons and key-collision fix moved to SETTINGS-10). `WhatTheyAsk.tsx` (add `+`, dismiss `−`, alt buttons `+ Name`) and `QuickMatchPanel.tsx` (same symbols, titles added) updated. Verdict marks (✓/✕) and header pills unchanged. `docs/specs/vacancy-view.md` §2 updated; `attention.md` exchange-rate placeholder note added; `log.md` scope correction recorded.
- [x] **L.5r7 — One unified money calculation function** (45–60 min). `quickmatch.EvaluateMoney(MoneyInput, *ProfileMoney, rates)` returns `MoneyOutput{ConvertedMin, ConvertedMax, ApproxEUR, Verdict, Reason}`; verdicts `"ok"/"unclear"/"mismatch"/"not_comparable"`. Both `vacancyview.ComputeMoneyResult` and `quickmatch.evaluateCompensation` call it. Private wrappers in `vacancyview` keep `TestToAnnual`/`TestConvertToProfile` working. New quickmatch money tests: three states, single figure, period conversion, unknown currency, USD by rate. Tax-basis pre-check stays in `evaluateCompensation`.

### 2026-11-13: ✓ added a whole phrase as a skill

Reported: ✓ added «Coding, debugging, reviewing» to Skills as one skill (old behaviour). With "Confirmed by you" in place, the ✓ rule is: a skill is added only when each part is a known technology / tool from the skill vocabulary (`config/skills`) or a short name (≤ 3 words, no verb forms like -ing activities); a list of such names is split into separate skills; anything else (activities, experience, phrases) goes to "Confirmed by you" as "You have: …". One targeted test with real lines: «Coding, debugging, reviewing» → Confirmed by you; «TypeScript, Node» → two skills; «Go or Rust» → the alternatives offered.
