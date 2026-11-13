# INBOX-1 — comments

Captured from the owner's report on 2026-11-08. No plan or work has started.

## Owner wording — continued discussion

```text
да, сайт целиком может не скачиваться - но нам же нужна инфа для оценки вакансии. т.е. текст вакансии должен быть целиком - ДО оценки. что думааешь?
```

```text
Вкладка Fit Assessment - это уже для более полного fit assessment by LLM. Это делается (уже вроде бы сделано) для карточек которые прошли triage и попали в "Preparation".
```

```text
кажется что штрафы тоже важны - пользователю сразу видно где он по мнению Sifty не проходит.
```

```text
И пользователь должен идти обновлять свой профиль. см. новый скриншот. [Image 1]
```

```text
нужно наверное еще сделать отдельную вкладку user в settings?
```

```text
что мы храним о пользователе? CV (может быть несколько), облако тэгов/скиллов, желаемая з/п, тип работы, страны, гда он может/хочет работать, что еще?
```

The supplied screenshots are proposals for Overview, Quick Match and inline profile correction, not evidence of implemented backend fields/scoring/profile writes.

## Proposed Profile screen — further owner-supplied mockup

The owner supplied a Profile design with identity/photo, completeness percentage, document upload and extraction status, multiple source files with a sent-with-applications marker, skill levels and source drilldown, target roles/industries, learned corrections/conflict review, minimum/target/day-rate pay, work mode/contract/seniority, work authorization/time zones/relocation/visa, languages/notice period and dealbreakers. It is a design proposal, not verified personal data or implementation approval; do not copy its example salary, citizenship, language levels or skill claims into the real profile.

Coordinator review recommendations, not yet individually approved:
- Use the document/skill/correction structure as a candidate basis. Source membership and application attachment selection are separate: a career note can support matching without being sent to an employer. Several managed CV versions need clear active/selected-purpose semantics.
- Persist user-confirmed corrections with provenance and allow source inspection for each skill. Document extraction should retain uncertainty and expose conflicts rather than silently overwrite explicit corrections. Imported mention alone must not imply a production-level skill. “Sent with applications” and automatic photo attachment must not claim functionality before implemented/authorized.
- Salary: cash floor versus desired cash/total package, currency, gross/net and period. Equity/tokens must not automatically satisfy a cash minimum or be treated as equivalent liquid guaranteed salary. Contract day rate is distinct from employee annual salary.
- Current Contract row mixes permanent/contract relationship, part-time commitment and freelance engagement. Separate legal/engagement form from full-/part-time hours; allow fractional scope separately where useful.
- Current location Paris and willingness to relocate to Lisbon are compatible facts, not necessarily a contradiction. Conflict detection must compare the same field, time and scope; distinguish present/desired/conditional facts.
- Work authorization, countries the person wants/is allowed to work from, relocation and visa sponsorship need separate fields. Unknown authorization must not be rendered as a confirmed inability (struck UK/US). Time-zone range is a preference/overlap constraint, not a substitute for eligible geography.
- Seniority labels and target role titles may overlap, but leadership scope and hands-on versus management preferences matter independently.
- A completeness percentage needs a defined meaningful denominator. Optional photo/day rate/extra document count should not imply worse fit or pressure irrelevant input; a list of missing matching-relevant fields may be clearer initially.
- Mockup proposes dealbreakers highlighted in Triage, never hidden. This default visibility versus hard exclusion remains a product decision; keep it distinct from shared catalogue retention and explicit user filters.
- After profile edit, immediately rescore the user's fast catalogue; mark prior model results stale without automatically starting paid model runs. Preserve full source vacancy data.

## Profile follow-up planning — 2026-11-09

Owner requested further planning after real Profile review. [SETTINGS-2](../../settings/SETTINGS-2_profile-workflow-and-fields/plan.md) owns workflow repairs; [SETTINGS-3](../../settings/SETTINGS-3_profile-fields/plan.md) owns fields and use-by-field semantics; [SETTINGS-4](../../settings/SETTINGS-4_profile-url-sources/plan.md) owns personal URL inputs; [ASM-5](../../assessment/ASM-5_shared-term-matching/plan.md) proposes shared vocabulary actually applied to existing ranking and Profile selection. Do not duplicate that matching work in the broader future scoring stage. No execution approval for this umbrella follows from planning those cards.

Collection coverage and vocabulary are independent: owner-selected niche sites (including web3.career) remain usable without JobSpy or a future aggregator inventory. Historical tiering proposed JobSpy, then optionally JSearch/SerpApi; adapter existence is not proof of live activation/coverage. Existing current FTS/BM25 ranking is not yet the proposed structured weighted engine. Fields' actual usage must be labelled honestly; free-text dealbreakers remain detailed LLM assessment inputs.

## Capture — 2026-11-09

**Kind:** idea

Owner's words added verbatim:

```text
надо сделать так, чтобы для начала в ручном режиме можно было получать вакансии, сортировать их, перемещать в triage или браковать. из папки preparation при клике на Apply - должно быть потом не только "ответил/не ответил", но и "вакансия старая/не подошло".

А насчет того, что попадает в триаж - еще надо определять дупликаты - потому что сегодня я опять увидел вакансии, на которые уже подавался на прошлой неделе. 
```

