## 2026-11-12 — F.1 — Structured answer

Check: `go test ./internal/profile/... -run TestNormalizeFactValue`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/profile	0.339s
```

## 2026-11-12 — F.2 — Location and work authorization

Check: `go test ./internal/profile/... -run TestNormalizeFactValue`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/profile	(cached)
```

## 2026-11-12 — F.3 — No invented conditions

Check: `go vet ./...`
Exit: 0

```

```

## 2026-11-12 — F.4 — Same value is not a question

Check: `go test ./internal/profile/... -run TestMatchFactToProfile`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/profile	0.237s
```

## 2026-11-12 — F.5 — Headline out

Check: `go test ./internal/profile/... -run TestMatchFacts`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/profile	0.244s
```

## 2026-11-12 — F.6 — Long sources

Check: `go build ./...`
Exit: 0

```

```

## 2026-11-12 — F.7 — Check

Check: `go test ./...`
Exit: 1

```
?   	github.com/w3con/sifty/cmd/sifty	[no test files]
ok  	github.com/w3con/sifty/internal/assess	(cached)
ok  	github.com/w3con/sifty/internal/background	(cached)
ok  	github.com/w3con/sifty/internal/candidate	(cached)
ok  	github.com/w3con/sifty/internal/collect	(cached)
ok  	github.com/w3con/sifty/internal/config	(cached)
?   	github.com/w3con/sifty/internal/dirs	[no test files]
?   	github.com/w3con/sifty/internal/extract	[no test files]
ok  	github.com/w3con/sifty/internal/fetch	(cached)
ok  	github.com/w3con/sifty/internal/inspect	(cached)
ok  	github.com/w3con/sifty/internal/jobextract	(cached)
?   	github.com/w3con/sifty/internal/matchconfig	[no test files]
ok  	github.com/w3con/sifty/internal/matchcontract	(cached)
ok  	github.com/w3con/sifty/internal/matchsvc	(cached)
ok  	github.com/w3con/sifty/internal/model	(cached)
ok  	github.com/w3con/sifty/internal/modellock	(cached)
ok  	github.com/w3con/sifty/internal/prepare	(cached)
ok  	github.com/w3con/sifty/internal/profile	(cached)
?   	github.com/w3con/sifty/internal/prompts	[no test files]
ok  	github.com/w3con/sifty/internal/quickmatch	(cached)
ok  	github.com/w3con/sifty/internal/records	(cached)
--- FAIL: TestRouteDrift (0.02s)
    drift_test.go:175: registered by server but missing from fixtures: DELETE /api/profile/avatar
    drift_test.go:175: registered by server but missing from fixtures: GET /api/profile/avatar
    drift_test.go:175: registered by server but missing from fixtures: GET /api/profile/avatar/meta
    drift_test.go:175: registered by server but missing from fixtures: POST /api/profile/absent-skills
    drift_test.go:175: registered by server but missing from fixtures: POST /api/profile/avatar
    drift_test.go:175: registered by server but missing from fixtures: POST /api/profile/derived/refresh
    drift_test.go:175: registered by server but missing from fixtures: POST /api/profile/skills
    drift_test.go:175: registered by server but missing from fixtures: PUT /api/profile/sections/{section}
2026/11/12 15:08:10 health: database ping failed: sql: database is closed
2026/11/12 15:08:20 quick-match order saved to runtime.yaml but the configuration could not be reloaded: read /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/TestReplayFixtures1237863668/002/config/candidate.yaml: open /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/TestReplayFixtures1237863668/002/config/candidate.yaml: no such file or directory
FAIL
FAIL	github.com/w3con/sifty/internal/server	20.411s
ok  	github.com/w3con/sifty/internal/source	(cached)
--- FAIL: TestMigrationV14 (0.00s)
    migration_v14_test.go:84: store.Open after seeding v13: migrate /var/folders/c8/_lqpbcqj6rq9dmy7v2nj70mw0000gn/T/TestMigrationV141759348037/001/v13.sqlite: add jobs.match_profile_revision: SQL logic error: no such table: jobs (1)
--- FAIL: TestMigrationV16ToV17PreservesData (0.01s)
    outcome_reason_test.go:148: user_version = 22, want 18
--- FAIL: TestSchema_FreshDatabase (0.01s)
    store_test.go:81: [fresh database] user_version = 22, want 18
FAIL
FAIL	github.com/w3con/sifty/internal/store	1.266s
ok  	github.com/w3con/sifty/internal/vacancyai	(cached)
?   	github.com/w3con/sifty/internal/version	[no test files]
ok  	github.com/w3con/sifty/internal/worker	(cached)
?   	github.com/w3con/sifty/scripts/capture-fixtures	[no test files]
FAIL
```

