# ASSESSMENT-1 — User calibration of Quick Match weights

**Epic:** assessment
**Created:** 2026-11-10
**Status:** backlog
**Kind:** idea

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Весам полезно придать смысл через сравнение случаев: что существеннее — небольшое отклонение по оплате, другая роль или несовместимая география? Такой подход к выбору весов описан в руководстве по многокритериальной оценке (https://analysisfunction.civilservice.gov.uk/policy-store/an-introductory-guide-to-mcda/). Оно также предупреждает против произвольных процентов без определённых шкал.  - это наверное будет отдельная фича (можешь записать в бэклог и приложить эту ссылку) для "калибровка весов пользователем".
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Deferred scope

Deferred from [the Quick Match behavior specification](../../../../docs/specs/quick-match.md), sections 10 and 12. Quick Match v1 uses fixed weights and does not expose a user editor; this card owns the later calibration feature. It is not authorized by that specification and must not be added silently to v1.

Method reference from the owner's wording: [An Introductory Guide to Multi-Criteria Decision Analysis](https://analysisfunction.civilservice.gov.uk/policy-store/an-introductory-guide-to-mcda/) (Government Analysis Function, published 1 May 2024). It recommends eliciting weights through comparisons and swing weighting, and warns against arbitrary percentages without defined scales.

## Provisional estimate — not approved

- **Complexity:** 5 — preference elicitation, UI and configuration/score-version coupling; moderate uncertainty because the method and user interaction are not yet designed.
- **AI preparation:** approximately 10–20 minutes to define cases, scales and interaction.
- **AI implementation:** approximately 35–60 minutes for the elicitation flow, configuration and tests.
- **AI acceptance:** approximately 10–20 minutes for review and correction.
- **Total AI elapsed:** approximately 55–100 minutes, excluding owner waiting. Existing subscription; monetary usage cost unknown. Not approved and not scheduled.
