# INBOX-2 L.1 — Inventory of listing text per source (executor, 2026-11-11)

Roles: the coordinator reviews this table and decides the scope of L.3; the executor produced it. No application code, no database and no server were touched.

## 1. Method and where the numbers come from

Stored samples and counts come from the staging database `data-staging/store.sqlite` of the base checkout, opened read-only (`mode=ro`) by `inventory_counts.py` in this folder; its verbatim output is `inventory_counts_output.txt`. The script counts the whole table, not only the samples.

The jobs table has no source column. The source of a vacancy here is the earliest row in `sightings` for it; 65 vacancies were seen by more than one source, so a vacancy is counted once, under the source that saw it first. `search.yaml` enables 15 sites; 13 of them have rows in `sightings`, and the database also has a source named Remotive that is not in `search.yaml`, so Table A has 14 sources. Welcome to the Jungle and Malt (enabled in `search.yaml`) have no rows at all.

Kinds are mutually exclusive and tested in this order: empty; html (a real tag such as p, li, ul, ol, br, div, h1-h6, strong, b, em, i, span, a is present in the stored text); escaped (contains `&lt;` or a tag name directly followed by `&gt;`, for example `strong&gt;`); metadata (no newline, shorter than 300 characters: skill and tag lists, error placeholders, one-paragraph short texts); glued (no newline, 300 characters or more); newlines (plain text with at least one newline).

The pages were fetched with `inventory_fetch.sh`: 42 requests in total, at most 3 per source, one every 4 seconds, with the project's own client identity `Sifty/1.0 (personal read-only job discovery)` (taken from `backend/internal/fetch/client.go`), no retries, no login. The list is `inventory_fetch_list.tsv`, the HTTP result of every request is `inventory_fetch_log.tsv`, and the summary of what each page delivers is `inventory_pages_output.txt` (produced by `inventory_pages.py`, which reads the saved page bodies; the bodies themselves are not kept in the repository). Page results are a snapshot of 2026-11-11 and are not reproducible by a script run.

## 2. Totals against the plan

| Measure | Plan | Counted now | Note |
|---|---|---|---|
| Vacancies | 2 780 | 2 783 | three more rows arrived in staging after the plan was written |
| Contain `<p>`, `<li>` or `<br>` | 161 | 161 | identical; all 161 also match the wider tag test |
| Contain `&amp;` | 126 | 126 | identical |
| Contain any entity | not stated | 168 | `&gt;` 74 rows, `&lt;` 60, `&amp;` 126, `&nbsp;` 12, `&ndash;` 1, `&L;` 1 (the last is page text, not an entity) |
| Contain a newline | about 2 100 | 2 107 | identical within rounding |
| Escaped or mangled markup (`strong&gt;`) | not stated | 13 | all Web3 Careers |
| No newline and 300 characters or more | not stated | 57 | 46 Web3 Careers, 9 A16z (skill lists, not descriptions), 1 Wellfound, 1 Multicoin (skill list) |
| Empty description | not stated | 64 | 58 Paradigm |

## 3. Table A: kind of stored text per source (counts and share of the source)

| Source | Vacancies | html | escaped | newlines | glued | metadata | empty |
|---|---|---|---|---|---|---|---|
| Paradigm | 836 | 80 (10%) | 0 | 697 (83%) | 0 | 1 (0%) | 58 (7%) |
| Wellfound | 400 | 0 | 0 | 399 (100%) | 1 (0%) | 0 | 0 |
| We Work Remotely | 265 | 27 (10%) | 0 | 237 (89%) | 0 | 0 | 1 (0%) |
| Remotive | 226 | 0 | 0 | 226 (100%) | 0 | 0 | 0 |
| Multicoin capital | 192 | 20 (10%) | 0 | 47 (24%) | 1 (1%) | 122 (64%) | 2 (1%) |
| A16z | 172 | 0 | 0 | 37 (22%) | 9 (5%) | 124 (72%) | 2 (1%) |
| Hacker News — Who is hiring? | 167 | 0 | 0 | 167 (100%) | 0 | 0 | 0 |
| Electric Capital | 141 | 5 (4%) | 0 | 79 (56%) | 0 | 57 (40%) | 0 |
| Web3 Careers | 132 | 2 (2%) | 13 (10%) | 6 (5%) | 46 (35%) | 65 (49%) | 0 |
| Framework ventures | 89 | 27 (30%) | 0 | 54 (61%) | 0 | 7 (8%) | 1 (1%) |
| Cryptocurrency Jobs | 65 | 0 | 0 | 61 (94%) | 0 | 4 (6%) | 0 |
| Remote3 | 41 | 0 | 0 | 40 (98%) | 0 | 1 (2%) | 0 |
| CryptoJobList | 29 | 0 | 0 | 29 (100%) | 0 | 0 | 0 |
| Braintrust | 28 | 0 | 0 | 28 (100%) | 0 | 0 | 0 |
| All | 2 783 | 161 (6%) | 13 (0%) | 2 107 (76%) | 57 (2%) | 381 (14%) | 64 (2%) |

