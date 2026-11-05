# COL-1 — execution log

Each entry records one closed step and the verbatim output of the check that proved it. Appended by `work-note`; never hand-edited to summarise.

## 2026-11-04 — 0.1 — Add a single outbound-HTTP client type with SSRF guard, response-size cap, and optional per-source proxy configuration

Check: `cd sifty && go test ./internal/fetch/ -run 'TestSSRFGuard|TestSizeCap' -v`
Exit: 0

```
=== RUN   TestSSRFGuard
=== RUN   TestSSRFGuard/loopback_IPv4
=== RUN   TestSSRFGuard/loopback_IPv6
=== RUN   TestSSRFGuard/RFC-1918_10.x
=== RUN   TestSSRFGuard/RFC-1918_172.16
=== RUN   TestSSRFGuard/RFC-1918_192.168
=== RUN   TestSSRFGuard/link-local_169.254
=== RUN   TestSSRFGuard/unique-local_IPv6_fc
=== RUN   TestSSRFGuard/unique-local_IPv6_fd
=== RUN   TestSSRFGuard/unspecified_IPv4
=== RUN   TestSSRFGuard/multicast
=== RUN   TestSSRFGuard/public_IPv4
=== RUN   TestSSRFGuard/public_IPv4_2
--- PASS: TestSSRFGuard (0.00s)
    --- PASS: TestSSRFGuard/loopback_IPv4 (0.00s)
    --- PASS: TestSSRFGuard/loopback_IPv6 (0.00s)
    --- PASS: TestSSRFGuard/RFC-1918_10.x (0.00s)
    --- PASS: TestSSRFGuard/RFC-1918_172.16 (0.00s)
    --- PASS: TestSSRFGuard/RFC-1918_192.168 (0.00s)
    --- PASS: TestSSRFGuard/link-local_169.254 (0.00s)
    --- PASS: TestSSRFGuard/unique-local_IPv6_fc (0.00s)
    --- PASS: TestSSRFGuard/unique-local_IPv6_fd (0.00s)
    --- PASS: TestSSRFGuard/unspecified_IPv4 (0.00s)
    --- PASS: TestSSRFGuard/multicast (0.00s)
    --- PASS: TestSSRFGuard/public_IPv4 (0.00s)
    --- PASS: TestSSRFGuard/public_IPv4_2 (0.00s)
=== RUN   TestSizeCap
--- PASS: TestSizeCap (0.00s)
=== RUN   TestSizeCapExactBoundary
--- PASS: TestSizeCapExactBoundary (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/fetch	(cached)
```


## 2026-11-04 — 0.2 — Add page cache with distinct TTLs for listing/detail pages and CAPTCHA/403 BlockedResult

Check: `cd sifty && go test ./internal/fetch/ -run 'TestCache|TestBlocked' -v`
Exit: 0

```
=== RUN   TestCacheListing
--- PASS: TestCacheListing (0.00s)
=== RUN   TestCacheDetail
--- PASS: TestCacheDetail (0.00s)
=== RUN   TestBlocked403
--- PASS: TestBlocked403 (0.00s)
=== RUN   TestBlockedCaptcha
=== RUN   TestBlockedCaptcha/<html>Please_verify_
=== RUN   TestBlockedCaptcha/<div_class='cf-chl-'
=== RUN   TestBlockedCaptcha/<div_class='captcha-
=== RUN   TestBlockedCaptcha/<title>Just_a_moment
--- PASS: TestBlockedCaptcha (0.00s)
    --- PASS: TestBlockedCaptcha/<html>Please_verify_ (0.00s)
    --- PASS: TestBlockedCaptcha/<div_class='cf-chl-' (0.00s)
    --- PASS: TestBlockedCaptcha/<div_class='captcha- (0.00s)
    --- PASS: TestBlockedCaptcha/<title>Just_a_moment (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/fetch	(cached)
```


## 2026-11-04 — 0.3 — Add per-domain traffic discipline: minimum interval, exponential backoff, 24h cooldown after a blocked result

Check: `cd sifty && go test ./internal/fetch/ -run 'TestTraffic' -v`
Exit: 0

```
=== RUN   TestTrafficInterval
--- PASS: TestTrafficInterval (0.00s)
=== RUN   TestTrafficCooldown
--- PASS: TestTrafficCooldown (0.00s)
=== RUN   TestTrafficBackoff
--- PASS: TestTrafficBackoff (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/fetch	(cached)
```


## 2026-11-04 — 1.1 — Define the Source interface and Registry; fallback resolves unknown hosts to the default

