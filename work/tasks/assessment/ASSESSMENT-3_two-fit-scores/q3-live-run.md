# Q.3 live run of both model passes

Run: 2026-11-11T01:44:24+01:00. Provider from configuration: stage extraction = `qwen3.5-9b`, type `ollama`, model `qwen3.5:9b`, endpoint `http://127.0.0.1:11434`, context 32000 chars, timeout 120000 ms, thinking off, temperature 0.1, at most 3 attempts.

Data: a copy of the staging store in `.local/assessment-3/data` (profile revision 20, 15 profile facts offered to the matcher). Extractor version `vacancyai.extract/1+d104a3a836`; matcher version `vacancyai.match/1+373086bbb2`.

## Job 84 — Tech Lead (Parity)

Description: 4552 characters, status `available structured description`.

### Pass 1 — vacancy extraction

- Time: 15.968s total, 1 attempt(s), last call 15.968s, request 8992 characters.
- Quote check: model returned 9 requirement(s); 9 kept with an exact quote; 0 dropped.
- Conditions kept: engagement="contract", timezone="European or closely aligned time zones", hours="full-time", workMode="hybrid".

<details><summary>Raw model answer</summary>

```json
{
  "conditions": {
    "compensation": null,
    "engagement": {
      "quote": "Team members will be taken on as a contractor via our UK entity.",
      "value": "contract"
    },
    "hours": {
      "quote": "This is a 50/50 hybrid role",
      "value": "full-time"
    },
    "location": null,
    "timezone": {
      "quote": "operating comfortably in European or closely aligned time zones.",
      "value": "European or closely aligned time zones"
    },
    "workMode": {
      "quote": "This is a 50/50 hybrid role",
      "value": "hybrid"
    }
  },
  "requirements": [
    {
      "kind": "experience",
      "level": "required",
      "quote": "Strong background as a Software Engineer, Tech Lead, or Engineering Manager with a proven ability to write production-level code and architect complex systems.",
      "text": "Software Engineer, Tech Lead, or Engineering Manager"
    },
    {
      "kind": "experience",
      "level": "required",
      "quote": "Experience managing or leading engineering teams, with a track record of supporting engineer growth, team culture, and delivery predictable execution.",
      "text": "managing or leading engineering teams"
    },
    {
      "kind": "experience",
      "level": "preferred",
      "quote": "Prior experience as a founder, co-founder, or early-stage startup lead is highly desirable.",
      "text": "founder, co-founder, or early-stage startup lead"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).",
      "text": "Web3, blockchain infrastructure, distributed systems"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).",
      "text": "Rust"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).",
      "text": "Substrate"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).",
      "text": "Go"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).",
      "text": "C++"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Excellent asynchronous communication skills, operating comfortably in European or closely aligned time zones.",
      "text": "asynchronous communication skills"
    }
  ]
}
```

</details>

Kept requirements (quote verified as an exact excerpt of the description):

0. **Software Engineer, Tech Lead, or Engineering Manager** — experience, required. Quote ✔=true: “Strong background as a Software Engineer, Tech Lead, or Engineering Manager with a proven ability to write production-level code and architect complex systems.”
1. **managing or leading engineering teams** — experience, required. Quote ✔=true: “Experience managing or leading engineering teams, with a track record of supporting engineer growth, team culture, and delivery predictable execution.”
2. **founder, co-founder, or early-stage startup lead** — experience, preferred. Quote ✔=true: “Prior experience as a founder, co-founder, or early-stage startup lead is highly desirable.”
3. **Web3, blockchain infrastructure, distributed systems** — skill, required. Quote ✔=true: “Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).”
4. **Rust** — skill, required. Quote ✔=true: “Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).”
5. **Substrate** — skill, required. Quote ✔=true: “Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).”
6. **Go** — skill, required. Quote ✔=true: “Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).”
7. **C++** — skill, required. Quote ✔=true: “Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).”
8. **asynchronous communication skills** — skill, required. Quote ✔=true: “Excellent asynchronous communication skills, operating comfortably in European or closely aligned time zones.”

### Pass 2 — matching against the profile

- Time: 9.295s total, 1 attempt(s), last call 9.295s, request 3314 characters.
- Profile-evidence check: 0 downgrade(s) to unclear.

