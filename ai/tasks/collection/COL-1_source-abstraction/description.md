# COL-1 — Source abstraction and ATS extraction waterfall

**Epic:** collection
**Created:** 2026-11-04
**Status:** founded

## Why this work exists

Job postings will come from many different sites, and the set will keep growing. Today the backend has deterministic parsing primitives (schema.org JSON-LD, RSS/Atom, generic HTML link extraction, a Getro special case) and a binding to the jobspy scraper worker, but these live as loose functions with no shared contract, and the daily collection run is a stub: it only re-enqueues existing jobs and never actually visits a source. The postings currently in the store were seeded from the earlier TypeScript collection; the Go backend does not collect anything itself.

Underneath that, the whole outbound-HTTP layer is missing in Go. The TypeScript backend's `network.ts` and `traffic.ts` were the foundation every fetch depended on: a request client with a server-side-request-forgery guard, per-source proxy profiles, a page cache with separate freshness windows for listings and detail pages, response-size caps, CAPTCHA/403 detection, and per-domain rate limiting with cooldowns. None of this exists in Go; `EnrichFn` and `POST /api/import` are explicit stubs (verified in the 2026-11-04 parity audit, `ai/sifty/go-parity-audit.md`). Parsers cannot run with nothing to fetch, so this card owns the fetch foundation as well as the abstraction above it. Its scope was widened, on the owner's instruction, to the whole collection layer rather than source adapters alone.

This card builds the real collection run around one light abstraction so that adding a new site later is a localised change behind a stable interface, not an edit threaded through the core. It also raises extraction quality and cost by ordering the mechanisms into a waterfall that prefers free, reliable structured data (the standards a site already publishes, and the public JSON endpoints that the dominant applicant-tracking platforms expose) and falls back to generic HTML only when those are absent. Extraction by a language model stays switched off here; the abstraction leaves a clearly named seam for a later learned-template cache (COL-2), but this card implements none of that learning.

The guiding decision, agreed with the owner: do the architecture correctly for future extension now, but with a light interface rather than a plugin framework, and reserve the model for the long tail rather than the common case.

## What this card delivers

A fetch foundation, a `Source` abstraction, and a working daily collection run built on them:

- An outbound-HTTP fetch layer that is the single path every source fetches through: a server-side-request-forgery guard that refuses private and loopback destinations, optional per-source proxy configuration, a response-size cap, CAPTCHA/403 detection, and a page cache with distinct freshness windows for listing pages and detail pages. This reproduces the load-bearing parts of the retired `network.ts`.
- A per-domain traffic discipline over that fetch layer: a minimum inter-request interval per domain, exponential backoff, and a cooldown that blocks a domain after a 403 or CAPTCHA, mirroring the retired `traffic.ts`. The existing Playwright pacing in `internal/background/discipline.go` is left as-is; this adds the equivalent for plain HTTP.
- A small interface that every source implements: discover candidate postings from a listing (returning references), and resolve one posting's detail page into a normalised `model.JobInput`. The existing parse primitives and the jobspy worker become implementations behind this interface rather than free functions called ad hoc.
- A dispatch that maps an enabled source from `config/search.yaml` to the right implementation: the jobspy worker for the major job boards it already covers, and the generic structured path (JSON-LD, then feed, then HTML links) for an owner-supplied site URL.
- Deterministic adapters for the dominant applicant-tracking platforms, keyed by the posting's host or platform fingerprint: Greenhouse and Lever (both expose public JSON posting endpoints), Ashby, and the existing Getro handling folded in behind the interface. These run without any model.
- An extraction waterfall with explicit order and validation: structured JSON-LD first, then the matching platform adapter, then the generic HTML extractor; each field that is kept must be present as exact page text, never a paraphrase. A source that yields no usable description ends in an honest "description pending / apply via listing" state, never a silent empty record and never a model call.
- The daily collection run (`defaultCollect`) wired to iterate enabled sources through this dispatch, dedupe by canonical URL, upsert postings, and enqueue them for assessment — replacing the current stub.
- The single-URL paths wired through the same fetch and waterfall: `POST /api/import` fetches and parses a real posting from an owner-supplied URL instead of writing a "Manual import" placeholder, and the `EnrichFn` behind `POST /api/jobs/{id}/read` re-fetches a role's detail page and updates its description instead of being a no-op.
- The named but unimplemented seam for COL-2: an interface for a per-host learned extraction hint and the rule that any such hint is validated on every use and invalidated (triggering re-learning) the moment it stops producing the required fields. Defined in code as types/interfaces with no behaviour, so COL-2 can fill it without reshaping COL-1.

## Must not touch

- The assessment and preparation pipelines, the assessment queue mechanics, and answer generation.
- The POST-submission blocking and the browser pacing, cooldown and concurrency discipline.
- The model-extraction fallback stays off; this card does not enable it and does not build the learned-template cache (that is COL-2).
- The public-subtree rules for `sifty/` (no card identifiers, tool-authorship trailers, absolute local paths or Russian prose in code or in commit messages touching `sifty/`).

## Notes

- Current primitives to reuse, not rewrite: `internal/collect/parse.go` (`StructuredJobs`, `ParseFeed`, `HTMLLinks`, `getroJobs`), `internal/collect/feed.go`, `internal/collect/text.go` (`cleanHTML`, `canonical`), and `internal/collect/jobspy.go` (`CollectViaJobspy`). The worker protocol is in `internal/worker/worker.go`.
- The daily run stub is `internal/background/scheduler.go` `defaultCollect`; the application wires a collection function in via the scheduler's injectable `CollectionFn`.
- Greenhouse and Lever public endpoints are the cheapest high-value adapters (verify the exact URL shape per platform during implementation); Ashby and Workday have machine interfaces too. Confirm each against a live posting before relying on it.
- Keep the abstraction light: an interface plus a small registry that selects an implementation from host/platform. No plugin loader, no configuration DSL.
- The learned-template cache, its storage, and its invalidation rule are COL-2; here only the interface and the invalidation contract are written, without behaviour.
- The stubs to replace, from the parity audit: `EnrichFn` no-op at `internal/server/routes.go:173`, `importURL` placeholder at `internal/server/routes.go:784`, `defaultCollect` at `internal/background/scheduler.go:153`. The retired sources to reproduce from, not port line by line, are `app/src/network.ts` and `app/src/traffic.ts`; read `ai/sifty/go-parity-audit.md` for what each provided.
- The fetch layer is new Go code, not a port of the primitives. Keep it a single client type the sources depend on, so the SSRF guard and the size cap cannot be bypassed by a source fetching directly.

## Steps

- [ ] To be defined in plan.md.

## Acceptance criteria

- To be defined in plan.md.