Entities per source (rows containing the entity; whole table, section 3 of the counts output): Paradigm 67 rows (`&gt;` 60, `&lt;` 60, `&amp;` 32); Web3 Careers 44 (`&amp;` 44, `&gt;` 13); Framework ventures 27 (`&amp;` 27); We Work Remotely 21 (`&amp;` 16, `&nbsp;` 12, `&ndash;` 1); Multicoin capital 6 (`&amp;` 5, `&gt;` 1); Electric Capital 2 (`&amp;` 2); Wellfound 1 (`&L;`, page text). Every other source has none.

Markdown and page chrome inside stored text (section 4b of the counts output): Wellfound 171 rows with `# ` heading lines, 276 with `**bold**`, 103 with `[text](https://…)` links, all rendered literally today; We Work Remotely 221 rows carry page chrome from the whole page body ("× All jobs …", "Posted 1 day ago"); Remote3 40 rows hold the text of a category page, not of a vacancy ("Public Relations (PR) Jobs (97 new in Sep 2025) … Stop applying — get discovered by hiring agents").

Structure hints inside plain text, useful for L.3 (section 4 of the counts output; rows with a bullet line, a numbered line, a short line ending with a colon): Paradigm 292 / 9 / 478; Wellfound 299 / 13 / 105; We Work Remotely 19 / 5 / 160; Remotive 72 / 1 / 113; Hacker News 59 / 2 / 56; Braintrust 10 / 1 / 13; all sources together 753 / 32 / 995. Numbered lines are rare, bullet lines exist in about a third of the rows that have newlines, colon lines are common.

## 4. Table B: how each source writes `jobs.description`

The writers are the legacy TypeScript adapters in `legacy/app/src/discovery.ts` (the staging data was collected by them) and the Go ports in `backend/internal/collect` and `backend/internal/source`. Every path that reaches the database goes through one flattening function (`cleanHtml` in TypeScript, `collect.CleanHTML` in Go) except the cases marked below.

| Source | Where the text comes from (status in the database) | Writer | Passes through the flattener |
|---|---|---|---|
| Paradigm | linked employer page: Ashby / Greenhouse / Lever JSON-LD or body ("available structured description" 497, "available cleaned html" 280), LinkedIn pages (the 80 html rows), 58 empty ("pending employer description", detail errors) | detail hydration, JSON-LD first, page body second | yes, but see section 5 point 1 |
| Wellfound | public search API description, markdown ("available public search description" 300); JSON-LD of the job page (99) | TypeScript `plain()` for the search text | yes; the markdown stays as markdown |
| We Work Remotely | page body ("available cleaned html" 222); RSS description (13); JSON-LD of linked pages (27 html) | feed parser, page body cleaner | yes |
| Remotive | JSON-LD of the job page (152), page body (73), public API (1) | detail hydration | yes |
| Multicoin capital, Electric Capital, Framework ventures | Getro listing carries only skills; the text comes from the linked employer page (Ashby, Lever, Greenhouse, LinkedIn) or the Getro job page JSON-LD | detail hydration | yes; skills-only rows stay as a comma-separated list |
| A16z | embedded portfolio dataset carries only skills; text from the linked employer page | detail hydration | yes |
| Hacker News | official API comment text, HTML with `<p>` separators and `<a>` links | TypeScript `plain()` | yes |
| Web3 Careers | JSON-LD of the job page | structured-job parser | yes, but the JSON-LD string is already flattened and damaged at the source (section 5 point 4) |
| Cryptocurrency Jobs | JSON-LD of the job page; skills only for the listing | structured-job parser | yes |
| Remote3 | page body of a category page (40) or skills line | page body cleaner | yes, but the page is the wrong one (section 5 point 5) |
| CryptoJobList | JSON-LD of the job page | structured-job parser | yes |
| Braintrust | public API description | TypeScript `plain()` | yes |

