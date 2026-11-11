# Q.7 re-measurement

Matcher after: ['vacancyai.match/3+33e7c9f182']. Vacancies re-matched by the new matcher: 81 (all), of them in the Q.6 set of 43: 43.

## Verdict shares

- Q.6 set (43 vacancies)
  - before: old matcher: 396 requirements; match 129 (32.6 %), partial 10 (2.5 %), no 0 (0.0 %), unclear 257 (64.9 %)
  - after:  new matcher: 396 requirements; match 157 (39.6 %), partial 31 (7.8 %), no 130 (32.8 %), unclear 78 (19.7 %)
- all re-matched (81 vacancies)
  - before: old matcher: 764 requirements; match 293 (38.4 %), partial 19 (2.5 %), no 0 (0.0 %), unclear 452 (59.2 %)
  - after:  new matcher: 764 requirements; match 306 (40.1 %), partial 49 (6.4 %), no 249 (32.6 %), unclear 160 (20.9 %)

Requirements decided by code without the model: 44 of 764 (5.8 %) over all re-matched vacancies.
In the Q.6 set: 10 of 396 (2.5 %).
Vacancies in which code decided every requirement (no model call at all): 0 of 81.

## Time per vacancy (matching only)

- all 81 timed vacancies: median 9.5 s, p90 13.9 s, mean 9.7 s, max 19.2 s
- the 81 that needed the model: median 9.5 s, p90 13.9 s, mean 9.7 s
- Q.6 for comparison (matching alone, old matcher): median 10.1 s, p90 16.4 s.

## You fit before and after

- before: min 50, median 70, max 100; below 50: 0 of 43
- after:  min 8, median 62, max 100; below 50: 10 of 43

| job | company / title | You fit before | You fit after |
|---|---|---|---|
| 2780 | Blackbird Labs / Product Marketing Manager **(unrelated role)** | 58 | 8 |
| 642 | Hadrian / Tech Lead, Operational Technology | 59 | 9 |
| 2773 | Kalshi / Regulatory Attorney - SEC/CFTC **(unrelated role)** | 50 | 11 |
| 1806 | Antares / Senior Manufacturing Engineer, Electrical Engineering | 52 | 11 |
| 1530 | Antares / Senior Accountant **(unrelated role)** | 50 | 11 |
| 2769 | Kalshi / Marketing & Advertising Compliance **(unrelated role)** | 50 | 17 |
| 2766 | MoonPay / Senior People Partner (ER) **(unrelated role)** | 50 | 25 |
| 2787 | BIT Official / Structured Products Trader **(unrelated role)** | 61 | 31 |
| 2770 | Phantom / Copywriter **(unrelated role)** | 50 | 42 |
| 658 | Hadrian / Tech Lead, Offensive Security | 58 | 46 |
| 2785 | BIT Official / Business Development Manager **(unrelated role)** | 67 | 50 |
| 2776 | Magic Eden / Senior Risk & Trading Analyst, Sportsbook | 63 | 50 |
| 2786 | Antares / Deployment Site Lead | 71 | 54 |
| 657 | Hadrian / Tech Lead, Detection & Response | 77 | 54 |
| 2694 | Mesh / Senior Engineering Manager, Cloud Infrastructure | 70 | 56 |
| 2775 | Phantom / Product Designer **(unrelated role)** | 79 | 57 |
| 2774 | Chainalysis / Director, Total Rewards | 60 | 57 |
| 2772 | Kalshi / Support Operations, Kalshi Prime **(unrelated role)** | 57 | 57 |
| 1089 | iSpot.TV / Manager, Engineering | 80 | 57 |
| 65 | Mysten Labs / Product Lead, AI | 80 | 60 |
| 907 | Mochi Health / Engineering Manager | 72 | 61 |
| 2771 | Monad Foundation / Trading Lead | 65 | 62 |
| 1619 | True Anomaly / Senior Manager, Infrastructure Engineering | 60 | 63 |
| 1001 | Courier Health / Engineering Manager | 72 | 63 |
| 1049 | Zenskar / Tech Lead (Backend) | 64 | 64 |
| 2767 | BIT Official / Trading Developer & Execution | 83 | 67 |
| 1133 | HIFI Labs / Senior Fullstack Engineer / Engineering Manager Americas | 83 | 67 |
| 2377 | Unknown / Engineering Manager | 92 | 67 |
| 1011 | GlossGenius / Senior Engineering Manager, Infrastructure  | 61 | 68 |
| 265 | True Anomaly / Senior Software Engineer, Digital Engineering | 81 | 72 |
| 1054 | Signifyd / Engineering Manager | 71 | 72 |
| 1584 | True Anomaly / Senior Manager, Digital Engineering | 71 | 73 |
| 2165 | Unknown / Growth Lead **(unrelated role)** | 50 | 73 |
| 641 | Hadrian / Senior Engineering Manager, Machine Learning | 67 | 75 |
| 854 | Svix (YC W21) / Team lead | 100 | 75 |
| 22 | Ethena Labs / Engineering Manager / Tech Lead **(named in the brief)** | 57 | 76 |
| 2109 | Prime Intellect / Product Lead | 88 | 77 |
| 1005 | Forus / Engineering Manager | 79 | 79 |
| 921 | Agave API / Engineering Manager | 96 | 81 |
| 84 | Parity / Tech Lead **(named in the brief)** | 88 | 88 |
| 1008 | Virta Health / Senior Engineering Manager - Client Group | 94 | 89 |
| 13 | Discernis / Senior Software Engineer | 90 | 90 |
| 855 | Check / Tech Lead Manager | 83 | 100 |