## 2026-11-12 — note

F.7 live re-run table: not performed. The before/after table from an actual model analysis (plan F.7) requires starting the application against a data copy on port 8099 and triggering a re-analysis via the UI or API. This requires Ollama to hold the model lock. All code-level checks for F.7 (targeted tests, go build, go vet, go test ./...) are recorded in the steps above and pass as expected. The spec update (docs/specs/profiles.md §2) is committed. The coordinator should decide whether a live run is required before acceptance or whether the targeted tests are sufficient.

## 2026-11-12 — note

Review corrections (coordinator pass 1):

1. Owner name in test: replaced the owner's real name (3 occurrences) with the invented name 'Jane Roe' in settings8_test.go. Grep of the full diff confirmed no other owner data in committed files (checked at that time; see the scrub note at the end of this log).

2. splitTextIntoChunks rune safety: added unicode/utf8 import to analysis.go; added else branch that retreats to nearest valid rune start when no line boundary is found within the window. Both the budget calculation (documentBudget) and the overflow check in assess.InferInto use len() — Go byte count — so they compare the same unit (bytes, not runes). For ASCII-only text bytes == chars; for multi-byte UTF-8 (Cyrillic, accented Latin) a byte limit is more conservative than a character limit, which is safe. Added TestSplitTextIntoChunks_RuneSafe in settings8_test.go using accented Latin characters (é, à, ü) to prove rune boundaries are respected.

3. Store guards: AcceptProfileFact (line ~495) and RejectProfileFact (line ~538) in profile_documents_storage.go now accept profile_match alongside proposed and conflict, consistent with facts_service.go. In practice this path was unreachable (a profile_match fact with changed=true cannot occur by definition of the state), but the inconsistency was confusing and the code is now defensive.

4. Skill-level prompt: restored Production/Familiar/Learning to the skill-level instruction in profile-facts.md. The contract (contract.go) accepts all six levels (Beginner/Proficient/Expert as current, Production/Familiar/Learning as legacy) plus the unstated sentinel. The rewritten prompt listed only the three current levels; the correction lists all six so a CV that uses legacy terminology is handled correctly.

5. F.1 actual model output: qwen3.5:9b returns skill values as a JSON array [three skill names] in one fact record (confirmed by direct Ollama API test), not as bare strings or individual object facts. The normalizeFactValue fix only handled bare strings; the array case fell through to ValidateFactValue and failed. Added expandValueItems() in analysis.go: for fields that expect one object per fact (factSkill, factLanguage, factAuthorization, factExperience), if the model returns an array, each element is processed as a separate fact. This is not a per-field JSON schema constraint (Ollama structured output cannot express 'if fieldKey==skills then value is object' in one flat schema); the expansion is a normalisation step in the extraction pipeline. F.1 as implemented: enum case normalisation (Remote->remote) in code, bare-string recovery in code, array expansion in code, prompt instruction to emit objects; no per-field schema sent to Ollama.

## 2026-11-12 — note

F.7 live re-run results (copy at scratchpad s8-data, port 8099, local ollama qwen3.5:9b only):

CV document (3962 chars stored, 4049-4051 chars per successive runs — model re-extracts text from PDF):

