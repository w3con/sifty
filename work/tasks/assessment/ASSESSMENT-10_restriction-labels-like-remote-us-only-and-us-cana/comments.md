# ASSESSMENT-10 — comments

Captured from the owner's report on 2026-11-13. No plan or work has started.

## 2026-11-13 — N.2 — Check on a production copy

Check: `TestLiveCompoundLabelResolution on scratch/data: 1934 evaluated; 6 changed ?→✕: 1153 'US only, US time zones', 1175 state list, 1196 'US and Canada', 1201 'US-based (remote within the US)', 1279 'UK, Ireland, Germany, Portugal, Spain or The Netherlands', 1294 'US & Canada'; 0 side effects; 1136 unchanged. Vacancy 17 'Remote (US only)' changes via paren extraction (code, not config), confirmed by unit test.`
Exit: 2

```
/bin/sh: -c: line 0: syntax error near unexpected token `('
/bin/sh: -c: line 0: `TestLiveCompoundLabelResolution on scratch/data: 1934 evaluated; 6 changed ?→✕: 1153 'US only, US time zones', 1175 state list, 1196 'US and Canada', 1201 'US-based (remote within the US)', 1279 'UK, Ireland, Germany, Portugal, Spain or The Netherlands', 1294 'US & Canada'; 0 side effects; 1136 unchanged. Vacancy 17 'Remote (US only)' changes via paren extraction (code, not config), confirmed by unit test.'
```
