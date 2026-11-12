# SETTINGS-8 plan (coordinator, 2026-11-12)

Diagnosis and agreed fixes: description.md. Headline: owner «а нужно ли это поле? может и не нужно.» — coordinator: not needed now (role matching uses target roles and experience; a CV title matters only for application preparation, which is out of scope). The headline is not proposed from documents any more and not sent to the matcher; stored values stay untouched; no editor field. It can return as a per-profile field with application preparation.

- [ ] **F.1 — Structured answer** (40–60 min): per-field JSON schema in the extraction request (Ollama structured output); skills and languages as {name, level}; a bare skill name becomes {name, level: unstated}; enum values normalised in code before validation ("Remote" → "remote"); fields the document does not state are omitted, not sent empty.
- [ ] **F.2 — Location and work authorization** (20–30 min): "citizen", "resident", "work permit", "visa" lines → `geography.workAuthorizations`; the address line → `geography.current`; both with quotes; never to the headline.
- [ ] **F.3 — No invented conditions** (15–20 min): a work-mode limit, dealbreaker or condition needs a quote that states it as the candidate's requirement; descriptions of past teams or roles are not limits.
- [ ] **F.4 — Same value is not a question** (30–45 min): a proposal equal to the current value (trimmed; case-insensitive for names) is recorded as "matches your profile" and not shown as a decision; a different value is a proposed change with both values; an experience proposal for a company with an overlapping period already in the profile is matched to that entry (no duplicate).
- [ ] **F.5 — Headline out** (10–15 min): stop proposing `identity.headline`; remove it from the matcher's profile facts; keep stored data.
- [ ] **F.6 — Long sources** (30–45 min): a document over the context budget is split into parts and the proposals merged (fixes the LinkedIn source "context overflow 34853 > 32000").
- [ ] **F.7 — Check** (20–30 min): re-run analysis on a copy for the owner's CV (the PDF of 2026-11-12) and LinkedIn; table before/after (skills, languages, authorization, location, invented items, duplicates); one targeted test for F.4 (equal value is not a decision) and one for F.1 (bare skill name kept). Spec `docs/specs/profiles.md` §2: how facts are proposed.

Estimate: complexity 5, AI time 2.75–4 h, local model only. Must not touch: production data (copies only), INBOX-2, SETTINGS-5.
