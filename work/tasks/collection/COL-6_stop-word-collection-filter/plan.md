---
estimate: 13
---
# COL-6 — plan

**Author:** coordinator
**Created:** 2026-11-05

## Statement of work

The triage inbox fills with whole categories of roles the owner never wants, wrong by their title alone (administrative, sales, and similar role types). The only tools today are dismissing each role one at a time and the per-listing permanent Delete; neither says "never any role whose title contains this word, and stop collecting them." This work adds a stop-word: a reversible rule matched whole-word and case-insensitive against the role title. Adding a stop-word hides every already-collected role whose title matches it from every view, and makes the collection loop skip matching postings from then on. Removing the stop-word brings the hidden roles back and resumes collecting them. Before a stop-word is committed, the owner sees how many stored roles it would hide. Management lives in Settings for now; a later navigation card relocates it into a Collection → Search section.

## Decisions settled with the owner (2026-11-05)

- Two cards: this one is the stop-word capability; the navigation restructure (single Collection tab with Sources/Search, single Archived tab with Applied/Dismissed, Pre-selected renamed Selected, relocation of block-rule and stop-word management) is a separate later card.
- The purge is reversible: a stop-word hides matching stored roles and stops collecting them; removing the stop-word reveals the hidden roles and resumes collection. Already-hidden roles are never deleted.
- A stop-word is a literal character sequence (for example `risk engineer`), matched case-insensitively as a substring of the role title. The UI shows the count of roles a stop-word would hide before it is committed.

## Coordinator's implementation decisions (settled, not for the owner)

- A stop-word is hidden by computing a `stopped` flag (and a `stoppedBy` reason) on each job at read time, the same pattern TRI-1 uses for `blocked`: the flag is derived from the active stop-words, so removal is automatically reversible and no stored role is mutated. A stopped role is excluded from every status listing (triage, preselected, applied, no), unlike a block rule which hides only from triage — a stop-word is a stronger "I never want to see this at all."
- Collection is filtered in the collection loop (`internal/background/collect.go`), before `Upsert`: a posting whose title matches an active stop-word is skipped, so junk is never fetched into the store and effort is saved. The filter is not placed inside `Upsert`, so a manually added or enriched role is never silently dropped.
- Storage is a `stop_words` table `(id, word, created_at)`, `UNIQUE(word)` on the normalised word, added idempotently with a `user_version` bump, mirroring `block_rules`.
- Substring match: the title and the stop-word are lower-cased, and the stop-word matches when it occurs anywhere in the title as a literal substring. `risk engineer` therefore hides "Senior Risk Engineer"; the match is literal, so the owner picks a sequence specific enough to avoid unwanted hits (a short fragment like `admin` would also hit "administrator").
- The preview count is a store method `StopWordMatchCount(word)` returning how many stored roles the word would hide, exposed as a read-only endpoint; it does not create the rule.

## Out of scope — do not touch

The `deleted_listings` tombstone and `DeleteRole` (the permanent, irreversible delete path; stop-words are reversible and separate). The dedup and status carry-forward in `store.Upsert`. TRI-1 block rules and the `block_rules` table. The stage model and its transitions. The assessment path. The collection fetch/source internals (this filters which postings are ingested, it does not change how pages are fetched). The navigation restructure (separate card).

## Acceptance criteria

- Adding a stop-word immediately hides every role whose title contains it (case-insensitive substring) from every view and from the per-stage counts.
- The collection loop skips a posting whose title matches an active stop-word, so matching roles are not re-ingested.
- No stopped role is deleted; removing the stop-word in Settings makes the matching roles reappear and collection resumes picking them up.
- The match is a case-insensitive literal substring of the title: a stop-word `risk engineer` hides "Senior Risk Engineer".
- Before committing a stop-word, Settings shows the number of stored roles it would hide, and that number equals the number actually hidden after it is added.
- Settings lists the active stop-words with a remove control.
- TRI-1 block rules and existing dismissal persistence are unchanged.
- `go test -race ./...` and `bin/work-test` pass.

## Checkpoints

**1 — Backend: storage and matching**

*Gate: `cd sifty && go test -race ./internal/store/...`*

- [x] 1.1 — Add the `stop_words` table (`word`, `created_at`, `UNIQUE(word)`) to the base schema and bump `user_version`; `go build ./...` is green.
- [x] 1.2 — Add store methods `AddStopWord`, `ListStopWords`, `RemoveStopWord`, and a `titleMatchesStopWords(title, words)` helper doing a case-insensitive substring match; a unit test covers a single word, a multi-word sequence (`risk engineer`), and case-insensitivity.
- [x] 1.3 — Populate `Stopped` and `StoppedBy` on the job returned by `Get`/`List`, and exclude a stopped role from every status listing; `go test -race ./...` is green.
- [x] 1.4 — Add `StopWordMatchCount(word)` returning how many stored roles the word would hide; a unit test covers it.

**2 — Backend: collection skip and API**

*Gate: `cd sifty && go test -race ./internal/...`*

- [x] 2.1 — In `internal/background/collect.go`, load the active stop-words once per run and skip a posting whose title matches before `Upsert`; a focused test or check confirms a matching posting is not stored.
- [x] 2.2 — `POST /api/stop-words {word}` (validates and inserts, returns the stored rule), `GET /api/stop-words`, `DELETE /api/stop-words/{id}`, and `GET /api/stop-words/preview?word=` (returns the count, creates nothing); handler tests cover add, list, remove, and preview, including that removal clears the `stopped` flag on affected jobs.

**3 — Frontend: management and hiding**

*Gate: `cd app && npx tsc --noEmit`*

- [x] 3.1 — Carry `stopped`/`stoppedBy` on the `Job` type and add `api` calls for add, list, remove, and preview stop-words.
- [x] 3.2 — Hide stopped roles from every view and exclude them from the per-stage counts.
- [x] 3.3 — In Settings, add a control to enter a stop-word that shows the preview count before it is committed, and list the active stop-words with a remove control that reveals the matching roles. (This panel is relocated into Collection → Search by the later navigation card.)

**4 — Verify end to end**

*Gate: `bin/work-test`*

- [x] 4.1 — `bin/work-test` passes.
- [x] 4.2 — Build the web bundle and the Go binary and confirm by hand against a copy of the live data: add a stop-word, see the preview count, watch matching roles vanish from every view, confirm collection skips them, remove the word, and watch the roles return. (Gate: manual, recorded in `log.md`.)

## Gate evidence

Дословный вывод каждой прогнанной проверки записывается сюда, без сокращений и пересказа.
