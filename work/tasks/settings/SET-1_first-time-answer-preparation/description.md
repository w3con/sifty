# SET-1 — First-time knowledge-base answer preparation during Settings setup

**Epic:** settings
**Created:** 2026-11-03
**Status:** founded
**Short card:** steps and acceptance criteria live here.

## Why this work exists

The default answers shown in the Application tab come from the knowledge base. That knowledge base is first populated when the user sets up the Settings tab: the workflow that turns the candidate's own material into reusable answers lives here, not in the Application epic. The Application epic (APP-3) only reads this store and writes refinements back to it; this card owns how it is first built.

The workflow itself is still to be designed with the owner. This card is a placeholder so the dependency and the boundary are recorded; its steps and acceptance criteria are written after that discussion.

## Open questions for the owner

Facts gathered before asking: the application already has a remembered-answers store (`<data dir>/answers/`, written by the "Remember" action on a single answer in the Application tab) and it is empty today; candidate material is configured in `candidate.yaml` (CV, leadership profile, positioning, an FAQ written for one company, the handover note, and public profiles), with review notes that flag unresolved facts (chronology overlap, relocation intent, French work authorisation).

1. Which store is "the knowledge base" for answers: the existing remembered-answers store, the Obsidian `kb/` of this repository, or something new?
2. What gets prepared first: a fixed list of common application questions (why this role, salary expectation, notice period, work authorisation, relocation, years of experience, links), the questions actually seen in inspected forms so far, or both?
3. Who writes the first version of each answer: the model drafts from the candidate material and the owner confirms each one, or the owner writes them?
4. Does an answer carry a scope: universal, per role type, per company, per language (English, French)?
5. For sensitive answers whose facts are unresolved in the candidate material (salary, work authorisation, relocation): may the model propose a draft, or are they always left for the owner?
6. What shape does it take in Settings: a one-time setup walk-through, or a list of answers the owner revisits and extends at any time?
7. How is a question in a real form matched to a stored answer: by the model with the match shown for confirmation, or only by exact wording?

## Steps

- [ ] To be defined: Settings answer-preparation workflow (discuss with owner before planning).

## Acceptance criteria

- To be defined once the Settings workflow is agreed.

## Must not touch

- To be defined.
