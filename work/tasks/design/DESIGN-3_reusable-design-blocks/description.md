# DESIGN-3 — Reusable design blocks from the Profile mockup

**Epic:** design
**Created:** 2026-11-09
**Status:** review
**Kind:** idea

**Integration status,2026-11-10:** owner confirmed the design is finished and authorized integrating the DESIGN-3 worktree content into main. Source published to main in commit `5bedae3`, final application source `58af432`. Technical acceptance stands; card remains review, not done; production is not updated. Functional owner acceptance/release is separate.

## Owner's words

Captured verbatim from the owner. This is the canonical record; do not paraphrase it.

```text
нет, пусть продолжает. потом сделаем миникарточку (или запусти ее паралельно другим агентом), который создаст базу "дизайн блоков", чтобы можно было их re-use.
```

## Preparation

This card records an incoming bug or idea. It does not authorize implementation, a worktree, a plan or a deploy. Add the plan and acceptance criteria when the coordinator prepares the work.

## Observable outcome and proposed adoption

The owner requests a small reusable visual-block base after current Profile fields; parallel work was offered as an alternative. Coordinator chooses sequential preparation to avoid competing Profile/CSS writers. Visual source: frontend/Claude outputs/profile.png. Propose only blocks Profile actually needs (section cards, field/stat rows, tags, option groups), reuse them in real Profile sections in this same future task and demonstrate one style change affecting multiple consumers. A disconnected gallery or unused library is not adoption. Preserve current field semantics; the image's personal examples are not application data.

DESIGN-1 centralizes tokens and DESIGN-2 separates presentation/effects while preserving appearance; neither owns this visual-block outcome. Reuse their resources where applicable. Exact blocks/layout scope and complete step/cost breakdown must be prepared before implementation; full application redesign is not inferred.

## Provisional estimate

- **Complexity:**5 — extract a bounded visual vocabulary and apply it to current Profile without changing editor/data behavior; exact block/layout scope remains uncertain.
- **AI preparation:**10–15min; **AI implementation:**35–75min including real reuse/builds/corrections; **AI acceptance:**15–30min including real browser comparison.
- **Total AI elapsed:**60–120min, excluding owner wait/external tool delays. Assumes a small Profile-used set, no new framework or standalone gallery system. Existing subscription, attributable cost unknown; no paid service assumed. Provisional backlog estimate, not execution approval.

## Refined proposal —2026-11-09

### Current Profile discussion —2026-11-09

Owner asks accepted cards to disappear from the right proposals column; top Profile becomes Profiles; add/switch profiles through a global dropdown, removing local profile tabs. Owner then says «Уже сделано в design-3.» — inspect/reuse the other agent's active changes, do not implement them twice or take that worktree while its writer is live.

Owner's scope answer: «первое. Потом (совсем потом) добавим еще logout. пока не заморачиваемся.» Profiles are alternative role positioning for ONE person, not different people/accounts. Logout/auth is deferred. Coordinator recommendation for discussion, not yet an approved backend build: common identity/factual base and original document library; each role profile selects documents/skills/experience and owns goals/work/pay/preferences. Active profile is app-wide; scores keyed by job+profile+revision, catalogue shows active-profile score, optional comparison in detail. Application history remains common with the used profile attached; changing profile cannot erase a prior submission. Confirm shared-versus-profile-specific behavior and complete bounded plan/cost before implementing multi-profile persistence/scoring. Current source still has one LocalProfileID/CurrentProfile; UI switcher currently disables new-profile creation and browser-only renaming is not server persistence.

### Incoming avatar ceiling correction —2026-11-09

Owner reports the existing "This photo is too large. The configured limit is2097152bytes" error should occur only above10MB. Required avatar ceiling is10MiB (10485760bytes), inclusive; keep client/server/capabilities and the actual configured effective limit consistent. Update the existing configuration/defaults path, not an independent UI-only limit; do not change document limits or add compression/cropping. Capture-only follow-up for the active Profile agent, no source/live config edits by this coordinator. Provisional added complexity1, preparation1–2min/implementation1–3min/acceptance1–2min, total3–7min excluding owner wait, existing subscription/attributable price unknown; verify allowed image above2MiB and at10MiB, refusal above ceiling on isolated data. No new paid service.

**Superseded estimate below:** owner challenged the inflated cost; coordinator withdrew115–190min and proposed25–45min for the same complete outcome. Owner approved «хорошо, делай одним прогоном». Current complexity5, preparation3–5/implementation17–30/acceptance5–10min, total25–45min. One combined D.1 stage; exact scope/results/costs in plan.md. Start17:26Z, stop18:11Z. Persistent photo remains included. History below explains the withdrawn proposal only.

Owner later explicitly requests user icon, file/photo addition and appearance against profile.png; the earlier claim that Skills/Experience did not change is withdrawn in comments.md. [Complete plan](plan.md) now proposes real Profile reuse, complete composition, discoverable existing file upload and persistent avatar. This is a coordinator-authored proposal, not already implemented or approved.

Refined **complexity8**, preparation15–25min/implementation80–125min/acceptance20–40min, **total115–190min** (previous60–120min). Added cost is real photo persistence and whole-screen composition/upload discoverability. Block-only/appearance scope excluding photo would be85–135min; full proposal includes photo. Complete step results/necessity/costs and exclusions are in plan.md; no dispatch until owner approval. Existing subscription; attributable price unknown, no paid service. One task delivers the actual Profile outcome and demonstrates immediate shared-block use; no standalone speculative library.
