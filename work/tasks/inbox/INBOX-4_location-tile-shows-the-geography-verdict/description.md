# INBOX-4 — Location tile shows the geography verdict

**Epic:** inbox
**Created:** 2026-11-13
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
10. Плитка «Место» должна показывать оценку по географии? Оба условия для этого уже выпущены. Да. Это отдельная маленькая карточка, после переделки отображения. - согласен со всеми твоими предлжоениями ... Заводить карточки? - да
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator

Coordinator (2026-11-13): the words above combine the coordinator's question 10 and recommendation (OPS-11 combined questions) with the owner's answers «14 вопросов - согласен со всеми твоими предлжоениями» and «да и да»; "..." marks the coordinator's shortening. Outcome: the Location tile of the vacancy view shows the geography verdict of Fits you (covered ✓ / not covered ✕ / unknown ?), the same as the number; on the INBOX-2 branch `band.go` leaves the verdict empty. After the INBOX-2 release; spec vacancy-view.md (draft in OPS-11 `specs-v2/`). Not planned, not estimated.

## Owner, 2026-11-13: unrestricted remote counts as a match

Owner: «объясни мне, почему Location Anywhere from listing не засчитывается как позитивный? если у меня в профиле стоит "Нигер" на пример, как желаемая локация, или Северный полюс - то Anywhere и Remote - мне подходит.»

Coordinator's finding: the Location tile has no verdict yet (this card), and Fits you leaves an unrestricted remote vacancy out of the geography group (`backend/internal/quickmatch/geography.go:40`: for a profile without preferred locations, `remoteUnrestricted` returns not applicable), so a vacancy restricted to the EU and covered by the FR authorization scores higher than "Anywhere"; with a preferred location, "Anywhere" has no country or city and gets unknown (0.5). Coordinator's three questions: (1) replace ✓/✕ buttons by + / − and defer Dismiss — INBOX-2; (2) count "Anywhere" as a geography match, and bare "Remote" too or as unknown (objection raised: many US companies write bare "Remote" and restrict in the text); (3) put the fix into this card. Owner: «да. напиши INBOX-2 агенту сам, спланируй другие карточки. И запусти что там следующее по roadmap.»

Coordinator's reading of «да» for question 2: both "Anywhere" and bare "Remote" count as a match, as the owner's own words say «Anywhere и Remote - мне подходит». In the code bare "Remote" without a location label is already the same class as "Anywhere" (`remoteUnrestricted`), so one rule covers both.

## Plan (coordinator, 2026-11-13)

Approved by the owner 2026-11-13: «да ... спланируй другие карточки. И запусти что там следующее по roadmap.» Stage G runs now (it touches only the scoring code, which the INBOX-2 branch does not change); stage T waits for the INBOX-2 release.

**Outcome.** A remote vacancy without a regional restriction ("Anywhere", "Worldwide", "Remote (Global)", or remote with no location stated) counts as a geography match in Fits you and shows a Matches line, whatever the owner's preferred locations, work authorizations or current location; after INBOX-2 the Location tile shows the same verdict as the number (✓ covered or unrestricted, ✕ not covered, ? unknown).

**Rule (stage G).** In `evaluateGeography` (`backend/internal/quickmatch/geography.go`): when the vacancy is `remoteUnrestricted` and the owner has any geography data (preferred locations, a work authorization or a current location), the geography group is applicable and gets one member with utility 1 and a new reason `remote_unrestricted` placed in the Matches section (`fit.go` `goalReasonSection`); with preferred locations this member replaces the location comparison, which cannot compare a label without a country or city. `remoteNoRemote`, `remoteRestricted` and `remoteUnknownLabel` are unchanged. Owner without any geography data: unchanged (not applicable). The line's job value is the vacancy's label, or "remote, no region stated" when there is none.

**Must not touch.** `frontend/`; `.worktrees/INBOX-2` and its branch; production data (copies only); the coverage rule for restricted remote (authorizations only, ASSESSMENT-8); the work mode comparison in `conditions.go`; region vocabulary.

**Acceptance criteria.**
- Targeted Go test: unrestricted remote with (a) authorizations only, (b) a preferred location such as Niger, (c) no geography data, (d) bare remote with no label; restricted covered / not covered / unknown cases unchanged.
- On a fresh production copy with the real profile: counts of vacancies whose geography verdict and Fits you change, a sample of ten read by the executor, no change for restricted or non-remote vacancies; 1136 stays not covered, 17 stays not covered.
- Stored results are re-evaluated without a model call (the executor verifies whether rules identity or evaluator version changes and bumps a version only if needed).
- Spec `docs/specs/quick-match.md` §14.1 (geography group applicability) and §4.1 describe the rule.

- [ ] **G.1 — Unrestricted remote is a match** (30–40 min): the rule, the reason code, spec §14.1 and §4.1, the targeted test.
- [ ] **G.2 — Check on a production copy** (15–20 min): extend the gated probe `backend/internal/matchsvc/live_restriction_test.go` or a sibling; report the counts and the sample.
- [ ] **T.1 — Location tile verdict** (after INBOX-2 release, 20–30 min): `band.go` fills the Location tile verdict from the geography group (covered or unrestricted ✓, not covered ✕, unknown or not applicable ?); spec `vacancy-view.md` rule 11.

**Estimate.** Complexity 2. Stage G: implementation 45–60 min, acceptance 15 min. Stage T: about 45 min including acceptance, after INBOX-2. No paid services, no model calls. Release is separate and needs the owner's «да».
