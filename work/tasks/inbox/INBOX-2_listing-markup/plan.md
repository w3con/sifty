# INBOX-2 — Listing text keeps its markup: plan (coordinator, 2026-11-11)

## What is wrong today (measured on data-staging, 2 780 vacancies)

- Every source turns the page into plain text through one function, `collect.CleanHTML` (`backend/internal/collect/text.go`): headings, list items and numbering are lost; only a newline before some block tags survives. Feeds and JSON-LD (`collect/parse.go`) go through the same `text()`.
- Some sources bypass it and store raw HTML in `jobs.description`: 161 vacancies contain `<p>`, `<li>` or `<br>` (e.g. LinkedIn via jobspy, web3.career detail pages). The Overview shows these tags literally.
- 126 vacancies contain `&amp;` and similar entities; some JSON-LD texts arrive glued without any newline ("About UsParity is…").
- About 2 100 have newlines, which the current `ListingText` renders as paragraphs and bullets — readable but without headings or numbering.
- `snapshots` keep only the normalised record, not the page, so the original markup of existing vacancies cannot be recovered without fetching the page again.

## Outcome

The Overview › Original shows the listing with its headings, bulleted and numbered lists and paragraphs, styled by the page CSS; no tags or entities are visible. Then, optionally, By section groups the same text under its headings.

## Decisions to keep

- **Two forms of one text.** A new stored form with structure (`jobs.description_html`, a sanitised subset: h2–h4, p, ul, ol, li, strong, em, br; everything else unwrapped, no attributes, no scripts, no links unless decided otherwise) and the existing plain `jobs.description`, derived from it by one function. Plain text stays the input of search (FTS), Quick Match facts and the ASSESSMENT-3 model passes.
- **No silent re-analysis.** For vacancies that exist today, `description` stays byte-identical unless the page is fetched again; otherwise every description hash changes and the model re-reads ~2 800 vacancies (~13 h). The structured form for them is built locally from what is stored.
- **Separation.** Conversion and sanitising live in the collect layer (one place, used by every source); storage in the store; rendering in a presentation component that maps the allowed tags to the page's existing typography classes. No inline styles, no `dangerouslySetInnerHTML` of unsanitised data.
- **Starts after ASSESSMENT-3 is merged** (both touch `RoleContent.tsx` and the description path); cut the worktree from `main` then.

## Steps