No stored vacancy has the status "jobspy description"; the only code path that stores raw HTML unmodified (`backend/workers/jobspy/index.mjs`, which copies `description` as is) has produced no rows in this database. The 161 vacancies with literal tags do not come from a path that skips the flattener (section 5 point 1).

## 5. Findings

1. The 161 vacancies with literal `<p>`, `<li>` or `<br>` come from JSON-LD descriptions that are themselves entity-escaped HTML (`&lt;p&gt;…`). LinkedIn pages show this: the page for vacancy 1417 delivers `&lt;p&gt;` in its JSON-LD, and the stored text starts with a literal `<p>`. The flattener parses the string as HTML, decodes the entities once and so produces text that contains tags. The same happens for the double-escaped `&lt;` and `&gt;` in the 60 rows that also hold a leftover `&lt;https://…&gt;` after a privacy-policy sentence. Where the origin is a plain-HTML JSON-LD (Ashby, Wellfound, Remotive, Getro) the flattener works as designed and the text is clean but without structure. By source the literal-tag rows are Paradigm 80, Framework 27, We Work Remotely 27, Multicoin 20, Electric 5, Web3 Careers 2.
2. The 126 rows with `&amp;` are the same double decoding problem in the other direction: the text was escaped twice at the source and decoded once. Paradigm 32, Web3 Careers 44, Framework 27, We Work Remotely 16, Multicoin 5, Electric 2. The 12 `&nbsp;` rows are all We Work Remotely.
3. Plain text with newlines is 76% of all rows and the dominant kind in 12 of 14 sources. It has paragraph breaks and, for roughly a third of the rows, bullet characters that were in the source text. It has no headings, no list markers produced from `<li>` and no numbering; headings survive only as a short line, often ending with a colon, which is the only cue L.3 can use. Page markup is flattened by one rule (a newline before p, br, div, li, h1-h3, section), so a heading, a paragraph and a list item look the same.
4. Web3 Careers is the damaged source. The JSON-LD description of the job page is a single flat string, the apostrophes are already gone ("You ll be the second person in BCB s data function"), `&` appears as `&amp;amp;`, and markup survived only as the remains `strong&amp;gt;About CoinList br&amp;gt; br&amp;gt; /strong&amp;gt;`. Of 132 vacancies, 46 are glued and 13 are mangled escaped markup (59 rows, 45%) and cannot be turned into structure without fetching again. The page body itself carries real markup (vacancy 28: 22 `<p>`, 11 `<ul>`, 51 `<li>`; vacancy 27: h3, h4, p, ul, li, br), so the refetch must read the body and not the JSON-LD.
5. Remote3: all 40 stored descriptions are the text of a category page (for example "Public Relations (PR) Jobs"), not of a vacancy. The vacancy page itself (vacancy 7) delivers h2, p, ul, li and strong but no JSON-LD, and the stored text for that vacancy is a skills line. This is a collection defect, not a markup one, and a refetch of the same address would give the same page.
6. We Work Remotely: 221 of the 222 body-cleaned rows contain the whole page chrome ("× All jobs Sales and Marketing …", "Posted 1 day ago"). The flattener takes the entire body. The RSS feed delivers the description as entity-escaped HTML with headings, paragraphs and lists.
7. Wellfound: 300 rows hold markdown from the search API. The current line renderer prints `# About Discernis`, `**bold**` and `[text](url)` literally in 171, 276 and 103 rows. The job page itself delivers HTML (h3, p, ul, ol, li, strong) in its JSON-LD.
8. Metadata-only rows are 381 (14%): A16z 124 of 172, Multicoin 122 of 192, Web3 Careers 65 of 132, Electric Capital 57 of 141. They hold a comma-separated list of skills or a short placeholder and are not listings; they have nothing to structure until the detail fetch has run. 64 more rows are empty.
9. Short real texts without newlines exist (Cryptocurrency Jobs: 4 rows of about 190 characters; vacancy 1343 is one paragraph of 187 characters) and are legitimate single paragraphs.

## 6. Table C: what the page delivers against what is stored

For each source the fetched pages (at most three) are named by vacancy number. "Delivers" is what the response contains; "stored" is the dominant kind in Table A.

