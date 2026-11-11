# Q.6 run analysis

Window start: 2026-11-11T02:22:00Z. Vacancies extracted in the window: 51; matched: 43.

## Timing

Vacancies timed: 50 (first row after a start and gaps over 600 s dropped).
- Per vacancy, extraction plus matching: median 23.9 s, p90 42.2 s, mean 24.6 s, max 59.5 s.
- Extraction pass alone: median 15.2 s, p90 28.4 s. Matching pass alone: median 10.1 s, p90 16.4 s.
- Projected backfill of 2273 vacancies still pending, at the median: 15.1 hours.
- Projected backfill of 2273 vacancies still pending, at the mean: 15.5 hours.
- Projected backfill of 2273 vacancies still pending, at the p90: 26.7 hours.

Special vacancies in the window:
- 84 Parity / Tech Lead: time 17.8 s, requirements 5, description 4552 characters
- 22 Ethena Labs / Engineering Manager / Tech Lead: time 31.9 s, requirements 11, description 5199 characters

## Verdicts

Requirements judged: 396 across 43 vacancies.
- match: 129 (32.6 %)
- partial: 10 (2.5 %)
- no: 0 (0.0 %)
- unclear: 257 (64.9 %)
Requirements per vacancy: median 7, max 22; vacancies with none: 8.

## Sample of 10 'unclear' lines (seed 6, pool 257)

1. job 1011 GlossGenius / Senior Engineering Manager, Infrastructure , requirement 7 (experience, preferred): Experience changing ops with AI
   quote: Fundamentally changed how your team operates because of AI, and you can name the system you designed, the work it absorbed, and what your engineers *stopped* doing as a result
   model reason: AI mentioned but no ops change context.
2. job 657 Hadrian / Tech Lead, Detection & Response, requirement 6 (experience, preferred): OT/ICS or manufacturing security exposure
   quote: Exposure to OT/ICS or manufacturing environment security
   model reason: No OT/ICS or manufacturing security mentioned.
3. job 2787 BIT Official / Structured Products Trader, requirement 1 (experience, required): 3+ years trading exotic options
   quote: 3+ years trading exotic options/structured products (TradFi or crypto); market-making or proprietary hedging experience preferred.
   model reason: No trading exotic options experience stated.
4. job 907 Mochi Health / Engineering Manager, requirement 2 (experience, required): Experience in regulated industries
   quote: You have managed teams in highly regulated, operationally heavy industries (e.g., healthcare, fintech, logistics) where failures have real-world consequences.
   model reason: No mention of regulated industries.
5. job 2772 Kalshi / Support Operations, Kalshi Prime, requirement 5 (skill, preferred): Coachability
   quote: Coachable and eager to take on more over time; you enjoy the pace and ambiguity of a startup
   model reason: No explicit coachability stated.
6. job 1530 Antares / Senior Accountant, requirement 12 (experience, preferred): Audit management experience
   quote: Experience supporting or managing a financial statement audit.
   model reason: Profile shows blockchain audits, not audit management roles.
7. job 2773 Kalshi / Regulatory Attorney - SEC/CFTC, requirement 4 (skill, required): Legal research, writing, and analytical skills
   quote: Exceptional legal research, writing, and analytical skills.
   model reason: No legal research or writing skills stated.
8. job 2786 Antares / Deployment Site Lead, requirement 0 (authorization, required): U.S. citizen, national, green card holder, Refugee, or Asylee
   quote: applicant must be a (i) U.S. citizen or national, (ii) U.S. lawful, permanent resident (aka green card holder), (iii) Refugee under 8 U.S.C. § 1157, or (iv) Asylee under 8 U.S.C. § 1158, or be eligibl
   model reason: No work authorization or citizenship stated.
9. job 2785 BIT Official / Business Development Manager, requirement 2 (skill, required): US stock market knowledge
   quote: Strong industry connections and US stock market knowledge
   model reason: No mention of US stock market knowledge.
