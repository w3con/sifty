# SETTINGS-9 — Geography editor shows location not set while view shows Lille, France

**Epic:** settings
**Created:** 2026-11-13
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Geography and authorization
Edit
Current locationLille, France
Legacy location text, used until a structured country/city is added. It is never parsed into a country.
Preferred regionsNot set
Preferred work locationsNot set

но при нажатии edit: пишет что location not set.

ЗАпиши как баг, пока это не важно.

```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Coordinator note (2026-11-13)

Likely cause (not verified): the view shows the legacy free-text `geography.current` ("Lille, France", accepted from the CV), while the editor reads only the structured country/city fields, which are empty, and says "not set". Saving from the editor could then drop the legacy text. Fix at planning: the editor shows the legacy text and offers to turn it into a structured country/city (France · Lille), never discarding it silently. Low priority per the owner («пока это не важно»); fits naturally into SETTINGS-5 S.1 (My data). Estimate: complexity 2, 20–30 min.