<details><summary>Raw model answer</summary>

```json
{
  "verdicts": [
    {
      "evidenceId": "P1",
      "index": 0,
      "reason": "Headline lists Software engineering and Leadership.",
      "verdict": "match"
    },
    {
      "evidenceId": "P1",
      "index": 1,
      "reason": "Headline explicitly states Leadership.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 2,
      "reason": "No facts mention founder or startup lead role.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "P5",
      "index": 3,
      "reason": "Solidity expertise implies Web3 blockchain knowledge.",
      "verdict": "match"
    },
    {
      "evidenceId": "P2",
      "index": 4,
      "reason": "Profile lists Rust as Proficient skill.",
      "verdict": "match"
    },
    {
      "evidenceId": "",
      "index": 5,
      "reason": "No facts mention Substrate framework.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "P9",
      "index": 6,
      "reason": "Profile lists Golang as Proficient skill.",
      "verdict": "match"
    },
    {
      "evidenceId": "",
      "index": 7,
      "reason": "No facts mention C++ programming language.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 8,
      "reason": "No facts describe asynchronous communication skills.",
      "verdict": "unclear"
    }
  ]
}
```

</details>

Verdicts after validation:

- Software Engineer, Tech Lead, or Engineering Manager → **match** (headline: Software engineering · Technical advisory · Leadership) Headline lists Software engineering and Leadership.
- managing or leading engineering teams → **unclear** (—) Headline explicitly states Leadership.
- founder, co-founder, or early-stage startup lead → **unclear** (—) No facts mention founder or startup lead role.
- Web3, blockchain infrastructure, distributed systems → **match** (skills: Solidity (Expert)) Solidity expertise implies Web3 blockchain knowledge.
- Rust → **match** (skills: Rust (Proficient)) Profile lists Rust as Proficient skill.
- Substrate → **unclear** (—) No facts mention Substrate framework.
- Go → **match** (skills: Golang (Proficient)) Profile lists Golang as Proficient skill.
- C++ → **unclear** (—) No facts mention C++ programming language.
- asynchronous communication skills → **unclear** (—) No facts describe asynchronous communication skills.

## Job 2616 — Software Engineer - Applied AI (Deepgram)

Description: 4928 characters, status `available structured description`.

### Pass 1 — vacancy extraction

- Time: 16.848s total, 1 attempt(s), last call 16.848s, request 9391 characters.
- Quote check: model returned 11 requirement(s); 11 kept with an exact quote; 2 dropped.
  - dropped: conditions.workMode: empty value or quote is not an exact excerpt of the vacancy text
  - dropped: conditions.location: empty value or quote is not an exact excerpt of the vacancy text
- Conditions kept: none.

<details><summary>Raw model answer</summary>

```json
{
  "conditions": {
    "compensation": null,
    "engagement": null,
    "hours": null,
    "location": {
      "quote": "We're a small team that works in person in San Francisco",
      "value": "San Francisco"
    },
    "timezone": null,
    "workMode": {
      "quote": "We're a small team that works in person in San Francisco",
      "value": "office"
    }
  },
  "requirements": [
    {
      "kind": "skill",
      "level": "required",
      "quote": "Experience building and shipping backend services that run in production in the cloud.",
      "text": "Backend services experience"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Operates AI-native: you build with modern AI coding tools as your default way of working, not occasionally, and use them to move fast without lowering the quality bar.",
      "text": "AI coding tools proficiency"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Learns quickly and takes on unfamiliar parts of the stack with little ramp-up.",
      "text": "Fast learning ability"
    },
    {
      "kind": "experience",
      "level": "required",
      "quote": "Has taken something from idea to production, at work or on your own time.",
      "text": "Idea to production experience"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Communicates clearly with teammates and is comfortable working directly with customers.",
      "text": "Communication skills"
    },
    {
      "kind": "language",
      "level": "required",
      "quote": "Comfortable working across languages. Our stack is mainly Kotlin and Python, with TypeScript on the front end.",
      "text": "Kotlin and Python"
    },
    {
      "kind": "experience",
      "level": "preferred",
      "quote": "Hands-on experience training, fine-tuning, or adapting models, through work, research, or substantial projects.",
      "text": "Model training experience"
    },
    {
      "kind": "skill",
      "level": "preferred",
      "quote": "Experience building evaluation or scoring for models or agents.",
      "text": "Evaluation and scoring experience"
    },
    {
      "kind": "skill",
      "level": "preferred",
      "quote": "Experience with audio, speech-to-text, or text-to-speech.",
      "text": "Audio STT TTS experience"
    },
    {
      "kind": "experience",
      "level": "preferred",
      "quote": "Experience as a founder or early startup engineer, or taking a system from zero to one.",
      "text": "Founder or startup experience"
    },
    {
      "kind": "experience",
      "level": "preferred",
      "quote": "Typically 2+ years of engineering experience, or equivalent demonstrated experience.",
      "text": "2+ years engineering experience"
    }
  ]
}
```