10. job 1011 GlossGenius / Senior Engineering Manager, Infrastructure , requirement 4 (skill, required): Debug complex distributed systems
   quote: Deep hands-on expertise across AWS, Kubernetes, Terraform, ArgoCD, and modern observability tooling, with the ability to debug a complex distributed system yourself when it matters
   model reason: No evidence of debugging complex distributed systems.

## You fit scores in the run

42 vacancies carry a You fit number. Minimum 50, median 69, maximum 100; by tenth: 50 (7 vacancies), 60 (11), 70 (9), 80 (8), 90 (5), 100 (2). The vacancies at 50 include Senior Accountant (Antares), Growth Lead, Senior People Partner (MoonPay). The verdict `no` appeared 0 times in 363 requirements, and `unclear` carries utility 0.5 (spec 14.2), so You fit cannot fall below 50: an irrelevant role scores the same as a role the profile half answers. Parity 84: You fit 88; Ethena Labs 22: 57 (Fits you 58).

## Hand check of "unclear" lines (done by the executor; the owner should confirm the judgements)

Profile at revision 25: skills Rust, TypeScript, Node, Solidity, React, JS, Python, Go, Backend, SQL, GraphQL; experience entries with prose summaries (Validité Founder & CTO 2025–present "built the platform ... including AI-assisted verification using document retrieval and connected tools" and "my own AI delivery system"; Pilier Network Co-Founder & Chief Architect; Futureverse Principal Smart Contracts Engineer "set smart-contract engineering standards ... and mentored engineers"); no education section, no team sizes, no infrastructure tools.

**Sample A: 10 random "unclear" lines from the whole run (seed 6, pool 244).** All 10 are right to be unclear: they come from roles unrelated to the profile (copywriter, attorney, trading, product marketing) and ask for things the profile does not contain (Gen-Z language, odds and liabilities, legal research, US stock market, NPS). One is arguable (job 2780: "Excitement about leveraging AI to scale marketing": the profile shows AI work, not excitement). Answerable from the profile: 0 clear, 1 arguable of 10.

**Sample B: 10 random "unclear" lines from vacancies with Fits you at least 0.5 (seed 6, pool 39; 15 vacancies).** Right to be unclear (6): CS degree (no education section), access to specialized subnetworks, asynchronous communication skills, Databricks/Spark/GCP, Firebase/GCP, prioritization and negotiation. Answerable from the profile, missed by the model (2 clear, 2 weaker):
- Ethena Labs, "agent-native development" (clear): the Validité entry describes an own AI delivery system and AI-assisted verification with connected tools.
- Ethena Labs, "Led small team of 3-8" (clear in part): the profile has Chief Architect, Principal Engineer who mentored engineers, and Founder & CTO; it lacks a team size, so partial, not unclear.
- Ethena Labs, "Owned production operations for money system" (weaker): Pilier Network lists network infrastructure, software upgrades and operational monitoring; partial at best.
- Ethena Labs, "Kubernetes, IaC and observability" (weaker): only "operational monitoring" at Pilier Network; partial at best.

**The two lines the brief named (Ethena Labs 22):** "Led small team of 3-8" stays unclear (partial would be the fair verdict); "Shipped production code in last year" stays unclear although Validité 2025–present says the owner built the platform (a clear match).

Reading: on relevant vacancies about 2 of 10 unclear lines are clearly answerable and 2 more partly; on irrelevant vacancies none. The misses are indirect answers: the facts are in prose summaries and the model asks for a literal statement. No model larger than qwen3.5:9b is installed on this machine, so a larger local model could not be tried here; qwen3.5:4b is smaller. What can be said from this run: the 9b model is too literal on indirect evidence, and a larger model (or a profile with explicit skills and team sizes) is the lever; the effect is unmeasured.

## Special vacancies

- Parity (84, glued headings, 4 552 characters): re-extracted in this run in 17.8 s, 5 requirements, You fit 88.
- Ethena Labs (22, 5 199 characters): 31.9 s, 11 requirements, You fit 57.
- Typeform (19): source unavailable ("detail unavailable: HTTP 403"); no extraction, never queued, both scores without a number, Refresh offered.
- Glued paragraph: Parity (84) is the glued case of the regression set; the other glued ones of that set (OKX 32, Zscaler 29) were not reached by the run.
