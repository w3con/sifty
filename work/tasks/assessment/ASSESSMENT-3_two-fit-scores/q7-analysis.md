# Q.7 re-measurement

Matcher after: ['vacancyai.match/3+93f26c9dd3']. Vacancies re-matched by the new matcher: 81 (all), of them in the Q.6 set of 43: 43.

## Verdict shares

- Q.6 set (43 vacancies)
  - before: old matcher: 396 requirements; match 129 (32.6 %), partial 10 (2.5 %), no 0 (0.0 %), unclear 257 (64.9 %)
  - after:  new matcher: 396 requirements; match 150 (37.9 %), partial 25 (6.3 %), no 136 (34.3 %), unclear 85 (21.5 %)
- all re-matched (81 vacancies)
  - before: old matcher: 764 requirements; match 293 (38.4 %), partial 19 (2.5 %), no 0 (0.0 %), unclear 452 (59.2 %)
  - after:  new matcher: 764 requirements; match 302 (39.5 %), partial 38 (5.0 %), no 261 (34.2 %), unclear 163 (21.3 %)

Requirements decided by code without the model: 44 of 764 (5.8 %) over all re-matched vacancies.
In the Q.6 set: 10 of 396 (2.5 %).
Vacancies in which code decided every requirement (no model call at all): 0 of 81.

## Time per vacancy (matching only)

- all 81 timed vacancies: median 8.9 s, p90 14.4 s, mean 9.1 s, max 17.4 s
- the 81 that needed the model: median 8.9 s, p90 14.4 s, mean 9.1 s
- Q.6 for comparison (matching alone, old matcher): median 10.1 s, p90 16.4 s.

## You fit before and after

- before: min 50, median 70, max 100; below 50: 0 of 43
- after:  min 8, median 62, max 100; below 50: 15 of 43

| job | company / title | You fit before | You fit after |
|---|---|---|---|
| 2780 | Blackbird Labs / Product Marketing Manager **(unrelated role)** | 58 | 8 |
| 642 | Hadrian / Tech Lead, Operational Technology | 59 | 9 |
| 2774 | Chainalysis / Director, Total Rewards | 60 | 10 |
| 1806 | Antares / Senior Manufacturing Engineer, Electrical Engineering | 52 | 11 |
| 1530 | Antares / Senior Accountant **(unrelated role)** | 50 | 11 |
| 2769 | Kalshi / Marketing & Advertising Compliance **(unrelated role)** | 50 | 17 |
| 2766 | MoonPay / Senior People Partner (ER) **(unrelated role)** | 50 | 25 |
| 2785 | BIT Official / Business Development Manager **(unrelated role)** | 67 | 33 |
| 2773 | Kalshi / Regulatory Attorney - SEC/CFTC **(unrelated role)** | 50 | 33 |
| 658 | Hadrian / Tech Lead, Offensive Security | 58 | 35 |
| 1619 | True Anomaly / Senior Manager, Infrastructure Engineering | 60 | 38 |
| 2772 | Kalshi / Support Operations, Kalshi Prime **(unrelated role)** | 57 | 40 |
| 2770 | Phantom / Copywriter **(unrelated role)** | 50 | 42 |
| 2787 | BIT Official / Structured Products Trader **(unrelated role)** | 61 | 44 |
| 65 | Mysten Labs / Product Lead, AI | 80 | 45 |
| 2776 | Magic Eden / Senior Risk & Trading Analyst, Sportsbook | 63 | 50 |
| 2786 | Antares / Deployment Site Lead | 71 | 54 |
| 2694 | Mesh / Senior Engineering Manager, Cloud Infrastructure | 70 | 56 |
| 657 | Hadrian / Tech Lead, Detection & Response | 77 | 58 |
| 1089 | iSpot.TV / Manager, Engineering | 80 | 61 |
| 2771 | Monad Foundation / Trading Lead | 65 | 62 |
| 22 | Ethena Labs / Engineering Manager / Tech Lead **(named in the brief)** | 57 | 62 |
| 2767 | BIT Official / Trading Developer & Execution | 83 | 67 |
| 907 | Mochi Health / Engineering Manager | 72 | 67 |
| 1133 | HIFI Labs / Senior Fullstack Engineer / Engineering Manager Americas | 83 | 67 |
| 2377 | Unknown / Engineering Manager | 92 | 67 |
| 1011 | GlossGenius / Senior Engineering Manager, Infrastructure  | 61 | 68 |
| 265 | True Anomaly / Senior Software Engineer, Digital Engineering | 81 | 69 |
| 13 | Discernis / Senior Software Engineer | 90 | 70 |
| 2775 | Phantom / Product Designer **(unrelated role)** | 79 | 71 |
| 1584 | True Anomaly / Senior Manager, Digital Engineering | 71 | 71 |
| 1049 | Zenskar / Tech Lead (Backend) | 64 | 72 |
| 2165 | Unknown / Growth Lead **(unrelated role)** | 50 | 73 |
| 641 | Hadrian / Senior Engineering Manager, Machine Learning | 67 | 75 |
| 854 | Svix (YC W21) / Team lead | 100 | 75 |
| 1001 | Courier Health / Engineering Manager | 72 | 75 |
| 1054 | Signifyd / Engineering Manager | 71 | 76 |
| 1008 | Virta Health / Senior Engineering Manager - Client Group | 94 | 78 |
| 1005 | Forus / Engineering Manager | 79 | 79 |
| 2109 | Prime Intellect / Product Lead | 88 | 85 |
| 84 | Parity / Tech Lead **(named in the brief)** | 88 | 88 |
| 921 | Agave API / Engineering Manager | 96 | 88 |
| 855 | Check / Tech Lead Manager | 83 | 100 |

