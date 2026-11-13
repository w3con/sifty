# Q.3 live run of both model passes

Run: 2026-11-11T01:43:06+01:00. Provider from configuration: stage extraction = `qwen3.5-9b`, type `ollama`, model `qwen3.5:9b`, endpoint `http://127.0.0.1:11434`, context 32000 chars, timeout 120000 ms, thinking off, temperature 0.1, at most 3 attempts.

Data: a copy of the staging store in `.local/assessment-3/data` (profile revision 20, 15 profile facts offered to the matcher). Extractor version `vacancyai.extract/1+f256d6ccc8`; matcher version `vacancyai.match/1+843b19003b`.

## Job 84 — Tech Lead (Parity)

Description: 4552 characters, status `available structured description`.

### Pass 1 — vacancy extraction

- Time: 15.596s total, 1 attempt(s), last call 15.595s, request 8856 characters.
- Quote check: model returned 5 requirement(s); 5 kept with an exact quote; 0 dropped.
- Conditions kept: workMode="hybrid", engagement="contract", timezone="European or closely aligned time zones", hours="full-time".

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
      "text": "Software Engineer Tech Lead experience"
    },
    {
      "kind": "experience",
      "level": "required",
      "quote": "Experience managing or leading engineering teams, with a track record of supporting engineer growth, team culture, and delivery predictable execution.",
      "text": "Engineering team leadership experience"
    },
    {
      "kind": "qualification",
      "level": "preferred",
      "quote": "Prior experience as a founder, co-founder, or early-stage startup lead is highly desirable.",
      "text": "Founder or startup lead experience"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).",
      "text": "Web3 blockchain distributed systems"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Excellent asynchronous communication skills, operating comfortably in European or closely aligned time zones.",
      "text": "Asynchronous communication skills"
    }
  ]
}
```

</details>

Kept requirements (quote verified as an exact excerpt of the description):

0. **Software Engineer Tech Lead experience** — experience, required. Quote ✔=true: “Strong background as a Software Engineer, Tech Lead, or Engineering Manager with a proven ability to write production-level code and architect complex systems.”
1. **Engineering team leadership experience** — experience, required. Quote ✔=true: “Experience managing or leading engineering teams, with a track record of supporting engineer growth, team culture, and delivery predictable execution.”
2. **Founder or startup lead experience** — qualification, preferred. Quote ✔=true: “Prior experience as a founder, co-founder, or early-stage startup lead is highly desirable.”
3. **Web3 blockchain distributed systems** — skill, required. Quote ✔=true: “Strong interest or background in Web3, blockchain infrastructure, distributed systems, or low-level systems programming (e.g., Rust, Substrate, Go, or C++).”
4. **Asynchronous communication skills** — skill, required. Quote ✔=true: “Excellent asynchronous communication skills, operating comfortably in European or closely aligned time zones.”

### Pass 2 — matching against the profile

- Time: 5.361s total, 1 attempt(s), last call 5.361s, request 3026 characters.
- Profile-evidence check: 0 downgrade(s) to unclear.

<details><summary>Raw model answer</summary>

```json
{
  "verdicts": [
    {
      "evidenceId": "",
      "index": 0,
      "reason": "No specific Tech Lead experience mentioned.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 1,
      "reason": "Leadership listed but no team management details.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 2,
      "reason": "No founder or startup lead experience stated.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "P5",
      "index": 3,
      "reason": "Solidity expertise indicates Web3 blockchain knowledge.",
      "verdict": "match"
    },
    {
      "evidenceId": "",
      "index": 4,
      "reason": "No specific asynchronous communication skills listed.",
      "verdict": "unclear"
    }
  ]
}
```

</details>

Verdicts after validation:

- Software Engineer Tech Lead experience → **unclear** (—) No specific Tech Lead experience mentioned.
- Engineering team leadership experience → **unclear** (—) Leadership listed but no team management details.
- Founder or startup lead experience → **unclear** (—) No founder or startup lead experience stated.
- Web3 blockchain distributed systems → **match** (skills: Solidity (Expert)) Solidity expertise indicates Web3 blockchain knowledge.
- Asynchronous communication skills → **unclear** (—) No specific asynchronous communication skills listed.

## Job 2616 — Software Engineer - Applied AI (Deepgram)

Description: 4928 characters, status `available structured description`.

### Pass 1 — vacancy extraction

- Time: 15.593s total, 1 attempt(s), last call 15.593s, request 9255 characters.
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
      "kind": "experience",
      "level": "required",
      "quote": "Experience building and shipping backend services that run in production in the cloud.",
      "text": "Backend services experience"
    },
    {
      "kind": "skill",
      "level": "required",
      "quote": "Operates AI-native: you build with modern AI coding tools as your default way of working, not occasionally, and use them to move fast without lowering the quality bar.",
      "text": "AI-native operations"
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
      "text": "Clear communication"
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
      "kind": "experience",
      "level": "preferred",
      "quote": "Experience building evaluation or scoring for models or agents.",
      "text": "Evaluation experience"
    },
    {
      "kind": "experience",
      "level": "preferred",
      "quote": "Experience with audio, speech-to-text, or text-to-speech.",
      "text": "Audio experience"
    },
    {
      "kind": "experience",
      "level": "preferred",
      "quote": "Experience as a founder or early startup engineer, or taking a system from zero to one.",
      "text": "Founder experience"
    },
    {
      "kind": "experience",
      "level": "required",
      "quote": "Typically 2+ years of engineering experience, or equivalent demonstrated experience.",
      "text": "2+ years experience"
    }
  ]
}
```