</details>

Kept requirements (quote verified as an exact excerpt of the description):

0. **Backend services experience** — skill, required. Quote ✔=true: “Experience building and shipping backend services that run in production in the cloud.”
1. **AI coding tools proficiency** — skill, required. Quote ✔=true: “Operates AI-native: you build with modern AI coding tools as your default way of working, not occasionally, and use them to move fast without lowering the quality bar.”
2. **Fast learning ability** — skill, required. Quote ✔=true: “Learns quickly and takes on unfamiliar parts of the stack with little ramp-up.”
3. **Idea to production experience** — experience, required. Quote ✔=true: “Has taken something from idea to production, at work or on your own time.”
4. **Communication skills** — skill, required. Quote ✔=true: “Communicates clearly with teammates and is comfortable working directly with customers.”
5. **Kotlin and Python** — language, required. Quote ✔=true: “Comfortable working across languages. Our stack is mainly Kotlin and Python, with TypeScript on the front end.”
6. **Model training experience** — experience, preferred. Quote ✔=true: “Hands-on experience training, fine-tuning, or adapting models, through work, research, or substantial projects.”
7. **Evaluation and scoring experience** — skill, preferred. Quote ✔=true: “Experience building evaluation or scoring for models or agents.”
8. **Audio STT TTS experience** — skill, preferred. Quote ✔=true: “Experience with audio, speech-to-text, or text-to-speech.”
9. **Founder or startup experience** — experience, preferred. Quote ✔=true: “Experience as a founder or early startup engineer, or taking a system from zero to one.”
10. **2+ years engineering experience** — experience, preferred. Quote ✔=true: “Typically 2+ years of engineering experience, or equivalent demonstrated experience.”

### Pass 2 — matching against the profile

- Time: 9.385s total, 1 attempt(s), last call 9.385s, request 3416 characters.
- Profile-evidence check: 0 downgrade(s) to unclear.

<details><summary>Raw model answer</summary>

```json
{
  "verdicts": [
    {
      "evidenceId": "P10",
      "index": 0,
      "reason": "Profile lists Backend as a proficient skill.",
      "verdict": "match"
    },
    {
      "evidenceId": "",
      "index": 1,
      "reason": "No mention of AI coding tools proficiency.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 2,
      "reason": "No explicit statement about fast learning ability.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 3,
      "reason": "No details on idea to production experience.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 4,
      "reason": "No explicit mention of communication skills.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "P8",
      "index": 5,
      "reason": "Profile lists Python as proficient.",
      "verdict": "partial"
    },
    {
      "evidenceId": "",
      "index": 6,
      "reason": "No mention of model training experience.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 7,
      "reason": "No mention of evaluation and scoring experience.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 8,
      "reason": "No mention of audio STT or TTS experience.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 9,
      "reason": "No mention of founder or startup experience.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 10,
      "reason": "No specific years of engineering experience stated.",
      "verdict": "unclear"
    }
  ]
}
```

</details>

Verdicts after validation:

