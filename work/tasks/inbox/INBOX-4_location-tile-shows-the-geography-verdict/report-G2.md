# INBOX-4 — G.2 production copy report

Executor run: 2026-11-13.

## Data copy

Production database backed up to `scratch/data/store.sqlite`; production config copied to `scratch/data/config`. No writes to `/Volumes/SSD/Dev/tknff/sifty/data` or ports 8080/8081.

## Before/after comparison (branch vs main@62bad1b)

`TestDumpRestrictionVerdicts` run against the base tree (commit `62bad1b`, temporary worktree) and the branch with the same `scratch/data` copy. One row per usable vacancy: `id\tverdict`.

- Vacancies with usable text: 1940
- Geography verdict changes: 15 (all `not_applicable` to `unrestricted`)
- No `covered`, `not_covered`, or `unknown` verdict changed.

Changed IDs: 910, 1134, 1135, 1143, 1168, 1169, 1174, 1182, 1190, 1220, 1235, 1262, 1263, 1267, 1297.

## Fits you score delta (TestLiveUnrestrictedRemote)

Old score simulated by removing geography data from profile (reproduces not-applicable path for unrestricted vacancies; production profile has locations=0).

- min delta: +6
- median delta: +12
- max delta: +16

## Sample of 10 changed vacancies (first by ID)

| ID | Company | Old | New | Delta | Label |
|----|---------|-----|-----|-------|-------|
| 910 | Bitwarden | 49 | 60 | +11 | remote, no region stated |
| 1134 | Descartes | 73 | 79 | +6 | remote, no region stated |
| 1135 | Innolitics | 27 | 43 | +16 | remote, no region stated |
| 1143 | Mastra | 50 | 61 | +11 | remote, no region stated |
| 1168 | Irash Tech | 36 | 50 | +14 | Worldwide |
| 1169 | Sourcegraph | 45 | 57 | +12 | remote, no region stated |
| 1174 | General Intelligence Holdings | 36 | 50 | +14 | remote, no region stated |
| 1182 | Matcha.fm | 27 | 43 | +16 | remote, no region stated |
| 1190 | Tether | 45 | 57 | +12 | Worldwide |
| 1220 | MaFi Games | 34 | 48 | +14 | remote, no region stated |

## Vacancy checks

- 1136 (Relevant Healthcare): verdict=`not_covered`, score=39. Unchanged.
- 17 (PrairieLearn Remote US): verdict=`not_covered`, score=25. Unchanged.

## Re-evaluation finding

`recomputeAll` in `matchsvc/service.go` calls `recomputeJob` for every vacancy unconditionally at startup. `recomputeJob` always recomputes from scratch. The `RulesIdentity` is a hash of the `Rules` struct; no new fields were added, so the hash is unchanged — but `recomputeAll` does not check stored identity before recomputing. Stored results are re-evaluated at startup without a model call. No version bump is needed.