| Source | Pages fetched (result) | What the page delivers | Stored | Richer than stored |
|---|---|---|---|---|
| Paradigm | Ashby page 9 (200), LinkedIn 8 (200), LinkedIn 1417 (200) | Ashby: JSON-LD description as raw HTML, 60 p, 22 ul, 30 li, 9 strong, 1 em, no h tags (the Ashby page body is a script-rendered shell, the JSON-LD is the only markup). LinkedIn: JSON-LD escaped HTML (11-13 p, 2-3 ul, 12-24 li, strong, br); the page body also has h2 and h3 | newlines 83%, html 10%, empty 7% | yes: lists, bold lead-ins, paragraphs |
| Wellfound | 13 (200), 2610 (404, listing gone), 2625 (200) | JSON-LD description raw HTML: h3 9, p 5-17, ul 3-5, ol 1, li 20-25, strong; the body repeats it with h1, h2 | markdown or plain newlines | yes: real headings and a numbered list |
| We Work Remotely | RSS feed (200, 100 items), page 2656 (403, Cloudflare "Just a moment", not retried), employer page 2168 (200) | RSS description entity-escaped HTML: h3 5, p 29, ul 13, li 40, strong 6, br 3 for the item with a full description; the job pages are blocked for a plain client | newlines 89%, chrome in 221 rows | yes (RSS only; job pages are blocked) |
| Remotive | page 1 (200), page 2676 (200), API (200, the `limit=2` parameter was ignored and the full 234 KB window came back) | JSON-LD and API description raw HTML: p 11-33 (up to 79 in the API), ul 2-9, ol 1, li 17-38, strong, em | newlines 100% | yes: lists, numbering, bold |
| Multicoin capital | 622 (200), 625 (200), 639 (200) | Getro job page: JSON-LD and `__NEXT_DATA__` currentJob.description raw HTML. Paragraphs are `<br>` (44 br against 0-5 p), ul 4, li 18, strong 5; vacancy 639 has an HTML description (h2, p) while the database holds only skills | newlines 24%, html 10%, metadata 64% | yes, and the 122 skill-only rows can be filled from the page |
| A16z | list page (200, 883 KB, embedded dataset, no per-job text), Ashby 450 (200), Coinbase 545 (403, Cloudflare, not retried) | Ashby JSON-LD raw HTML: h2 7, p 37, ul 5, li 31, br 6, strong 14; the list page carries only titles and skills | newlines 22%, metadata 72% | yes for the employer pages; the board page itself adds nothing |
| Hacker News | API items 16, 1275, 1276 (200) | comment text as raw HTML: `<p>` separators (6-10) and `<a href>` links (3 and 5 in two of the three), no lists, no headings | newlines 100% | little: paragraphs already kept; links lost |
| Electric Capital | 368 (200), Ashby 10 (200), 410 (200) | 368: Getro JSON-LD and `__NEXT_DATA__` raw HTML, br 34, ul 3, li 15, strong 6, em 10; Ashby 10: h2 4, p 58, ul 5, li 37; 410: JSON-LD description is 28 characters (the title), nothing more exists | newlines 56%, metadata 40% | yes for 368 and 10; none for 410 |
| Web3 Careers | 27, 28, 75 (all 200) | JSON-LD description: one flat string with apostrophes lost and `&amp;amp;`, markup remains `strong&amp;gt;`; page body: h1, h3 16, h4 10, p 23-40, ul 11-19, li 51-80, br | glued 35%, escaped 10%, metadata 49% | yes, only through the page body, never the JSON-LD |
| Framework ventures | 24, 732, 805 (all 200) | same Getro page shape: br 28-36, ul 3-4, li 10-17, strong 7-9, em 2, h2 2 in the body | newlines 61%, html 30%, metadata 8% | yes (`<br>` paragraphs and lists) |
| Cryptocurrency Jobs | 21, 1343, 1375 (all 200) | JSON-LD description raw HTML: h2 6, p 5-6, ul 2-3, ol 1, li 14-25, strong 3-7; vacancy 1343 has one short p | newlines 94% | yes: headings, a numbered list |
| Remote3 | 1376 and 1415 (200, category pages: h1, 10 p), 7 (200, vacancy page: h2 7, p 23, ul 5, li 32, strong 5; no JSON-LD) | the stored 40 rows are category pages (see finding 5); a vacancy page has real markup | newlines 98% (wrong page) | yes, if the right page is fetched |
| CryptoJobList | all three 403 (Cloudflare "Just a moment", not retried) | blocked for a plain client; markup unknown | newlines 100% | unknown (blocked) |
| Braintrust | API jobs 159, 185, 186 (200) | API description raw HTML: p 21-60, br 3-7, strong 5-7, b 5-7; no lists, no headings | newlines 100% | little: paragraphs and bold |