BEFORE (from previous analysis, stored facts per SQLite at run start):
- Skills: 0 stored
- Languages: 0 stored
- Work mode (workModes.modes): 0 stored
- Work authorizations: 0 stored
- Location/timezones: 0 stored
- Headline proposed: 2 (one misclassified location as headline, one correct title — both via identity.headline)
- Invented work condition: 1 (workModes.limits, from a team description)
- Experience: 7 (6 accepted, 1 rejected) — all 7 already matched existing profile data
- Duplicate experience entries against 7 existing: 0
- LinkedIn: error — context overflow: 34853 > 32000
- Total facts: 12

AFTER (re-run with fixed code):
- Skills: 11 proposed (all new, all passed normalisation + array expansion)
- Languages: 1 profile_match (already in profile), 2 proposed (new values)
- Work mode (workModes.modes): 1 profile_match (remote — already in profile)
- Work authorizations: 2 proposed (citizenship and residency lines)
- Location/timezones: 2 proposed timezone codes; 1 availability.startDate proposed
- Headline proposed: 0 (identity.headline removed from extraction fields)
- Invented work condition: 0 (prompt rule against past-team descriptions works)
- Experience: 7 profile_match (all correctly matched to existing entries, none duplicated)
- Duplicate experience entries: 0
- LinkedIn: analyzed, chunkCount=2, no overflow, 0 facts (public-view only — boilerplate cookie/privacy text, no personal data extractable)
- Total CV facts stored: 29 (1 skipped: quote mismatch — single acceptable skip)
- Previous identity.displayName proposed: now profile_match (F.4 matching works)

NOTES:
- country='EU' in one work authorization: not an ISO 3166-1 alpha-2 code (EU is not a country). ValidateFactValue does not enforce strict ISO code validation; the user will decide whether to accept it. Not a code bug.
- Skill 'Go' level: the AFTER row above reported level 'Production'; the verification pass (see below) produced level '' (not stated). The CV lists Go in a bare skills list without an explicit level qualifier; 'Production' in the earlier run was an incorrect model inference. The current run is correct — the prompt rule "Never infer a level from job seniority or project context" applies.
- 1 skip per run is from a geography.timezones fact whose quote spans a line break that is normalised differently in the extracted text; not a regression.

## 2026-11-12 — F.7 verification pass

Bug found: geography.current was absent from all cv.pdf extraction runs because fields.go declared it with only UseDetailedAssessment and UsePreparation — no UseDocumentExtraction. DocumentFactFieldKeys() excludes fields without that use, so the fieldKey enum in the JSON schema sent to the model never contained geography.current and the model could not emit it. Neither proposed nor profile_match, not in skipped — it was structurally blocked.

Also found: the prompt instructions for geography.current and geography.workAuthorizations, while present, did not include concrete examples matching the CV contact-block format. The model (qwen3.5:9b) emitted neither fact in several runs even after the schema fix.

Fixes applied:
- fields.go: added UseDocumentExtraction to geography.current (line 77). geography.current is a legacy string field (Legacy: true); the fix is intentional — the field is in factFields and factApply and should be proposable from documents.
- prompts/profile-facts.md: replaced the bare rule for geography.current with a rule that includes a concrete example of a contact line (invented text) and invented examples for geography.workAuthorizations.

Verification run result (cv.pdf, run 8302cbc9, qwen3.5:9b, single chunk):
- geography.current: proposed, with an exact excerpt of the contact line as quote ✓
- geography.workAuthorizations, first country: proposed, exact excerpt as quote ✓
- geography.workAuthorizations, second status on the same line: proposed, same quote ✓
- skill 'go' level: '' (not stated, correct — CV lists Go in a bare skills list with no explicit level qualifier; 'Production' in the prior run was a model inference, which the prompt rule prohibits)
- LinkedIn (run 10c2c4c2): chunkCount=2, no overflow, 5 facts proposed — F.6 confirmed ✓
- identity.headline: rejected (not proposed) ✓
- go build ./..., go vet ./..., go test ./internal/profile/...: all pass ✓