## Sample of 20 'unclear' lines (seed 7, pool 78)

1. job 265 True Anomaly / Senior Software Engineer, Digital Engineering (You fit 72), skill required: **Programming language**
   quote: Proficiency in one or more modern programming languages (e.g., Go, Python, TypeScript).
   model reason: 
2. job 2769 Kalshi / Marketing & Advertising Compliance (You fit 17), skill required: **Strong writing skills**
   quote: Strong writing skills and attention to detail.
   model reason: Writing skills not stated in profile facts.
3. job 2694 Mesh / Senior Engineering Manager, Cloud Infrastructure (You fit 56), skill preferred: **Payments or crypto domain knowledge**
   quote: Payments or crypto domain knowledge is a plus. QA/testing infrastructure experience is a bonus. But intellectual curiosity and a fast ramp matter more than a specific background.
   model reason: 
4. job 2774 Chainalysis / Director, Total Rewards (You fit 57), skill required: **Communication skills**
   quote: Excellent communication skills with the ability to translate complex concepts for diverse audiences
   model reason: Communication skills not stated in profile.
5. job 2772 Kalshi / Support Operations, Kalshi Prime (You fit 57), skill required: **Excellent written communication**
   quote: Excellent written communication: you can explain margin, an account hold or a document request clearly to a retail trader and to an institutional client
   model reason: Soft skill not stated in facts.
6. job 1530 Antares / Senior Accountant (You fit 11), skill preferred: **Automation interest**
   quote: Interest in using technology and automation to improve accounting processes.
   model reason: Automation interest not explicitly stated.
7. job 2772 Kalshi / Support Operations, Kalshi Prime (You fit 57), skill preferred: **Coachability**
   quote: Coachable and eager to take on more over time; you enjoy the pace and ambiguity of a startup
   model reason: Soft skill not stated in facts.
8. job 1806 Antares / Senior Manufacturing Engineer, Electrical Engineering (You fit 11), authorization required: **U.S. citizen, national, permanent resident, Refugee, or Asylee**
   quote: applicant must be a (i) U.S. citizen or national, (ii) U.S. lawful, permanent resident (aka green card holder), (iii) Refugee under 8 U.S.C. § 1157, or (iv) Asylee under 8 U.S.C. § 1158, or be eligible to obtain the required authorizations from the U.S. Depart
   model reason: 
9. job 2773 Kalshi / Regulatory Attorney - SEC/CFTC (You fit 11), skill required: **Commercial judgment**
   quote: Strong commercial judgment and the ability to translate complex regulatory issues into practical business advice.
   model reason: Commercial judgment not stated in profile
10. job 1089 iSpot.TV / Manager, Engineering (You fit 57), experience required: **System Modernization experience**
   quote: Proven track record of upgrading legacy systems and refactoring technical debt into scalable, modern platform architectures.
   model reason: 
11. job 658 Hadrian / Tech Lead, Offensive Security (You fit 46), skill required: **Reverse engineering skills**
   quote: Strong reverse engineering skills (Ghidra, IDA Pro, or Binary Ninja) and a track record of finding and weaponizing real vulnerabilities.
   model reason: 
12. job 2785 BIT Official / Business Development Manager (You fit 50), skill required: **communication and negotiation skills**
   quote: Proficient in communication, negotiation, and results-driven partnership management
   model reason: Soft skills not stated in CV
13. job 2772 Kalshi / Support Operations, Kalshi Prime (You fit 57), skill preferred: **Learning agility**
   quote: Comfort learning new tools quickly and juggling several queues and priorities at once
   model reason: Soft skill not stated in facts.
14. job 907 Mochi Health / Engineering Manager (You fit 61), skill required: **Hands-on technical leadership**
   quote: You will be a hands-on technical leader who takes deep pride in team growth. While you won't be on the critical path for daily feature delivery, you will be in the codebase—reviewing PRs, pairing with junior engineers to unblock them, and actively recruiting t
   model reason: 
