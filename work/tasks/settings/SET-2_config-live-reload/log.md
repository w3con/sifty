# SET-2 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-03 — 1.1

Check: `go test -race -count=1 ./internal/config/...`
Exit: 1

```
# ./internal/config/...
pattern ./internal/config/...: directory prefix internal/config does not contain main module or its selected dependencies
FAIL	./internal/config/... [setup failed]
FAIL
```


## 2026-11-03 — 1.1

Check: `cd sifty && go test -race -count=1 ./internal/config/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/config	1.216s
```


## 2026-11-03 — 1.2

Check: `cd sifty && go test -run TestHolderReload ./internal/config/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/config	(cached)
```


## 2026-11-03 — 2.1

Check: `cd sifty && go test -run TestProviderDiagnosticsReflectsNewModelAfterSave ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-03 — 2.2

Check: `cd sifty && go test -run TestPortChangeSaveReturnsRestartNotice ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	0.350s
```


## 2026-11-03 — 3.1

Check: `cd sifty && go test -run 'TestProviderDiagnosticsReflectsNewModelAfterSave|TestPortChangeSaveReturnsRestartNotice' ./internal/server/...`
Exit: 0

```
ok  	github.com/w3con/sifty/internal/server	0.367s
```

