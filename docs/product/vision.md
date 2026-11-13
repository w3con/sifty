# Sifty — product vision

Status: owner's development vision, recorded 2026-11-13. This is a direction for development, not a sales description; it holds enough to write a user-facing description later. Prices and quotas are the owner's first figures, to be recalculated from measured costs (§4). The owner's original note, verbatim, is in [`work/decisions/product-vision.md`](../../work/decisions/product-vision.md). Sections 1–3 restate the owner; section 5 is the coordinator's analysis, which the owner has not yet reviewed.

Roles: **owner** — the person who decides what Sifty is; **coordinator** — the planning agent that records decisions and plans work; **user** — anyone running or subscribing to Sifty.

## 1. Three editions

1. **Self-hosted, free.** Sifty under the [Elastic License 2.0](../../LICENSE). Anyone can run it on their own machine at no cost, with their own model and search keys.
2. **Hosted, bring your own keys.** Sifty running on our server with the frontend available to users by subscription, at a very low price (owner's first figure: about $1 per week). The user enters their own API keys, so model and search costs are theirs.
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

## 5. Coordinator's notes

Implications and risks the vision brings, to be decided by the owner.

1. **Multi-user architecture.** Sifty today assumes one owner: one data directory, one profile. The hosted editions need accounts, isolation of each user's data, a shared vacancy catalogue separated from per-user state (stages, verdicts, answers), and billing. Collection becomes shared, and today's per-owner search expressions become filters over the shared catalogue. New code should avoid deepening single-owner assumptions.
2. **Payment fees on small charges.** Card processors charge a fixed fee per charge (about $0.30) plus a percentage, so a $1 weekly charge loses about a third. The low tier should be billed monthly or yearly, with the price quoted per week if wanted.
3. **Storing users' keys.** The bring-your-own-keys edition keeps users' paid API keys on our server; they must be encrypted, and a leak costs users money directly.
4. **Quota versus cost, rough estimate (not measured).** One shortlisted vacancy with assessment, answers and a cover letter is of the order of 15–20 thousand input tokens and a few thousand output tokens, about $0.05–0.10 on a mid-range model. 100 preparations per day is about 700 per week, $35–70 per user per week, which a $10 tier does not cover. A person cannot apply to 100 vacancies a day by hand, so real use is likely far below such a quota; the measured funnel (§4) decides.
5. **Legal risk of the hosted editions.** A self-hosted user collects vacancies for themself. The hosted service collects vacancies from other sites and shows them to paying users; the terms of many job boards, LinkedIn above all, forbid that. Sources for the hosted editions must be chosen by their terms, for example applicant-tracking systems' public job boards and sources that permit reuse.
6. **Licence.** The Elastic License 2.0 allows free self-hosting and forbids others from offering Sifty as a hosted service, which protects the hosted editions. It is source-available, not open source. If outside developers contribute, a contributor agreement is needed for the owner to keep using their code in the hosted editions.

## 6. Open questions

1. Prices, billing period and daily quotas of each tier — after §4.
2. Which models run shared extraction, and which per-user work, in each edition.
3. Whether the hosted editions keep the self-hosted feature set or add hosted-only features.
4. Which collection sources the hosted editions may use (§5.5).
5. When multi-user work starts relative to the application stages in the [roadmap](../../work/ROADMAP.md).