## Sample of 20 'unclear' lines (seed 7, pool 85)

1. job 1584 True Anomaly / Senior Manager, Digital Engineering (You fit 71), skill required: **Cloud and enterprise technologies**
   quote: Strong understanding of cloud and enterprise technologies, API and integration patterns, data modeling, and keeping data consistent across systems of record.
   model reason: 
2. job 2771 Monad Foundation / Trading Lead (You fit 62), skill required: **Comfortable with numbers**
   quote: Comfortable with numbers. You can model whether an incentive program retains volume after it ends, explain a trading team’s revenue drivers, and hold your own with engineers and quants.
   model reason: Comfort with numbers not explicitly stated or shown.
3. job 265 True Anomaly / Senior Software Engineer, Digital Engineering (You fit 69), skill required: **Programming language**
   quote: Proficiency in one or more modern programming languages (e.g., Go, Python, TypeScript).
   model reason: 
4. job 2780 Blackbird Labs / Product Marketing Manager (You fit 8), other required: **Scrappy, cross-functional leader with a passion for hospitality**
   quote: Scrappy, cross-functional leader with a passion for hospitality and building programs from scratch.
   model reason: 
5. job 2774 Chainalysis / Director, Total Rewards (You fit 10), experience preferred: **Rapid scaling/Advising experience**
   any of: building total rewards programs at companies undergoing rapid scaling | advising on sales compensation plan design and governance
   quote: Background in building total rewards programs at companies undergoing rapid scaling

Experience advising on sales compensation plan design and governance
   model reason: 
6. job 1054 Signifyd / Engineering Manager (You fit 76), skill required: **Team leadership and coaching**
   quote: Genuine care for engineer growth and development, with experience giving structured feedback and building career paths.
   model reason: 
7. job 2772 Kalshi / Support Operations, Kalshi Prime (You fit 40), skill required: **Excellent written communication**
   quote: Excellent written communication: you can explain margin, an account hold or a document request clearly to a retail trader and to an institutional client
   model reason: Soft skill not stated in profile facts.
8. job 1619 True Anomaly / Senior Manager, Infrastructure Engineering (You fit 38), authorization required: **US citizenship or work authorization**
   quote: you must be a U.S. citizen, lawful permanent resident of the U.S., protected individual as defined by 8 U.S.C. 1324b(a)(3), or eligible to obtain the required authorizations from the U.S. Department of State.
   model reason: 
9. job 1530 Antares / Senior Accountant (You fit 11), skill preferred: **Attention to detail**
   quote: Strong attention to detail with the ability to understand how individual transactions flow through the broader financial statements.
   model reason: Attention to detail not explicitly stated.
10. job 2774 Chainalysis / Director, Total Rewards (You fit 10), skill required: **Communication skills**
   quote: Excellent communication skills with the ability to translate complex concepts for diverse audiences
   model reason: Communication skills not stated in CV.
11. job 921 Agave API / Engineering Manager (You fit 88), skill required: **Explaining tradeoffs**
   quote: Can explain tradeoffs to founders and customers, not just engineers.
   model reason: 
12. job 84 Parity / Tech Lead (You fit 88), skill required: **Excellent asynchronous communication skills**
   quote: Excellent asynchronous communication skills
   model reason: 
13. job 2785 BIT Official / Business Development Manager (You fit 33), qualification required: **Bachelor's degree**
   quote: Bachelor’s degree
   model reason: 
14. job 2772 Kalshi / Support Operations, Kalshi Prime (You fit 40), experience required: **2+ years experience**
   quote: 2+ years in customer support, operations, client onboarding, compliance operations or a related role
   model reason: 
