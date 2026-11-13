# Application preparation — specification

Status: owner decisions of 2026-11-13 applied (OPS-11); delivery stages and the application-path design agreed with the owner on 2026-11-13 (§3.0, §3.6); answer drafting still partly open (§5); rule tags show the implementation state.

## 1. Purpose

The owner shortlists a vacancy and Sifty does two kinds of model-assisted work for it. First, a model assessment compares the vacancy against the owner's candidate evidence and returns a structured result: a summary, evidence items with exact quotes, confirmed mismatches, candidate unknowns, and employer unknowns. Second, it reads the employer's application form through a headless browser and drafts answers to its questions from the owner's candidate context and from a knowledge base of prepared answers. The owner reviews the assessment result, edits the draft answers, opens the employer page, applies by hand, and records the outcome. Sifty never fills or submits a form on the owner's behalf.

## 2. Scope

This specification covers model assessment, form inspection, and answer drafting for a shortlisted vacancy. It does not cover:

- Apply outcomes (Applied, Dismissed with reason, Not yet): these belong to the [triage specification](triage.md).
- Fits you and You fit scores, the fit worker, and the shared model lock: the [Quick Match specification](quick-match.md).
- The confirmed profile as an evidence source: the [profiles specification](profiles.md).
- The first-time answer knowledge-base setup in Settings: SET-1 (not yet designed; it blocks answer drafting).

## What exists today

The code has two separate workers — an assessment worker and a preparation worker — both running since the Go port. The assessment worker produces structured results but omits the evidence bank from the request, so the model receives no candidate text to cite even though the prompt instructs it to do so; the production assessment queue has been off since 2026-11-05 with 2 107 pending rows. The preparation worker can read application forms via a headless browser (inspection is wired and completes on real roles), but the generation path has never worked: the inference function is not wired in the server, the provider description passed to preparation lacks the type and endpoint, and the candidate source text is serialised through a lossy JSON path that discards the actual text before the evidence bank is built; no model-generated draft has ever been recorded by the Go backend. The remembered-answers store exists on disk but the "Remember" call fails at runtime and nothing reads the store before drafting. The full code-level picture, with line references and production counts, is in [the as-built file](../../work/tasks/ops/OPS-11_spec-coverage/as-built/application.md).

## 3. Rules

Each rule ends with a status tag. Only rules the owner has decided are listed here.

### 3.0 Delivery stages

Owner decision, 2026-11-13: a complete cycle comes first, and the quality of each step is improved later (owner: "I want a full cycle already; the quality of each stage can be improved later"). The full cycle is collect, Inbox, Shortlist, prepare, apply, record the outcome. Application work is delivered in three stages; the owner's words on the goal: "the full cycle is with answers and even filled in automatically. That is the ideal we are moving toward. Iteratively."

- **Stage 1 — direct link (APP-9).** For a shortlisted vacancy Sifty finds the application form and, where possible, the same vacancy on the employer's own site or applicant-tracking system. An Apply button opens that page in a separate browser window; the owner applies by hand and records the outcome. The cycle is closed after this stage.
- **Stage 2 — prepared answers.** Sifty collects the form's questions, proposes answers from the answer bank, the owner's facts and previous answers, and the owner approves or edits them. Each answer has a Copy button; the owner pastes it into the form open in the separate window.
- **Stage 3 — the ideal: filled forms.** A browser extension with a side panel opens beside the employer's form in the owner's own browser, where the owner is already signed in, shows the answers and fills the fields. The owner presses Submit. Server-side form filling by a headless browser is not planned, because sites detect and block it.

### 3.1 Model assessment

