---
estimate: 9
---
# SET-3 — plan

**Author:** coordinator
**Created:** 2026-11-04

## Statement of work

The owner must be able to pick, as a persistent default, which configured provider runs each model stage, from the Configuration tab rather than by editing YAML. Fit assessment has a YAML-only default (`stages.assessment`) and only a per-vacancy override in the catalogue; application preparation has no stage at all and is hardcoded to the `openai` provider; extraction runs on scripts and uses a model only as a fallback gated by the `model_extraction` runtime flag. This card adds a `preparation` stage to the Go config and its validation, changes the preparation worker's default provider from the hardcoded `openai` to `stages.preparation` while keeping the per-operation override, adds a small endpoint to read the stage→provider map with the configured provider list and to set one stage (writing the `stages` block and reloading live via SET-2), and adds a "Stage models" section to the Configuration tab with one labelled dropdown per stage populated from the configured providers. Assessment and preparation are the required fields; extraction appears as a third field labelled as the fallback used only when `model_extraction` is enabled. It does not remove the per-role assessment override, add providers, or change config formats beyond the new stage key.

## Steps

### Phase 1 — Backend: a preparation stage

- [x] 1.1 — Add `Preparation` to `config.Providers.Stages` and include it in both validation loops (config.go and save.go), tolerating an empty value as "unset"; check: a providers.yaml with `stages.preparation` naming a defined provider validates, and one naming an unknown provider is rejected.
- [x] 1.2 — Change `defaultPrepareJob` to default the provider to `cfg.Providers.Stages.Preparation` when the operation request carries none, keeping the request override and erroring clearly if neither is set; check: a preparation operation with no request provider runs on the configured `stages.preparation`, and a request-level provider still overrides it.

### Phase 2 — Backend: read and set stage defaults

- [x] 2.1 — Add an endpoint that returns the current stage→provider selections and the list of configured providers (name, model); check: the endpoint returns assessment, preparation and extraction selections and every configured provider.
- [x] 2.2 — Add an endpoint that sets one stage's provider by writing the `stages` block in providers.yaml and reloading live, rejecting an unknown provider; check: setting preparation to a valid provider updates the file and the next read reflects it with no restart, and an unknown provider is rejected.

### Phase 3 — Frontend: dropdowns in Configuration

- [x] 3.1 — Add a "Stage models" section to the Configuration tab with a labelled `<select>` per stage (Fit assessment, Application preparation, and Extraction labelled as the `model_extraction` fallback), options being the configured providers shown as `name — model`, current value preselected, onChange saving via the set endpoint and reflecting the result; check: the built UI shows the three selects populated with the configured providers and the current selections.
- [x] 3.2 — Rebuild the frontend bundle so the Go-served app carries the new section; check: `app/dist` is rebuilt and the served Configuration page renders the stage-model selects.

### Phase 4 — Live verification

- [x] 4.1 — On the running demo over the owner's real data dir, set Application preparation to a chosen provider and confirm a new preparation run uses it with no restart, and set Fit assessment and confirm the default changes without touching any vacancy; check: a recorded run shows preparation using the chosen provider and the assessment default changed from Configuration alone.

## Acceptance criteria

- The Configuration tab offers a dropdown per stage, each listing the configured providers, that sets the persistent default for fit assessment and application preparation (and extraction as the labelled fallback), saved live with no restart.
- Application preparation runs on the selected `stages.preparation` provider by default, with the per-operation override still honoured; the hardcoded `openai` default is gone.
- Setting a stage rejects an unknown provider and never corrupts the providers file; the per-role assessment override in the catalogue is unchanged.
- On the running demo, changing a stage's model from Configuration takes effect on the next run without a restart.
