# ASSESSMENT-8 plan (coordinator, 2026-11-12)

Owner approved the order («да», 2026-11-12). Diagnosis and proposal: description.md. Depends on SETTINGS-8 (released: work authorizations and current location are now proposed from the CV; the owner accepts them in the profile).

**Backend only.** INBOX-2 (not merged yet) rebuilds the vacancy screen and its Location tile; this card must not touch `frontend/`. The tile's colour follows automatically once INBOX-2 reads the verdict this card stores; if it does not, that is a 10-minute follow-up after both are merged.

- [ ] **G.1 — Region vocabulary in configuration** (15–20 min): region names and aliases → sets of countries (US, Canada, North America, Americas, LATAM, EU, EEA, Europe, UK, APAC, "worldwide / anywhere"); one file next to the other quick-match configuration; no hard-coded countries in code.
- [ ] **G.2 — Restriction from the vacancy** (30–40 min): from the merged location facts (source field, section code, the model's `location` with its quote) decide: unrestricted (worldwide / anywhere), restricted to a set of countries ("Remote (US/Canada only)", "Remote — EU", "must be based in", "US-based"), or unknown. Ambiguity stays unknown (spec quick-match §4.1).
- [ ] **G.3 — Eligibility and the geography group** (30–40 min): compare the restriction with the owner's work authorizations and current location → covered / not covered / unknown. The geography group takes part in Fits you whenever the vacancy states a restriction and the owner has authorizations or a current location (not only when the owner set a desired place). Not covered → a mismatch line ("Remote only in US/Canada — you can work in EU, New Zealand"); unknown → To clarify. A remote work mode with a regional restriction never counts as a plain remote match. Store the verdict with the location fact so views can colour it.
- [ ] **G.4 — Check** (20–30 min): one targeted test (restricted remote vs authorizations: covered, not covered, unknown); on a fresh production copy with the owner's accepted authorizations: vacancy 1136 (Relevant Healthcare) and the count of vacancies whose Fits you changes, before/after. Spec `docs/specs/quick-match.md` §4.1 and §14.1 updated in the same commit.

Estimate: complexity 4, AI time 1.5–2 h. No model calls, no paid services. Must not touch: `frontend/`, INBOX-2 and its worktree, production data (copies only).
If the owner has not accepted the authorizations by G.4, use the copy's profile with the facts accepted there and say so in the report.