15. job 1806 Antares / Senior Manufacturing Engineer, Electrical Engineering (You fit 11), authorization required: **U.S. citizen, national, permanent resident, Refugee, or Asylee**
   quote: applicant must be a (i) U.S. citizen or national, (ii) U.S. lawful, permanent resident (aka green card holder), (iii) Refugee under 8 U.S.C. § 1157, or (iv) Asylee under 8 U.S.C. § 1158, or be eligible to obtain the required authorizations from the U.S. Depart
   model reason: 
16. job 1806 Antares / Senior Manufacturing Engineer, Electrical Engineering (You fit 11), qualification required: **Bachelor's degree in engineering or physics**
   quote: Bachelor’s degree in an engineering discipline or physics
   model reason: 
17. job 2774 Chainalysis / Director, Total Rewards (You fit 10), skill preferred: **Equity/incentive design (Web3/tech)**
   any of: equity compensation programs | incentive plan design
   quote: Experience with equity compensation programs and incentive plan design in the Web3 or technology industry
   model reason: 
18. job 22 Ethena Labs / Engineering Manager / Tech Lead (You fit 62), skill required: **Review native iOS app**
   quote: Native mobile. Enough to review a native iOS app (Swift / SwiftUI) and reason about mobile ↔ backend contracts, failure states, and on-device auth.
   model reason: 
19. job 1089 iSpot.TV / Manager, Engineering (You fit 61), skill required: **Communication skills**
   quote: Exceptional communication skills to align engineering efforts with Product, Data Science, and Operations objectives.
   model reason: Communication skills not stated in profile.
20. job 1806 Antares / Senior Manufacturing Engineer, Electrical Engineering (You fit 11), skill preferred: **Master's degree or 5+ years in electrical/aerospace engineering**
   quote: Master's degree or 5+ years of professional experience in electrical or aerospace engineering
   model reason: 

## Sample of 10 'no' lines (seed 7, pool 136)

1. job 2780 Blackbird Labs / Product Marketing Manager (You fit 8), skill required: **Expertise in lifecycle marketing, retention, and engagement**
   quote: Expertise in lifecycle marketing, retention, and engagement across email, push, and in-app channels.
   model reason: No lifecycle marketing retention or engagement
2. job 2773 Kalshi / Regulatory Attorney - SEC/CFTC (You fit 33), qualification required: **J.D. and active state bar membership**
   quote: J.D. and active membership in good standing in at least one U.S. state bar.
   model reason: No bar membership or legal qualification facts.
3. job 642 Hadrian / Tech Lead, Operational Technology (You fit 9), skill required: **Knowledge of industrial environments**
   quote: Deep knowledge of industrial environments including PLCs, SCADA/DCS, HMIs, RTUs, and industrial protocols.
   model reason: No industrial environment experience found
4. job 907 Mochi Health / Engineering Manager (You fit 67), experience required: **Experience in regulated industries**
   quote: You have managed teams in highly regulated, operationally heavy industries (e.g., healthcare, fintech, logistics) where failures have real-world consequences.
   model reason: No regulated industry experience in profile.
5. job 2785 BIT Official / Business Development Manager (You fit 33), experience preferred: **background in leading online brokerages**
   quote: Background in leading online brokerages or international markets
   model reason: No online brokerage leadership listed
6. job 642 Hadrian / Tech Lead, Operational Technology (You fit 9), experience required: **10+ years OT/ICS/embedded/security experience**
   quote: 10+ years across OT/ICS, embedded/product security, manufacturing, or supply chain security, including meaningful hands-on ownership.
   model reason: No OT/ICS/embedded/security experience found
7. job 2785 BIT Official / Business Development Manager (You fit 33), experience preferred: **experience in US stocks**
   quote: Deployable partner resources and experience in US stocks
   model reason: No US stock experience in profile
8. job 2773 Kalshi / Regulatory Attorney - SEC/CFTC (You fit 33), experience required: **Regulatory advisory experience**
   quote: Experience advising on complex regulatory matters involving financial products, trading platforms, or regulated market participants.
   model reason: Profile is engineering, not regulatory advisory.
9. job 1619 True Anomaly / Senior Manager, Infrastructure Engineering (You fit 38), experience preferred: **Aerospace/defense experience**
   quote: Experience in aerospace, defense, or national security technology companies where infrastructure spans classification levels
   model reason: No aerospace or defense experience mentioned in profile
10. job 1089 iSpot.TV / Manager, Engineering (You fit 61), experience required: **Agile Execution experience**
   quote: Mastery of effective sprint planning to balance rapid delivery of new platform features with long-term infrastructure stability.
   model reason: No agile execution facts offered.
