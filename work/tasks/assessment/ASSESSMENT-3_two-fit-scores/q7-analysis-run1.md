# Q.7 re-measurement

Matcher after: ['vacancyai.match/3+40e4b3e740']. Vacancies re-matched by the new matcher: 81 (all), of them in the Q.6 set of 43: 43.

## Verdict shares

- Q.6 set (43 vacancies)
  - before: old matcher: 396 requirements; match 129 (32.6 %), partial 10 (2.5 %), no 0 (0.0 %), unclear 257 (64.9 %)
  - after:  new matcher: 396 requirements; match 157 (39.6 %), partial 25 (6.3 %), no 147 (37.1 %), unclear 67 (16.9 %)
- all re-matched (81 vacancies)
  - before: old matcher: 764 requirements; match 293 (38.4 %), partial 19 (2.5 %), no 0 (0.0 %), unclear 452 (59.2 %)
  - after:  new matcher: 764 requirements; match 302 (39.5 %), partial 29 (3.8 %), no 294 (38.5 %), unclear 139 (18.2 %)

Requirements decided by code without the model: 41 of 764 (5.4 %) over all re-matched vacancies.
In the Q.6 set: 8 of 396 (2.0 %).
Vacancies in which code decided every requirement (no model call at all): 0 of 81.

## Time per vacancy (matching only)

- all 81 timed vacancies: median 9.3 s, p90 14.7 s, mean 9.4 s, max 20.0 s
- the 81 that needed the model: median 9.3 s, p90 14.7 s, mean 9.4 s
- Q.6 for comparison (matching alone, old matcher): median 10.1 s, p90 16.4 s.

## You fit before and after

- before: min 50, median 70, max 100; below 50: 0 of 43
- after:  min 5, median 58, max 100; below 50: 14 of 43

| job | company / title | You fit before | You fit after |
|---|---|---|---|
| 1806 | Antares / Senior Manufacturing Engineer, Electrical Engineering | 52 | 5 |
| 1530 | Antares / Senior Accountant **(unrelated role)** | 50 | 7 |
| 2780 | Blackbird Labs / Product Marketing Manager **(unrelated role)** | 58 | 8 |
| 2773 | Kalshi / Regulatory Attorney - SEC/CFTC **(unrelated role)** | 50 | 8 |
| 642 | Hadrian / Tech Lead, Operational Technology | 59 | 9 |
| 2769 | Kalshi / Marketing & Advertising Compliance **(unrelated role)** | 50 | 17 |
| 2785 | BIT Official / Business Development Manager **(unrelated role)** | 67 | 21 |
| 2766 | MoonPay / Senior People Partner (ER) **(unrelated role)** | 50 | 25 |
| 1049 | Zenskar / Tech Lead (Backend) | 64 | 25 |
| 2772 | Kalshi / Support Operations, Kalshi Prime **(unrelated role)** | 57 | 33 |
| 2787 | BIT Official / Structured Products Trader **(unrelated role)** | 61 | 36 |
| 65 | Mysten Labs / Product Lead, AI | 80 | 45 |
| 657 | Hadrian / Tech Lead, Detection & Response | 77 | 46 |
| 658 | Hadrian / Tech Lead, Offensive Security | 58 | 46 |
| 2776 | Magic Eden / Senior Risk & Trading Analyst, Sportsbook | 63 | 50 |
| 1133 | HIFI Labs / Senior Fullstack Engineer / Engineering Manager Americas | 83 | 50 |
| 2774 | Chainalysis / Director, Total Rewards | 60 | 53 |
| 1619 | True Anomaly / Senior Manager, Infrastructure Engineering | 60 | 53 |
| 2786 | Antares / Deployment Site Lead | 71 | 54 |
| 2771 | Monad Foundation / Trading Lead | 65 | 54 |
| 1089 | iSpot.TV / Manager, Engineering | 80 | 57 |
| 2770 | Phantom / Copywriter **(unrelated role)** | 50 | 58 |
| 2694 | Mesh / Senior Engineering Manager, Cloud Infrastructure | 70 | 61 |
| 907 | Mochi Health / Engineering Manager | 72 | 61 |
| 2767 | BIT Official / Trading Developer & Execution | 83 | 67 |
| 1011 | GlossGenius / Senior Engineering Manager, Infrastructure  | 61 | 68 |
| 1054 | Signifyd / Engineering Manager | 71 | 69 |
| 2165 | Unknown / Growth Lead **(unrelated role)** | 50 | 69 |
| 2775 | Phantom / Product Designer **(unrelated role)** | 79 | 71 |
| 265 | True Anomaly / Senior Software Engineer, Digital Engineering | 81 | 72 |
| 641 | Hadrian / Senior Engineering Manager, Machine Learning | 67 | 75 |
| 1001 | Courier Health / Engineering Manager | 72 | 75 |
| 2377 | Unknown / Engineering Manager | 92 | 75 |
| 22 | Ethena Labs / Engineering Manager / Tech Lead **(named in the brief)** | 57 | 76 |
| 921 | Agave API / Engineering Manager | 96 | 81 |
| 1584 | True Anomaly / Senior Manager, Digital Engineering | 71 | 82 |
| 1005 | Forus / Engineering Manager | 79 | 86 |
| 84 | Parity / Tech Lead **(named in the brief)** | 88 | 88 |
| 1008 | Virta Health / Senior Engineering Manager - Client Group | 94 | 89 |
| 13 | Discernis / Senior Software Engineer | 90 | 90 |
| 854 | Svix (YC W21) / Team lead | 100 | 92 |
| 2109 | Prime Intellect / Product Lead | 88 | 92 |
| 855 | Check / Tech Lead Manager | 83 | 100 |

