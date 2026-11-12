# ASSESSMENT-8 — Restricted remote (US/Canada only) counted as a match

**Epic:** assessment
**Created:** 2026-11-12
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
смотри какой BS я нашел у себя
(screenshot: Relevant Healthcare — Lead Engineer, Hacker News, "Remote (US/Canada only) | Full-time"; Fits you 59 Partial; Matches: Role Lead Engineer, Work mode remote; To clarify: Salary, Contract type; Requirements: No requirements stated)

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator diagnosis (2026-11-12, production data, job 1136)

- The model read the condition correctly: `workMode = remote` and `location = "US/Canada"`, both quoting "Remote (US/Canada only)".
- Fits you used only three groups: role (utility 0.5), compensation (unknown) and conditions (work mode remote = 1). **There is no geography group**, because the owner's profile has no geography goal to compare with: `geography.desired`, `preferredLocations` and `workAuthorizations` are empty, `eligibility` is `["Yes"]` (not a usable value), `current` is "New Zealand / Europe". A group without an owner goal is left out (spec quick-match §14.1), so the US/Canada restriction had nothing to count against and "remote" alone counted as a full match.
- The role line is shown with ✓ under Matches although its utility is 0.5 (Lead Engineer vs target Technical Lead) — a partial shown as a match.
- "No requirements stated" is literally true for this Hacker News comment; the full posting is on the employer's page it links to.

## Proposed fix (to plan)

1. **Restricted remote is a location condition.** "Remote (US only)", "Remote — EU", "remote within X time zones" are eligibility constraints, not a plain remote. When the vacancy restricts where remote work may be done, Fits you compares the region with the owner's work authorizations and current location: not covered → geography mismatch (listed under Mismatches: "Remote limited to US/Canada — you are in Europe / New Zealand"); owner data missing → "To clarify: can you work from US/Canada?". The work-mode line must not show ✓ remote when the region excludes the owner.
2. **The geography group exists whenever the vacancy states a restriction and the owner has a current location or authorizations**, not only when the owner set a desired location.
3. **Partial lines are marked partial** (the role line here), not ✓.
4. Owner action (not code): fill work authorizations (e.g. EU, New Zealand) and clean `eligibility` in the profile; without them item 1 can only ask.
5. Later, not here: follow the link of a Hacker News comment to the full posting (COL-2, per-source rules).

Spec: quick-match §14.1 and §4.1 (ambiguity and geography) to be updated with item 1–2 at planning.

## Estimate (preliminary)

Complexity 4, AI time 45–75 min (region parsing of the extracted location against authorizations/current location with a small region vocabulary in configuration — US, Canada, EU, UK, time-zone bands; the group rule; partial marking; one targeted test; spec). Free.