15. job 854 Svix (YC W21) / Team lead (You fit 75), skill required: **Technical, architectural, and troubleshooting skills**
   quote: Strong technical, architectural, and troubleshooting skills.
   model reason: 
16. job 1089 iSpot.TV / Manager, Engineering (You fit 57), skill required: **Communication skills**
   quote: Exceptional communication skills to align engineering efforts with Product, Data Science, and Operations objectives.
   model reason: Communication skills not explicitly stated.
17. job 2771 Monad Foundation / Trading Lead (You fit 62), skill required: **Strong communicator**
   quote: A strong communicator on paper, on stage, and in 1:1 conversations with founders.
   model reason: Strong communication skills not explicitly stated.
18. job 2780 Blackbird Labs / Product Marketing Manager (You fit 8), other required: **Scrappy, cross-functional leader with a passion for hospitality**
   quote: Scrappy, cross-functional leader with a passion for hospitality and building programs from scratch.
   model reason: 
19. job 1584 True Anomaly / Senior Manager, Digital Engineering (You fit 73), qualification preferred: **Bachelor's degree in Computer Science or Engineering**
   quote: Bachelor's degree in Computer Science, Engineering, or equivalent professional experience.
   model reason: 
20. job 1530 Antares / Senior Accountant (You fit 11), qualification required: **Bachelor's degree in Accounting**
   quote: Bachelor’s degree in Accounting, Finance, or a related field.
   model reason: 

## Sample of 10 'no' lines (seed 7, pool 130)

1. job 2780 Blackbird Labs / Product Marketing Manager (You fit 8), experience required: **Track record marketing to multiple audiences and leading complex launches**
   quote: Proven track record marketing to multiple audiences and leading complex launches.
   model reason: No marketing to multiple audiences or launches
2. job 2773 Kalshi / Regulatory Attorney - SEC/CFTC (You fit 11), experience preferred: **Direct SEC/CFTC experience**
   quote: Experience working directly with the SEC or CFTC, either as a regulator or representing regulated financial institutions.
   model reason: No direct SEC or CFTC experience in profile
3. job 642 Hadrian / Tech Lead, Operational Technology (You fit 9), qualification required: **Experience with IEC 62443 and NIST SP 800-82**
   quote: Experience with standards such as IEC 62443 and NIST SP 800-82.
   model reason: No IEC 62443 or NIST SP 800-82 mentioned
4. job 1054 Signifyd / Engineering Manager (You fit 72), skill required: **MLOps toolkits**
   quote: Strong knowledge of software development best practices, cloud computing platforms, big data technologies, MLOps toolkits, and SLO-driven reliability management.
   model reason: No MLOps toolkits listed in facts.
5. job 2785 BIT Official / Business Development Manager (You fit 50), experience preferred: **experience in US stocks**
   quote: Deployable partner resources and experience in US stocks
   model reason: Profile shows Web3 not US stocks
   evidence: experience: Head of Development, Project Manager at Nival Interactive, 2004 to 2007: Led production of Ubisoft's Heroes of Might and Magic V across offices in Russia, Belarus, Cyprus and USA, managing large teams
6. job 642 Hadrian / Tech Lead, Operational Technology (You fit 9), skill required: **Knowledge of industrial environments**
   quote: Deep knowledge of industrial environments including PLCs, SCADA/DCS, HMIs, RTUs, and industrial protocols.
   model reason: No industrial environment experience found
7. job 2785 BIT Official / Business Development Manager (You fit 50), experience required: **3+ years BD in online brokerages or fintech**
   quote: 3+ years in Business Development in online brokerages or fintech with partnership successes
   model reason: Profile shows gaming industry not fintech
   evidence: experience: Head of Development, Project Manager at Nival Interactive, 2004 to 2007: Led production of Ubisoft's Heroes of Might and Magic V across offices in Russia, Belarus, Cyprus and USA, managing large teams
8. job 2773 Kalshi / Regulatory Attorney - SEC/CFTC (You fit 11), experience preferred: **Background at leading firms**
   quote: Background at a leading financial services law firm, exchange, trading firm, or high-growth fintech.
   model reason: No background at leading firms stated in profile
9. job 265 True Anomaly / Senior Software Engineer, Digital Engineering (You fit 72), skill preferred: **GitOps**
   quote: Experience with Go, infrastructure-as-code (e.g., Terraform), and GitOps.
   model reason: No GitOps experience stated in profile.
10. job 1133 HIFI Labs / Senior Fullstack Engineer / Engineering Manager Americas (You fit 67), experience required: **startup experience with many hats**
   quote: worked in startups where you take on many hats
   model reason: No specific startup experience facts offered.
