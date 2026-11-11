# Q.3b regression: expected versus received (extraction pass, qwen3.5:9b)

Six vacancies, expectations written by hand from the vacancy text before any run (`q3b-regression/expected.json`): Parity 84, Deepgram 2616, Dropbox 2520 (the three of the Q.3 live run) and three more chosen for what they test: Chalk 834 ("This is not a hybrid role", in the office five days), OKX 32 (glued paragraph, "PhD or Master's", right-to-work preference), Zscaler 29 (glued, "such as C, C++, Python", a `#LI-Hybrid` tag, "is a plus"). Model `qwen3.5:9b` on local Ollama, thinking off, temperature 0.1, one attempt per vacancy. The text of each vacancy is the one in the staging copy (`.local/assessment-3/data`); for Parity it is the 4 552-character text taken from the ASSESSMENT-2 review copy, as in the Q.3 run.

Two prompt versions were run twice each: the **old** one (commit `ff47b0a`, extractor `vacancyai.extract/1+d104a3a836`, run from a temporary checkout of that commit) and the **new** one (this commit, `vacancyai.extract/2+d9df4c7197`, run three times). The model is not deterministic, so single runs differ by a few items. `compare.py` matches an expected requirement to a received one by regular expressions on its text, alternatives and quote (listed in `expected.json`), so a match is a judgement of the table, not of the model; the "other requirements received" lines in the tables below are left for a human to judge.

## Summary

| Run | Expected requirements found (of 52) | Level correct | Groups returned with alternatives (of 6) | Forbidden items received | Conditions correct (of 36) |
|---|---|---|---|---|---|
| old, run 1 | 36 | 34 | 0 | 1 | 32 |
| old, run 2 | 35 | 32 | 0 | 4 | 32 |
| new, run 1 | 42 | 41 | 5 | 1 | 33 |
| new, run 2 | 42 | 40 | 4 | 1 | 32 |
| new, run 3 | 42 | 41 | 5 | 1 | 33 |

Requirements kept per vacancy (84, 2616, 2520, 834, 32, 29): old 5, 12, 2, 3, 8, 7; new 5, 12, 7, 5, 8, 8. Time for the six vacancies: old 45–64 s, new 58–73 s.

## What the iterations on the way to the new version showed

- First new prompt, schema edit did not apply (my own mistake: the substitution that added `alternatives` to the schema matched nothing and went unnoticed): the model never returned `alternatives` (0 of 6 groups, 34 and 30 of 52 found). A direct request to Ollama with the field in the schema returned it, so the field must be in the schema, not only in the prompt.
- Second version, schema fixed, title and company removed from the request so that the key and the input both be the description alone: Dropbox 2520 returned **zero requirements** in every run (and the answer would have been stored as final "no requirements stated"), where the old version returned two. Measured outside the pipeline on that vacancy: the bare description 0 requirements (twice), a frame `DESCRIPTION:` without title 0, title and company plus description 7, description with the trailing "other jobs" list cut off 6. The page opens with navigation text, and the title orients the model. The title and company therefore stay in the request; the extraction key remains the description alone.
- Final version also asks again when the model returns an empty list for a text that contains requirement wording; only an answer that stays empty through every attempt is final.

Results of those two intermediate versions are kept in `first-attempt/` and `second-attempt/`.

## Still wrong in the final version

- Chalk 834: the model returns 5 of 11 expected requirements and files the task "Write C++, Python and Rust" as requirements ("C++ programming" is received in place of "Rust" as preferred).
- Parity 84: the work mode `remote` is not reported (the model says `hybrid`, quoting the "50/50 hybrid role" sentence, and the guard drops it; null is better than a wrong value). Chalk's office is likewise dropped because the model quotes the negation "This is not a hybrid role." The prompt now asks for the sentence that states the actual place, and that did not change the model's choice: in all three final runs it quotes the negation for Chalk and the "50/50 hybrid role" sentence for Parity.
- Dropbox 2520 gets `engagement = employment` in all three runs from the "Job type Full-Time" line; the expectation was null (a judgement call: a job type of full-time is employment more often than not).
- "Sprint management / dependency tracking" (Parity), "Finds problems and proposes solutions" (Deepgram), "Track record delivering software solutions" (Dropbox) are skipped in some runs: the 9B model drops some of the softer requirements.

## Tables

## Run `q3b-regression/old-run1.json` — extractor `vacancyai.extract/1+d104a3a836`, model `qwen3.5:9b`

### 84 — Tech Lead | Parity (glued paragraph; '50/50 hybrid role' is about duties)

