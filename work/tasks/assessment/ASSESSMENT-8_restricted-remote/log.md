# ASSESSMENT-8 executor log

## 2026-11-12–13 — G.1 through G.4

Executor: Claude Sonnet 4.6, branch work/ASSESSMENT-8, worktree .worktrees/ASSESSMENT-8.

**G.1:** `RegionEntry` added to `matchcontract.Rules`; `Regions []QuickMatchRegion` added to `config.QuickMatch`; `matchconfig.Rules()` maps config to contract; `profile.QuickMatchSnapshot` projects `CurrentLocation`; `config/runtime.yaml` now carries the full region vocabulary (26 entries: worldwide/anywhere, US, Canada, North America, Americas, LATAM, EU, EEA, Europe, UK, APAC, plus individual countries NZ, AU, DE, FR, NL, ES, SE, CH, PL, PT, IE, BR, IN, SG, JP). No countries are hard-coded in Go.

**G.2:** `classifyRemoteRestriction` in `geography.go` classifies each vacancy as `remoteNoRemote` / `remoteUnrestricted` / `remoteRestricted` / `remoteUnknownLabel`. It checks all usable hiring location facts and resolves compound labels ("US/Canada" → ["US", "Canada"]) against the vocabulary. `resolveRestrictionLabel` tries stripped forms ("US only" → "US") and falls back to `worldwide` for unrestricted labels. `hasRemoteWithRestriction` is the cheap boolean used by conditions evaluation.

**G.3:** `evaluateGeography` gains a second path (profile has no preferred locations): checks restriction classification, then compares against owner's authorized countries and current location. `ownerCoverageCountries` collects ISO codes from all eligible authorizations plus the current location. `evaluateRemoteRestriction` builds the geography group comparison with reason codes `remote_restricted_covered`, `remote_restricted_not_covered` or `remote_restricted_unknown`. `evaluateWorkModes` accepts a `restricted bool`; when the vacancy remote mode is restricted, the work mode comparison uses `work_mode_remote_restricted` (midpoint utility) instead of `work_mode_compatible`.

**G.4 targeted test:** `TestEvaluateRestrictedRemote` in `evaluate_test.go` — 7 sub-tests covering covered (NZ/FR owner, US/Canada restriction), covered via current location, not covered, unknown label, unknown label via needs-corroboration, unrestricted (worldwide), and no-vocabulary backward compat. All pass.

**G.4 data probe:** scratch copy of data at `.worktrees/ASSESSMENT-8/scratch/data`. Profile updated in scratch only: added NZ (eligible) and FR (eligible) work authorizations and NZ as current location (owner has neither in production profile as of 2026-11-13; probe used scratch-only authorizations as the plan allows). Config: region vocabulary applied from updated `config/runtime.yaml` template. `TestLiveRemoteRestriction` in `matchsvc/live_restriction_test.go` runs both the old rules (no regions) and the new rules (26 regions) over all 1 936 vacancies with usable text, merging stored model conditions. Result: 387 vacancies changed, all going down. Vacancy 1136 (Relevant Healthcare, Lead Engineer, "Remote (US/Canada only)"): stored score 59 (old code, work mode counted as full match); probe score 59→50 (new code, US/Canada restriction not covered by NZ/FR authorizations, work mode not counted as full remote match, geography group contributes mismatch).

**Spec:** `docs/specs/quick-match.md` §4.1 extended with restricted-remote behaviour (covered/not covered/unknown, region vocabulary from config, geography group applicability conditions); §14.1 extended with geography group applicability under case (b) — restriction present and owner has authorizations/current location.

**Tests:** `go test ./internal/quickmatch/ ./internal/matchcontract/ ./internal/config/ ./internal/matchsvc/` — all pass. `go vet ./...` — clean.