1. When the owner moves a vacancy to Shortlist, an assessment is queued automatically at default priority. [implemented; `backend/internal/store/jobs.go:814-822`; note: the queue has been Off in production since 2026-11-05 with 2 107 pending rows, so queued rows are not being claimed]
2. The owner can run assessment immediately with "Assess now", which re-queues the role at high priority and accepts an optional provider override. [implemented; `frontend/RoleContent.tsx:54,58`]
3. The assessment result contains a summary, evidence items (each citing an exact original quote from candidate evidence), confirmed mismatches, candidate unknowns, and employer unknowns. [implemented; `backend/internal/assess/assess.go:97-105`]
4. A result is marked stale when the vacancy text, candidate context version, or confirmed profile revision differs from what was recorded when the result was computed; a stale result is kept and displayed with a warning. [implemented; `backend/internal/store/jobs.go:121-160`]
5. Assessment cites only candidate evidence; each evidence claim references an exact quote; a claim with no grounding fails validation rather than appearing in the result. [deviates — the Go backend omits the evidence bank from the assessment request so the model has no candidate text to cite; `backend/internal/assess/assess.go:308`; no card]
6. Assessment runs only for shortlisted roles; the queue does not sweep the whole vacancy catalogue. Owner, 2026-11-13: accepted the coordinator's recommendation. [deviates — the code sweeps all non-hidden roles every 30 seconds while the queue is On; no card for the fix; same decision as triage.md §3.12 rule 33]
7. The owner can re-run failed assessments on an explicitly chosen model without changing the stage model for the whole queue; the failed count and retry model are shown before the action runs. [not implemented — ASM-1, deferred by the owner on 2026-11-04]
8. When a vacancy enters Shortlist, a second assessment on a stronger owner-chosen model is stored beside the initial result, labelled with the model name. [owner decision needed — the owner requested this in ASSESSMENT-6 but whether it runs automatically or on explicit request, and which model is recommended, has not been decided]

### 3.2 Form inspection

9. Inspection starts the moment a vacancy is moved to Shortlist; it does not wait for the Application tab to open. [implemented; `backend/internal/server/routes.go:163-178`]
10. Inspection is model-free and never waits behind inference work or takes the shared model lock. [implemented; `backend/internal/background/preparation.go:137-205`]
11. Inspection clicks through ATS "Apply" controls (Ashby, Greenhouse, Lever and similar applicant-tracking systems) to reach the actual application form rather than reading the job-listing landing page. [not implemented — APP-4]
12. Browser automation paces requests per host, respects blocks with a per-host cooldown period, and treats a challenge page or captcha as a block rather than retrying into it. [not implemented — APP-5]
13. An unresolvable or unreadable destination ends inspection with a named honest state, not a raw error; the owner is offered a direct link to the original listing. [not implemented — APP-4]

### 3.3 Answer drafting

14. Each inspected question is answered by default from the knowledge base of prepared answers, not by per-question model generation across all shortlisted roles. [not implemented — APP-3, blocked on SET-1]
15. Answer drafting is lazy and on demand; it does not run eagerly for every shortlisted vacancy; the vacancy the owner is actively viewing is processed first. [not implemented — APP-3]
16. Next to each answer the owner can open a per-question "think" modal; the model works through the question with the owner, asks for what it needs, and on resolution writes the agreed answer to the knowledge base so later questions on other roles are answered from it. [not implemented — APP-3]
17. Remembered answers saved from previous applications are injected as owner-confirmed evidence before drafting; a question the owner has already answered is drafted from that answer rather than from scratch. [not implemented — APP-6]

### 3.4 Draft editing and revision safety

18. Saving a field's edited value persists it in the stored draft and advances the draft revision. [deviates — the PATCH handler decodes the request body into the draft record directly, so the field value is not updated; `backend/internal/server/routes.go:945-949`; APP-7 assumes the edit works]
19. A background generation that completes while the owner has unsaved edits must not silently overwrite those edits; a draft edit made against an out-of-date revision is refused with a conflict response rather than applied. [not implemented — APP-7]

### 3.5 Application and outcome recording

Application submission, outcome recording, and confirmation dialog behaviour are part of the triage workflow and are specified in the [triage specification](triage.md).

The two rules that belong here: Sifty never fills or submits an application form from the server; the browser worker blocks all non-data POST requests. [implemented; `backend/workers/playwright/index.mjs:228-245`; stage 3 (§3.0) adds filling by a browser extension in the owner's own browser, and submission stays the owner's act] Only the owner's explicit confirmation moves a vacancy to Applied; opening the employer page does not. [implemented; `backend/internal/store/jobs.go:741-812`]

