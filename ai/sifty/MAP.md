# Sifty — map of where everything lives

This file exists so a session opening cold knows where the Sifty product thinking is recorded and does not have to reconstruct it from a compacted conversation. Sifty is the public product built on the internal Sift engine under `app/`. This repository (FCTO) is private; the public product will live in its own separate MIT repository (see the backlog). Update this map whenever a Sifty document is added, moved, or substantially changed.

## The documents and what each holds

`kb/Strategy/sift-product.md` — the business-value view of the product: what Sifty is, who it serves, the tier model, the real asset (ghost-job / dead-posting detection from collection-side signals), the blockchain verdict, open decisions, and the name/domain decision. It is the current source of truth for product framing. It is explicitly not a build plan.

`ai/research/sift-public-service-research.md` — the detailed research behind the product: LLM cost at real volume (section 1), similar tools and reuse verdict (section 2), the provider-abstraction gap (section 4), the deterministic pre-filter cost lever (section 5), career-ops design borrowings (section 6), collection strategy tiers (section 7), self-hosted inference economics (section 8), business-model options and the refined data/inference matrix (section 9), true p2p feasibility and the blockchain community-layer discussion (section 9a), and the candidate plan additions awaiting decision (section 10). This is where the Resume Matcher reference and the weighted-scorer design live.

`ai/tasks/job-search-assistant.md` — the build card for the Sift application under `app/`: the delivered stages (Stage 1 collection + fit, Stage 2 application-answer preparation, both complete and verified), the development-machine transfer tooling, and the acceptance records. It is the source of truth for what has actually been built. New product build work is tracked in the backlog below until it gets its own card.

`ai/sifty/backlog.md` — the Sifty product backlog: the open tasks, the decisions locked on 2026-11-01, and the sequence. This is the entry point for "what are we building next".

`CLAUDE.md` (repository root) — points a session at the build card and the knowledge base; it is project instructions, not product thinking.

## What is not here

User reviews and a company-reputation layer were considered and dropped; the objective liveness detector replaces them. Blockchain is out of the core product per `sift-product.md`, with a noted tension against research section 9a that the backlog records. The business model, pricing numbers, and the legal questions around scraping and a community data pool are not settled and are listed as open in the backlog, not as decisions.

## The internal engine versus the public product

The code is named Sift and keeps that name internally. Sifty is the public product. The domain sifty.work is owned. The engine currently lives inside this private repository and reads and writes some user data into `kb/` paths; separating it into a self-contained public repository is the first task in the backlog.
