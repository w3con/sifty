# Coordinator handoff — Sifty (2026-11-13)

Written by the outgoing coordinator session for the next coordinator. Read this, then follow AGENTS.md. Answer the owner in Russian.


## 0. If you run in Claude Code on the owner's Mac (update 2026-11-13)

The owner will continue in Claude Code on the Mac, so the next coordinator can orchestrate executors itself:
- Launch executors as subagents (AGENTS.md: "through the live session's visible subagent tool"), one bounded stage per subagent, with the card, worktree, step IDs, checks and stop point in the brief (the cards' plan.md / brief files are written for that). The owner chose **Sonnet 5.5** for executors (ASSESSMENT-3: «Для исправления запускай Sonnet 5.5»); AGENTS.md line 7 still names OpenCode/DeepSeek — ask the owner once which applies and record it.
- Two executor sessions started by the owner may still be running (INBOX-2 lead, ASSESSMENT-9). Ask the owner whether to let them finish or take over; never have two writers in one worktree (AGENTS.md: one coordinator per card, record ownership in attention.md).
- Section 2's VM-specific rules (no Go, no localhost, "prunable" worktrees, lock files without delete permission) do **not** apply on the Mac; the rest of section 2 does (production via the release tool, copies of data only, never start `bin/sifty` after a worktree build, leave `legacy/app/shoot-L57.mjs`).
- You can now check things yourself: run `go test`, start review servers on free ports, curl 8080/8094, take Playwright screenshots — still read-only towards production data.

## 1. Your role and the people

- **Owner:** Alexander. He decides scope, approves plans and every production release. His words go verbatim into cards.
- **You (coordinator):** plan cards, write plans/briefs/specs, accept or return executor work, keep `work/ROADMAP.md`, `docs/MAP.md` and `docs/specs/` current. You do not implement (AGENTS.md).
- **Executors:** Claude Code sessions on the owner's Mac. You cannot message them; the owner relays. Give the owner one ready-to-paste line per instruction. Two run in parallel:
  - **Executor 1 / lead session** — INBOX-2 (it runs its own sub-executor and does a first review; its reports say "готово к приёмке").
  - **Executor 2** — small backend cards in sequence (did SETTINGS-7, OPS-10, SETTINGS-8, ASSESSMENT-8; now ASSESSMENT-9).
- Acceptance is yours; release needs the owner's explicit "да".

## 2. Environment rules (learned the hard way)

- Repo: `/Volumes/SSD/Dev/tknff/sifty` on the owner's Mac. From the coordinator's bridge VM it is `$HOME/mnt/sifty` (device_bash). The VM has no Go and cannot reach Ollama or localhost:8080/8094 — reproduce or run nothing there; read code, data (`python3` + `sqlite3` module, read-only `file:...?mode=ro`) and git.
- **Never create, prune or remove worktrees from the VM** (paths differ; `git worktree list` shows them "prunable" — that is a VM artefact, do NOT prune). Worktrees are under `.worktrees/<ID>`; read a branch with `git --git-dir=$HOME/mnt/sifty/.git/worktrees/<ID> --work-tree=.` inside the worktree.
- **Git locks:** committing from the VM needs delete permission on the sifty folder (`device_request_delete_permission` for `/Volumes/SSD/Dev/tknff/sifty`), otherwise git leaves `.git/HEAD.lock` / `next-index-*.lock` that block the owner's git. After a failed unlink, remove the stale lock (or `mv` it aside without permission).
- `bin/work-finish --reject` and anything resolving worktree absolute paths must run on the Mac (owner's Terminal or an executor).
- Untracked file `legacy/app/shoot-L57.mjs` (INBOX-2's) sits in main; executors move it aside temporarily for `work-finish` cleanliness and put it back. Leave it.
- Production: port 8080, started by the release tool (`bin/sift-promote` via `work-finish`). If the owner restarts by hand use `make run` from main (rebuilds from main). Never start `bin/sifty` directly after a worktree build. Staging preview: 8081/8082. Review copies: 8094 (INBOX-2), other free ports for executors.
- Data: production `data/store.sqlite`, staging `data-staging/store.sqlite`. Copies only for executors.
- Commit trailer for your commits: `Co-Authored-By: Claude Opus 5.5 <noreply@anthropic.com>` and the session line.

## 3. Where things are

- Process: `AGENTS.md`, `work/WORKFLOW.md`, card tools in `bin/work-*` (`work-capture` for new bugs/ideas with the owner's exact words, `work-status`, `work-list`, `work-handover` needs `- [ ]` steps in plan.md or description.md, `work-finish` = squash to main + release).
- Order and status: `work/ROADMAP.md` (top "Current planning work" block is the live sequence), `docs/MAP.md` (card index; run `bin/map-check` after edits; requirement tags must exist, e.g. R-ASM, R-SET, R-COL, R-REL, R-HARNESS, R-SEARCH).
- Specs: `docs/specs/quick-match.md` (§14 two scores, §14.14 matching off / You fit hidden, §15 agreed pipeline: profile = fact store, vacancy by sections), `docs/specs/profiles.md` (multi-profile, agreed), `docs/specs/vacancy-view.md` (approved vacancy design, money/location rules).
- Approved vacancy design: canvas "Sifty — vacancy view" https://claude.ai/artifact/NV9n28fNGjTVruEqPsoRkp (private; executors use `work/tasks/inbox/INBOX-2_listing-markup/design/vacancy-view-design.png` and `.html` in the INBOX-2 worktree). Design system: https://claude.ai/artifact/7E2LqDinz5ynWV99YbXynB.
- Acceptance records: in each card — `log.md` (append "Coordinator review/acceptance" entries with date), `plan.md` (tick/annotate steps, owner decisions with verbatim quotes), `attention.md` (open findings, known limits), executor reports `report-*.md` / `acceptance.md`.

## 4. Live state (2026-11-13)

| Card | State | Next |
|---|---|---|
| **INBOX-2** listing markup + vacancy view (worktree `.worktrees/INBOX-2`, branch `work/INBOX-2`, review copy 8094) | L.1–L.4d accepted; L.5 returned; L.5r0 fill rates done; L.5r1–r3 built (Fits you band, What they ask/offer, full listing open), partly accepted | Executor 1 must finish: (1) "Confirmed by you" — ✓/✕ on any requirement records a fact, same text elsewhere marked; (2) ✓ rule: only known technologies / short names become skills, lists split, phrases → Confirmed by you (plan.md sections dated 2026-11-12/13); (3) one money-conversion function for tile and Fits you; (4) verify the executor's real model (3 commits signed "Sonnet 4.6"). Then owner review on 8094 (L.6), owner fills exchange rates in `config/vacancy-view.yaml` (EUR per unit; placeholders now), release with the Remote3 cleanup (40 category rows, keep id 7; owner approved). Branch migrations 24–27 (main at 22). |
| **ASSESSMENT-9** model-read regional restriction (worktree exists) | handed to executor 2 | Accept M.1–M.2: vacancy 1136 must become "not covered"; then release with owner's yes. |
| **SETTINGS-5** profiles | plan approved (plan.md §10 + owner additions), spec `profiles.md` | Start right after INBOX-2 is released (schema conflict otherwise). Includes search moving into the profile, "Copy to search", headline dropped. |
| **COLLECTION-2** Remotive category pages (73) + WWR foreign salary blocks | planned R.1–R.4 | After ASSESSMENT-9 for executor 2; removal needs owner's yes at release. |
| **SETTINGS-6** edit source URL | proposed | small, after INBOX-2 |
| **OPS-9** work-tooling defects (9 items) | scope agreed | stale pid record, worktree builds writing prod artefacts, readiness before approval, work-note exit codes, ASM-6 closing path, 4 failing tests |
| **ASSESSMENT-5** matching quality (gold set) / **ASSESSMENT-7** Laya trial | planned; ASSESSMENT-7 runnable by a separate agent anytime | needs owner yes for Haiku labelling and small model downloads |
| **COL-2** per-source parsing rules + repair | direction agreed (spec §15.3) | after ASSESSMENT-5 |
| **ASSESSMENT-6** shortlist re-check, **ASSESSMENT-4** Dismiss, **SETTINGS-9** geography editor bug (low), **COLLECTION-3** sites behind login (discuss) | backlog | — |
| ASM-6 | review | closes via OPS-9 (released outside work-finish) |

Owner to-dos: accept facts from the CV re-analysis on 8080 (French B2, skills; reject English with empty level); enter work authorizations by hand (New Zealand citizen, France resident — the model did not propose them on production); remove "Yes" from eligibility and the duplicate Holvi.finance experience entry.

Known limits: model is qwen3.5:9b (local); You fit hidden (matching off); CV extraction varies run to run; LinkedIn import stores a sign-in page (load profile as PDF instead).

## 5. Done in the last day (2026-11-12 → 13), all in main unless noted

- **SETTINGS-7** released (`320dbf9`): profile files can be deleted; accepted facts stay.
- **OPS-10** released (`f7190bf`): production hang fixed — SQLite driver 1.62.0 + libc 1.81.0 (mutex leak), worker status from cache, one writer + read pool, `/api/health` 503 in 2 s.
- **SETTINGS-8** released (`b2cf419`): CV analysis keeps skills/languages, proposes authorization and location, no invented limits, same value is "matches your profile", experience matched, headline dropped, long sources split.
- **ASSESSMENT-8** released (`eddc94a`): regional remote restrictions vs work authorizations; region vocabulary in config; residence alone is not authorization. Follow-up ASSESSMENT-9.
- Cards created/planned: OPS-9, OPS-10, SETTINGS-5 (re-planned, approved), SETTINGS-6/7/8/9, ASSESSMENT-4…9, COLLECTION-2, COLLECTION-3, INBOX-3 (folded into INBOX-2). Cancelled/closed: ASM-5, COL-5 (duplicate), ASSESSMENT-2 (rejected).
- Specs written: `quick-match.md` §15 (pipeline), `profiles.md`, `vacancy-view.md` (approved design, money conversion with configured rates: ✓ lower bound ≥ minimum, ? minimum inside range, ✕ upper bound below).
- INBOX-2 (branch only): structured description form, original page stored, collection fixes (Web3 Careers address → web3.career, Remote3 via RSS, WWR via RSS), poll without refetching the 35 MB jobs list, server-down banner, sections and classes, new vacancy view per design.

## 6. How to continue

1. Read AGENTS.md, this file, `work/ROADMAP.md` top block, the INBOX-2 plan.md (end sections) and log.md.
2. Wait for the owner to paste the next executor report; review against the card plan and spec (read diffs read-only from the worktree), record your verdict in the card's log.md, give the owner one line for the executor.
3. Keep cards and specs updated in the same turn as decisions; commit from the VM with delete permission granted.