### 3.6 Application path and answers

Owner decisions of 2026-11-13. The owner proposed the nine-step path (find the form link, find the employer's page, keep only the direct link, read the questions, propose answers, owner approval, learn from edits, keep every answer, assemble answers before the cover letter); the coordinator's amendments (separate window instead of an iframe, applicant-tracking-system boards before web search, confirmed facts from edits, reuse of facts rather than text, sensitive questions left to the owner) were accepted by the owner ("I agree with all your other proposals"; rule 31: "ok").

Employer page lookup (stage 1):

20. When a vacancy is fetched and parsed, the link to its application form is recorded with the vacancy. [not implemented — APP-9; today inspection reads the landing page, see rule 11]
21. For a shortlisted vacancy, and for any vacancy whose Apply the owner presses, Sifty looks for the same vacancy on the employer's own site or applicant-tracking system, in this order: links on the aggregator's vacancy page; the public job boards of applicant-tracking systems (Greenhouse, Lever, Ashby, Workable) under the company name; a web search for the company site and its careers page, whatever it is called; the vacancy on that page. [not implemented — APP-9]
22. A found vacancy is matched to the original by title and location; when the match is not exact, the model confirms that it is the same vacancy. [not implemented — APP-9]
23. Once a direct link is found, all further work (form reading, Apply) uses it; the aggregator link is kept as a secondary fallback. Reason, owner: aggregators such as Wellfound have many good vacancies but technically refuse candidates outside the US, while the employer's own form usually needs no registration. [not implemented — APP-9]
24. The vacancy page is opened in a separate browser window beside Sifty, not in an iframe: most job sites and applicant-tracking systems forbid embedding their pages, and an iframe would stay blank. [not implemented — APP-9]
25. Web search runs through a configurable chain of providers with fallback (§4). [not implemented — APP-9]

Questions (stage 2):

26. Form questions are read where the form is readable without signing in: applicant-tracking-system data (Greenhouse and Ashby publish question lists; Lever serves a plain form page) and plain form pages. [not implemented]
27. Forms behind a sign-in (LinkedIn, Wellfound) are handled as decided in §5 question 9. [owner decision needed]

Answers (stage 2):

28. Proposed answers are stored in the database per vacancy and per question. [not implemented]
29. Typical questions (salary expectation, notice period, work authorisation, visa sponsorship, relocation, years with a named technology) are answered from a bank of prepared answers kept in the profile data, without calling the model. [not implemented]
30. The model is called only for the cover letter and for non-typical questions; it uses the owner's facts, previous answers, and the vacancy text. [not implemented]
31. The owner approves or edits each proposed answer. When the owner edits one, the model extracts candidate facts from the change and proposes them; each fact is saved to the profile only after the owner confirms it with + (or rejects it with −). Reason: some edits are true for one vacancy or for a limited time, and silent saving would pollute the profile. [not implemented]
32. Every answer is stored, including cover letters and company-specific answers ("why do you want to work at XYZ"); later drafting reuses facts extracted from them, never their text, so one company's name never reaches another company's letter. [not implemented]
33. Sensitive questions (gender, ethnic origin, disability, criminal record and similar) are never answered by Sifty; they are left for the owner. [not implemented]

## 4. Configuration

- Stage models for assessment, preparation, and extraction are set in Settings, Models panel; the selection is written to `providers.yaml` under `stages.assessment`, `stages.preparation`, and `stages.extraction`. The preparation stage is optional; without it generation cannot start.
- The assessment queue is enabled or disabled in Settings, Assessment queue; while off, the worker does nothing, including after "Assess now" (the row is queued but never claimed). [implemented; known defect: the "Assess now" notice does not reflect the off state]
- Browser automation timing (`browser_interval_ms`, `browser_jitter_ms`, `browser_concurrency`, `browser_cooldown_ms`) lives in `runtime.yaml`; defaults are 5 000 ms interval, 3 000 ms jitter, concurrency 1, cooldown 300 000 ms. [not implemented — APP-5]
- Candidate evidence sources and their scope are listed in `candidate.yaml`.
- Web search for the employer lookup (rule 21) uses an ordered chain of providers set in YAML for now; fields that prove necessary move into Settings later (owner: keys for Tavily, Brave or any search engine, and the fallbacks, must be settable; "for now, for simplicity, in yaml"). Each provider entry names the environment variable or secret file that holds its key; keys are never written into the YAML itself. Default order recommended by the coordinator: Tavily, then Brave when a key exists, then DuckDuckGo's HTML page without a key, with a pause between requests; SearXNG is added only if DuckDuckGo starts refusing. Expected volume is two or three queries per shortlisted vacancy. [not implemented — APP-9]

## 5. Questions for the owner

Design questions that must be answered before answer drafting can be built. Questions 1–3 and 5 were answered by the owner's decisions of 2026-11-13 (§3.6); the rest stay open for stage 2 planning.

1. Which store is the knowledge base for prepared answers? **Answered 2026-11-13:** a bank of prepared answers kept in the profile data (rule 29); whether it reuses `data/answers/` is decided in the stage 2 plan.
2. What is populated into the answer knowledge base first? **Answered 2026-11-13:** a fixed list of typical questions first; questions seen in read or pasted forms are added as they appear.
3. Who writes the first version of each stored answer? **Answered 2026-11-13:** the model proposes from the owner's facts and previous answers; the owner approves or edits (rule 31).
4. Does a stored answer carry a scope (universal, per role type, per company, per language), and how is a form question matched to a stored answer — by the model with the match shown for confirmation, or by exact wording? (open)
5. Sensitive answers. **Answered 2026-11-13:** demographic and criminal-record questions are never answered by Sifty (rule 33); salary, work authorisation and relocation come from the answer bank. Open: what happens when the bank has no answer for one of them.
6. When exactly does drafting trigger? Partly answered: the employer lookup runs on shortlist and on Apply (rule 21). Drafting trigger is open: when the Application tab opens, on an explicit "Prepare" press, or as a low-priority background pre-warm.
7. What is the shape of the "think" modal: a separate overlay or inline in the Application tab, and what happens to any existing stub answer while the conversation runs?
8. For the stronger-model shortlist re-run (ASSESSMENT-6): should it run automatically on every shortlist entry, or only when the owner requests it, and what is the recommended default model?
9. Forms behind a sign-in (LinkedIn, Wellfound). The coordinator's options: (a) Sifty signs in with the owner's stored credentials or session — not recommended, LinkedIn blocks accounts for automated access and stored credentials are a risk; (b) a catalogue of the fixed part of each aggregator's form (contacts, résumé, work authorisation, sponsorship, a short note) — covers only the fixed part, employer-added questions differ per vacancy; (c) the owner pastes the form's text into Sifty and the model extracts the questions — works on any site, and pasted questions grow the catalogue; (d) the stage 3 extension reads the form in the owner's signed-in browser. Recommendation: (b) and (c) in stage 2, (d) replaces (c) in stage 3.

## 6. Open cards

- APP-3 — KB-sourced answers with per-question think modal; blocked on SET-1.
- APP-4 — ATS click-through in form inspection (Ashby, Greenhouse, Lever); blocked on APP-1 (done).
- APP-5 — Browser pacing and cooldown discipline; blocked on APP-4.
- APP-6 — Remembered answers injected as preparation evidence; backlog.
- APP-7 — Draft revision check to prevent concurrent overwrites; backlog.
- APP-8 — Negative outcomes in the confirmation dialog; released in main `a3ead8b`; card stays in review until OPS-9 closes released review cards.
- SET-1 — First-time answer knowledge-base setup; placeholder, design not started; blocks APP-3.
- APP-9 — Stage 1 (§3.0): application-form link, employer page lookup, search provider chain, Apply in a separate window; next after INBOX-2.
- ASM-1 — Re-run failed assessments on a chosen model; deferred by the owner on 2026-11-04.
- ASSESSMENT-6 — Stronger-model re-run at shortlist entry; backlog, not planned.