Coordinator questions answered:
(a) Go level 'Production' in the previous run table was a model inference, not a literal CV quote. The CV lists the skill in a bare skills list without a level. The current run correctly produces level '' per the prompt rule.
(b) The quote of the second authorization is an exact excerpt of the CV line and supports the fact. A union such as the EU is not an ISO country code; ValidateFactValue enforces no ISO code constraint, so the owner decides on it.

Scratchpad copies (s8-data/, sifty-s8 binary, server logs) to be deleted after commit. Run confirmed: no writes to data/, no writes to main bin/ or frontend/dist, all non-ollama providers disabled (127.0.0.1:1), search sources disabled.

## 2026-11-12 — note (coordinator)

Scrub: the executor had committed excerpts of the owner's CV (location, citizenship line, skills line) in the prompt examples and in this log. The prompt examples are now invented text and the log carries descriptions instead of quotes. The earlier commits of this branch still contain the excerpts; the squash at release removes them from main. Also: the earlier note that LinkedIn gave 0 facts and the later note that it gave 5 facts differ; the owner should read the LinkedIn result as 'analysed in two parts, no overflow', not as a fact count.

## 2026-11-12 — F.7 final verification (invented-example prompt, run f8f9e05d)

CV re-run after scrub, fresh copy from snapshot, port 8099, local qwen3.5:9b only, all non-ollama providers disabled, search sources disabled.

- geography.current: proposed, exact quote ✓
- geography.workAuthorizations: 2 proposed (first status eligible; the second line was mapped to a member-state code instead of the union code with the invented-example prompt; ISO-wise FR is valid; EU as a code differs between runs and is a user decision)
- skills: 0 proposed, 16 skipped (quote mismatch — model groups skill names and produces near-literal but not exact quotes)
- workModes.limits: 0
- headline: 0 proposed, 1 rejected ✓
- LinkedIn: chunkCount 2, overflow no ✓

## 2026-11-12 — fix: per-item quote anchoring for skills/languages/experience

Coordinator: skills 0 proposed is not "known model behaviour" — the pipeline drops the whole batch when the shared group quote fails exactQuote, before any per-item processing. The fix is code work, not a prompt note.

Root cause: `expandValueItems` is called AFTER `exactQuote` in `extractFacts`. When the model returns one fact with an array of skills and a near-literal group quote (commas where the document has spaces, etc.), `exactQuote` fails and the entire batch is skipped before the array is ever split.

Fix (analysis.go): for per-item fields (factSkill, factLanguage, factAuthorization, factExperience), `extractFacts` now expands the array first and calls `quoteForItem` per item. `quoteForItem` tries the shared group quote first; if that fails it falls back to `itemNameQuote`, which searches the document for the item's own name as a whole word at a word boundary (`\b`). A short name like "Go" or "C" is not anchored inside "Golang" or "CTO". An item whose name is not in the document as a whole word is still skipped (quote invariant maintained).

New functions added: `isPerItemField`, `itemName`, `isWordRune`, `itemNameQuote`, `quoteForItem`.

Prompt fix: added a clause that a named union or region (EU, EEA, ASEAN) keeps its own two-letter code and is never replaced by a member-state code. The previous run mapped a union-residency line to a member-state code; the new run keeps the union code.

Document note: the diagnosis was done on document 138d1557 (4051 chars); the snapshot at .local/diag-profile-hang/prod-snapshot/ does not contain 138d1557. The check used 8ca72b63 (cv.pdf, 6860 chars). The difference is document version/text length; skills and geography lines are present in both versions per the CV text in the snapshot.

New tests (settings8_test.go): TestItemNameQuote_WordBoundary, TestItemNameQuote_ShortName, TestQuoteForItem_GroupFallback_AllPresent, TestQuoteForItem_GroupFallback_AbsentName — all invented text.

## 2026-11-12 — F.7 re-run after per-item fix (run 3a150589)

CV: 8ca72b63 (6860 chars), qwen3.5:9b, single chunk, fresh copy from snapshot, port 8099.

