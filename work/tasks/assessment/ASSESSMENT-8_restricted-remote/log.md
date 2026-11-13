# ASSESSMENT-8 executor log

## 2026-11-12–13 — G.1 through G.4

Executor: Claude Sonnet 4.6, branch work/ASSESSMENT-8, worktree .worktrees/ASSESSMENT-8.

**G.1:** `RegionEntry` added to `matchcontract.Rules`; `Regions []QuickMatchRegion` added to `config.QuickMatch`; `matchconfig.Rules()` maps config to contract; `profile.QuickMatchSnapshot` projects `CurrentLocation`; `config/runtime.yaml` now carries the full region vocabulary (26 entries: worldwide/anywhere, US, Canada, North America, Americas, LATAM, EU, EEA, Europe, UK, APAC, plus individual countries NZ, AU, DE, FR, NL, ES, SE, CH, PL, PT, IE, BR, IN, SG, JP). No countries are hard-coded in Go. *(Correction in next commit: vocabulary moved to embedded `config/regions.default.yaml`; `config/runtime.yaml` now holds only a comment stub — the binary carries the vocabulary.)*

**G.2:** `classifyRemoteRestriction` in `geography.go` classifies each vacancy as `remoteNoRemote` / `remoteUnrestricted` / `remoteRestricted` / `remoteUnknownLabel`. It checks all usable hiring location facts and resolves compound labels ("US/Canada" → ["US", "Canada"]) against the vocabulary. `resolveRestrictionLabel` tries stripped forms ("US only" → "US") and falls back to `worldwide` for unrestricted labels. `hasRemoteWithRestriction` is the cheap boolean used by conditions evaluation.

**G.3:** `evaluateGeography` gains a second path (profile has no preferred locations): checks restriction classification, then compares against owner's authorized countries. `ownerCoverageCountries` collects ISO codes from all eligible authorizations; the current location contributes to group applicability but not to coverage (spec §6.2 — residence is not a work right; corrected in next section). `evaluateRemoteRestriction` builds the geography group comparison with reason codes `remote_restricted_covered`, `remote_restricted_not_covered` or `remote_restricted_unknown`. `evaluateWorkModes` accepts a `restricted bool`; when the vacancy remote mode is restricted, the work mode comparison uses `work_mode_remote_restricted` (midpoint utility) instead of `work_mode_compatible`.

**G.4 targeted test:** `TestEvaluateRestrictedRemote` in `evaluate_test.go` — 7 sub-tests covering covered (NZ/FR owner, US/Canada restriction), covered via current location, not covered, unknown label, unknown label via needs-corroboration, unrestricted (worldwide), and no-vocabulary backward compat. All pass.

**G.4 data probe:** scratch copy of data at `.worktrees/ASSESSMENT-8/scratch/data`. Profile updated in scratch only: added NZ (eligible) and FR (eligible) work authorizations and NZ as current location (owner has neither in production profile as of 2026-11-13; probe used scratch-only authorizations as the plan allows). Config: region vocabulary applied from updated `config/runtime.yaml` template. `TestLiveRemoteRestriction` in `matchsvc/live_restriction_test.go` runs both the old rules (no regions) and the new rules (26 regions) over all 1 936 vacancies with usable text, merging stored model conditions. Result: 387 vacancies changed, all going down. Vacancy 1136 (Relevant Healthcare, Lead Engineer, "Remote (US/Canada only)"): stored score 59 (old code, work mode counted as full match); probe score 59→50 (new code, US/Canada restriction not covered by NZ/FR authorizations, work mode not counted as full remote match, geography group contributes mismatch).

**Spec:** `docs/specs/quick-match.md` §4.1 extended with restricted-remote behaviour (covered/not covered/unknown, region vocabulary from config, geography group applicability conditions); §14.1 extended with geography group applicability under case (b) — restriction present and owner has authorizations/current location.

**Tests:** `go test ./internal/quickmatch/ ./internal/matchcontract/ ./internal/config/ ./internal/matchsvc/` — all pass. `go vet ./...` — clean.

## 2026-11-13 — Correction: current location does not cover a restriction (spec §6.2)

Executor: Claude Sonnet 4.6, branch work/ASSESSMENT-8, worktree .worktrees/ASSESSMENT-8.

