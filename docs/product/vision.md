# Sifty — product vision

Status: owner's development vision, recorded 2026-11-13. This is a direction for development, not a sales description; it holds enough to write a user-facing description later. Prices and quotas are the owner's first figures, to be recalculated from measured costs (§4). The owner's original note, verbatim, is in [`work/decisions/product-vision.md`](../../work/decisions/product-vision.md). Sections 1–3 restate the owner; section 5 is the coordinator's analysis, which the owner has not yet reviewed.

Roles: **owner** — the person who decides what Sifty is; **coordinator** — the planning agent that records decisions and plans work; **user** — anyone running or subscribing to Sifty.

## 1. Three editions

1. **Self-hosted, free.** Sifty under the [Elastic License 2.0](../../LICENSE). Anyone can run it on their own machine at no cost, with their own model and search keys.
2. **Hosted, bring your own keys.** Sifty running on our server with the frontend available to users by subscription, at a very low price (owner's first figure: about $1 per week, revised to about $10 per month because of payment fees, §5.2). The user's own API keys pay for model and search use. Our server does not store the keys; they stay on the user's computer (owner, 2026-11-13; options in §5.3).
3. **Hosted, all inclusive.** The user enters no keys. Several tiers, starting at about $10 per week (owner's first figure), each including a generous daily quota of prepared applications (owner's example: 100 preparations per day). Tiers are set after the costs in §4 are measured.

## 2. Where the cost sits

When Sifty runs on our server, work splits into two kinds.

- **Shared work, paid once for all users.** Collection, fetching and data extraction from vacancies are the same for every user. They can run on expensive, high-quality models, because the cost is spread across all users. Users then pick what they receive with filters and search over the shared catalogue.
- **Per-user work, paid for each user separately.** Everything done for a vacancy a user has shortlisted: fit assessment, the employer-site lookup and its web searches, reading the form's questions, proposed answers and the cover letter (see the [application specification](../specs/application.md) §3.0, §3.6).

## 3. Positioning

Sifty is more technical than its competitors: much is configured by the user. The owner considers it learnable, and UI and UX are worked on later.

Reference competitor, found by the owner via We Work Remotely: **AI Job Copilot** — $12 per week, everything included, up to 20 applications per day, one profile, and the service submits applications automatically on the user's behalf. Sifty differs: the user reviews every application before it is sent, several search profiles are planned, the hosted price is noticeably lower, and the self-hosted edition is free.

## 4. What must be measured before prices are set

1. **The funnel:** how many collected vacancies become relevant, how many are shortlisted, how many are applied to — per user per week. The owner's own production data gives the first figures.
2. **Shared cost:** tokens and money per vacancy for extraction, and per week for the whole collection.
3. **Per-user cost:** tokens and money per shortlisted vacancy, including fit assessment, employer lookup searches, answers and the cover letter.
4. **Cost of an active user per week** = shortlisted vacancies per week × cost per shortlisted vacancy + that user's share of the shared cost.
5. From these: weekly and monthly prices and the daily quota of each tier.

Timing, owner decision 2026-11-13: not now. The product is built first; the owner then uses Sifty for their own job search, and that real use gives the figures.

## 5. Coordinator's notes

Implications and risks the vision brings, to be decided by the owner.

1. **Multi-user architecture.** Sifty today assumes one owner: one data directory, one profile. The hosted editions need accounts, isolation of each user's data, a shared vacancy catalogue separated from per-user state (stages, verdicts, answers), and billing. Collection becomes shared, and today's per-owner search expressions become filters over the shared catalogue. New code should avoid deepening single-owner assumptions.
2. **Payment fees on small charges.** Card processors charge a fixed fee per charge (about $0.30) plus a percentage, so a $1 weekly charge loses about a third. Owner's answer, 2026-11-13: then about $10 per month.
3. **Where users' keys live.** Owner, 2026-11-13: we do not store keys; they stay on the user's computer, in the browser or on disk; to be decided separately. Coordinator's options: (a) keys kept in browser storage and the browser calls the model provider directly, so the key never reaches our server — per-user work runs only while Sifty is open, and not every provider accepts calls from a browser (Anthropic does; others need checking); (b) keys kept in the browser and sent to our server with each request, used in memory and never written — works with every provider, but the key passes through our server; (c) per-user work runs in a program on the user's computer (the stage 3 browser extension or a small helper) while our server provides only the shared catalogue — most private, needs an install. Coordinator's recommendation: (a), with (b) as the fallback for providers that refuse browser calls. Not decided.
4. **Quota versus cost, rough estimate (not measured).** One shortlisted vacancy with assessment, answers and a cover letter is of the order of 15–20 thousand input tokens and a few thousand output tokens, about $0.05–0.10 on a mid-range model; 100 preparations per day would be $35–70 per user per week. Owner's answer, 2026-11-13: Haiku 5 gives good enough quality and is cheap, comparable small models from other vendors likely too; if needed, our own compute running an open model of about 100 billion parameters (for example Qwen), shared across users. To be decided.
5. **Legal risk of the hosted editions.** A self-hosted user collects vacancies for themself. The hosted service collects vacancies from other sites and shows them to paying users; the terms of many job boards, LinkedIn above all, forbid that. Today Sifty collects with its own code and only from open sources: applicant-tracking systems' public APIs (Greenhouse, Lever, Ashby) and public job-board pages (structured data, feeds, links); no third-party scraping library is used, and sign-in sources are not collected (COLLECTION-3, no behaviour agreed). Owner, 2026-11-13: consider paid third-party libraries or job-data services if that removes the problem. Coordinator's note: a paid provider removes the risk only if it obtains its data lawfully; reselling data scraped against a site's terms keeps the problem. Candidates named from memory and not yet checked: TheirStack and Coresignal (job-data APIs), Apify (a marketplace of scrapers). Research is needed before the hosted editions, not now.
6. **Licence.** The Elastic License 2.0 allows free self-hosting and forbids others from offering Sifty as a hosted service, which protects the hosted editions. It is source-available, not open source. If outside developers contribute, a contributor agreement is needed for the owner to keep using their code in the hosted editions.

## 6. Open questions

1. Prices, billing period and daily quotas of each tier — after §4.
2. Which models run shared extraction, and which per-user work, in each edition.
3. Whether the hosted editions keep the self-hosted feature set or add hosted-only features.
4. Which collection sources the hosted editions may use, and whether a paid job-data provider replaces own collection (§5.5).
5. When multi-user work starts relative to the application stages in the [roadmap](../../work/ROADMAP.md).
6. Where users' keys live (§5.3).
7. Which models the hosted editions run: a small commercial model or own compute (§5.4).
