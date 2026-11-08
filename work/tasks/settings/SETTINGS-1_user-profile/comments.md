# SETTINGS-1 — comments

Captured from the owner's report on 2026-11-08. No plan or work has started.

## Owner clarification — CV and follow-up search

```text
не заставляй писать слишком много тестов - только критические, которые ловят ошибки, которые я НЕ  способен увидеть глазами в приложении.

это стоит записать где-то в правилах проекта
```

This corrected owner wording supersedes the preceding message that omitted НЕ. Project rules and the executor brief now allow only minimal critical hidden-failure tests within approved work; normal visible UI acceptance is performed in the running application. No test/source command was executed merely to record this instruction.

```text
LLM анализ документов - должен быть прост - у нас уже все есть. надо только определить как все будет сохраняться.

Стоит тебе продумать структуру таблицы пользователя в DB?
```

Coordinator inspected migrate.go, Store.Context/SaveContext and assess.Infer/RawModel before drafting the four-table storage proposal in description.md. Provider adapters, PDF/text reading and context snapshots exist. Profile current-state/documents/assertions/revisions do not. No runtime database was opened/migrated or personal CV content read for this proposal. New output schema/decoder uses existing transport; no independent LLM framework is proposed.

```text
хотя некоторые вещи, типа dealbreakers не получится полностью реализовать без LLM - типа как модуль оценки поймет написаное от руки "travel > 25%" - т.е. либо нам нужно будет подумать наперед и добавить таких полей (не давай вводить их вручную), либо этот блок оценивать только на этапе подготовки ответа на вакансию. ну
```

```text
и да, эти данные должны влиять на оценку - видимо нужно будет добавить функциональности к тому модулю оценки, что у нас есть.
```

```text
ну, CV - это то что пользователь добавил для Sifty.

Возможно, позже, мы будем генерировать CV для польователя под каждую вакансию.
```

```text
кстати - надо будет (потом) сделать так, чтобы поиск по умолчанию создавался или соответствовал этим данным о пользователе. make sense?
```

## Screenshots and approval meaning

## Library research — documentation checked 2026-11-08

Owner asked whether DOCX support is simply library integration and requested ready-made libraries for this card. Search used `/Volumes/SSD/Dev/ai/bin/websearch` first (Tavily answered all3queries; no provider failure), followed by primary project README/license/package metadata. No packages were installed, benchmarked or exercised on personal CVs during this research; version compatibility is documentation-level until the chosen implementation is built.

| Need | Candidate and verified facts | Coordinator recommendation |
|---|---|---|
| DOCX→text | Mammoth `https://github.com/mwilliamson/mammoth.js`: `extractRawText` accepts Node path/buffer or browser ArrayBuffer; paragraphs separated by blank lines; converter supports tables/lists/headings/text boxes. BSD-2-Clause verified in LICENSE. Raw text intentionally drops layout. | Leading candidate for extraction, not Word editing. Backend can call a small local helper using the already present Node worker runtime; no new external service. Need verify representative CV ordering/tables before final selection. Do not render unsanitized converted HTML; this task needs text/evidence. |
| Native Go DOCX | `https://github.com/gomutex/godocx`: pure Go, MIT, `OpenDocument` parses DOCX; README and parsing source inspected. | Credible Go-native alternative, but no claim that it already offers equivalent convenient text extraction or is proven on our CVs. Select only after a focused source/sample check; no need to implement a DOCX parser from scratch. |
| Other Go DOCX | `https://github.com/fumiama/go-docx`: parse/read support, AGPL-3.0 documented. | Not the default choice for the current project's licensing direction; do not casually add it merely because it is a Go package. |
| Section editors | `https://github.com/react-hook-form/react-hook-form`: MIT; package peerDependency explicitly includes React19. `https://github.com/react-hook-form/resolvers` documents Zod resolver and typed input/output. | Use React Hook Form plus resolver and existing Zod for dirty/errors/reset/submit; not a custom generic form engine. Backend Go validation still required. |
| Document picker/drop area | `https://github.com/react-dropzone/react-dropzone`: MIT, React18+ in current metadata, accepted-file/type/size/rejection handling. README explicitly says it is NOT a server uploader. | Small suitable optional helper for the mockup drop area; upload endpoint/storage/processing statuses remain our work. Native file input may suffice initially. |
| PDF and plain text | Existing `candidate.fileText` uses pdftotext; Markdown/plain text can be read directly. Zod and SQLite already dependencies. | Reuse existing parts; do not replace PDF stack or add document-viewer/generator frameworks for text import. OCR/Word rendering not included. |

