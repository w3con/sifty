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