- [ ] **L.1 — Inventory per source** (30–40 min). For each source in `search.yaml`, three stored samples: raw form (HTML / escaped HTML / plain with newlines / glued), entities, what the page actually offers (fetch at most 3 pages per source, politely). Table in the card. Check: coordinator reviews the table; it decides L.3.
- [ ] **L.2 — Structured form at collection** (60–90 min). One converter `collect.StructuredHTML` (sanitised subset above) plus `collect.PlainFromStructured` (the plain text, entities decoded, blocks separated by blank lines, list items as "- " / "1. " lines). All adapters, feeds and JSON-LD use them; jobspy/LinkedIn HTML goes through them too. Additive migration for `description_html`. One targeted test for the sanitiser (scripts, attributes, event handlers removed) and one for the plain derivation. Check: a fresh collection on the review copy (collection enabled for this step only, one or two sources) stores both forms.
- [ ] **L.3 — Existing vacancies, locally** (40–60 min). Without network: stored raw HTML → structured form via the converter; plain text with newlines → structured form from lines (bullets → ul, "1." lines → ol, short lines ending with ":" → h3); glued text → structured form of one paragraph, marked for refetch. `description` unchanged for all of them. Refetch only when the owner presses Refresh or opens a vacancy marked for refetch (existing detail path, existing rate limits). Check: counts per kind before/after; hashes of `description` identical for all existing rows.
- [ ] **L.4 — Original view** (30–45 min). `ListingText` renders `descriptionHtml` through a whitelist renderer (React elements, not innerHTML) with the existing listing-text typography; fallback to the current line renderer when absent. Check in the browser: Parity, one LinkedIn vacancy with HTML, one with numbered list, one glued; screenshots in the card.
- [ ] **L.4b — Sticky "Failed to fetch" banner (INBOX-3, owner agreed)** (20–30 min). Per INBOX-3 diagnosis: an error raised by the background poll clears on the next successful poll; network failure to the own server is classified in `frontend/messages.ts` as Retry with the text "Sifty server is not responding. Retrying…"; errors from the owner's actions unchanged. Check in the browser: stop and start the review server, the banner appears and disappears by itself.
- [ ] **L.4c — Catalogue row placeholder** (10–15 min, coordinator, 2026-11-11, from the owner's screenshot after the ASSESSMENT-3 release). With matching off, a row whose vacancy waits for extraction shows two dashes ("– –"): the second is a leftover You fit placeholder. Show one placeholder for Fits you only; a row without usable text keeps its single "–". Root cause found on production data: these rows have no stored two-score result at all (`fit` is null, so `youFitOff` is false and two cells are drawn); they were collected at 15:14 after the startup rescoring at 15:13 and get a result only when the worker reaches them. Fix both: (a) the row draws one cell when there is no result; (b) collection rescoring: after a collection run, compute the two-score result locally (no model) for the new vacancies at once, so Fits you appears immediately and the worker only refines it. Owner's words: «переключился в newest - и спустя 30 секунд - все еще ничего.», «обновляются только вторые цифры, но не первые». +15–20 min. Check on production data in the browser (Newest, first rows).
- [ ] **L.5 — By section (optional)** (45–75 min). Sections come from the headings of the structured form, in original order; a heading matched to a small fixed vocabulary (About, Role, Requirements, Nice to have, Benefits, Process) gets that label, others keep their own text. The toggle is enabled only when a listing has at least two headings; otherwise it stays disabled with the existing tooltip. Presentation only: same typography, each section a heading and its content, sections collapsible. Spec for the listing view updated. Check: screenshots for three vacancies.
- [ ] **L.6 — Owner review on the review copy (8094)** (20–30 min, owner present).

## Acceptance criteria

Added by the coordinator on 2026-11-11 when work-handover refused the card for lacking this section; every line restates the Outcome or a step's Check above, nothing new is decided here.

- Overview › Original shows headings, bulleted and numbered lists and paragraphs in the page typography; no HTML tags or entities are visible (checked in the browser on Parity, a LinkedIn vacancy with HTML, one with a numbered list, one glued).
- A fresh collection stores both `description` and `description_html`; the sanitiser removes scripts, attributes and event handlers (targeted test).
- For vacancies that exist before the change, `description` is byte-identical (hashes unchanged); counts per kind are reported before and after.
- Search, Quick Match facts and the ASSESSMENT-3 passes still read the plain `description`.
- Without a stored `description_html` the view falls back to the current line renderer.
- If L.5 is built: By section is enabled only for listings with at least two headings, and sections are stored with the vacancy and exposed through the API.

## Estimate

Complexity 8. Required L.1–L.4 + L.6: about 3–4.5 h of AI time. Optional L.5: +45–75 min. Total with L.5: 4–5.5 h; with L.4b (INBOX-3) +20–30 min. No paid model or service; L.1 and L.2 fetch a few pages from job boards.

## Addition (coordinator, 2026-11-11): one section splitter for the view and the extraction

The ASSESSMENT-3 research found that the extraction also needs the text divided into sections (56 of 97 texts split by headings or label lines, 16 glued). There must be one splitter: L.5 builds sections in Go from the structured form, stores them with the vacancy (not only in the frontend) and exposes them through the API; the view uses them for By section. Feeding sections to the extraction is not part of this card; it is available for ASSESSMENT-5 or a later card. Estimate of L.5 unchanged except storing sections (+15–20 min).

## Approval

2026-11-11, owner: «с ней, утверждаю.» — the plan with the optional L.5 (By section) is approved. Start after ASSESSMENT-3 is merged into main.
