# SET-3 — Stage model selection: pick the model per stage from a dropdown in Configuration

**Epic:** settings
**Created:** 2026-11-04
**Status:** founded

## Why this work exists

The owner needs to choose, as a persistent default, which configured provider runs each model stage — fit assessment and application preparation — and to do it from the Configuration tab, not by editing YAML. Today that is not possible:

- **Fit assessment** has a default in `stages.assessment` (providers.yaml), but the only way to set it is editing raw YAML. The per-role provider dropdown in the catalogue is a one-off override for a single vacancy, not a global default, so it does not answer the need.
- **Application preparation** has no stage at all: the preparation worker is hardcoded to the `openai` provider (`defaultPrepareJob`, preparation.go). It cannot be chosen.
- **Extraction** is done by scripts (`detailFromHtml`); a model runs only as a fallback and only when the runtime flag `model_extraction` is enabled (off by default), so `stages.extraction` is inert in normal operation.

This card adds a `preparation` stage, wires the preparation worker to it, and gives the Configuration tab a labelled dropdown per stage, each populated from the configured providers, saved live (the server reloads config on save as of SET-2).

## Must not touch

- The per-role assessment override in the catalogue (RoleContent) stays as it is; this card adds a global default, it does not remove the override. The config validation, backup and file formats from earlier cards. The `model_extraction` runtime flag's meaning. The provider definitions themselves (this card selects among them, it does not add providers).

## Notes

- Go config: `config.Providers.Stages` has `Assessment` and `Extraction` only (config.go); add `Preparation`. Validation lives in config.go (~line 209) and save.go (~line 172) — both iterate the stage map and must include preparation.
- Preparation provider resolution: `defaultPrepareJob` (preparation.go ~348) defaults to `"openai"` when the request carries no provider; change the default to `cfg.Providers.Stages.Preparation`, keeping the per-operation request override.
- Assessment already falls back to `cfg.Providers.Stages.Assessment` (assessment.go ~197); `/api/assessment-options` already returns `selected: stages.assessment` and the provider list — reuse that shape.
- Setting a stage means rewriting the `stages` block in providers.yaml and reloading. Prefer a small dedicated endpoint (read the stage→provider map and the provider list; set one stage) over making the UI hand-edit YAML; it can reuse `config.Save` + the SET-2 reload, or write the stages block directly then reload.
- Frontend: `app/web/Settings.tsx`, the `panel==='settings'` block. Add a "Stage models" section above or beside the raw YAML editor with one `<select>` per stage, options = configured provider names (show the model, e.g. `luna — openai/gpt-6-luna`), current value = the stage's selection, onChange saves and reflects the new value. The UI is served from `app/dist` (built from `app/web`); a rebuild of the frontend is part of the work.
- Extraction: include it as a third dropdown, clearly labelled as the fallback used only when `model_extraction` is enabled. If the owner does not want it shown, drop that one field; the two required fields are assessment and preparation.
- Verify live on the running demo: set preparation to a chosen provider in Configuration, confirm a new preparation run uses it with no restart; set assessment and confirm the default changes without touching a vacancy.