- Backend services experience → **match** (skills: Backend (Proficient)) Profile lists Backend as a proficient skill.
- AI coding tools proficiency → **unclear** (—) No mention of AI coding tools proficiency.
- Fast learning ability → **unclear** (—) No explicit statement about fast learning ability.
- Idea to production experience → **unclear** (—) No details on idea to production experience.
- Communication skills → **unclear** (—) No explicit mention of communication skills.
- Kotlin and Python → **partial** (skills: Python (Proficient)) Profile lists Python as proficient.
- Model training experience → **unclear** (—) No mention of model training experience.
- Evaluation and scoring experience → **unclear** (—) No mention of evaluation and scoring experience.
- Audio STT TTS experience → **unclear** (—) No mention of audio STT or TTS experience.
- Founder or startup experience → **unclear** (—) No mention of founder or startup experience.
- 2+ years engineering experience → **unclear** (—) No specific years of engineering experience stated.

## Job 2520 — Backend Product Software Engineer (Unknown)

Description: 4510 characters, status `available cleaned html`.

### Pass 1 — vacancy extraction

- Time: 8.986s total, 1 attempt(s), last call 8.986s, request 8975 characters.
- Quote check: model returned 5 requirement(s); 5 kept with an exact quote; 0 dropped.
- Conditions kept: hours="full-time", location="Anywhere in the World".

<details><summary>Raw model answer</summary>

```json
{
  "conditions": {
    "compensation": null,
    "engagement": null,
    "hours": {
      "quote": "Job type Full-Time",
      "value": "full-time"
    },
    "location": {
      "quote": "Anywhere in the World",
      "value": "Anywhere in the World"
    },
    "timezone": null,
    "workMode": null
  },
  "requirements": [
    {
      "kind": "experience",
      "level": "required",
      "quote": "5+ years of software engineering or related industry experience.",
      "text": "5+ years software engineering"
    },
    {
      "kind": "qualification",
      "level": "required",
      "quote": "BS degree in Computer Science or related technical field involving coding(e.g., physics or mathematics), or equivalent technical experience.",
      "text": "BS degree CS or related"
    },
    {
      "kind": "experience",
      "level": "required",
      "quote": "Proven track record in identifying and delivering software solutions through a set of milestones spanning a specific product focus or a multi-component system",
      "text": "deliver software solutions"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Developed and shipped large-scale web applications with a user-centric mindset.",
      "text": "developed large-scale web apps"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Proficiency in building user-facing software and supporting systems and libraries for web and/or desktop platforms.",
      "text": "proficiency web desktop platforms"
    }
  ]
}
```

</details>

Kept requirements (quote verified as an exact excerpt of the description):

0. **5+ years software engineering** — experience, required. Quote ✔=true: “5+ years of software engineering or related industry experience.”
1. **BS degree CS or related** — qualification, required. Quote ✔=true: “BS degree in Computer Science or related technical field involving coding(e.g., physics or mathematics), or equivalent technical experience.”
2. **deliver software solutions** — experience, required. Quote ✔=true: “Proven track record in identifying and delivering software solutions through a set of milestones spanning a specific product focus or a multi-component system”
3. **developed large-scale web apps** — skill, required. Quote ✔=true: “Developed and shipped large-scale web applications with a user-centric mindset.”
4. **proficiency web desktop platforms** — skill, required. Quote ✔=true: “Proficiency in building user-facing software and supporting systems and libraries for web and/or desktop platforms.”

### Pass 2 — matching against the profile

- Time: 4.54s total, 1 attempt(s), last call 4.54s, request 3125 characters.
- Profile-evidence check: 0 downgrade(s) to unclear.

<details><summary>Raw model answer</summary>

```json
{
  "verdicts": [
    {
      "evidenceId": "",
      "index": 0,
      "reason": "No experience duration stated in profile.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 1,
      "reason": "No degree information provided in profile.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 2,
      "reason": "No specific project delivery details listed.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 3,
      "reason": "No mention of large-scale web apps built.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 4,
      "reason": "No mention of desktop platform proficiency.",
      "verdict": "unclear"
    }
  ]
}
```

</details>

Verdicts after validation:

- 5+ years software engineering → **unclear** (—) No experience duration stated in profile.
- BS degree CS or related → **unclear** (—) No degree information provided in profile.
- deliver software solutions → **unclear** (—) No specific project delivery details listed.
- developed large-scale web apps → **unclear** (—) No mention of large-scale web apps built.
- proficiency web desktop platforms → **unclear** (—) No mention of desktop platform proficiency.

