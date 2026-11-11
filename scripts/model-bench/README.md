# Model benchmark for Sifty

Sifty reads each vacancy in two model passes: **extraction** (turn the vacancy text into a list of requirements and
conditions) and **matching** (compare those requirements with your profile and give a verdict for each). New models
appear every few months, and which one is best depends on your machine and your vacancies. These scripts let you
test a model on your own data and print a comparison table, so you do not have to trust anyone's benchmark.

Nothing here changes your data: every script works on a copy.

## What you need

- A **copy** of your Sifty data directory (`cp -R data data-bench`). It should contain vacancies that Sifty has already
  read and matched, so that the store holds extractions and verdicts.
- Python 3.9 or newer (no packages), `sqlite3` on the path, and a Go toolchain (the passes run from the source tree).
- The model, defined as a provider in `data-bench/config/providers.yaml`, for example:

  ```yaml
  providers:
    qwen-35b:
      type: ollama
      model: qwen3.5:35b-a3b
      endpoint: http://127.0.0.1:11434
      timeout_ms: 120000
      context_chars: 32000
  ```

## Privacy and cost

- **Extraction reads only the vacancy text.** It may run on a cloud model (add `--allow-paid`; the vacancy texts leave
  your machine, your profile does not). A cloud run of a few dozen vacancies costs a few cents (roughly 0.1 to 0.2 US cent per
  vacancy with a small hosted model, an estimate from token counts; check current prices).
- **Matching sends the profile facts closest to each requirement to the model** (a few facts per requirement, not the
  whole CV). Which provider receives them is your choice: a local model keeps them on your machine, a hosted model
  receives them. For a provider that is neither local nor marked `free: true`, add `--allow-paid`; Sifty refuses a bulk
  pass on it otherwise, so a backlog is never billed by surprise.

## 1. Extraction

```sh
scripts/model-bench/run-extraction.sh --data-dir data-bench --provider qwen-35b \
    --jobs 84,2616,2520 --out runs/qwen-35b-1.json
# a hosted model: add --allow-paid
```

Run it twice per model if you want to see how repeatable the model is, and use the same `--jobs` for every model.
Job ids are the numbers in your store (`sqlite3 data-bench/store.sqlite "select id, company, title from jobs limit 50"`).

Compare:

```sh
scripts/model-bench/report.py extraction \
    --run "Qwen3.5 9B=runs/9b-1.json,runs/9b-2.json" \
    --run "Qwen3.5 35B-A3B=runs/qwen-35b-1.json" \
    --expected expected.json --out report.md
```

`--expected` is optional. Without it you get volume, speed, repeatability and agreement between models. With it you
also get a score per model. Expectations are written **by hand from the vacancy text, before looking at any model's
output**, so nobody can tune them to a model:

```json
{ "jobs": [ {
  "id": 84, "label": "Tech Lead | Example Co",
  "requirements": [
    { "name": "Leads engineering teams", "level": "required", "match": ["lead.*team", "managing"] },
    { "name": "Any of Rust, Go, C++", "level": "preferred", "group": true, "match": ["rust"] }
  ],
  "forbidden": [ { "name": "A task, not a requirement", "match": ["write c\\+\\+"] } ],
  "conditions": { "workMode": "remote", "engagement": "contract", "hours": "null", "location": "either" }
} ] }
```

`match` holds regular expressions tried against the requirement's text, alternatives and quote. `level` is
`required`, `preferred` or `any`. `group: true` means the vacancy lists alternatives ("any of ...") and the model should
return them as one requirement. A condition is the expected value as lower-case text, `"null"` when the vacancy says
nothing, or `"either"` when it should not be scored. Six to ten vacancies chosen for what they test (negations, glued
paragraphs, several languages in one sentence) tell you more than fifty random ones.

## 2. Matching

```sh
cp -R data-bench data-bench-2          # a fresh copy for the second model
scripts/model-bench/run-matching.sh --data-dir data-bench --provider qwen-9b --out runs/match-9b.sqlite
scripts/model-bench/run-matching.sh --data-dir data-bench --provider qwen-35b --out runs/match-35b.sqlite
# a hosted model: add --allow-paid
scripts/model-bench/report.py matching \
    --run "Qwen3.5 9B=runs/match-9b.sqlite" --run "Qwen3.5 35B-A3B=runs/match-35b.sqlite" --out match-report.md
```

The script re-matches every vacancy that already has verdicts, on a temporary server, and waits for it to finish
(minutes for dozens of vacancies on a recent laptop). For repeatability, run the same model twice from the same starting
copy and pass both stores to one label: `--run "35B=runs/a.sqlite,runs/b.sqlite"`.

If one store holds the verdicts of several models, select a model with `store.sqlite:VERSION`, where `VERSION` is a
part of the matcher version recorded with the verdicts (`select distinct matcher_version from match_requirement_matches`).

## How to read the matching report

There is no answer key for matching: only you know your profile. The report therefore shows signals, not a grade.

- **Verdicts.** A matcher that says `unclear` to most requirements gives you nothing; one that never says `no` is
  guessing. A requirement for a skill your profile lacks should be `no`.
- **"You fit".** Check the vacancies you know. A role far from your profile should score low, a close one high.
- **Repeatability.** The same model should give the same verdict when asked twice. Low numbers mean the score on any
  single vacancy is noise.
- **Agreement and disagreements.** Where two models differ, the report prints a random sample. Decide each one from your
  own profile; the model that is right more often on that sample is the better matcher for you.
- **Time.** Seconds per vacancy decide whether a backlog of thousands of vacancies is practical.

## Limits

- Results hold for your profile, your vacancies and Sifty's prompts at the time of the run; repeat after major prompt
  changes. The extractor and matcher versions are recorded in the files.
- Models are not deterministic. Two runs of the same model differ a little; treat gaps of a few percent as noise.
- The agreement table compares wording with a simple word-overlap test, so it understates agreement between models that
  phrase a requirement differently. Use it to spot large gaps, not small ones.
