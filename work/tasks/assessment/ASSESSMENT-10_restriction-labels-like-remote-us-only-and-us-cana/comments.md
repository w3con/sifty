# ASSESSMENT-10 — comments

Captured from the owner's report on 2026-11-13. No plan or work has started.

## 2026-11-13 — N.1 — Label forms

Check: `cd backend && go test ./internal/quickmatch/ -run TestResolveRestrictionLabelForms -v 2>&1`
Exit: 0

```text
=== RUN   TestResolveRestrictionLabelForms
=== RUN   TestResolveRestrictionLabelForms/vacancy_17:_paren_extraction_+_trailing_qualifier_strip
=== RUN   TestResolveRestrictionLabelForms/vacancy_1153:_comma_split_+_trailing_qualifier_strip_for_each_part
=== RUN   TestResolveRestrictionLabelForms/vacancy_1175:_state_list_split_on_comma_and_'or',_each_state_→_US
=== RUN   TestResolveRestrictionLabelForms/vacancy_1196:_'and'_separator
=== RUN   TestResolveRestrictionLabelForms/vacancy_1201:_paren_extraction,_'-based'_strip,_'remote_within_the'_prefix_strip
=== RUN   TestResolveRestrictionLabelForms/vacancy_1279:_multi-country,_'or'_separator,_'The_Netherlands'_alias
=== RUN   TestResolveRestrictionLabelForms/vacancy_1294:_'&'_separator
=== RUN   TestResolveRestrictionLabelForms/vacancy_1119:_city_names_NYC_and_Dallas_are_not_in_the_vocabulary
=== RUN   TestResolveRestrictionLabelForms/vacancy_1281:_time-zone_statement_—_no_resolvable_structure
=== RUN   TestResolveRestrictionLabelForms/vocabulary_entry_'Americas'_name_contains_'and':_full-label_lookup_must_succeed_before_split
--- PASS: TestResolveRestrictionLabelForms (0.00s)
    --- PASS: TestResolveRestrictionLabelForms/vacancy_17:_paren_extraction_+_trailing_qualifier_strip (0.00s)
    --- PASS: TestResolveRestrictionLabelForms/vacancy_1153:_comma_split_+_trailing_qualifier_strip_for_each_part (0.00s)
    --- PASS: TestResolveRestrictionLabelForms/vacancy_1175:_state_list_split_on_comma_and_'or',_each_state_→_US (0.00s)
    --- PASS: TestResolveRestrictionLabelForms/vacancy_1196:_'and'_separator (0.00s)
    --- PASS: TestResolveRestrictionLabelForms/vacancy_1201:_paren_extraction,_'-based'_strip,_'remote_within_the'_prefix_strip (0.00s)
    --- PASS: TestResolveRestrictionLabelForms/vacancy_1279:_multi-country,_'or'_separator,_'The_Netherlands'_alias (0.00s)
    --- PASS: TestResolveRestrictionLabelForms/vacancy_1294:_'&'_separator (0.00s)
    --- PASS: TestResolveRestrictionLabelForms/vacancy_1119:_city_names_NYC_and_Dallas_are_not_in_the_vocabulary (0.00s)
    --- PASS: TestResolveRestrictionLabelForms/vacancy_1281:_time-zone_statement_—_no_resolvable_structure (0.00s)
    --- PASS: TestResolveRestrictionLabelForms/vocabulary_entry_'Americas'_name_contains_'and':_full-label_lookup_must_succeed_before_split (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/quickmatch	0.171s
```

## 2026-11-13 — N.2 — Check on a production copy (corrected)

True before/after diff using `git archive b6e7c58 backend` (pre-N.1 state) for the before binary and the current branch for the after binary. Both compile and run against the same scratch DB copy.

Before step (exit 0):

```text
cd <scratch>/before_ASSESSMENT10/backend
SIFTY_PROBE_DATA_DIR=.../scratch/data SIFTY_VERDICT_OUT=.../verdicts_before.tsv \
  go test ./internal/matchsvc/ -run TestDumpRestrictionVerdicts -count=1 -v
wrote 1938 rows to .../verdicts_before.tsv
--- PASS: TestDumpRestrictionVerdicts (2.11s)
```

After step (exit 0):

```text
cd <worktree>/backend
SIFTY_PROBE_DATA_DIR=.../scratch/data SIFTY_VERDICT_OUT=.../verdicts_after.tsv \
  go test ./internal/matchsvc/ -run TestDumpRestrictionVerdicts -count=1 -v
wrote 1938 rows to .../verdicts_after.tsv
--- PASS: TestDumpRestrictionVerdicts (1.86s)
```

Diff (8 lines changed):

```text
17   unknown → not_covered   "Remote (US only)"                     — paren extraction → US only → US; correct
1153 unknown → not_covered   "US only, US time zones"               — comma split + "time zones" qualifier strip; correct
1175 unknown → not_covered   "Alabama, Florida, ... or Virginia"    — comma + "or" separator → state aliases → US; correct
1196 unknown → not_covered   "US and Canada"                        — "and" separator; correct
1201 unknown → not_covered   "US-based (remote within the US)"      — paren + "remote within the" prefix + "-based" qualifier; correct
1262 unknown → not_applicable "Remote (Global)"                     — paren extraction → "Global" = worldwide alias; vacancy is globally unrestricted (correct resolution; label was misread as unknown before)
1279 unknown → not_covered   "UK, Ireland, Germany, Portugal, Spain or The Netherlands" — "or" separator + "The Netherlands" alias; correct
1294 unknown → not_covered   "US & Canada"                         — "&" separator; correct
```

Unchanged: 1119 (unknown), 1281 (unknown), 1136 (not_covered). All correct.