**Rule change applied:** `ownerCoverageCountries` in `geography.go` previously included the owner's current location country in the coverage set alongside eligible authorizations. Spec §6.2 states that residence is not a work right. The fix removes current location from coverage: only eligible work authorizations determine covered/not-covered. Current location still contributes to `present` (whether the group is applicable), so an owner with only a current location and no authorizations gets an "unknown" verdict (midpoint) rather than "not covered". An owner with authorizations but none in the restriction set remains "not covered".

**Test changes:** Sub-test "covered via current location — owner in Canada, vacancy US/Canada only" replaced with "current location alone does not cover — owner in Canada, no authorizations, vacancy US/Canada only" (asserts unknown verdict, group applicable). New sub-test "current location plus non-covering authorization is not covered" (CA current location + NZ authorization, US/Canada restriction → not covered). 9 sub-tests in `TestEvaluateRestrictedRemote`, all pass.

**Spec:** §4.1 Covered bullet changed to authorizations only; Unknown bullet extended with "owner has no eligible authorizations even when a current location is known"; applicability paragraph updated to note that current location makes the check applicable but does not cover. §14.1 same clarification added.

**New probe test:** `TestLiveRemoteRestrictionCurrentLocationOnly` in `matchsvc/live_restriction_test.go` runs the old-vs-new comparison with an overridden profile that has only current location NZ and no authorizations. Expected: restricted vacancies produce unknown verdicts (midpoint, not not-covered).

**Probe results (authorization profile):** 1936 vacancies evaluated, 387 changed, all down. Vacancy 1136 (Relevant Healthcare, Lead Engineer, "Remote (US/Canada only)"): 59→50. Unchanged from prior probe — the fix did not affect this profile because authorization countries already included NZ without needing current location.

**Probe results (location-only profile):** 1936 vacancies evaluated, 387 changed. Vacancy 1136: 59→50. Same change count and same score for 1136 as the authorization profile because (a) work mode changes from compatible to restricted for all 387 restricted vacancies regardless of coverage verdict, and (b) for vacancy 1136, the scratch profile has a preferred location incompatible with US/Canada, making geography utility 0 in both runs — the restriction member (unknown vs not-covered) is the second member of a two-member group that is already driven to 0 by the location mismatch.

**Tests (all):** `go build ./...`, `go vet ./...`, `go test -count=1 ./internal/quickmatch/ ./internal/matchcontract/ ./internal/config/ ./internal/matchsvc/` — all clean and passing.

## 2026-11-13 — build and vet after corrections

Check: `cd /Volumes/SSD/Dev/tknff/sifty/.worktrees/ASSESSMENT-8/backend && go build ./... && go vet ./... && echo 'build and vet clean'`
Exit: 0

```
build and vet clean
```

## 2026-11-13 — targeted tests after corrections

Check: `cd /Volumes/SSD/Dev/tknff/sifty/.worktrees/ASSESSMENT-8/backend && go test ./internal/quickmatch/ ./internal/matchcontract/ ./internal/config/ ./internal/matchsvc/ -timeout 60s`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/quickmatch	(cached)
ok  	github.com/w3con/sifty/internal/matchcontract	(cached)
ok  	github.com/w3con/sifty/internal/config	(cached)
ok  	github.com/w3con/sifty/internal/matchsvc	(cached)
```

## 2026-11-13 — live probe after corrections

Check: `cd /Volumes/SSD/Dev/tknff/sifty/.worktrees/ASSESSMENT-8/backend && SIFTY_PROBE_DATA_DIR=/Volumes/SSD/Dev/tknff/sifty/.worktrees/ASSESSMENT-8/scratch/data go test ./internal/matchsvc/ -run TestLiveRemoteRestriction -v -timeout 120s 2>&1 | grep -E '(profile:|config:|vacancies evaluated|vacancy 1136|score:|status:|--- PASS|--- FAIL)'`
Exit: 0

```
    live_restriction_test.go:71: profile: authorizations=2 currentLocation=&{NZ }
    live_restriction_test.go:73: config: regions=26
    live_restriction_test.go:161: === vacancies evaluated: 1936, changed: 387 ===
    live_restriction_test.go:171: === vacancy 1136: Relevant Healthcare / Lead Engineer ===
    live_restriction_test.go:172:   score: 59 → 50
    live_restriction_test.go:173:   status: scored → scored
