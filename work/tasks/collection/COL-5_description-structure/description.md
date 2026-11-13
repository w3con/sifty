# COL-5 — Preserve heading and list structure in the job description view

**Epic:** collection
**Created:** 2026-11-04
**Status:** founded (not scheduled; plan.md before execution)

## Why this work exists

The job description reads as one undifferentiated block of text: headings and lists from the original posting are flattened, so the structure that makes a long description scannable is lost. The cause is at collection: every adapter runs the captured HTML through `collect.CleanHTML` (`sifty/internal/collect/text.go`), which strips all tags to plain text (it removes `script/style/nav/…`, inserts newlines before block elements, and takes `body.Text()`). The UI then renders that plain text as a text node in `app/web/RoleContent.tsx` (`<div className="description">{job.description}</div>`), so there is no markup left to style.

The owner wants the description to keep only heading and list structure (`<h1>`–`<h4>`, `<ul>`/`<ol>`/`<li>`), nothing else, and styled with our own CSS — the source's CSS, classes and inline styles must not come through.

## The hard constraint — do not change `job.Description`

`job.Description` must stay plain text. It is consumed by:
- the assessment exact-quote validation (`sifty/internal/assess/assess.go` builds `jobQuoteText` from `job.Description` and checks quotes against it),
- the assessment prompt and candidate context (`internal/background/assessment.go`, `assess.go`),
- the catalogue skill/keyword filters in the UI (`app/web/CatalogueFilters.tsx`, which match on description text).

Turning `job.Description` into HTML would break all three. So the structured version must live in a separate field; `job.Description` stays exactly as it is today.

## What this card should deliver

- A new sanitiser (e.g. `collect.StructuredHTML`) that, from the same source HTML `CleanHTML` receives, produces a minimal HTML string keeping only `<h1>`–`<h4>`, `<ul>`, `<ol>`, `<li>` (and their text content), with every other tag unwrapped to its text, every attribute removed (no `class`, `style`, `id`, no `<script>`/`<style>`), and the source's CSS gone. Whether paragraph breaks are kept as plain text/newlines or as `<p>` is a plan.md decision, but headings and lists are the only rich structure kept.
- A new persisted field on the job (e.g. `DescriptionHTML` / `description_html`) with a store migration to add the column, populated by the adapters that have real HTML (`greenhouse.go` `j.Content`, `ashby.go` `descriptionHtml`, `lever`, `getro`, and the generic HTML path); left empty where only plain text is available.
- `app/web/RoleContent.tsx` renders the sanitised HTML (via `dangerouslySetInnerHTML`) when `job.descriptionHtml` is present, falling back to the current plain-text rendering when it is not.
- Our own CSS in `app/web/style.css` styling `.description h1/h2/h3/h4`, `.description ul/ol/li` so headings and lists read clearly.

## Must not touch

- `job.Description` (plain text) and everything that consumes it: assessment exact-quote validation, the assessment prompt/candidate context, and the catalogue keyword/skill filters.
- The COL-1 fetch/source internals (SSRF guard, cache, traffic discipline, extraction waterfall order) and COL-3 triggers — this adds a second rendering of the same captured HTML, it does not change how pages are fetched or which tags the waterfall extracts.
- The public-subtree rules for `sifty/`.

## Notes

- Keep only `<h1>`–`<h4>`, `<ul>`, `<ol>`, `<li>`. No other tags, no attributes, no source CSS/classes/inline styles — the owner was explicit: our CSS, not theirs.
- The sanitiser should reuse the same `goquery` parse `CleanHTML` already uses; the two can share the initial strip of `script/style/nav/footer/header/noscript/svg`.
- `dangerouslySetInnerHTML` is safe here only because the sanitiser emits a fixed tag whitelist with no attributes; the plan must ensure the sanitiser is the single source of the HTML and no raw source HTML ever reaches the DOM.

## Steps

- [ ] To be defined in plan.md.

## Acceptance criteria

- To be defined in plan.md.