Primary references checked: Mammoth README and BSD license at `https://raw.githubusercontent.com/mwilliamson/mammoth.js/master/{README.md,LICENSE}`; Godocx README/document.go and docx/document.go on its main branch; React Hook Form package.json/LICENSE; resolvers README; react-dropzone README/package.json. These checks establish documented capabilities/terms, not a compatibility test of pinned released versions. Choose/pin actual published versions at implementation, not a development-branch version string.

DOCX reading itself is not the hard part and should not be planned as a custom parser. The existing30–50min documents row includes upload/API/storage/list/statuses/format dispatch and useful errors, not30–50min solely for one extraction call. Text→structured facts/provenance/conflict confirmation remains a separate30–50min row; no document library decides production skill level or resolves candidate claims. The library findings do not justify pretending that these integration tasks disappear, nor an automatic whole-card time reduction without checking the chosen integration.

## Executor retrospective — November6–7, checked against runtime metadata

The owner asked to inspect previous sessions, noting that DeepSeek wrote the large refactoring and is inexpensive. Coordinator queried OpenCode session/message metadata for project executor sessions created on November6–7: **14sessions,1497assistant messages, exclusively `opencode-go/deepseek-v4.1-flash`**, no other assistant provider/model in that group. Message counts are volume evidence, not a success rate, active time or billing amount.

November6 includes repository/lifecycle command work, publication checks/setup, relocation, task state/navigation/capture, requirement/data map, continuous preview, isolated deploy/recovery and checked squash release. Evidence: OPS-4 worktree log and commits such as `9a60b4a`, `66db415`. November7 includes all four frontend separation stages, not just small fixes. Frontend diff through8cab4ce touches66files; this includes formatting/extraction, not66independent features. Foundation diff includes many renames, so its692file count must not be called692new implementations.

Review findings were substantive: lifecycle rejection/ignored-file preservation/invalid-plan gaps; deploy snapshot/recovery ordering; frontend Sort DOM nesting. The same executors implemented corrections and coordinator records show subsequent acceptance. Coordinator mistakes also mattered: an incorrect instruction on backup timing, excessive/repeated verification, broad stage assignments and idle/communication failures. The recent api.ts proxy/MIME error was November8: a real implementation error missed by coordinator acceptance, not enough by itself to characterize the two prior days.

Conclusion/recommendation: keep DeepSeek as the default coder for this card. The logs support broad implementation ability with review/correction, not merely tiny tasks. No controlled comparison establishes a better model or that a more expensive default would reduce total cost. Nominal cost/token fields in OpenCode metadata are not proof of the owner's OpenCode Go invoice. Measure useful accepted results and correction effort; do not blame coordinator overhead on model weakness or silently change provider. No model change is authorized by this retrospective.

Latest owner wording:

```text
ну.. если при отборе заявки из triage - мы включяем LLM - то она может обновлять оценку. может быть так будет проще? для начала.
```

```text
ок. тогда давай сделаем план - что нам нужно сделать сначала, что потом.. где какие зависимости.
```

```text
полагаю начать лучше с профиля, потом перейти к поиску, затем модуль оценки, затем карточка вакансии?
```

The first «ок» accepts the recommended simpler product direction and asks for planning; it is not approval to execute the newly written plan. The coordinator's plan retains quick/refined scores separately and postpones an arbitrary-text quick-rule engine.

## Mockup notes and approval meaning

The first Profile mockup and individual-block edit screenshot establish desired editable sections (Edit/Save/Cancel), not a promise that every illustrated behavior is built. The owner approved the coordinator's six corrections: evidence documents versus sending, cash versus equity, engagement versus hours/work mode, present versus relocation geography, meaningful missing data rather than arbitrary completeness percentage, and explicit dealbreaker visibility.

The updated mockup has Profile access in the header, missing-data notices, separated cash/day-rate/equity fields, separated engagement/hours/work mode and separate residence/relocation. It still shows Can send/Default and automatic photo attachment, which are superseded by the owner's CV-as-input clarification unless separately approved. The “newer conflicting value wins until selection” text remains a coordinator-identified open issue, not an agreed conflict policy. Mockup examples are not personal facts to seed in a profile.
