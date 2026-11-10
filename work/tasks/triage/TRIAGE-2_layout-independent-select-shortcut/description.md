# TRIAGE-2 — Layout-independent Select shortcut

**Epic:** triage
**Created:** 2026-11-10
**Status:** backlog
**Kind:** bug

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
Можно сделать так, чтобы Y (select) работало на любой раскладке, не только на английской?
```

## Why this work exists

Observable outcome: pressing the physical Y key selects on every keyboard layout, not only the English one. Beneficiary: the owner triaging jobs. Concrete use: in the existing triage/select view, `н` on a Russian layout and `y` on an English layout, both with `event.code === 'KeyY'`, must run the same Select action as the current `y/KeyY` path; typing in an editable field must not select. Demonstrated within this task in the existing browser tooling against the running frontend.

Approval: owner «да. потом - Давай все закоммитим, squash commits / push?» approved the exact small plan shown in chat immediately before this card was opened — fix only Y/Select across layouts.

## Preparation — coordinator plan (transcribed)

Scope: use the physical `KeyboardEvent.code === 'KeyY'` instead of the locale-dependent `y`; preserve the existing editable-target, modifier (Ctrl/Alt/Meta), composition and dialog guards, and leave every other shortcut untouched. The existing `event.code` is the data/config/domain input, the existing frontend handler in `frontend/main.tsx` is the logic, and the existing button is presentation. No persisted data or settings are added.

Preliminary estimate: complexity 1; AI preparation ~1 min, fix ~1 min, frontend build 1–2 min, real-browser verification 1–2 min; total AI elapsed 3–5 min with the upper boundary measured from approval. No external monetary costs.

Excluded: other shortcut keys, backend changes, user-data changes, deploy/release, history rewrite, integration and push.

## Steps

- [ ] K.1 — Use physical KeyY for Select; build frontend and verify layout-independent action and input guard in a real browser.

## Acceptance criteria

- `y` on an English layout and `н` on a Russian layout, both with `event.code === 'KeyY'`, invoke Select.
- Shift behavior is unchanged and the existing editable/modifier/composition/dialog guards still suppress Select.
- The frontend build passes.

## Must not touch

Other shortcuts, backend, user data, persistence, settings, deploy, history rewrite, integration and push.