Check: `cd sifty && go test ./internal/source/ -run 'TestRegistry' -v`
Exit: 0

```
=== RUN   TestRegistry
=== RUN   TestRegistry/boards-api.greenhouse.io
=== RUN   TestRegistry/api.lever.co
=== RUN   TestRegistry/api.ashbyhq.com
=== RUN   TestRegistry/jobs.example.com
=== RUN   TestRegistry/api.lever.co#01
--- PASS: TestRegistry (0.00s)
    --- PASS: TestRegistry/boards-api.greenhouse.io (0.00s)
    --- PASS: TestRegistry/api.lever.co (0.00s)
    --- PASS: TestRegistry/api.ashbyhq.com (0.00s)
    --- PASS: TestRegistry/jobs.example.com (0.00s)
    --- PASS: TestRegistry/api.lever.co#01 (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/source	(cached)
```


## 2026-11-04 — 1.2 — GenericSource wrapping JSON-LD/feed/HTML-link primitives behind the Source interface

Check: `cd sifty && go test ./internal/source/ -run 'TestGenericSource' -v`
Exit: 0

```
=== RUN   TestGenericSourceJSONLD
--- PASS: TestGenericSourceJSONLD (0.00s)
=== RUN   TestGenericSourceFeed
--- PASS: TestGenericSourceFeed (0.00s)
=== RUN   TestGenericSourceHTMLLinks
--- PASS: TestGenericSourceHTMLLinks (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/source	(cached)
```


## 2026-11-04 — 1.3 — JobspySource wrapping CollectViaJobspy behind the Source interface

Check: `cd sifty && go test ./internal/source/ -run 'TestJobspySource' -v`
Exit: 0

```
=== RUN   TestJobspySource
--- PASS: TestJobspySource (0.02s)
PASS
ok  	github.com/w3con/sifty/internal/source	(cached)
```


## 2026-11-04 — 2.1 — Greenhouse adapter using boards-api.greenhouse.io JSON endpoint with fixture test

Check: `cd sifty && go test ./internal/source/ -run 'TestGreenhouse' -v`
Exit: 0

```
=== RUN   TestGreenhouse
--- PASS: TestGreenhouse (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/source	(cached)
```


## 2026-11-04 — 2.2 — Lever adapter using api.lever.co/v0/postings endpoint with fixture test

Check: `cd sifty && go test ./internal/source/ -run 'TestLever' -v`
Exit: 0

```
=== RUN   TestLever
--- PASS: TestLever (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/source	(cached)
```


## 2026-11-04 — 2.3 — Ashby adapter (api.ashbyhq.com/posting-api) and Getro adapter; both wired through registry

Check: `cd sifty && go test ./internal/source/ -run 'TestAshby|TestGetro' -v`
Exit: 0

```
=== RUN   TestAshby
--- PASS: TestAshby (0.00s)
=== RUN   TestGetro
--- PASS: TestGetro (0.00s)
PASS
ok  	github.com/w3con/sifty/internal/source	(cached)
```


## 2026-11-04 — 3.1 — Replace defaultCollect stub: dispatch iterates enabled sources, dedupes, upserts and enqueues

Check: `cd sifty && go test ./internal/background/ -run 'TestDefaultCollect' -v`
Exit: 0

```
=== RUN   TestDefaultCollect
--- PASS: TestDefaultCollect (0.01s)
PASS
ok  	github.com/w3con/sifty/internal/background	(cached)
```


## 2026-11-04 — 3.2 — Wire importURL and EnrichFn through fetch client and source registry, replacing stubs

Check: `cd sifty && go test ./internal/server/ -run 'TestImportURL|TestEnrichFn' -v`
Exit: 0

```
=== RUN   TestImportURLFetch
--- PASS: TestImportURLFetch (0.01s)
=== RUN   TestEnrichFn
--- PASS: TestEnrichFn (0.01s)
=== RUN   TestImportURLRoute
--- PASS: TestImportURLRoute (0.01s)
=== RUN   TestImportURL
--- PASS: TestImportURL (3.95s)
PASS
ok  	github.com/w3con/sifty/internal/server	(cached)
```


## 2026-11-04 — 3.3 — ExtractionHint, HintValidator, HintStore types written as behaviourless COL-2 seam

Check: `cd sifty && go build ./internal/source/... && grep -n 'ExtractionHint' sifty/internal/source/hint.go`
Exit: 2

```
grep: sifty/internal/source/hint.go: No such file or directory
```