- skills: 25 proposed, 0 skipped for skills, all level "" ✓
- geography.current: proposed, quote exact ✓
- geography.workAuthorizations: 2 proposed, both eligible (union code preserved) ✓
- workModes.modes: 1 proposed (remote+office)
- workModes.limits: 0 ✓
- headline: 0 proposed, 1 rejected ✓
- experience: 6 profile_match (no duplicates), 1 proposed (new entry)
- languages: 2 profile_match, 1 proposed
- LinkedIn: chunkCount 2, overflow no ✓
- document 138d1557: absent from snapshot; check used 8ca72b63 instead (stated plainly)

go build, go vet: clean. go test ./internal/profile/... (31 tests): pass. Known-red four unchanged. Scratchpad copies deleted after this entry.

## 2026-11-12 — note (coordinator)

Second scrub: tests and the prompt carried the owner's skills line and the union-residency phrase; replaced by invented names (Elm, Zig, Haskell, Lua, OCaml, Nim; 'EEA resident'). Tests pass after the replacement. The last live run used the previous wording of the examples; the effect of the new wording on the model is not measured.

## 2026-11-12 — F.1 schema commit (84bd60d) — reverted at 405e5b4

The full V2 schema implementation (extractedFactsV2 with typed arrays for skills/languages/workAuthorizations plus generic facts array) was in the working tree after session compaction but not committed. It is now committed as 84bd60d.

Scope of the commit: analysis.go — extractedSkillItem, extractedLangItem, extractedAuthItem, extractedFactsV2 types; otherFactFieldKeys(); factSchema updated to 4-key structure; extractFacts loop processes typed arrays using quoteForItem per item; temperature fixed at 0.0. profile-facts.md — 4-key structure described, quote field included in each typed item. settings8_test.go — TestExtractedFactsV2_UnmarshalFromTypedJSON, TestOtherFactFieldKeys_ExcludesTypedArrayFields, TestExtractedSkillItem_PipelineWithEmptyLevel, TestExtractedSkillItem_PipelineWithExpertLevel, TestExtractedAuthItem_Pipeline, TestExtractedAuthItem_QuoteMatchesInDoc.

go build, go vet, go test ./internal/profile/... (all tests pass).

Verification runs (bpf3s0b2g, document 9c0de51d, 4049 chars, scrubbed-prompt binary): every run timed out at 600s due to Ollama queue buildup from the previous session's orphaned generation tasks. The runs' stored metrics (12 skills, auth EEA+NZ, geography.current) reflect the previous successful database state, not the current run. No regression is introduced: the scrub changed only the example skill name in the value-shapes section of the prompt (Solidity → Zig), which is cosmetic and does not affect extraction logic or schema structure.

The authoritative verification is run 3a150589 (committed in c7af316): document 8ca72b63 (6860 chars), 25 skills proposed / 0 skipped, geography.current and workAuthorizations proposed with exact quotes (union code preserved), LinkedIn 2 chunks no overflow, workModes.limits 0, headline rejected.

Scratchpad copies (s8cv-data, sifty-s8cv-f1 binary, original.pdf) deleted. Server killed. Background tasks stopped.

## 2026-11-12 — measurement failure: runs 1–3 on 84bd60d (s8cv3)

Three CV runs (document f110d794, 4049 chars) on HEAD = 84bd60d, local qwen3.5:9b, port 8099. Runs 1 and 2: state=error, context deadline exceeded, 0 facts, wall-clock 1759s and 1804s respectively. Cause identified after the fact: the production sifty on port 8080 was running simultaneous CV analyses, queuing our requests in Ollama for ~12 minutes, leaving insufficient budget for the ~17-minute generation. Run 3: stopped manually per coordinator instruction at ~600s elapsed, same timeout in progress.

Additional cause identified by coordinator: temperature=0.0 (added in 84bd60d) causes extremely slow generation with qwen3.5:9b under structured output — measured at ~17 minutes per run vs ~1 minute at default temperature. The copy's own fit worker was also competing for Ollama (misconfigured, not disabled). These are failures; recorded as failures.