## Sample of 20 'unclear' lines (seed 7, pool 67)

1. job 265 True Anomaly / Senior Software Engineer, Digital Engineering (You fit 72), experience required: **Project leadership experience**
   quote: Experience leading a project or feature area from design through delivery.
   model reason: 
2. job 2770 Phantom / Copywriter (You fit 58), skill required: **Excellent communication and storytelling skills**
   quote: Excellent communication and storytelling skills. You can explain product value and also shape how people feel about the brand.
   model reason: Communication skills not explicitly stated
3. job 907 Mochi Health / Engineering Manager (You fit 61), skill required: **Collaborate with clinical teams**
   quote: You will partner deeply with Operations, Product, and Clinical leadership. You will act as a collaborative translation layer—distilling complex medical realities and regulatory workflows into clear, pragmatic technical roadmaps for your team.
   model reason: 
4. job 2773 Kalshi / Regulatory Attorney - SEC/CFTC (You fit 8), skill required: **Commercial judgment**
   quote: Strong commercial judgment and the ability to translate complex regulatory issues into practical business advice.
   model reason: Commercial judgment not stated in profile
5. job 2780 Blackbird Labs / Product Marketing Manager (You fit 8), other required: **Scrappy, cross-functional leader with a passion for hospitality**
   quote: Scrappy, cross-functional leader with a passion for hospitality and building programs from scratch.
   model reason: 
6. job 921 Agave API / Engineering Manager (You fit 81), other preferred: **Self-motivated and high ownership**
   quote: Self-motivated, high ownership, low ego; desire to work on a fast-paced, intense, fun team. Passion for building a world-class engineering culture.
   model reason: Soft skill not stated in profile facts.
7. job 1584 True Anomaly / Senior Manager, Digital Engineering (You fit 82), authorization required: **U.S. security clearance**
   quote: Ability to obtain and maintain a U.S. security clearance. (US Citizenship is required to qualify for a security clearance)
   model reason: 
8. job 2109 Prime Intellect / Product Lead (You fit 92), skill required: **Learn directly from customers, write clearly**
   quote: You learn directly from customers, write clearly, and can turn an ambiguous problem into a concrete next step.
   model reason: No facts offered for customer learning or writing clearly.
9. job 1049 Zenskar / Tech Lead (Backend) (You fit 25), skill preferred: **Security and compliance awareness**
   quote: Security and compliance awareness (SOC 2, DPDPA, or similar) with a grasp of the engineering implications
   model reason: 
10. job 1619 True Anomaly / Senior Manager, Infrastructure Engineering (You fit 53), authorization required: **Active US Secret clearance**
   quote: Active U.S. security clearance, such as Secret, Top Secret or TS/SCI clearance
   model reason: 
11. job 2785 BIT Official / Business Development Manager (You fit 21), experience preferred: **background in leading online brokerages**
   quote: Background in leading online brokerages or international markets
   model reason: 
12. job 1011 GlossGenius / Senior Engineering Manager, Infrastructure  (You fit 68), qualification required: **Right to work in US**
   quote: You must be commutable to our San Francisco office and will operate in an in-person environment, as we default to being in-office 3-4 days per week.
   model reason: No work authorization details stated.
13. job 2772 Kalshi / Support Operations, Kalshi Prime (You fit 33), experience preferred: **Trading familiarity**
   quote: Familiarity with futures, perpetuals, margin or derivatives trading
   model reason: Trading familiarity not stated in facts.