--- PASS: TestLiveRemoteRestriction (2.57s)
```

## 2026-11-13 — Correction: current location does not cover a restriction (spec §6.2)

**Rule change:** `ownerCoverageCountries` removed the current location country from the coverage set. Only eligible work authorizations determine covered/not-covered. Current location still contributes to `present` (group applicability). An owner with only a current location and no authorizations gets unknown (midpoint) rather than not-covered.

**Test:** Rewrote "covered via current location" sub-test → "current location alone does not cover". Added "current location plus non-covering authorization is not covered". 9 sub-tests in `TestEvaluateRestrictedRemote`, all pass.

**Spec:** §4.1 Covered bullet — authorizations only. Unknown bullet — extended with "owner has no eligible authorizations even when a current location is known". Applicability paragraph — current location makes check applicable but does not cover. §14.1 — same clarification.

**New probe test:** `TestLiveRemoteRestrictionCurrentLocationOnly` — location-only profile override (no authorizations, NZ current location), old-vs-new comparison.

## 2026-11-13 — build, vet, tests after §6.2 correction

Check: `cd /Volumes/SSD/Dev/tknff/sifty/.worktrees/ASSESSMENT-8/backend && go build ./... && go vet ./... && go test -count=1 ./internal/quickmatch/ ./internal/matchcontract/ ./internal/config/ ./internal/matchsvc/`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/quickmatch	0.175s
ok  	github.com/w3con/sifty/internal/matchcontract	0.428s
ok  	github.com/w3con/sifty/internal/config	0.816s
ok  	github.com/w3con/sifty/internal/matchsvc	0.948s
```

## 2026-11-13 — live probe after §6.2 correction (both profiles)

Check: `cd /Volumes/SSD/Dev/tknff/sifty/.worktrees/ASSESSMENT-8/backend && SIFTY_PROBE_DATA_DIR=/Volumes/SSD/Dev/tknff/sifty/.worktrees/ASSESSMENT-8/scratch/data go test -count=1 -v ./internal/matchsvc/ -run 'TestLiveRemoteRestriction$|TestLiveRemoteRestrictionCurrentLocationOnly' -timeout 300s 2>&1 | grep -E '(authorizations|regions|vacancies evaluated|vacancy 1136|score:|status:|--- PASS|--- FAIL)'`
Exit: 0

```
    live_restriction_test.go:71: profile: authorizations=2 currentLocation=&{NZ }
    live_restriction_test.go:73: config: regions=26
    live_restriction_test.go:161: === vacancies evaluated: 1936, changed: 387 ===
    live_restriction_test.go:171: === vacancy 1136: Relevant Healthcare / Lead Engineer ===
    live_restriction_test.go:172:   score: 59 → 50
    live_restriction_test.go:173:   status: scored → scored
--- PASS: TestLiveRemoteRestriction (2.72s)
    live_restriction_test.go:233: profile override: authorizations=0 currentLocation=NZ
    live_restriction_test.go:234: config: regions=26
    live_restriction_test.go:319: === vacancies evaluated: 1936, changed: 387 (location-only profile) ===
    live_restriction_test.go:331: === vacancy 1136 (location-only): Relevant Healthcare / Lead Engineer ===
    live_restriction_test.go:332:   score: 59 → 50
    live_restriction_test.go:333:   status: scored → scored
--- PASS: TestLiveRemoteRestrictionCurrentLocationOnly (2.30s)
```

Authorization profile: 1936 evaluated, 387 changed, vacancy 1136: 59→50 (unchanged — fix did not affect profile with explicit authorizations, current location was redundant).
Location-only profile: 1936 evaluated, 387 changed, vacancy 1136: 59→50. Same score because (a) work mode changes from compatible to restricted for all 387 restricted vacancies regardless of coverage verdict, and (b) for vacancy 1136 the scratch profile has a preferred location incompatible with US/Canada, so the geography group is already driven to 0 by the location mismatch; the restriction member (unknown vs not-covered) is the second member of a two-member group whose first member is 0.
