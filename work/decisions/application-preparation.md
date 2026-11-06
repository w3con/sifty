# Application preparation — decisions

Append-only record of why the application-preparation design is shaped as it is. Written by the coordinator; approved in conversation by the owner on 2026-11-03.

## 2026-11-03 — Build the target shape, decomposed by component (not TypeScript parity)

The coordinator first proposed restoring the Go chain to TypeScript parity before any redesign. The owner pushed back, and the coordinator withdrew that proposal: the generate step in the TypeScript app (the model drafts each answer) is not the target, so building it faithfully would be throwaway work. The decision is to build the target shape from the start, decomposed by component so the chain becomes visible early with nothing built to be discarded:

- Form inspection (reading the questions) is unchanged between parity and target, so it is wired first and makes the chain visible on real data (APP-1).
- Answer drafting is built directly in its target form (knowledge-base sourced), never as the TypeScript per-question model generation.

## 2026-11-03 — Where apply destination and form questions come from

Apply-destination resolution already happens at collection and is stored on the job as `apply_url` plus a classifying `apply_status` (about eleven classes, e.g. reachable employer ATS, unresolved intermediary, requires account). The board or aggregator a job was found on is stored as its `sources[]`. No new `vacancySource` enum or `applicationFormURL` field is needed; the distinction between "apply on the source" and "apply on the employer site" is already in the data.

Reading the application form's actual questions is a separate, heavier step (a full Playwright browser read) and is deliberately NOT done at collection: it would be run across every collected job when only a few are ever pre-selected. It is reserved for selection onward.

## 2026-11-03 — Trigger timing

Inspection runs the moment a job is pre-selected, not only when the Application tab is opened, so questions are ready by the time the user looks. Inspection is cheap enough per job and needs no model.

Answer drafting is lazy: it runs on demand (tab open or a per-question action), ordered by priority with the job the user is actively viewing first. It is not run eagerly for every pre-selected job, because the model budget would be spent on drafts that may never be read and the active job could be delayed. A low-priority background pre-warm of other drafts, behind the active job, is an optional later nicety.

## 2026-11-03 — Pipelines

Four pipelines, as independent as the resource allows:

1. Collection (fetch) — on demand and on the daily schedule. Fully independent.
2. Form inspection — model-free, runs early (at selection), never takes the inference lock.
3. Assessment — model, serialized behind the single shared inference lock.
4. Answer drafting — model, serialized behind the same inference lock, priority-ordered with the active job first.

Collection, assessment and preparation are already separate tables and workers today; assessment and preparation share one inference lock on purpose (only one model runs at a time on the laptop). What is newly separated is inspection as its own model-free path, split out of the combined inspect+generate operation.

## 2026-11-03 — Answers come from the knowledge base, not per-question generation

By default each inspected question is answered from the knowledge base, which holds material prepared earlier during Settings setup (owned by the Settings epic, SET-1). Next to each answer is a "think" affordance that opens a modal: the model works the question through in conversation, asks the user what it needs, and on resolution writes the outcome back to the knowledge base so later questions are answered better. The Settings workflow that first populates this store is to be designed separately with the owner.

## 2026-11-03 — Application tab states

While inspecting, the tab shows an indeterminate spinner, not a percentage bar (the question count is unknown in advance). Questions appear as soon as they are read, before any answers exist, each with its own answer state. Distinct honest states replace the single status banner: reading the form; questions found, answers pending; reachable form with no questions ("apply directly at <link>", not an error); form needs owner sign-in; job not pre-selected.
