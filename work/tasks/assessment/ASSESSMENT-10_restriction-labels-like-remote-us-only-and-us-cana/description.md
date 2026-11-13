# ASSESSMENT-10 — Restriction labels like 'Remote (US only)' and 'US & Canada' are not res

**Epic:** assessment
**Created:** 2026-11-13
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
тдельный вопрос, без срочности. По ASSESSMENT-9 девять вакансий с явным ограничением («Remote (US only)», «US & Canada») по-прежнему получают «неизвестно», потому что словарь регионов не разбирает такие формулировки места. Это кандидат на отдельную небольшую карточку. Заведу её, если скажете. - да
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator (2026-11-13)

After ASSESSMENT-9 (released 2026-11-13), 38 model-read locations became usable, but nine remote vacancies with an explicit restriction still get "?" because `resolveRestrictionLabel` in `backend/internal/quickmatch` does not resolve their labels: 17 "Remote (US only)", 1119 "REMOTE (US only) or NYC/Dallas", 1153 "US only, US time zones", 1175 US state list, 1196 "US and Canada", 1201 "US-based (remote within the US)", 1279 "UK, Ireland, Germany, Portugal, Spain or The Netherlands", 1281 "ET timezone in the US", 1294 "US & Canada" (production copy of 2026-11-13; list in ASSESSMENT-9 `attention.md`). Likely work: strip wrapper words ("Remote", parentheses, "-based"), split on "&", "and", "or", "/", ",", resolve each part through the region vocabulary, map US state names to US; separators and wrapper words in configuration next to the regions, not in code. Spec quick-match §4.1.

Owner, 2026-11-13, on the order of work: «1» (wait for the INBOX-2 release before COLLECTION-2 and do ASSESSMENT-10 meanwhile).

## Plan (coordinator, 2026-11-13; awaiting the owner's approval)

**Outcome.** A remote vacancy whose usable location label names the allowed countries in a common compound form gets a definite "covered" / "not covered" verdict instead of "?". The owner sees it in triage as the Fits you score and a Matches or Mismatches line for geography. It is demonstrated on a fresh production copy with the list of every changed vacancy.

**Rule.** `resolveRestrictionLabel` (pure, `internal/quickmatch/geography.go`) also tries: the text inside and outside parentheses ("Remote (US only)" → "US only"; "US-based (remote within the US)" → "US-based"); leading wrapper words ("remote", "remote within the", "within the"); trailing qualifiers, including the ones now hard-coded in `strippedForms` ("only", "-based", "based") and time-zone wording ("time zones", "timezone"); and the separators "&", "and", "or", ";" besides "/" and ",". The full label is always tried first, so a vocabulary name that contains "and" is never split. Every part must resolve or the whole label stays unknown (unchanged conservative rule). US states and DC are a vocabulary entry mapping to US. All words, separators and states live in `backend/internal/config/regions.default.yaml` with a `quick_match` override, like the regions and phrases; no word list stays in Go code.

**Expected on the nine known vacancies.** Resolve: 17, 1153, 1175, 1196, 1201, 1279, 1294. Stay "?": 1119 (city names NYC/Dallas are not in the vocabulary) and 1281 ("ET timezone in the US" is a time-zone statement).

**Must not touch.** `frontend/`; `.worktrees/INBOX-2` and its branch; production data (copies only); the coverage rule itself (authorizations only, ASSESSMENT-8); the reliability rule (ASSESSMENT-9). Excluded: city names, time-zone-only restrictions, EU freedom of movement for coverage.

**Acceptance criteria.**
- The seven labels above resolve to the right countries and the two stay unknown, in one targeted test with the real labels; a label containing a vocabulary name with "and" is not split.
- On a fresh production copy with the real profile: every vacancy whose verdict changes is listed with its label and quote and read by the executor; no vacancy changes for any other reason; vacancy 1136 stays "not covered".
- Stored results are re-evaluated after the change without a model call (rules identity or evaluator version changes; the executor verifies which and bumps a version only if needed).
- Spec `docs/specs/quick-match.md` §4.1 describes the label forms.

- [ ] **N.1 — Label forms** (30–40 min): the rule above, the configuration keys, spec §4.1, one targeted test. Needed because these labels are the reason the nine vacancies stay "?".
- [ ] **N.2 — Check on a production copy** (15–20 min): extend the gated probe `backend/internal/matchsvc/live_restriction_test.go`; report the counts "?" → ✕ and "?" → ✓ and every changed vacancy. Needed to show the outcome on real data and catch over-splitting.

**Estimate.** Complexity 2: one pure function and vocabulary; the risk is wrong splitting, covered by the real-label test and the probe. AI time: preparation 15 min (done), implementation 45–60 min, coordinator acceptance 15 min; total about 1.25–1.5 h. No paid services, no model calls. Release is separate and needs the owner's «да».