5 requirements kept, 15.2 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| Software engineer / tech lead background, production code and architecture | required | Software Engineer Tech Lead experience [experience, required] | ✔ | ✔ |  |
| Managing or leading engineering teams | required | Software Engineer Tech Lead experience [experience, required] | ✔ | ✔ |  |
| Founder / early-stage startup experience | preferred | Founder or startup lead experience [qualification, preferred] | ✔ | ✔ |  |
| Interest or background in Web3, blockchain, distributed or low-level systems (Rust, Substrate, Go, C++ are examples) | preferred | Web3 blockchain distributed systems skills [skill, required] | ✔ | ✗ required | ✗ (no alternatives) |
| Sprint management, dependency tracking, cross-team communication | required | — | ✗ | | |
| Asynchronous communication, European or aligned time zones | required | Asynchronous communication skills [skill, required] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- Engineering team leadership experience [experience, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | remote | hybrid | ✗ |
| engagement | contract | contract | ✔ |
| hours | null | full-time | ✗ |
| timezone | either | European or closely aligned time zones | ✔ |
| compensation | null | null | ✔ |
| location | either | null | ✔ |

### 2616 — Software Engineer - Applied AI | Deepgram (stack sentence, in-person, '9-to-5' negated)

12 requirements kept, 16.2 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| Backend services shipped in production in the cloud | required | Backend services experience [skill, required] | ✔ | ✔ |  |
| AI-native way of working with AI coding tools | required | AI coding tools proficiency [skill, required] | ✔ | ✔ |  |
| Learns quickly, unfamiliar parts of the stack | required | Fast learning ability [skill, required] | ✔ | ✔ |  |
| Finds problems and proposes solutions | required | Proactive problem solving [skill, required] | ✔ | ✔ |  |
| Took something from idea to production | required | Idea to production experience [experience, required] | ✔ | ✔ |  |
| Clear communication, works with customers | required | Clear communication skills [skill, required] | ✔ | ✔ |  |
| Comfortable working across languages (stack: Kotlin, Python, TypeScript) | required | Kotlin and Python proficiency [language, required] | ✔ | ✔ |  |
| Model training / fine-tuning experience | preferred | Model training experience [experience, preferred] | ✔ | ✔ |  |
| Evaluation or scoring for models or agents | preferred | Evaluation and scoring experience [skill, preferred] | ✔ | ✔ |  |
| Audio, speech-to-text or text-to-speech | preferred | Audio STT TTS experience [skill, preferred] | ✔ | ✔ |  |
| Founder or early startup, zero to one | preferred | Founder or startup experience [experience, preferred] | ✔ | ✔ |  |
| 2+ years of engineering experience | preferred | 2+ years engineering experience [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | office | null | ✗ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | null | ✔ |

### 2520 — Backend Product Software Engineer | Dropbox (list page: other listings and pay ranges follow the text)

2 requirements kept, 6.0 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 5+ years of software engineering experience | required | 5+ years software engineering [experience, required] | ✔ | ✔ |  |
| BS in computer science or equivalent experience | required | BS degree CS or related field [qualification, required] | ✔ | ✔ |  |
| Track record delivering software solutions through milestones | required | — | ✗ | | |
| Shipped large-scale web applications | required | — | ✗ | | |
| Building user-facing software for web / desktop | required | — | ✗ | | |
| Open-source contributions | preferred | — | ✗ | | |
| Full-stack development | preferred | — | ✗ | | |
| Technologies: MySQL, Python, Go, Java, Rust | preferred | — | ✗ | | |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | null | null | ✔ |
| engagement | null | null | ✔ |
| hours | either | full-time | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Anywhere in the World | ✔ |

### 834 — Software Engineer | Chalk ('This is not a hybrid role', in the office 5 days)

3 requirements kept, 5.5 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 4+ years of professional backend engineering | required | 4+ years backend experience [experience, required] | ✔ | ✔ |  |
| Strong computer science fundamentals | required | — | ✗ | | |
| Comfort leading development of new components, ownership | required | — | ✗ | | |
| Learns new concepts quickly | required | — | ✗ | | |
| Works autonomously, pragmatic tradeoffs | required | — | ✗ | | |
| Strong communication skills | required | — | ✗ | | |
| Bachelor's degree in Computer Science or equivalent | required | Bachelor CS degree [qualification, required] | ✔ | ✔ |  |
| Data infrastructure experience | preferred | — | ✗ | | |
| Distributed systems / compilers / query planners | preferred | — | ✗ | | |
| Rust experience | preferred | C++ Python Rust [skill, required] | ✔ | ✗ required |  |
| PaaS-like software | preferred | — | ✗ | | |

Forbidden items received: 1
- ✗ C++ Python Rust [skill, required] — 'Write C++, Python and Rust' is a task, not a requirement

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | office | office | ✔ |
| engagement | null | null | ✔ |
| hours | null | full-time | ✗ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | office | ✔ |

### 32 — Principal Engineer | OKX (glued, 'PhD or Master's', right-to-work preference, 'e.g.' in the role description)

8 requirements kept, 10.8 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| PhD or Master's in CS, AI, ML, Robotics or a related field | required | PhD or Master in CS/AI/ML/Robotics [qualification, required] | ✔ | ✔ |  |
| 10+ years of industry experience | required | 10+ years industry experience [experience, required] | ✔ | ✔ |  |
| Deep expertise in long-term memory, knowledge graphs or context management | required | Expertise in long-term memory/knowledge graphs/context management [skill, required] | ✔ | ✔ | ✗ (no alternatives) |
| Right to work in Singapore without sponsorship | any | Right to work in Singapore [authorization, required] | ✔ | ✔ |  |
| Experience in top-tier AI labs or tech leaders | preferred | Experience in top-tier AI labs or Tech industry leaders [experience, preferred] | ✔ | ✔ |  |
| Experience in high-frequency trading, fintech, Web3 or crypto | preferred | Experience in HFT/fintech/Web3/crypto [experience, preferred] | ✔ | ✔ |  |
| Startups or research-intensive environments, comfortable with ambiguity | preferred | Experience in startups or research-intensive environments [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- No visa sponsorship required [authorization, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | null | null | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Singapore | ✔ |

### 29 — Staff Software Development Engineer | Zscaler (glued, 'such as C, C++, Python', '#LI-Hybrid' tag, 'is a plus')

7 requirements kept, 10.4 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 5+ years in software development or sustenance engineering | required | 5+ years software development [experience, required] | ✔ | ✔ |  |
| Proficiency in programming languages such as C, C++, Python and Linux | required | C, C++, Python, Linux proficiency [skill, required] | ✔ | ✔ | ✗ (no alternatives) |
| Hands-on experience with network security products (firewalls, VPNs, data protection) | required | firewalls, VPNs, data protection [skill, required] | ✔ | ✔ | ✗ (no alternatives) |
| Experience developing cloud and customer reliability tools | required | cloud reliability tools experience [experience, required] | ✔ | ✔ |  |
| Interest in troubleshooting customer issues; public cloud / Kubernetes knowledge is a plus | any | public cloud, kubernetes knowledge [skill, preferred] | ✔ | ✔ |  |
| Bachelor's degree in Computer Science or engineering | preferred | Bachelor's degree CS or Engineering [qualification, preferred] | ✔ | ✔ |  |
| Cloud reliability engineering experience | preferred | cloud reliability engineering experience [experience, preferred] | ✔ | ✔ |  |
| Delivering products as cloud-based services | preferred | — | ✗ | | |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | either | hybrid | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Burnaby, Canada | ✔ |

**Totals for this run:** expected requirements found 36/52; level correct 34/36; groups with alternatives 0/6; forbidden items received 1; conditions correct 32/36.


## Run `q3b-regression/old-run2.json` — extractor `vacancyai.extract/1+d104a3a836`, model `qwen3.5:9b`

### 84 — Tech Lead | Parity (glued paragraph; '50/50 hybrid role' is about duties)

5 requirements kept, 8.0 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| Software engineer / tech lead background, production code and architecture | required | Software Engineer Tech Lead experience [experience, required] | ✔ | ✔ |  |
| Managing or leading engineering teams | required | Software Engineer Tech Lead experience [experience, required] | ✔ | ✔ |  |
| Founder / early-stage startup experience | preferred | Founder or startup lead experience [qualification, preferred] | ✔ | ✔ |  |
| Interest or background in Web3, blockchain, distributed or low-level systems (Rust, Substrate, Go, C++ are examples) | preferred | Web3 blockchain distributed systems skill [skill, required] | ✔ | ✗ required | ✗ (no alternatives) |
| Sprint management, dependency tracking, cross-team communication | required | — | ✗ | | |
| Asynchronous communication, European or aligned time zones | required | Asynchronous communication skill [skill, required] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- Engineering team leadership experience [experience, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | remote | hybrid | ✗ |
| engagement | contract | contract | ✔ |
| hours | null | full-time | ✗ |
| timezone | either | European or closely aligned time zones | ✔ |
| compensation | null | null | ✔ |
| location | either | null | ✔ |

### 2616 — Software Engineer - Applied AI | Deepgram (stack sentence, in-person, '9-to-5' negated)

13 requirements kept, 13.0 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| Backend services shipped in production in the cloud | required | Backend services experience [skill, required] | ✔ | ✔ |  |
| AI-native way of working with AI coding tools | required | Modern AI coding tools [skill, required] | ✔ | ✔ |  |
| Learns quickly, unfamiliar parts of the stack | required | Quick learner [skill, required] | ✔ | ✔ |  |
| Finds problems and proposes solutions | required | — | ✗ | | |
| Took something from idea to production | required | Idea to production experience [experience, required] | ✔ | ✔ |  |
| Clear communication, works with customers | required | Communication skills [skill, required] | ✔ | ✔ |  |
| Comfortable working across languages (stack: Kotlin, Python, TypeScript) | required | Kotlin [language, required] | ✔ | ✔ |  |
| Model training / fine-tuning experience | preferred | Model training experience [experience, preferred] | ✔ | ✔ |  |
| Evaluation or scoring for models or agents | preferred | Evaluation experience [skill, preferred] | ✔ | ✔ |  |
| Audio, speech-to-text or text-to-speech | preferred | Audio experience [experience, preferred] | ✔ | ✔ |  |
| Founder or early startup, zero to one | preferred | Founder experience [experience, preferred] | ✔ | ✔ |  |
| 2+ years of engineering experience | preferred | 2+ years experience [experience, required] | ✔ | ✗ required |  |

Forbidden items received: 3
- ✗ Kotlin [language, required] — the company's own stack as a spoken-language requirement or separate requirement
- ✗ Python [language, required] — the company's own stack as a spoken-language requirement or separate requirement
- ✗ TypeScript [language, required] — the company's own stack as a spoken-language requirement or separate requirement

Other requirements received (not in the expected list; judge by hand): 2
- Python [language, required]
- TypeScript [language, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | office | null | ✗ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | null | ✔ |

### 2520 — Backend Product Software Engineer | Dropbox (list page: other listings and pay ranges follow the text)

2 requirements kept, 3.4 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 5+ years of software engineering experience | required | 5+ years software engineering [experience, required] | ✔ | ✔ |  |
| BS in computer science or equivalent experience | required | BS degree CS or related [qualification, required] | ✔ | ✔ |  |
| Track record delivering software solutions through milestones | required | — | ✗ | | |
| Shipped large-scale web applications | required | — | ✗ | | |
| Building user-facing software for web / desktop | required | — | ✗ | | |
| Open-source contributions | preferred | — | ✗ | | |
| Full-stack development | preferred | — | ✗ | | |
| Technologies: MySQL, Python, Go, Java, Rust | preferred | — | ✗ | | |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | null | null | ✔ |
| engagement | null | null | ✔ |
| hours | either | full-time | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Anywhere in the World | ✔ |

### 834 — Software Engineer | Chalk ('This is not a hybrid role', in the office 5 days)

3 requirements kept, 4.2 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 4+ years of professional backend engineering | required | 4+ years backend experience [experience, required] | ✔ | ✔ |  |
| Strong computer science fundamentals | required | — | ✗ | | |
| Comfort leading development of new components, ownership | required | — | ✗ | | |
| Learns new concepts quickly | required | — | ✗ | | |
| Works autonomously, pragmatic tradeoffs | required | — | ✗ | | |
| Strong communication skills | required | — | ✗ | | |
| Bachelor's degree in Computer Science or equivalent | required | Bachelor CS degree [qualification, required] | ✔ | ✔ |  |
| Data infrastructure experience | preferred | — | ✗ | | |
| Distributed systems / compilers / query planners | preferred | — | ✗ | | |
| Rust experience | preferred | C++ Python Rust [skill, required] | ✔ | ✗ required |  |
| PaaS-like software | preferred | — | ✗ | | |

Forbidden items received: 1
- ✗ C++ Python Rust [skill, required] — 'Write C++, Python and Rust' is a task, not a requirement

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | office | office | ✔ |
| engagement | null | null | ✔ |
| hours | null | full-time | ✗ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | office | ✔ |

### 32 — Principal Engineer | OKX (glued, 'PhD or Master's', right-to-work preference, 'e.g.' in the role description)

8 requirements kept, 8.6 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| PhD or Master's in CS, AI, ML, Robotics or a related field | required | PhD or Master in CS/AI/ML/Robotics [qualification, required] | ✔ | ✔ |  |
| 10+ years of industry experience | required | 10+ years industry experience [experience, required] | ✔ | ✔ |  |
| Deep expertise in long-term memory, knowledge graphs or context management | required | Expertise in long-term memory/knowledge graphs/context management [skill, required] | ✔ | ✔ | ✗ (no alternatives) |
| Right to work in Singapore without sponsorship | any | Right to work in Singapore [authorization, required] | ✔ | ✔ |  |
| Experience in top-tier AI labs or tech leaders | preferred | Experience in top-tier AI labs or Tech industry leaders [experience, preferred] | ✔ | ✔ |  |
| Experience in high-frequency trading, fintech, Web3 or crypto | preferred | Experience in HFT/fintech/Web3/crypto [experience, preferred] | ✔ | ✔ |  |
| Startups or research-intensive environments, comfortable with ambiguity | preferred | Experience in startups or research-intensive environments [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- No visa sponsorship required [authorization, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | null | null | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Singapore | ✔ |

### 29 — Staff Software Development Engineer | Zscaler (glued, 'such as C, C++, Python', '#LI-Hybrid' tag, 'is a plus')

7 requirements kept, 7.8 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 5+ years in software development or sustenance engineering | required | 5+ years software development [experience, required] | ✔ | ✔ |  |
| Proficiency in programming languages such as C, C++, Python and Linux | required | C, C++, Python, Linux proficiency [skill, required] | ✔ | ✔ | ✗ (no alternatives) |
| Hands-on experience with network security products (firewalls, VPNs, data protection) | required | firewalls, VPNs, data protection [skill, required] | ✔ | ✔ | ✗ (no alternatives) |
| Experience developing cloud and customer reliability tools | required | cloud and customer reliability tools experience [experience, required] | ✔ | ✔ |  |
| Interest in troubleshooting customer issues; public cloud / Kubernetes knowledge is a plus | any | public cloud or kubernetes knowledge [skill, preferred] | ✔ | ✔ |  |
| Bachelor's degree in Computer Science or engineering | preferred | Bachelor's degree CS or Engineering [qualification, preferred] | ✔ | ✔ |  |
| Cloud reliability engineering experience | preferred | cloud reliability engineering experience [experience, preferred] | ✔ | ✔ |  |
| Delivering products as cloud-based services | preferred | — | ✗ | | |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | either | hybrid | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Burnaby, Canada | ✔ |

**Totals for this run:** expected requirements found 35/52; level correct 32/35; groups with alternatives 0/6; forbidden items received 4; conditions correct 32/36.


## Run `q3b-regression/new-run1.json` — extractor `vacancyai.extract/2+d9df4c7197`, model `qwen3.5:9b`

### 84 — Tech Lead | Parity (glued paragraph; '50/50 hybrid role' is about duties)

5 requirements kept, 12.0 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| Software engineer / tech lead background, production code and architecture | required | Software Engineer, Tech Lead, or Engineering Manager [experience, required] | ✔ | ✔ |  |
| Managing or leading engineering teams | required | Software Engineer, Tech Lead, or Engineering Manager [experience, required] | ✔ | ✔ |  |
| Founder / early-stage startup experience | preferred | Founder, co-founder, or early-stage startup lead [experience, preferred] | ✔ | ✔ |  |
| Interest or background in Web3, blockchain, distributed or low-level systems (Rust, Substrate, Go, C++ are examples) | preferred | Web3, blockchain infrastructure, distributed systems, or low-level systems programming [skill, preferred] | ✔ | ✔ | ✔ |
| Sprint management, dependency tracking, cross-team communication | required | — | ✗ | | |
| Asynchronous communication, European or aligned time zones | required | Excellent asynchronous communication skills [skill, required] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- Experience managing or leading engineering teams [experience, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | remote | null | ✗ |
| engagement | contract | contract | ✔ |
| hours | null | null | ✔ |
| timezone | either | European or closely aligned time zones | ✔ |
| compensation | null | null | ✔ |
| location | either | null | ✔ |

### 2616 — Software Engineer - Applied AI | Deepgram (stack sentence, in-person, '9-to-5' negated)

12 requirements kept, 16.6 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| Backend services shipped in production in the cloud | required | Cloud backend experience [skill, required] | ✔ | ✔ |  |
| AI-native way of working with AI coding tools | required | AI coding tools [skill, required] | ✔ | ✔ |  |
| Learns quickly, unfamiliar parts of the stack | required | Fast learner [skill, required] | ✔ | ✔ |  |
| Finds problems and proposes solutions | required | — | ✗ | | |
| Took something from idea to production | required | Idea to production experience [experience, required] | ✔ | ✔ |  |
| Clear communication, works with customers | required | Customer communication [skill, required] | ✔ | ✔ |  |
| Comfortable working across languages (stack: Kotlin, Python, TypeScript) | required | Multiple languages [language, required] | ✔ | ✔ |  |
| Model training / fine-tuning experience | preferred | Model training experience [experience, preferred] | ✔ | ✔ |  |
| Evaluation or scoring for models or agents | preferred | Evaluation experience [experience, preferred] | ✔ | ✔ |  |
| Audio, speech-to-text or text-to-speech | preferred | Audio/Speech skills [skill, preferred] | ✔ | ✔ |  |
| Founder or early startup, zero to one | preferred | Founder experience [experience, preferred] | ✔ | ✔ |  |
| 2+ years of engineering experience | preferred | Engineering experience [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- Backend language [skill, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | office | office | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | San Francisco | ✔ |

### 2520 — Backend Product Software Engineer | Dropbox (list page: other listings and pay ranges follow the text)

7 requirements kept, 11.7 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 5+ years of software engineering experience | required | 5+ years software engineering [experience, required] | ✔ | ✔ |  |
| BS in computer science or equivalent experience | required | BS degree CS or related [qualification, required] | ✔ | ✔ |  |
| Track record delivering software solutions through milestones | required | — | ✗ | | |
| Shipped large-scale web applications | required | shipped large-scale web apps [skill, required] | ✔ | ✔ |  |
| Building user-facing software for web / desktop | required | build user-facing software [skill, required] | ✔ | ✔ |  |
| Open-source contributions | preferred | open source contributions [skill, preferred] | ✔ | ✔ |  |
| Full-stack development | preferred | Full-Stack Development [skill, preferred] | ✔ | ✔ |  |
| Technologies: MySQL, Python, Go, Java, Rust | preferred | Technologies [skill, preferred] | ✔ | ✔ | ✔ |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | null | null | ✔ |
| engagement | null | employment | ✗ |
| hours | either | full-time | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Anywhere in the World | ✔ |

### 834 — Software Engineer | Chalk ('This is not a hybrid role', in the office 5 days)

5 requirements kept, 7.8 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 4+ years of professional backend engineering | required | 4+ years backend software engineering [experience, required] | ✔ | ✔ |  |
| Strong computer science fundamentals | required | Strong computer science fundamentals [skill, required] | ✔ | ✔ |  |
| Comfort leading development of new components, ownership | required | — | ✗ | | |
| Learns new concepts quickly | required | — | ✗ | | |
| Works autonomously, pragmatic tradeoffs | required | — | ✗ | | |
| Strong communication skills | required | — | ✗ | | |
| Bachelor's degree in Computer Science or equivalent | required | Bachelor's degree in Computer Science [qualification, required] | ✔ | ✔ |  |
| Data infrastructure experience | preferred | — | ✗ | | |
| Distributed systems / compilers / query planners | preferred | — | ✗ | | |
| Rust experience | preferred | C++ [skill, required] | ✔ | ✗ required |  |
| PaaS-like software | preferred | — | ✗ | | |

Forbidden items received: 1
- ✗ C++ [skill, required] — 'Write C++, Python and Rust' is a task, not a requirement

Other requirements received (not in the expected list; judge by hand): 1
- Backend language [skill, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | office | null | ✗ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | the office | ✔ |

### 32 — Principal Engineer | OKX (glued, 'PhD or Master's', right-to-work preference, 'e.g.' in the role description)

8 requirements kept, 12.2 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| PhD or Master's in CS, AI, ML, Robotics or a related field | required | PhD or Master's in CS/AI/ML/Robotics [qualification, required] | ✔ | ✔ |  |
| 10+ years of industry experience | required | 10+ years industry experience [experience, required] | ✔ | ✔ |  |
| Deep expertise in long-term memory, knowledge graphs or context management | required | Expertise in memory/graphs/context [skill, required] | ✔ | ✔ | ✔ |
| Right to work in Singapore without sponsorship | any | Right to work in Singapore [authorization, required] | ✔ | ✔ |  |
| Experience in top-tier AI labs or tech leaders | preferred | Top-tier AI lab experience [experience, preferred] | ✔ | ✔ |  |
| Experience in high-frequency trading, fintech, Web3 or crypto | preferred | HFT/fintech/Web3/crypto experience [experience, preferred] | ✔ | ✔ |  |
| Startups or research-intensive environments, comfortable with ambiguity | preferred | Startup/research environment experience [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- No visa sponsorship required [authorization, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | null | null | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Singapore | ✔ |

### 29 — Staff Software Development Engineer | Zscaler (glued, 'such as C, C++, Python', '#LI-Hybrid' tag, 'is a plus')

8 requirements kept, 12.7 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 5+ years in software development or sustenance engineering | required | 5+ years software development [experience, required] | ✔ | ✔ |  |
| Proficiency in programming languages such as C, C++, Python and Linux | required | Programming languages [skill, required] | ✔ | ✔ | ✔ |
| Hands-on experience with network security products (firewalls, VPNs, data protection) | required | Network security products [skill, required] | ✔ | ✔ | ✔ |
| Experience developing cloud and customer reliability tools | required | Cloud reliability tools experience [experience, required] | ✔ | ✔ |  |
| Interest in troubleshooting customer issues; public cloud / Kubernetes knowledge is a plus | any | Public cloud or Kubernetes knowledge [skill, preferred] | ✔ | ✔ |  |
| Bachelor's degree in Computer Science or engineering | preferred | Bachelor's degree in CS or Engineering [qualification, preferred] | ✔ | ✔ |  |
| Cloud reliability engineering experience | preferred | Cloud reliability engineering experience [experience, preferred] | ✔ | ✔ |  |
| Delivering products as cloud-based services | preferred | Success delivering cloud-based products [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | either | null | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Burnaby, Canada | ✔ |

**Totals for this run:** expected requirements found 42/52; level correct 41/42; groups with alternatives 5/6; forbidden items received 1; conditions correct 33/36.


## Run `q3b-regression/new-run2.json` — extractor `vacancyai.extract/2+d9df4c7197`, model `qwen3.5:9b`

### 84 — Tech Lead | Parity (glued paragraph; '50/50 hybrid role' is about duties)

5 requirements kept, 8.6 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| Software engineer / tech lead background, production code and architecture | required | Software Engineer, Tech Lead, or Engineering Manager [experience, required] | ✔ | ✔ |  |
| Managing or leading engineering teams | required | Software Engineer, Tech Lead, or Engineering Manager [experience, required] | ✔ | ✔ |  |
| Founder / early-stage startup experience | preferred | Founder, co-founder, or early-stage startup lead [experience, preferred] | ✔ | ✔ |  |
| Interest or background in Web3, blockchain, distributed or low-level systems (Rust, Substrate, Go, C++ are examples) | preferred | Web3, blockchain infrastructure, distributed systems, or low-level systems programming [skill, preferred] | ✔ | ✔ | ✔ |
| Sprint management, dependency tracking, cross-team communication | required | — | ✗ | | |
| Asynchronous communication, European or aligned time zones | required | Excellent asynchronous communication skills [skill, required] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- Experience managing or leading engineering teams [experience, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | remote | null | ✗ |
| engagement | contract | contract | ✔ |
| hours | null | null | ✔ |
| timezone | either | European or closely aligned time zones | ✔ |
| compensation | null | null | ✔ |
| location | either | null | ✔ |

### 2616 — Software Engineer - Applied AI | Deepgram (stack sentence, in-person, '9-to-5' negated)

12 requirements kept, 13.8 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| Backend services shipped in production in the cloud | required | Cloud backend experience [skill, required] | ✔ | ✔ |  |
| AI-native way of working with AI coding tools | required | AI coding tools [skill, required] | ✔ | ✔ |  |
| Learns quickly, unfamiliar parts of the stack | required | Fast learner [skill, required] | ✔ | ✔ |  |
| Finds problems and proposes solutions | required | — | ✗ | | |
| Took something from idea to production | required | Idea to production experience [experience, required] | ✔ | ✔ |  |
| Clear communication, works with customers | required | Customer communication [skill, required] | ✔ | ✔ |  |
| Comfortable working across languages (stack: Kotlin, Python, TypeScript) | required | Multiple languages [language, required] | ✔ | ✔ |  |
| Model training / fine-tuning experience | preferred | Model training experience [experience, preferred] | ✔ | ✔ |  |
| Evaluation or scoring for models or agents | preferred | Evaluation experience [experience, preferred] | ✔ | ✔ |  |
| Audio, speech-to-text or text-to-speech | preferred | Audio/Speech skills [skill, preferred] | ✔ | ✔ |  |
| Founder or early startup, zero to one | preferred | Founder experience [experience, preferred] | ✔ | ✔ |  |
| 2+ years of engineering experience | preferred | 2+ years engineering experience [experience, required] | ✔ | ✗ required |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- Backend language [skill, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | office | office | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | San Francisco | ✔ |

### 2520 — Backend Product Software Engineer | Dropbox (list page: other listings and pay ranges follow the text)

7 requirements kept, 9.1 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 5+ years of software engineering experience | required | 5+ years software engineering [experience, required] | ✔ | ✔ |  |
| BS in computer science or equivalent experience | required | BS degree CS or related [qualification, required] | ✔ | ✔ |  |
| Track record delivering software solutions through milestones | required | — | ✗ | | |
| Shipped large-scale web applications | required | shipped large-scale web apps [skill, required] | ✔ | ✔ |  |
| Building user-facing software for web / desktop | required | build user-facing software [skill, required] | ✔ | ✔ |  |
| Open-source contributions | preferred | open source contributions [skill, preferred] | ✔ | ✔ |  |
| Full-stack development | preferred | Full-Stack Development [skill, preferred] | ✔ | ✔ |  |
| Technologies: MySQL, Python, Go, Java, Rust | preferred | Technologies [skill, preferred] | ✔ | ✔ | ✔ |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | null | null | ✔ |
| engagement | null | employment | ✗ |
| hours | either | full-time | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Anywhere in the World | ✔ |

### 834 — Software Engineer | Chalk ('This is not a hybrid role', in the office 5 days)

5 requirements kept, 6.3 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 4+ years of professional backend engineering | required | 4+ years backend software engineering [experience, required] | ✔ | ✔ |  |
| Strong computer science fundamentals | required | Computer science fundamentals [skill, required] | ✔ | ✔ |  |
| Comfort leading development of new components, ownership | required | — | ✗ | | |
| Learns new concepts quickly | required | — | ✗ | | |
| Works autonomously, pragmatic tradeoffs | required | — | ✗ | | |
| Strong communication skills | required | — | ✗ | | |
| Bachelor's degree in Computer Science or equivalent | required | Bachelor's degree in Computer Science [qualification, required] | ✔ | ✔ |  |
| Data infrastructure experience | preferred | — | ✗ | | |
| Distributed systems / compilers / query planners | preferred | — | ✗ | | |
| Rust experience | preferred | C++ [skill, required] | ✔ | ✗ required |  |
| PaaS-like software | preferred | — | ✗ | | |

Forbidden items received: 1
- ✗ C++ [skill, required] — 'Write C++, Python and Rust' is a task, not a requirement

Other requirements received (not in the expected list; judge by hand): 1
- Backend language [skill, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | office | null | ✗ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | office | ✔ |

### 32 — Principal Engineer | OKX (glued, 'PhD or Master's', right-to-work preference, 'e.g.' in the role description)

8 requirements kept, 10.3 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| PhD or Master's in CS, AI, ML, Robotics or a related field | required | PhD or Master's in CS/AI/ML/Robotics [qualification, required] | ✔ | ✔ |  |
| 10+ years of industry experience | required | 10+ years industry experience [experience, required] | ✔ | ✔ |  |
| Deep expertise in long-term memory, knowledge graphs or context management | required | Expertise in memory/graphs/context [skill, required] | ✔ | ✔ | ✔ |
| Right to work in Singapore without sponsorship | any | Right to work in Singapore [authorization, required] | ✔ | ✔ |  |
| Experience in top-tier AI labs or tech leaders | preferred | Top-tier AI lab experience [experience, preferred] | ✔ | ✔ |  |
| Experience in high-frequency trading, fintech, Web3 or crypto | preferred | Fintech/Web3/crypto experience [experience, preferred] | ✔ | ✔ |  |
| Startups or research-intensive environments, comfortable with ambiguity | preferred | Startup/research environment experience [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- No visa sponsorship required [authorization, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | null | null | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Singapore | ✔ |

### 29 — Staff Software Development Engineer | Zscaler (glued, 'such as C, C++, Python', '#LI-Hybrid' tag, 'is a plus')

8 requirements kept, 10.1 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 5+ years in software development or sustenance engineering | required | 5+ years software development [experience, required] | ✔ | ✔ |  |
| Proficiency in programming languages such as C, C++, Python and Linux | required | Programming languages [skill, required] | ✔ | ✔ | ✔ |
| Hands-on experience with network security products (firewalls, VPNs, data protection) | required | Network security products [experience, required] | ✔ | ✔ | ✗ (no alternatives) |
| Experience developing cloud and customer reliability tools | required | Cloud reliability tools [experience, required] | ✔ | ✔ |  |
| Interest in troubleshooting customer issues; public cloud / Kubernetes knowledge is a plus | any | Public cloud or Kubernetes [skill, preferred] | ✔ | ✔ |  |
| Bachelor's degree in Computer Science or engineering | preferred | Bachelor's degree [qualification, preferred] | ✔ | ✔ |  |
| Cloud reliability engineering experience | preferred | Cloud reliability engineering experience [experience, preferred] | ✔ | ✔ |  |
| Delivering products as cloud-based services | preferred | Delivering cloud-based services [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | either | null | ✔ |
| engagement | null | employment | ✗ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Burnaby, Canada | ✔ |

**Totals for this run:** expected requirements found 42/52; level correct 40/42; groups with alternatives 4/6; forbidden items received 1; conditions correct 32/36.


## Run `q3b-regression/new-run3.json` — extractor `vacancyai.extract/2+d9df4c7197`, model `qwen3.5:9b`

### 84 — Tech Lead | Parity (glued paragraph; '50/50 hybrid role' is about duties)

5 requirements kept, 8.4 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| Software engineer / tech lead background, production code and architecture | required | Software Engineer, Tech Lead, or Engineering Manager [experience, required] | ✔ | ✔ |  |
| Managing or leading engineering teams | required | Software Engineer, Tech Lead, or Engineering Manager [experience, required] | ✔ | ✔ |  |
| Founder / early-stage startup experience | preferred | Founder, co-founder, or early-stage startup lead [experience, preferred] | ✔ | ✔ |  |
| Interest or background in Web3, blockchain, distributed or low-level systems (Rust, Substrate, Go, C++ are examples) | preferred | Web3, blockchain infrastructure, distributed systems, or low-level systems programming [skill, preferred] | ✔ | ✔ | ✔ |
| Sprint management, dependency tracking, cross-team communication | required | — | ✗ | | |
| Asynchronous communication, European or aligned time zones | required | Excellent asynchronous communication skills [skill, required] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- Experience managing or leading engineering teams [experience, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | remote | null | ✗ |
| engagement | contract | contract | ✔ |
| hours | null | null | ✔ |
| timezone | either | European or closely aligned time zones | ✔ |
| compensation | null | null | ✔ |
| location | either | null | ✔ |

### 2616 — Software Engineer - Applied AI | Deepgram (stack sentence, in-person, '9-to-5' negated)

12 requirements kept, 13.3 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| Backend services shipped in production in the cloud | required | Cloud backend experience [skill, required] | ✔ | ✔ |  |
| AI-native way of working with AI coding tools | required | AI-native operations [skill, required] | ✔ | ✔ |  |
| Learns quickly, unfamiliar parts of the stack | required | Fast learning [skill, required] | ✔ | ✔ |  |
| Finds problems and proposes solutions | required | — | ✗ | | |
| Took something from idea to production | required | Idea to production experience [experience, required] | ✔ | ✔ |  |
| Clear communication, works with customers | required | Customer communication [skill, required] | ✔ | ✔ |  |
| Comfortable working across languages (stack: Kotlin, Python, TypeScript) | required | Cross-language comfort [language, required] | ✔ | ✔ |  |
| Model training / fine-tuning experience | preferred | Model training experience [experience, preferred] | ✔ | ✔ |  |
| Evaluation or scoring for models or agents | preferred | Model evaluation experience [experience, preferred] | ✔ | ✔ |  |
| Audio, speech-to-text or text-to-speech | preferred | Audio/Speech skills [skill, preferred] | ✔ | ✔ |  |
| Founder or early startup, zero to one | preferred | Founder/Startup experience [experience, preferred] | ✔ | ✔ |  |
| 2+ years of engineering experience | preferred | Engineering experience [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- Backend language [skill, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | office | office | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | San Francisco | ✔ |

### 2520 — Backend Product Software Engineer | Dropbox (list page: other listings and pay ranges follow the text)

8 requirements kept, 10.0 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 5+ years of software engineering experience | required | 5+ years software engineering [experience, required] | ✔ | ✔ |  |
| BS in computer science or equivalent experience | required | BS degree CS or related [qualification, required] | ✔ | ✔ |  |
| Track record delivering software solutions through milestones | required | — | ✗ | | |
| Shipped large-scale web applications | required | shipped large-scale web apps [skill, required] | ✔ | ✔ |  |
| Building user-facing software for web / desktop | required | build user-facing software [skill, required] | ✔ | ✔ |  |
| Open-source contributions | preferred | open source contributions [skill, preferred] | ✔ | ✔ |  |
| Full-stack development | preferred | Full-Stack Development [skill, preferred] | ✔ | ✔ |  |
| Technologies: MySQL, Python, Go, Java, Rust | preferred | MySQL Python Go Java Rust [skill, preferred] | ✔ | ✔ | ✔ |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- AI fluency [other, preferred]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | null | null | ✔ |
| engagement | null | employment | ✗ |
| hours | either | full-time | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Anywhere in the World | ✔ |

### 834 — Software Engineer | Chalk ('This is not a hybrid role', in the office 5 days)

5 requirements kept, 6.3 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 4+ years of professional backend engineering | required | 4+ years backend experience [experience, required] | ✔ | ✔ |  |
| Strong computer science fundamentals | required | Computer science fundamentals [skill, required] | ✔ | ✔ |  |
| Comfort leading development of new components, ownership | required | — | ✗ | | |
| Learns new concepts quickly | required | — | ✗ | | |
| Works autonomously, pragmatic tradeoffs | required | — | ✗ | | |
| Strong communication skills | required | — | ✗ | | |
| Bachelor's degree in Computer Science or equivalent | required | Bachelor's degree in CS [qualification, required] | ✔ | ✔ |  |
| Data infrastructure experience | preferred | — | ✗ | | |
| Distributed systems / compilers / query planners | preferred | — | ✗ | | |
| Rust experience | preferred | C++ [skill, required] | ✔ | ✗ required |  |
| PaaS-like software | preferred | — | ✗ | | |

Forbidden items received: 1
- ✗ C++ [skill, required] — 'Write C++, Python and Rust' is a task, not a requirement

Other requirements received (not in the expected list; judge by hand): 1
- Backend language [skill, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | office | null | ✗ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | office | ✔ |

### 32 — Principal Engineer | OKX (glued, 'PhD or Master's', right-to-work preference, 'e.g.' in the role description)

8 requirements kept, 9.8 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| PhD or Master's in CS, AI, ML, Robotics or a related field | required | PhD or Master's in CS/AI/ML/Robotics [qualification, required] | ✔ | ✔ |  |
| 10+ years of industry experience | required | 10+ years industry experience [experience, required] | ✔ | ✔ |  |
| Deep expertise in long-term memory, knowledge graphs or context management | required | Expertise in memory/graphs/context [skill, required] | ✔ | ✔ | ✔ |
| Right to work in Singapore without sponsorship | any | Right to work in Singapore [authorization, required] | ✔ | ✔ |  |
| Experience in top-tier AI labs or tech leaders | preferred | Top-tier AI lab experience [experience, preferred] | ✔ | ✔ |  |
| Experience in high-frequency trading, fintech, Web3 or crypto | preferred | HFT/fintech/Web3/crypto experience [experience, preferred] | ✔ | ✔ |  |
| Startups or research-intensive environments, comfortable with ambiguity | preferred | Startup/research environment experience [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 1
- No visa sponsorship required [authorization, required]

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | null | null | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Singapore | ✔ |

### 29 — Staff Software Development Engineer | Zscaler (glued, 'such as C, C++, Python', '#LI-Hybrid' tag, 'is a plus')

8 requirements kept, 10.1 s, 1 attempt(s).

| Expected | Level | Received | Found | Level ok | Group ok |
|---|---|---|---|---|---|
| 5+ years in software development or sustenance engineering | required | 5+ years software development [experience, required] | ✔ | ✔ |  |
| Proficiency in programming languages such as C, C++, Python and Linux | required | Programming languages [skill, required] | ✔ | ✔ | ✔ |
| Hands-on experience with network security products (firewalls, VPNs, data protection) | required | Network security products [skill, required] | ✔ | ✔ | ✔ |
| Experience developing cloud and customer reliability tools | required | Cloud reliability tools experience [experience, required] | ✔ | ✔ |  |
| Interest in troubleshooting customer issues; public cloud / Kubernetes knowledge is a plus | any | Public cloud or Kubernetes knowledge [skill, preferred] | ✔ | ✔ |  |
| Bachelor's degree in Computer Science or engineering | preferred | Bachelor's degree in CS or Engineering [qualification, preferred] | ✔ | ✔ |  |
| Cloud reliability engineering experience | preferred | Cloud reliability engineering experience [experience, preferred] | ✔ | ✔ |  |
| Delivering products as cloud-based services | preferred | Success delivering cloud-based products [experience, preferred] | ✔ | ✔ |  |

Forbidden items received: 0

Other requirements received (not in the expected list; judge by hand): 0

| Condition | Expected | Received | OK |
|---|---|---|---|
| workMode | either | null | ✔ |
| engagement | null | null | ✔ |
| hours | null | null | ✔ |
| timezone | null | null | ✔ |
| compensation | null | null | ✔ |
| location | either | Burnaby, Canada | ✔ |

**Totals for this run:** expected requirements found 42/52; level correct 41/42; groups with alternatives 5/6; forbidden items received 1; conditions correct 33/36.