14. job 2785 BIT Official / Business Development Manager (You fit 21), skill required: **communication and negotiation skills**
   quote: Proficient in communication, negotiation, and results-driven partnership management
   model reason: 
15. job 2774 Chainalysis / Director, Total Rewards (You fit 53), skill required: **Communication skills**
   quote: Excellent communication skills with the ability to translate complex concepts for diverse audiences
   model reason: Communication skills not stated in CV.
16. job 642 Hadrian / Tech Lead, Operational Technology (You fit 9), authorization required: **U.S. citizen or authorized for ITAR**
   quote: To conform to U.S. Government export regulations, including the International Traffic in Arms Regulations (ITAR) you must be a U.S. citizen or national, lawful permanent resident, protected individual as defined by 8 U.S.C. 1324b(a)(3), or eligible to obtain t
   model reason: No work authorization or citizenship stated
17. job 22 Ethena Labs / Engineering Manager / Tech Lead (You fit 76), skill required: **Kubernetes, IaC and observability**
   quote: Platform & reliability. Enough Kubernetes, IaC, and observability (OTel / Grafana-class) to own the deploy/rollback story and hold the reliability bar — you know the difference between a dashboard that exists and an alert that pages.
   model reason: 
18. job 1530 Antares / Senior Accountant (You fit 7), skill preferred: **Communication skills**
   quote: Strong written and verbal communication skills.
   model reason: Communication skills not explicitly stated.
19. job 2771 Monad Foundation / Trading Lead (You fit 54), skill required: **Strong communicator**
   quote: A strong communicator on paper, on stage, and in 1:1 conversations with founders.
   model reason: Strong communicator not stated in profile facts.
20. job 1530 Antares / Senior Accountant (You fit 7), skill preferred: **Time management skills**
   quote: Ability to independently manage multiple priorities and meet deadlines in a fast-paced environment.
   model reason: Time management skills not explicitly stated.

## Sample of 10 'no' lines (seed 7, pool 147)

1. job 2377 Unknown / Engineering Manager (You fit 75), qualification preferred: **Degree in CS, Engineering or related**
   quote: Degree in Computer Science, Engineering or a related discipline is desirable, but not essential.
   model reason: No degree listed in profile.
   evidence: headline: Software engineering · Technical advisory · Leadership
2. job 921 Agave API / Engineering Manager (You fit 81), skill required: **Explaining tradeoffs**
   quote: Can explain tradeoffs to founders and customers, not just engineers.
   model reason: No specific fact shows explaining tradeoffs.
3. job 2780 Blackbird Labs / Product Marketing Manager (You fit 8), skill required: **Expertise in lifecycle marketing, retention, and engagement**
   quote: Expertise in lifecycle marketing, retention, and engagement across email, push, and in-app channels.
   model reason: No lifecycle marketing retention or engagement
4. job 65 Mysten Labs / Product Lead, AI (You fit 45), experience required: **Startup/entrepreneurial experience**
   quote: Startup or similarly entrepreneurial experience, with direct ownership of building and shipping under uncertainty.
   model reason: No startup or entrepreneurial experience
5. job 2773 Kalshi / Regulatory Attorney - SEC/CFTC (You fit 8), experience preferred: **Direct SEC/CFTC experience**
   quote: Experience working directly with the SEC or CFTC, either as a regulator or representing regulated financial institutions.
   model reason: No direct SEC or CFTC experience shown
6. job 1049 Zenskar / Tech Lead (Backend) (You fit 25), experience preferred: **Early-stage startup experience**
   quote: Early-stage startup experience: comfortable with ambiguity, evolving requirements, wearing multiple hats
   model reason: Early-stage startup experience not explicitly stated in profile facts.
7. job 1806 Antares / Senior Manufacturing Engineer, Electrical Engineering (You fit 5), skill preferred: **Part inspection methods and technologies**
   any of: part inspection methods | technologies | micrometers | CMM | white light
   quote: Experience with various part inspection methods and technologies (micrometers, CMM, white light, etc.)
   model reason: No part inspection methods or technologies shown.
8. job 2785 BIT Official / Business Development Manager (You fit 21), language required: **Basic English skills**
   quote: Basic English skills for international partnerships
   model reason: US stocks experience not stated
9. job 1049 Zenskar / Tech Lead (Backend) (You fit 25), skill preferred: **Cloud platform experience**
   quote: Cloud platform experience (AWS preferred), deploying and operating services at scale
   model reason: No cloud platform experience explicitly stated in offered facts.
10. job 2785 BIT Official / Business Development Manager (You fit 21), skill required: **US stock market knowledge**
   quote: Strong industry connections and US stock market knowledge
   model reason: US stock market knowledge not stated