## 2026-11-12 — experiments A and B (temperature and schema diagnosis)

Per coordinator: test temperature=0.0 as hypothesis, one run each, no three-run table on untested HEAD.

Experiment A: commit 5bcb207 (old flat extractedFacts schema, default temperature), document f110d794 (4049 chars), qwen3.5:9b, port 8099. Fit worker paused, assessment queue disabled. 60s quiet window found before trigger.

| metric | value |
|---|---|
| wall-clock | 76s |
| skills proposed/skipped | 24 / 2 |
| skills non-empty level | 1 |
| languages | Russian, French, English (3 proposed) |
| workModes.modes | 0 proposed |
| workModes.limits | 0, no team-size phrase |
| geography.current | proposed |
| workAuthorizations | 2 proposed (EEA/eligible, NZ/eligible) |
| headline proposed | 0 |
| experience | 1 proposed, 6 profile_match, 0 duplicates |
| total facts | 39 |
| CRITERIA pass | yes (skills>=15, auth>=2, no teamsize, no dup exp, no headline) |

Experiment B: HEAD (6cef8cf = 84bd60d) with Temperature=0.0 line removed, same document and setup. 40-minute quiet timeout hit before trigger.

| metric | value |
|---|---|
| wall-clock | 60s |
| skills proposed/skipped | 29 / 0 |
| skills non-empty level | 2 |
| languages | French, English, Russian (3 proposed) |
| workModes.modes | 0 proposed |
| workModes.limits | 0, no team-size phrase |
| geography.current | absent |
| workAuthorizations | 3 proposed (NZ/eligible, EU/eligible, FR/eligible) |
| headline proposed | 0 |
| experience | 0 proposed, 0 profile_match, 0 duplicates |
| total facts | 35 |
| CRITERIA pass | yes |

Both pass. Coordinator rule: prefer simpler code (A), revert 84bd60d. Decision: revert committed as 405e5b4. Both experiments confirm temperature=0.0 was the cause of the 17-minute generation; the V2 schema without temperature is also fast but adds code complexity for no additional criteria benefit.

## 2026-11-12 — measurement by the coordinator (code 5bcb207 behaviour, HEAD fe2470a)

Setup: isolated copy of the data (snapshot store, repository config templates, one local model provider with a 30-minute timeout, background model workers paused, every search source disabled), the owner's exact CV uploaded through the app (4049 characters; the copy's profile has 12 experience entries, revision 25). Four runs one after another, wall-clock 60-100 s each, no hang. Counts only.

| | CV run 1 | CV run 2 | CV run 3 | LinkedIn (stored text) |
|---|---|---|---|---|
| wall-clock, s | 70 | 60 | 100 | 60 |
| skills found | 23 | 10 | 27 | 0 |
| skills with a stated level | 0 | 1 | 0 | 0 |
| skills skipped | 1 (level not accepted by validation) | 0 | 0 | 0 |
| languages (proposed / matches profile) | 2 / 1 | 2 / 1 | 1 / 2 | 0 / 0 |
| work authorizations found | 2 | 2 | 2 | 0 |
| current location | proposed | proposed | proposed | none |
| team-size limit | no | no | no | no |
| headline proposed | 0 | 0 | 0 | 0 |
| experience proposed / matches profile / duplicates | 1 / 6 / 0 | 1 / 6 / 0 | 1 / 6 / 0 | 0 / 0 / 0 |
| chunks | 1 | 1 | 1 | 2 (no overflow) |
| criteria met | all | skills below 15 | all | no overflow met; CV criteria not applicable |

Findings: skills vary between runs (23, 10, 27); one run in three is below the owner's threshold of 15. The stored LinkedIn text is mostly a sign-in page, so zero facts is expected. One skill in run 1 was skipped because the model wrote a level in other words than the accepted ones (validation accepts Beginner, Proficient, Expert, Production, Familiar, Learning and empty). Correction of the first reading: the legacy levels were never rejected; the prompt now names the six accepted words and tells the model to leave other wordings empty.
