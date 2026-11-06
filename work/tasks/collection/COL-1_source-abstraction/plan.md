---
estimate: 16
---
# COL-1 — plan

**Author:** coordinator
**Created:** 2026-11-04

## Statement of work

Build the whole Go collection layer: first the outbound-HTTP fetch foundation that is missing entirely in Go, then a light `Source` abstraction over it, then the real daily collection run. The fetch foundation is a single HTTP client with a server-side-request-forgery guard, optional per-source proxy, a response-size cap, CAPTCHA/403 detection, a listing/detail page cache, and a per-domain traffic discipline (minimum interval, backoff, cooldown on block) — reproducing the load-bearing parts of the retired `network.ts` and `traffic.ts`, not porting them line by line. Every source fetches only through this client, so the guards cannot be bypassed. On top of it, introduce the `Source` abstraction so that adding a new site later is a localised implementation behind a stable interface. Every source implements two operations: discover candidate postings from a listing into references, and resolve one posting into a normalised `model.JobInput`. The existing parse primitives (JSON-LD, feed, HTML links, Getro) and the jobspy worker become implementations behind this interface. Detail extraction runs as a waterfall — structured JSON-LD, then a deterministic platform adapter (Greenhouse, Lever, Ashby, Getro), then the generic HTML extractor — where every kept field must appear as exact page text and a source with no usable description ends in an honest pending state, never a model call. The daily run stub `defaultCollect` is replaced by a dispatch that iterates enabled sources from `config/search.yaml`, routes each to the right implementation, dedupes by canonical URL, upserts and enqueues. A model is not used for extraction anywhere in this card; the per-host learned-hint interface and its validate-on-use/invalidate-on-failure contract are written as types with no behaviour, as the seam COL-2 will fill.

## Steps

### Phase 0 — The fetch foundation

- [x] 0.1 — Add a single outbound-HTTP client type with a server-side-request-forgery guard (refuse private, link-local and loopback destinations unless explicitly configured), a response-size cap, and optional per-source proxy configuration; check: a table test proves the guard refuses private/loopback hosts and allows a public host, and a response over the cap is rejected.
- [x] 0.2 — Add a page cache to the client with distinct freshness windows for listing pages and detail pages, and CAPTCHA/403 detection that surfaces a typed blocked result; check: a second fetch within the window is served from cache without a network call, and a stubbed 403/CAPTCHA response produces the typed blocked result.
- [x] 0.3 — Add a per-domain traffic discipline over the client: a minimum inter-request interval per domain, exponential backoff, and a cooldown that blocks a domain after a 403 or CAPTCHA; check: a test with a fake clock proves two requests to one domain are spaced by at least the interval and that a blocked domain is refused until its cooldown elapses.

### Phase 1 — The interface and the generic source

- [x] 1.1 — Define the `Source` interface (discover listing → posting references; resolve reference → `model.JobInput`) and a small registry that selects an implementation by host/platform, with a default; check: the package compiles and a table test resolves a known host to its implementation and an unknown host to the default.
- [x] 1.2 — Wrap the existing JSON-LD, feed and HTML-link primitives as the generic source behind the interface, with the waterfall order JSON-LD → feed/HTML and the exact-quote validation preserved; check: fixture pages (one JSON-LD, one feed, one plain HTML listing) each produce the same postings through the interface as the primitives do directly.
- [x] 1.3 — Wrap the jobspy worker as the source for the boards it covers; check: a stubbed worker response is normalised to postings through the interface, matching the current `CollectViaJobspy` output.

### Phase 2 — Platform adapters

- [x] 2.1 — Add a deterministic Greenhouse adapter using its public JSON posting endpoint, selected by host fingerprint; check: a recorded Greenhouse response yields title, company, description and location with each field an exact quote, no model called.
- [x] 2.2 — Add a deterministic Lever adapter the same way; check: a recorded Lever response yields the four fields with exact-quote validation.
- [x] 2.3 — Add an Ashby adapter and fold the existing Getro handling in behind the interface; check: recorded Ashby and Getro fixtures each resolve through the registry to their adapter and produce valid postings.

### Phase 3 — Wire the run and name the future seam

- [x] 3.1 — Replace the `defaultCollect` stub with a run that iterates enabled sources from `config/search.yaml`, dispatches through the registry, fetches through the Phase 0 client, dedupes by canonical URL, upserts and enqueues; check: a run over a fixture source set upserts the expected postings and enqueues them, and a source with no readable description produces a pending-state record rather than an empty one or a model call.
- [x] 3.2 — Wire the single-URL paths through the same fetch and waterfall: `importURL` (behind `POST /api/import`) fetches and parses a real posting instead of writing the "Manual import" placeholder, and `EnrichFn` (behind `POST /api/jobs/{id}/read`) re-fetches a role's detail page and updates its description instead of being a no-op; check: against a fixture detail page, import produces a populated `JobInput` and enrich updates an existing role's description, both through the guarded client.
- [x] 3.3 — Write the learned-hint interface and its contract as behaviourless types: a per-host extraction hint, a validate-on-every-use step, and invalidation that triggers re-learning when the hint stops producing the required fields; check: the types compile, are documented as the COL-2 seam, and nothing in COL-1 reads or writes a hint.

## Acceptance criteria

- A single outbound-HTTP client exists with an SSRF guard, response-size cap, optional proxy, listing/detail page cache, CAPTCHA/403 detection and per-domain traffic discipline; every source fetches through it and cannot bypass the guard.
- A `Source` interface and host/platform registry exist; the JSON-LD/feed/HTML primitives and the jobspy worker are implementations behind it, producing the same results as before.
- Deterministic adapters for Greenhouse, Lever, Ashby and Getro resolve postings without any model call, with every kept field validated as exact page text.
- The daily collection run visits enabled sources through the dispatch, dedupes, upserts and enqueues, replacing the stub; a source without a usable description ends in an honest pending state.
- `POST /api/import` and the `POST /api/jobs/{id}/read` enrich path fetch and parse real postings through the guarded client, replacing the "Manual import" placeholder and the no-op `EnrichFn`.
- Model-based extraction remains off and the learned-template cache is not implemented; only its interface and invalidation contract are present, as the COL-2 seam.
- No change to assessment, preparation, answer generation, POST-blocking or browser discipline; the `sifty/` public-subtree rules hold.
