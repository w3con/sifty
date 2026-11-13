# Listing view specification

- **Status:** implemented in INBOX-2 (`work/INBOX-2`); not yet released to production. The structured form, section storage, API and frontend toggle are all implemented; the OPS-10 poll-reduction step (L.4e) remains open.
- **Last updated:** 2026-11-12.

This is a persistent behavior specification for how vacancy descriptions are stored, served and displayed. Task execution steps, evidence and estimates live in the [INBOX-2 card](../../work/tasks/inbox/INBOX-2_listing-markup/plan.md).

## 1. Two stored forms, one plain derivation

Every collected vacancy carries two stored forms:

- **`jobs.description`** — plain text, derived from `description_html` by one function (`collect.PlainFromStructured`): block elements separated by blank lines, list items as `- ` / `1.` lines, entities decoded. This form is the input of full-text search, Quick Match fact extraction and all model passes. It is never changed for a vacancy that already has it, except when that vacancy is explicitly refetched (the refetch path). Hash-identical `description` is the invariant for all backfill and migration steps.
- **`jobs.description_html`** — a sanitised subset of the page's markup: `h2`–`h4`, `p`, `ul`, `ol`, `li`, `strong`, `em`, `br`; every other tag is unwrapped to its text; no attributes, no scripts, no inline styles, no links. This form is normalised on write (bold-only paragraphs → `h3`, double line breaks → paragraph break, empty paragraphs removed) and is the source for section splitting. It is added additively; existing rows get it from the stored `description` by backfill on startup.

A third table, `job_originals`, holds the raw page or feed body as received, compressed. It is never rendered and never returned by the API. It is the source for future re-parsing without a network fetch.

The separation means: `description` and all its derivatives (search, Quick Match, assessments) are unaffected by changes to the HTML rendering or section logic.

## 2. Section storage

Sections are derived from `description_html` by the splitter in `backend/internal/collect/sections.go` and stored in `job_sections(job_id, sections TEXT)`. The `sections` column is a JSON array of objects: `{heading, class, content}`.

- **heading** — the text of the heading that opens the section, or absent when the section precedes the first heading.
- **class** — one of the vocabulary classes in `config/sections.yaml` (About, Role, Responsibilities, Requirements, Nice to have, Compensation, Benefits, Location, Process, Legal/EEO) or `other` when no heading matches.
- **content** — the HTML content of the section (the same sanitised subset as `description_html`).

Section data is derived from `description_html` and is recomputed on startup for all vacancies. It is not the input of any score or model pass; it is presentation data only. Routing section classes to extraction is not part of this specification (reserved for ASSESSMENT-5).

## 3. API surface

**`GET /api/jobs`** (list endpoint) does not return `sections`. The list payload must not grow materially; the per-job section payload (~5–8 KB) would multiply across thousands of rows.

**`GET /api/jobs/{id}`** (single-job endpoint) returns two additional optional fields:

```
descriptionHtml?: string   — the sanitised HTML; absent when not stored
sections?: Section[]       — the section array; absent when none stored
```

where `Section` is:
```
{ heading?: string, class: string, content: string }
```

Both fields use `omitempty`; they never appear in the list response.

## 4. Frontend behavior

### 4.1 Listing text modes

The listing panel in Overview has two modes, toggled by two buttons ("By section" / "Original") in the panel header:

- **Original** (default): renders `descriptionHtml` through a whitelist element renderer (`renderHtmlNodes`). Falls back to the plain-text line renderer (`ListingText`) when `descriptionHtml` is absent. No `innerHTML` or `dangerouslySetInnerHTML`; the renderer maps allowed tags to React elements and drops everything else. Headings, lists and emphasis use the page's existing typography classes.
- **By section**: renders sections as collapsible `<details>`/`<summary>` blocks. The first section is open by default.

Switching tabs or opening a different vacancy resets mode to Original.

### 4.2 Section availability

"By section" is enabled only when the vacancy has at least two sections that carry a heading (i.e. `headedSectionCount >= 2`). With fewer headed sections the button is `disabled` with the tooltip "Available for listings with at least two headings".

The list endpoint does not return sections. When a vacancy is opened, `RoleContent` fetches the full single-job response in the background to obtain sections; the `localSections` state is populated from that response. An `active` flag prevents a stale callback from a previous vacancy populating the state of the current one.

### 4.3 Section labels

Each section block is labelled as follows:

- If `class` is a vocabulary class (not `other`), the label is the class name: "Requirements", "Responsibilities", etc.
- If `class` is `other`, the label is the section's own heading text, or "Preamble" when the section has no heading (the text before the first heading).

### 4.4 Fallback

When neither `descriptionHtml` nor `description` is available, the panel shows the message: "A full description has not been retrieved. Open the original listing; missing text is not an empty job."

## 5. Source normalization rules

The `collect.StructuredHTML` converter accepts four input forms and produces the same sanitised subset from each:

- **HTML** — parsed directly; tags outside the allowed set are unwrapped.
- **Entity-escaped HTML** — decoded once, then parsed as HTML.
- **Markdown** — headings, bold, lists and links converted to the allowed subset; links unwrapped to their text.
- **Plain text with lines** — bullet lines (`- `, `* `, `1.`, etc.) → `ul`/`ol`; short lines ending with `:` → `h3`; paragraphs separated by blank lines.

The normaliser (`collect.NormalizeHTML`) runs on the sanitised output:

- A paragraph whose entire content is bold text becomes an `h3`.
- A paragraph whose first run is bold and the rest is plain text has its bold run promoted to an `h3` and the rest moved to the following paragraph.
- Consecutive `<br>` elements that separate paragraph-sized content become `<p>` boundaries and the `<br>` elements are removed.
- Empty `<p>` elements are removed.

## 6. What is not stored, returned or rendered

- The raw page body (`job_originals`) is never returned by any API endpoint and never rendered to the owner.
- Section data is not the input of Quick Match, full-text search, or any model pass.
- No inline styles, event handlers or non-allowlisted attributes survive the sanitiser.
- `description` is not changed by the section or HTML logic except through the explicit refetch path.