## 7. Table D: three stored samples per source (first characters; the verbatim 240-character quotes are in section 7 of `inventory_counts_output.txt`)

| Source | Sample vacancy and kind | What it shows |
|---|---|---|
| Paradigm | 9 newlines; 8 html; 238 metadata | "About fab2" heading line then paragraphs, no list markers; `<p>AMBR is building…</p><p><br></p>` literal tags; skills line "Smart Contracts, Solidity, …" |
| Wellfound | 13 newlines (markdown); 2610 glued; 2625 newlines | `# About Discernis` markdown heading; one 431-character paragraph; plain paragraphs |
| We Work Remotely | 5 newlines (feed); 2168 html; 2656 newlines | "Headquarters: New York, NY" label lines; `<p>Account Executive … <strong>About US<br><br></strong>` literal tags; "× All jobs Sales and Marketing Director of Affiliates … Posted 1 day ago" page chrome |
| Remotive | 1, 2677, 2676 newlines | "Oct 27, 2026 - Lemon.io is hiring…" paragraphs; "Role Description" label line with an indented first word |
| Multicoin capital | 639 metadata; 625 newlines; 622 html | "Blockchain, Ethereum, Digital Asset Management"; "About Sei Labs" with single newlines; `<strong>Hadrian - Manufacturing the Future<br><br></strong>` literal tags |
| A16z | 453 metadata; 450 newlines; 545 glued | skills list after a 404; "TL;DR" with paragraphs; a 419-character skills list with no newline |
| Hacker News | 16, 1276, 1275 newlines | first line "Company | url | place | terms", paragraphs, "- " bullets in some |
| Electric Capital | 10 newlines; 410 metadata; 368 html | plain paragraphs; "Engineering Manager, Product" (title only); literal tags |
| Web3 Careers | 75 metadata; 28 glued; 27 escaped | "project manager non tech senior +1" tags line; "The Role You ll be the second person in BCB s data function … Data &amp; Analytics" with no newline; `strong&gt;About CoinList br&gt; br&gt; /strong&gt;` |
| Framework ventures | 24 newlines; 732 html; 805 metadata | "About Chainlink" paragraphs; `<strong>About Chainlink<br><br></strong>` literal tags; "Computer Science, Algorithms, …" skills |
| Cryptocurrency Jobs | 21 newlines; 1343 metadata; 1375 newlines | "Role Overview" label line then paragraphs; one 187-character paragraph; "About Us" label line |
| Remote3 | 1376 newlines; 7 metadata; 1415 newlines | "Public Relations (PR) Jobs (97 new in Sep 2025) ✌ Discover top …" category page text; "at Ondo Finance - Full-Time - Worldwide"; the same category text |
| CryptoJobList | 1315, 2605, 2421 newlines | paragraphs and "Responsibilities" label lines; non-breaking spaces (`\xa0`) inside text |
| Braintrust | 159, 186, 185 newlines | paragraphs; "Role Overview" label line; "AI Build Consultant:" colon lines; en dashes |

## 8. What this means for L.3 (executor's reading, for the coordinator to decide)

- Locally recoverable structure: plain text with newlines (2 107 rows), the 161 literal-tag rows (real tags, decode with the converter) and, with a decode pass, the 13 mangled escaped rows only partly (the opening bracket of every tag is gone, so only the shape `tag&gt;` can be guessed).
- Not recoverable without the network: the 46 glued rows (Web3 Careers, whose JSON-LD is damaged, so the refetch must parse the page body), the 13 mangled rows, the 381 metadata rows (nothing to structure), the 64 empty rows, the Remote3 category-page rows (wrong page), and the 221 We Work Remotely rows with page chrome (the chrome can be stripped heuristically from the text but cannot be separated from the vacancy reliably).
- Sources whose job pages cannot be refetched with a plain client today: We Work Remotely (Cloudflare 403; the RSS feed still works for the current 100 items), CryptoJobList (Cloudflare 403), Coinbase pages (403), LinkedIn pages answered 200 and carried the JSON-LD description, but this is a snapshot of two requests and LinkedIn is known to throttle.
- The Wellfound markdown (300 rows, 171 with headings) is a fourth input form that the planned converter does not mention; it needs its own branch in L.3 (`#` lines to headings, `**` to strong, `[text](url)` to text).
- Counts that differ from the plan: vacancies 2 783 against 2 780. All other numbers reproduce.
