# NAVIGATION-2 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-05 — 1.1

Check: `cd sifty && go test -race ./internal/config/... ./internal/store/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/config	(cached)
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-05 — 1.2

Check: `cd sifty && go test -race ./internal/config/... ./internal/store/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/config	(cached)
ok  	github.com/w3con/sifty/internal/store	(cached)
```


## 2026-11-05 — 1.3

Check: `cd sifty && go test -race ./internal/config/... ./internal/store/... ./internal/background/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/config	(cached)
ok  	github.com/w3con/sifty/internal/store	(cached)
ok  	github.com/w3con/sifty/internal/background	(cached)
```


## 2026-11-05 — 2.1

Check: `cd sifty && go test -race ./internal/store/... ./internal/background/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	(cached)
ok  	github.com/w3con/sifty/internal/background	(cached)
```


## 2026-11-05 — 2.2

Check: `cd sifty && go test -race ./internal/store/... ./internal/background/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	(cached)
ok  	github.com/w3con/sifty/internal/background	(cached)
```


## 2026-11-05 — 2.3

Check: `cd sifty && go test -race ./internal/store/... ./internal/background/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/store	(cached)
ok  	github.com/w3con/sifty/internal/background	(cached)
```


## 2026-11-05 — 3.1

GET/PUT /api/search/expressions implemented via YAML node surgery (updateSearchExpressionsInYAML). POST /api/block-rules retired (returns 400). Fixture 38 updated to expect 400+error. server tests green.


## 2026-11-05 — 3.2

GET/POST /api/search/sites, DELETE /api/search/sites/{name}, PATCH /api/search/sites/{name} implemented via updateSearchSiteInYAML ops (add/remove/enable/disable). Drift test normalisation added for {name} segment. Fixtures 43-48 added. Fixture 47 uses NonExistent site (returns 400, accepted by statusOK). Replay seeds search.yaml. server tests green.


## 2026-11-06 — 3.3

Added TestMigrationV14 (store): seeds v13 db with company+source+title block rules and a pre-existing stop_word, opens via store.Open, verifies company rules copied to stop_words (deduplicated), source rules deleted, company rules remain in block_rules, user_version=14. Added 12 integration tests in search_endpoints_test.go covering GET/PUT /api/search/expressions and GET/POST/DELETE/PATCH /api/search/sites. Full go test -race ./... passes, all packages green.