</details>

Kept requirements (quote verified as an exact excerpt of the description):

0. **Backend services experience** — experience, required. Quote ✔=true: “Experience building and shipping backend services that run in production in the cloud.”
1. **AI-native operations** — skill, required. Quote ✔=true: “Operates AI-native: you build with modern AI coding tools as your default way of working, not occasionally, and use them to move fast without lowering the quality bar.”
2. **Fast learning ability** — skill, required. Quote ✔=true: “Learns quickly and takes on unfamiliar parts of the stack with little ramp-up.”
3. **Idea to production experience** — experience, required. Quote ✔=true: “Has taken something from idea to production, at work or on your own time.”
4. **Clear communication** — skill, required. Quote ✔=true: “Communicates clearly with teammates and is comfortable working directly with customers.”
5. **Kotlin and Python** — language, required. Quote ✔=true: “Comfortable working across languages. Our stack is mainly Kotlin and Python, with TypeScript on the front end.”
6. **Model training experience** — experience, preferred. Quote ✔=true: “Hands-on experience training, fine-tuning, or adapting models, through work, research, or substantial projects.”
7. **Evaluation experience** — experience, preferred. Quote ✔=true: “Experience building evaluation or scoring for models or agents.”
8. **Audio experience** — experience, preferred. Quote ✔=true: “Experience with audio, speech-to-text, or text-to-speech.”
9. **Founder experience** — experience, preferred. Quote ✔=true: “Experience as a founder or early startup engineer, or taking a system from zero to one.”
10. **2+ years experience** — experience, required. Quote ✔=true: “Typically 2+ years of engineering experience, or equivalent demonstrated experience.”

### Pass 2 — matching against the profile

- Time: 9.922s total, 1 attempt(s), last call 9.922s, request 3242 characters.
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
      "reason": "No facts mention AI-native operations.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 2,
      "reason": "No facts demonstrate fast learning ability.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 3,
      "reason": "No facts mention idea to production experience.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "P1",
      "index": 4,
      "reason": "Headline implies clear communication skills.",
      "verdict": "match"
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
      "reason": "No facts mention model training experience.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 7,
      "reason": "No facts mention evaluation experience.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 8,
      "reason": "No facts mention audio experience.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 9,
      "reason": "No facts mention founder experience.",
      "verdict": "unclear"
    },
    {
      "evidenceId": "",
      "index": 10,
      "reason": "No facts specify years of experience.",
      "verdict": "unclear"
    }
  ]
}
```

</details>

Verdicts after validation:

- Backend services experience → **match** (skills: Backend (Proficient)) Profile lists Backend as a proficient skill.
- AI-native operations → **unclear** (—) No facts mention AI-native operations.
- Fast learning ability → **unclear** (—) No facts demonstrate fast learning ability.
- Idea to production experience → **unclear** (—) No facts mention idea to production experience.
- Clear communication → **match** (headline: Software engineering · Technical advisory · Leadership) Headline implies clear communication skills.
- Kotlin and Python → **partial** (skills: Python (Proficient)) Profile lists Python as proficient.
- Model training experience → **unclear** (—) No facts mention model training experience.
- Evaluation experience → **unclear** (—) No facts mention evaluation experience.
- Audio experience → **unclear** (—) No facts mention audio experience.
- Founder experience → **unclear** (—) No facts mention founder experience.
- 2+ years experience → **unclear** (—) No facts specify years of experience.

## Job 2520 — Backend Product Software Engineer (Unknown)

Description: 4510 characters, status `available cleaned html`.

### Pass 1 — vacancy extraction

- Time: 6.101s total, 1 attempt(s), last call 6.101s, request 8839 characters.
- Quote check: model returned 2 requirement(s); 2 kept with an exact quote; 0 dropped.
- Conditions kept: location="Anywhere in the World", hours="full-time".

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
    }
  ]
}
```

</details>

Kept requirements (quote verified as an exact excerpt of the description):

0. **5+ years software engineering** — experience, required. Quote ✔=true: “5+ years of software engineering or related industry experience.”
1. **BS degree CS or related** — qualification, required. Quote ✔=true: “BS degree in Computer Science or related technical field involving coding(e.g., physics or mathematics), or equivalent technical experience.”

### Pass 2 — matching against the profile

- Time: 2.249s total, 1 attempt(s), last call 2.249s, request 2825 characters.
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
    }
  ]
}
```

</details>

Verdicts after validation:

- 5+ years software engineering → **unclear** (—) No experience duration stated in profile.
- BS degree CS or related → **unclear** (—) No degree information provided in profile.

