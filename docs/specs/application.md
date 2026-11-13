# Application preparation — specification

Status: owner decisions of 2026-11-13 applied (OPS-11); NOT DESIGNED; rule tags show the implementation state.

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

The two rules that belong here: Sifty never fills or submits an application form; the browser worker blocks all non-data POST requests. [implemented; `backend/workers/playwright/index.mjs:228-245`] Only the owner's explicit confirmation moves a vacancy to Applied; opening the employer page does not. [implemented; `backend/internal/store/jobs.go:741-812`]

## 4. Configuration

- Stage models for assessment, preparation, and extraction are set in Settings, Models panel; the selection is written to `providers.yaml` under `stages.assessment`, `stages.preparation`, and `stages.extraction`. The preparation stage is optional; without it generation cannot start.
- The assessment queue is enabled or disabled in Settings, Assessment queue; while off, the worker does nothing, including after "Assess now" (the row is queued but never claimed). [implemented; known defect: the "Assess now" notice does not reflect the off state]
- Browser automation timing (`browser_interval_ms`, `browser_jitter_ms`, `browser_concurrency`, `browser_cooldown_ms`) lives in `runtime.yaml`; defaults are 5 000 ms interval, 3 000 ms jitter, concurrency 1, cooldown 300 000 ms. [not implemented — APP-5]
- Candidate evidence sources and their scope are listed in `candidate.yaml`.

## 5. Questions for the owner

Design questions that must be answered before answer drafting can be built.

1. Which store is the knowledge base for prepared answers: the existing remembered-answers store (`data/answers/`), the Obsidian `kb/` vault, or something new that SET-1 must define? (SET-1)
2. What is populated into the answer knowledge base first: a fixed list of common application questions (why this role, notice period, work authorisation, salary expectation), the questions already seen in inspected forms, or both? (SET-1)
3. Who writes the first version of each stored answer: the model drafts from candidate material and the owner confirms each one, or the owner writes them directly? (SET-1)
4. Does a stored answer carry a scope (universal, per role type, per company, per language), and how is a form question matched to a stored answer — by the model with the match shown for confirmation, or by exact wording? (SET-1)
5. For sensitive answers whose facts are unresolved in the candidate material (salary, work authorisation, relocation): may the model propose a draft, or are they always left blank for the owner to fill? (SET-1)
6. When exactly does drafting trigger: when the Application tab opens, only on an explicit "Prepare" button press, or as a low-priority background pre-warm for other shortlisted roles? The 2026-11-03 decisions record says "tab open or a per-question action"; APP-3 says "on demand, the viewed role first"; these need alignment before APP-3 can be planned.
7. What is the shape of the "think" modal: a separate overlay or inline in the Application tab, and what happens to any existing stub answer while the conversation runs?
8. For the stronger-model shortlist re-run (ASSESSMENT-6): should it run automatically on every shortlist entry, or only when the owner requests it, and what is the recommended default model?

## 6. Open cards

- APP-3 — KB-sourced answers with per-question think modal; blocked on SET-1.
- APP-4 — ATS click-through in form inspection (Ashby, Greenhouse, Lever); blocked on APP-1 (done).
- APP-5 — Browser pacing and cooldown discipline; blocked on APP-4.
- APP-6 — Remembered answers injected as preparation evidence; backlog.
- APP-7 — Draft revision check to prevent concurrent overwrites; backlog.
- APP-8 — Negative outcomes in the confirmation dialog; released in main `a3ead8b`; card stays in review until OPS-9 closes released review cards.
- SET-1 — First-time answer knowledge-base setup; placeholder, design not started; blocks APP-3.
- ASM-1 — Re-run failed assessments on a chosen model; deferred by the owner on 2026-11-04.
- ASSESSMENT-6 — Stronger-model re-run at shortlist entry; backlog, not planned.
