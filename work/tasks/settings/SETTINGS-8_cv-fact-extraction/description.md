# SETTINGS-8 — CV analysis loses skills, languages and work authorization

**Epic:** settings
**Created:** 2026-11-12
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
что не так?
ну там такой же pdf. ок - вот он
(owner uploaded "SENIOR _ PRINCIPAL BLOCKCHAIN ENGINEER.pdf" to the profile; the same CV as AT_Blockchain_engineer.pdf)

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator diagnosis (2026-11-12, production document 138d1557…, qwen3.5:9b)

The CV (2 pages, 4 051 characters) produced 12 proposals, all `proposed`:
- **Kept, correct:** display name, headline, 7 experience entries (Futureverse, Immutable, ThinkAgents.ai, Holvi.finance, Blockchain Labs, Pilier Network, Validité), target roles (a proposal inferred from the title).
- **Wrong:** "Lille, France" proposed as a second `identity.headline` (it is the current location); `workModes.limits = "Team size < 10"` invented from "an approximately eight-person product team".
- **Lost by validation (in `analysis_metadata.skipped`):** every skill (3 × "skills must be an object with key, name and level" — the model returned plain names: Solidity, EVM, Foundry, Go, TypeScript, Rust…), languages ("must be an object with name and optional level" — English fluent, Russian native, French B2), work mode ("unsupported value \"Remote\"" — capitalised), plus empty compensation / hours / availability fields the CV does not state (correctly skipped, but the model should not emit them).
- **Not proposed at all:** work authorization ("New Zealand citizen · EU resident") and current location — exactly what ASSESSMENT-8 needs; education (the profile has no section).
- **Duplicates:** the profile already holds 12 experience entries (revision 24); the 7 proposals are not matched against them, so accepting them would duplicate roles.
- Separate, same pipeline: the LinkedIn source is in error "context overflow: 34853 > 32000".

## Proposed fix (to plan)

1. **Shape the answer, not the validator:** give the model the JSON schema of each field (Ollama structured output) so skills come as {name, level} and languages as {name, level}; normalise case of enum values ("Remote" → "remote") in code before validation; a plain skill name becomes {name, level: unstated} rather than being dropped.
2. **Location and authorization:** map "citizen", "resident", "work permit", "visa" lines to `geography.workAuthorizations` and the address line to `geography.current`, with quotes; never to the headline.
3. **No invented limits:** a work-mode limit, dealbreaker or condition needs a quote that states it as the candidate's requirement; descriptions of past teams are not limits.
4. **Do not emit empty fields:** fields the document does not state are omitted, not sent empty.
5. **Match experience to existing entries:** a proposal for a company and overlapping period already in the profile is shown as "already in profile" or as a proposed update of that entry, not as a new one.
6. **Long sources:** split a document over the context budget into parts and merge proposals (fixes LinkedIn), or raise the budget where the model allows it.

Spec: profile spec `docs/specs/profiles.md` §2 (facts) to state how facts are proposed; a short section to add at planning.

## Estimate (preliminary)

Complexity 5, AI time 2–3 h, local model only, re-run on this CV and on LinkedIn as the check. Related: ASSESSMENT-8 needs item 2; SETTINGS-5 S.1 moves these facts to "My data".

## Owner, 2026-11-12 (after the diagnosis)

«да, и не только. например, у меня уже задано имя, но предлагается его задать еще раз. Это наверное нормально? А вот поменять Headline руками я не могу.»

- **Same value proposed again** (name, and any field): a proposal equal to the current value (after trimming and case for names) is recorded as "matches your profile" and not shown as a decision; a different value is a proposed change with both values side by side. Item 5 above becomes this general rule, experience entries included.
- **Headline cannot be edited:** `backend/internal/profile/contract.go` keeps `identity.headline` as "legacy data … the current editor no longer offers it", yet the extractor proposes it and the matcher uses it (MatchFacts). Restore an editable Headline field in the identity section now (+10–15 min). In SETTINGS-5 the headline becomes a profile field (each profile has its own: "Senior Blockchain Engineer", "Engineering leader") — noted there.

Estimate now 2.25–3.25 h.
